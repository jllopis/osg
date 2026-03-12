package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/render"
)

const buildCacheVersion = 3

type buildCache struct {
	Version       int                  `json:"version"`
	ConfigHash    string               `json:"config_hash"`
	TemplatesHash string               `json:"templates_hash"`
	Templates     map[string]string    `json:"templates,omitempty"`
	AssetsHash    string               `json:"assets_hash"`
	StaticHash    string               `json:"static_hash,omitempty"`
	SassHash      string               `json:"sass_hash,omitempty"`
	PluginsHash   string               `json:"plugins_hash"`
	Content       map[string]fileStamp `json:"content"`
	Outputs       map[string]string    `json:"outputs"`
	// Dependency graph for smart incremental builds.
	PageTemplates map[string]string   `json:"page_templates,omitempty"` // source_path -> template_name
	SectionPages  map[string][]string `json:"section_pages,omitempty"`  // section_path -> [source_paths]
	GeneratedAt   string              `json:"generated_at"`
}

type fileStamp struct {
	ModTime int64 `json:"mod_time"`
	Size    int64 `json:"size"`
}

type buildConfigSignature struct {
	BaseURL        string                  `json:"base_url"`
	Theme          string                  `json:"theme"`
	ContentDir     string                  `json:"content_dir"`
	PublicDir      string                  `json:"public_dir"`
	TemplatesDir   string                  `json:"templates_dir"`
	StaticDir      string                  `json:"static_dir"`
	ThemesDir      string                  `json:"themes_dir"`
	PluginsDir     string                  `json:"plugins_dir"`
	PluginsEnabled []string                `json:"plugins_enabled"`
	SassDir        string                  `json:"sass_dir"`
	IncludeDrafts  bool                    `json:"include_drafts"`
	CompileSass    bool                    `json:"compile_sass"`
	CleanPublic    bool                    `json:"clean_public"`
	Taxonomies     []config.TaxonomyConfig `json:"taxonomies"`
}

func buildCacheFrom(cfg config.Config, files []string) (*buildCache, error) {
	configHash, err := hashConfig(cfg)
	if err != nil {
		return nil, err
	}
	templatesHash, err := hashTemplates(cfg)
	if err != nil {
		return nil, err
	}
	perTemplateHashes, err := hashTemplatesPerFile(cfg)
	if err != nil {
		return nil, err
	}
	assetsHash, err := hashAssets(cfg)
	if err != nil {
		return nil, err
	}
	staticHash, err := hashStatic(cfg)
	if err != nil {
		return nil, err
	}
	sassHash, err := hashSass(cfg)
	if err != nil {
		return nil, err
	}
	pluginsHash, err := hashPlugins(cfg)
	if err != nil {
		return nil, err
	}

	content, err := hashContent(files)
	if err != nil {
		return nil, err
	}

	return &buildCache{
		Version:       buildCacheVersion,
		ConfigHash:    configHash,
		TemplatesHash: templatesHash,
		Templates:     perTemplateHashes,
		AssetsHash:    assetsHash,
		StaticHash:    staticHash,
		SassHash:      sassHash,
		PluginsHash:   pluginsHash,
		Content:       content,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func buildCachePath(cfg config.Config) string {
	dir := strings.TrimSpace(cfg.BuildCacheDir)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "build.json")
}

func loadBuildCache(path string) (*buildCache, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cache buildCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func saveBuildCache(path string, cache *buildCache) error {
	if cache == nil || strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func diffContent(prev map[string]fileStamp, current map[string]fileStamp) (map[string]bool, []string) {
	changed := map[string]bool{}
	if prev == nil {
		for path := range current {
			changed[path] = true
		}
		return changed, nil
	}

	removed := []string{}
	for path, stamp := range current {
		if prevStamp, ok := prev[path]; !ok || prevStamp != stamp {
			changed[path] = true
		}
	}
	for path := range prev {
		if _, ok := current[path]; !ok {
			removed = append(removed, path)
		}
	}
	return changed, removed
}

func hashConfig(cfg config.Config) (string, error) {
	signature := buildConfigSignature{
		BaseURL:        cfg.BaseURL,
		Theme:          cfg.Theme,
		ContentDir:     cfg.ContentDir,
		PublicDir:      cfg.PublicDir,
		TemplatesDir:   cfg.TemplatesDir,
		StaticDir:      cfg.StaticDir,
		ThemesDir:      cfg.ThemesDir,
		PluginsDir:     cfg.PluginsDir,
		PluginsEnabled: cfg.PluginsEnabled,
		SassDir:        cfg.SassDir,
		IncludeDrafts:  cfg.IncludeDrafts,
		CompileSass:    cfg.CompileSass,
		CleanPublic:    cfg.CleanPublic,
		Taxonomies:     cfg.Taxonomies,
	}
	data, err := json.Marshal(signature)
	if err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

func hashTemplates(cfg config.Config) (string, error) {
	hashes := []string{}
	builtins, err := render.BuiltinsSignature()
	if err != nil {
		return "", err
	}
	hashes = append(hashes, builtins)

	userSig, err := hashDir(cfg.TemplatesDir, isTemplateFile)
	if err != nil {
		return "", err
	}
	themeSig, err := hashDir(themeTemplatesDir(cfg), isTemplateFile)
	if err != nil {
		return "", err
	}
	hashes = append(hashes, userSig, themeSig)
	return hashStrings(hashes), nil
}

// hashTemplatesPerFile returns a map of relative template path -> hash.
// This allows diffing which individual templates changed between builds.
func hashTemplatesPerFile(cfg config.Config) (map[string]string, error) {
	result := make(map[string]string)
	dirs := []string{
		cfg.TemplatesDir,
		themeTemplatesDir(cfg),
	}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if shouldSkipDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isTemplateFile(path, d) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(rel)
			result[key] = hashBytes(data) // user templates override theme
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// hashStatic hashes only static assets (not sass).
func hashStatic(cfg config.Config) (string, error) {
	hashes := []string{}
	s1, err := hashDir(cfg.StaticDir, includeAllFiles)
	if err != nil {
		return "", err
	}
	s2, err := hashDir(filepath.Join(cfg.ThemesDir, cfg.Theme, "static"), includeAllFiles)
	if err != nil {
		return "", err
	}
	hashes = append(hashes, s1, s2)
	return hashStrings(hashes), nil
}

// hashSass hashes only sass/scss assets.
func hashSass(cfg config.Config) (string, error) {
	hashes := []string{}
	s1, err := hashDir(cfg.SassDir, includeAllFiles)
	if err != nil {
		return "", err
	}
	s2, err := hashDir(filepath.Join(cfg.ThemesDir, cfg.Theme, "sass"), includeAllFiles)
	if err != nil {
		return "", err
	}
	hashes = append(hashes, s1, s2)
	return hashStrings(hashes), nil
}

func hashAssets(cfg config.Config) (string, error) {
	hashes := []string{}
	staticSig, err := hashDir(cfg.StaticDir, includeAllFiles)
	if err != nil {
		return "", err
	}
	themeStaticSig, err := hashDir(filepath.Join(cfg.ThemesDir, cfg.Theme, "static"), includeAllFiles)
	if err != nil {
		return "", err
	}
	sassSig, err := hashDir(cfg.SassDir, includeAllFiles)
	if err != nil {
		return "", err
	}
	themeSassSig, err := hashDir(filepath.Join(cfg.ThemesDir, cfg.Theme, "sass"), includeAllFiles)
	if err != nil {
		return "", err
	}
	hashes = append(hashes, staticSig, themeStaticSig, sassSig, themeSassSig)
	return hashStrings(hashes), nil
}

func hashPlugins(cfg config.Config) (string, error) {
	return hashDir(cfg.PluginsDir, func(path string, d fs.DirEntry) bool {
		if d.IsDir() {
			return false
		}
		return strings.HasSuffix(strings.ToLower(d.Name()), ".wasm")
	})
}

func hashContent(files []string) (map[string]fileStamp, error) {
	out := make(map[string]fileStamp, len(files))
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		out[path] = fileStamp{
			ModTime: info.ModTime().UnixNano(),
			Size:    info.Size(),
		}
	}
	return out, nil
}

func hashDir(root string, include func(string, fs.DirEntry) bool) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !info.IsDir() {
		return "", nil
	}

	type entry struct {
		Path    string
		ModTime int64
		Size    int64
	}
	entries := []entry{}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if include != nil && !include(path, d) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, entry{
			Path:    filepath.ToSlash(rel),
			ModTime: info.ModTime().UnixNano(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	hasher := sha256.New()
	for _, entry := range entries {
		_, _ = fmt.Fprintf(hasher, "%s|%d|%d\n", entry.Path, entry.ModTime, entry.Size)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashStrings(values []string) string {
	hasher := sha256.New()
	for _, value := range values {
		_, _ = io.WriteString(hasher, value)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func isTemplateFile(path string, _ fs.DirEntry) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".xml", ".txt":
		return true
	default:
		return false
	}
}

func includeAllFiles(_ string, d fs.DirEntry) bool {
	return !d.IsDir()
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".")
}
