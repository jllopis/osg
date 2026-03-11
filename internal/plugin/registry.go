package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---- GitHub Release Install ----

// githubRepoPattern matches references like "github.com/user/repo" or
// "github.com/user/repo@v1.2.3".
var githubRepoPattern = regexp.MustCompile(
	`^github\.com/([a-zA-Z0-9._-]+)/([a-zA-Z0-9._-]+?)(?:@(.+))?$`,
)

// IsGitHubRef returns true if ref looks like a GitHub repository reference.
func IsGitHubRef(ref string) bool {
	return githubRepoPattern.MatchString(ref)
}

// ParseGitHubRef extracts owner, repo, and optional tag from a reference.
// Returns ("", "", "") if the format is invalid.
func ParseGitHubRef(ref string) (owner, repo, tag string) {
	m := githubRepoPattern.FindStringSubmatch(ref)
	if m == nil {
		return "", "", ""
	}
	owner = m[1]
	repo = m[2]
	if len(m) > 3 {
		tag = m[3]
	}
	return owner, repo, tag
}

// GitHubRelease represents a GitHub release with its assets.
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a downloadable file in a release.
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int    `json:"size"`
}

// InstallFromGitHub downloads a .wasm plugin from a GitHub release.
// If tag is empty, the latest release is used.
// Returns the installed plugin name (without .wasm extension).
func InstallFromGitHub(ctx context.Context, owner, repo, tag, pluginsDir string) (string, error) {
	if pluginsDir == "" {
		return "", fmt.Errorf("plugins dir is not configured")
	}

	release, err := fetchGitHubRelease(ctx, owner, repo, tag)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}

	asset := findWASMAsset(release.Assets)
	if asset == nil {
		return "", fmt.Errorf("no .wasm asset found in release %s of %s/%s", release.TagName, owner, repo)
	}

	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return "", fmt.Errorf("create plugins dir: %w", err)
	}

	dest := filepath.Join(pluginsDir, asset.Name)
	if err := downloadFile(ctx, asset.BrowserDownloadURL, dest); err != nil {
		return "", fmt.Errorf("download %s: %w", asset.Name, err)
	}

	name := strings.TrimSuffix(asset.Name, ".wasm")

	// Record in lock file.
	lock, _ := LoadLockFile(filepath.Dir(pluginsDir))
	lock.Set(name, LockEntry{
		Source:  fmt.Sprintf("github.com/%s/%s", owner, repo),
		Version: release.TagName,
	})
	_ = lock.Save()

	return name, nil
}

// githubAPIBase is the base URL for the GitHub API.
// Tests can override this to point at a mock server.
var githubAPIBase = "https://api.github.com"

func fetchGitHubRelease(ctx context.Context, owner, repo, tag string) (*GitHubRelease, error) {
	var url string
	if tag == "" || tag == "latest" {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, owner, repo)
	} else {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", githubAPIBase, owner, repo, tag)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "osg-plugin-installer")

	// Support GITHUB_TOKEN for rate-limited or private repos.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		if tag == "" || tag == "latest" {
			return nil, fmt.Errorf("no releases found for %s/%s", owner, repo)
		}
		return nil, fmt.Errorf("release %s not found for %s/%s", tag, owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}

func findWASMAsset(assets []GitHubAsset) *GitHubAsset {
	for i := range assets {
		if strings.HasSuffix(strings.ToLower(assets[i].Name), ".wasm") {
			return &assets[i]
		}
	}
	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "osg-plugin-installer")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, resp.Body); err != nil {
		// Clean up partial download.
		_ = os.Remove(dest)
		return err
	}
	return nil
}

// ---- Plugin Lock File ----

// LockEntry records the source and version of an installed plugin.
type LockEntry struct {
	Source  string `json:"source"`
	Version string `json:"version"`
}

// LockFile tracks installed plugin versions.
type LockFile struct {
	mu      sync.RWMutex
	path    string
	Plugins map[string]LockEntry `json:"plugins"`
}

const lockFileName = "plugins.lock.json"

// LoadLockFile reads the lock file from the .osg directory under baseDir.
// Returns an empty lock file if the file doesn't exist.
func LoadLockFile(baseDir string) (*LockFile, error) {
	lf := &LockFile{
		path:    filepath.Join(baseDir, ".osg", lockFileName),
		Plugins: make(map[string]LockEntry),
	}

	data, err := os.ReadFile(lf.path)
	if err != nil {
		if os.IsNotExist(err) {
			return lf, nil
		}
		return lf, err
	}

	if err := json.Unmarshal(data, lf); err != nil {
		return lf, fmt.Errorf("parse lock file: %w", err)
	}
	if lf.Plugins == nil {
		lf.Plugins = make(map[string]LockEntry)
	}
	return lf, nil
}

// Set records a plugin entry.
func (lf *LockFile) Set(name string, entry LockEntry) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	lf.Plugins[name] = entry
}

// Get returns the lock entry for a plugin and whether it exists.
func (lf *LockFile) Get(name string) (LockEntry, bool) {
	lf.mu.RLock()
	defer lf.mu.RUnlock()
	e, ok := lf.Plugins[name]
	return e, ok
}

// Remove deletes a plugin entry from the lock file.
func (lf *LockFile) Remove(name string) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	delete(lf.Plugins, name)
}

// Save writes the lock file to disk.
func (lf *LockFile) Save() error {
	lf.mu.RLock()
	defer lf.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(lf.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lf.path, data, 0o644)
}

// Names returns sorted plugin names.
func (lf *LockFile) Names() []string {
	lf.mu.RLock()
	defer lf.mu.RUnlock()
	names := make([]string, 0, len(lf.Plugins))
	for name := range lf.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ---- Curated Plugin Index ----

const defaultIndexURL = "https://raw.githubusercontent.com/jllopis/osg/master/plugins-index.json"

// IndexEntry describes a plugin in the curated index.
type IndexEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Repo        string `json:"repo"`
	Version     string `json:"version,omitempty"`
}

// PluginIndex is a list of curated plugins.
type PluginIndex struct {
	Plugins []IndexEntry `json:"plugins"`
}

// FetchIndex downloads the curated plugin index from the default URL.
func FetchIndex(ctx context.Context) (*PluginIndex, error) {
	return FetchIndexFrom(ctx, defaultIndexURL)
}

// FetchIndexFrom downloads the curated plugin index from a custom URL.
func FetchIndexFrom(ctx context.Context, url string) (*PluginIndex, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "osg-plugin-installer")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("index returned HTTP %d", resp.StatusCode)
	}

	var idx PluginIndex
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	return &idx, nil
}

// SearchIndex filters the index entries by a query string.
// The query is matched against name, description, and author (case-insensitive).
func SearchIndex(index *PluginIndex, query string) []IndexEntry {
	if query == "" {
		return index.Plugins
	}
	q := strings.ToLower(query)
	var results []IndexEntry
	for _, e := range index.Plugins {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Description), q) ||
			strings.Contains(strings.ToLower(e.Author), q) {
			results = append(results, e)
		}
	}
	return results
}

// ---- Plugin Update ----

// CheckUpdate compares the installed version (from lock file) against the
// latest GitHub release. Returns the new tag if an update is available,
// or "" if already up-to-date.
func CheckUpdate(ctx context.Context, name string, lock *LockFile) (string, error) {
	entry, ok := lock.Get(name)
	if !ok {
		return "", fmt.Errorf("plugin %q not in lock file (was it installed from GitHub?)", name)
	}
	if !IsGitHubRef(entry.Source) && !strings.HasPrefix(entry.Source, "github.com/") {
		return "", fmt.Errorf("plugin %q source is not a GitHub repo: %s", name, entry.Source)
	}

	// Parse the source as "github.com/owner/repo".
	source := entry.Source
	if !strings.HasPrefix(source, "github.com/") {
		return "", fmt.Errorf("unexpected source format: %s", source)
	}
	parts := strings.SplitN(strings.TrimPrefix(source, "github.com/"), "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected source format: %s", source)
	}
	owner, repo := parts[0], parts[1]

	release, err := fetchGitHubRelease(ctx, owner, repo, "latest")
	if err != nil {
		return "", err
	}

	if release.TagName == entry.Version {
		return "", nil
	}
	return release.TagName, nil
}

// UpdatePlugin downloads the latest version of a plugin from GitHub.
// Returns the new version tag.
func UpdatePlugin(ctx context.Context, name string, pluginsDir string, lock *LockFile) (string, error) {
	entry, ok := lock.Get(name)
	if !ok {
		return "", fmt.Errorf("plugin %q not in lock file", name)
	}

	source := entry.Source
	if !strings.HasPrefix(source, "github.com/") {
		return "", fmt.Errorf("plugin %q source is not a GitHub repo: %s", name, source)
	}
	parts := strings.SplitN(strings.TrimPrefix(source, "github.com/"), "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected source format: %s", source)
	}

	installedName, err := InstallFromGitHub(ctx, parts[0], parts[1], "", pluginsDir)
	if err != nil {
		return "", err
	}
	// Re-read the lock entry to get the new version.
	newEntry, _ := lock.Get(installedName)
	return newEntry.Version, nil
}

// ---- Auto-install Official Plugins ----

// EnsureOfficialPlugins checks each name in enabledPlugins. If a plugin
// is not present in pluginsDir and is listed in the curated plugin index,
// it is downloaded from the corresponding GitHub release.
// Bundled plugins (embedded in the binary) are skipped since
// EnsureBundledPlugins handles those.
func EnsureOfficialPlugins(ctx context.Context, pluginsDir string, enabledPlugins []string, logger *slog.Logger) error {
	if pluginsDir == "" || len(enabledPlugins) == 0 {
		return nil
	}

	// Build set of bundled plugin names to skip.
	bundled := make(map[string]bool, len(BundledPlugins))
	for _, name := range BundledPlugins {
		bundled[name] = true
	}

	// Determine which enabled plugins are missing from disk.
	var missing []string
	for _, name := range enabledPlugins {
		if bundled[name] {
			continue
		}
		wasmPath := filepath.Join(pluginsDir, name+".wasm")
		if _, err := os.Stat(wasmPath); err == nil {
			continue // already installed
		}
		missing = append(missing, name)
	}

	if len(missing) == 0 {
		return nil
	}

	// Fetch the curated index to resolve plugin repos.
	index, err := FetchIndex(ctx)
	if err != nil {
		logger.Warn("could not fetch plugin index, skipping auto-install", "error", err)
		return nil // non-fatal: network may be unavailable
	}

	// Build a name -> IndexEntry lookup.
	byName := make(map[string]IndexEntry, len(index.Plugins))
	for _, e := range index.Plugins {
		byName[e.Name] = e
	}

	for _, name := range missing {
		entry, ok := byName[name]
		if !ok {
			logger.Warn("plugin not found in official index, skipping", "name", name)
			continue
		}
		if entry.Repo == "" {
			logger.Warn("plugin has no repo in index, skipping", "name", name)
			continue
		}

		owner, repo, tag := ParseGitHubRef(entry.Repo)
		if owner == "" {
			logger.Warn("invalid repo reference in index", "name", name, "repo", entry.Repo)
			continue
		}

		logger.Info("installing plugin", "name", name, "source", entry.Repo)
		_, err := InstallFromGitHub(ctx, owner, repo, tag, pluginsDir)
		if err != nil {
			logger.Warn("failed to install plugin", "name", name, "error", err)
			continue // non-fatal: keep going with remaining plugins
		}
		logger.Info("installed plugin", "name", name)
	}

	return nil
}
