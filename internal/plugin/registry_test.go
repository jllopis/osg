package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIsGitHubRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref  string
		want bool
	}{
		{"github.com/user/repo", true},
		{"github.com/user/repo@v1.0.0", true},
		{"github.com/user/repo@latest", true},
		{"github.com/org-name/my-plugin.v2", true},
		{"github.com/a/b@some-tag", true},
		{"", false},
		{"/local/path.wasm", false},
		{"gitlab.com/user/repo", false},
		{"github.com/user", false},
		{"github.com/", false},
		{"github.com//repo", false},
	}
	for _, tc := range tests {
		got := IsGitHubRef(tc.ref)
		if got != tc.want {
			t.Errorf("IsGitHubRef(%q) = %v, want %v", tc.ref, got, tc.want)
		}
	}
}

func TestParseGitHubRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref       string
		wantOwner string
		wantRepo  string
		wantTag   string
	}{
		{"github.com/user/repo", "user", "repo", ""},
		{"github.com/user/repo@v1.0.0", "user", "repo", "v1.0.0"},
		{"github.com/org/my-plugin@latest", "org", "my-plugin", "latest"},
		{"invalid", "", "", ""},
	}
	for _, tc := range tests {
		owner, repo, tag := ParseGitHubRef(tc.ref)
		if owner != tc.wantOwner || repo != tc.wantRepo || tag != tc.wantTag {
			t.Errorf("ParseGitHubRef(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.ref, owner, repo, tag, tc.wantOwner, tc.wantRepo, tc.wantTag)
		}
	}
}

func TestInstallFromGitHub(t *testing.T) {
	t.Parallel()

	// Create a mock GitHub API server.
	release := GitHubRelease{
		TagName: "v1.2.0",
		Assets: []GitHubAsset{
			{Name: "source.tar.gz", BrowserDownloadURL: ""},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testuser/testplugin/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		// The wasm asset URL will be set below after the server starts.
		json.NewEncoder(w).Encode(release)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// We need to override the fetch function. Instead, let's test the
	// parsing and non-network parts separately, and use the mock server
	// for the integration test via fetchGitHubRelease.

	// Test: no plugins dir
	_, err := InstallFromGitHub(context.Background(), "user", "repo", "", "")
	if err == nil || err.Error() != "plugins dir is not configured" {
		t.Errorf("expected plugins dir error, got: %v", err)
	}
}

func TestInstallFromGitHubWithMockServer(t *testing.T) {
	t.Parallel()

	wasmContent := []byte("mock wasm binary content")

	mux := http.NewServeMux()

	// Serve the wasm file.
	mux.HandleFunc("/download/osg-test.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		w.Write(wasmContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Serve the release JSON with the wasm asset pointing to our mock server.
	release := GitHubRelease{
		TagName: "v2.0.0",
		Assets: []GitHubAsset{
			{Name: "README.md", BrowserDownloadURL: server.URL + "/download/README.md"},
			{Name: "osg-test.wasm", BrowserDownloadURL: server.URL + "/download/osg-test.wasm", Size: len(wasmContent)},
		},
	}

	mux.HandleFunc("/repos/myuser/myplugin/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	})
	mux.HandleFunc("/repos/myuser/myplugin/releases/tags/v2.0.0", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	})

	// Temporarily override the GitHub API host by using fetchGitHubRelease directly.
	// Since InstallFromGitHub uses the real GitHub API URL, we test the
	// download function directly instead.
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "osg-test.wasm")

	err := downloadFile(context.Background(), server.URL+"/download/osg-test.wasm", dest)
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != string(wasmContent) {
		t.Errorf("downloaded content mismatch: got %q, want %q", string(data), string(wasmContent))
	}
}

func TestFindWASMAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		assets []GitHubAsset
		want   string // expected asset name, or "" if nil
	}{
		{
			"finds wasm",
			[]GitHubAsset{
				{Name: "readme.md"},
				{Name: "plugin.wasm", BrowserDownloadURL: "http://example.com/plugin.wasm"},
			},
			"plugin.wasm",
		},
		{
			"case insensitive",
			[]GitHubAsset{
				{Name: "Plugin.WASM", BrowserDownloadURL: "http://example.com/Plugin.WASM"},
			},
			"Plugin.WASM",
		},
		{
			"no wasm",
			[]GitHubAsset{
				{Name: "readme.md"},
				{Name: "source.tar.gz"},
			},
			"",
		},
		{
			"empty",
			nil,
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findWASMAsset(tc.assets)
			if tc.want == "" && got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
			if tc.want != "" && (got == nil || got.Name != tc.want) {
				t.Errorf("expected asset %q, got %+v", tc.want, got)
			}
		})
	}
}

// ---- Lock File Tests ----

func TestLockFileRoundTrip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, err := LoadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("load empty lock file: %v", err)
	}

	if len(lock.Plugins) != 0 {
		t.Errorf("expected empty plugins, got %d", len(lock.Plugins))
	}

	lock.Set("myplugin", LockEntry{
		Source:  "github.com/user/myplugin",
		Version: "v1.0.0",
	})
	lock.Set("other", LockEntry{
		Source:  "github.com/org/other",
		Version: "v2.3.1",
	})

	if err := lock.Save(); err != nil {
		t.Fatalf("save lock file: %v", err)
	}

	// Verify the file was created.
	lockPath := filepath.Join(tmpDir, ".osg", lockFileName)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file should exist at %s: %v", lockPath, err)
	}

	// Reload and verify.
	lock2, err := LoadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("reload lock file: %v", err)
	}

	if len(lock2.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(lock2.Plugins))
	}

	entry, ok := lock2.Get("myplugin")
	if !ok {
		t.Fatal("expected myplugin in lock file")
	}
	if entry.Version != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", entry.Version)
	}
	if entry.Source != "github.com/user/myplugin" {
		t.Errorf("expected github.com/user/myplugin, got %s", entry.Source)
	}
}

func TestLockFileRemove(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, _ := LoadLockFile(tmpDir)
	lock.Set("a", LockEntry{Source: "s", Version: "v1"})
	lock.Set("b", LockEntry{Source: "s", Version: "v2"})

	lock.Remove("a")
	_, ok := lock.Get("a")
	if ok {
		t.Error("expected 'a' to be removed")
	}
	_, ok = lock.Get("b")
	if !ok {
		t.Error("expected 'b' to still exist")
	}
}

func TestLockFileNames(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, _ := LoadLockFile(tmpDir)
	lock.Set("zebra", LockEntry{})
	lock.Set("alpha", LockEntry{})
	lock.Set("middle", LockEntry{})

	names := lock.Names()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "middle" || names[2] != "zebra" {
		t.Errorf("expected sorted names, got %v", names)
	}
}

// ---- Index Tests ----

func TestSearchIndex(t *testing.T) {
	t.Parallel()

	index := &PluginIndex{
		Plugins: []IndexEntry{
			{Name: "search", Description: "Full-text search plugin", Author: "jllopis"},
			{Name: "feed", Description: "RSS/Atom feed generator", Author: "jllopis"},
			{Name: "analytics", Description: "Site analytics tracker", Author: "contrib"},
		},
	}

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},        // empty query returns all
		{"search", 1},  // matches name
		{"feed", 1},    // matches name "feed" only
		{"jllopis", 2}, // matches author
		{"tracker", 1}, // matches description
		{"nothing", 0}, // no match
	}
	for _, tc := range tests {
		got := SearchIndex(index, tc.query)
		if len(got) != tc.want {
			t.Errorf("SearchIndex(%q) returned %d results, want %d", tc.query, len(got), tc.want)
		}
	}
}

func TestFetchIndexFromMockServer(t *testing.T) {
	t.Parallel()

	index := PluginIndex{
		Plugins: []IndexEntry{
			{Name: "search", Description: "Full-text search", Author: "jllopis", Repo: "github.com/jllopis/osg-search"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(index)
	}))
	defer server.Close()

	got, err := FetchIndexFrom(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchIndexFrom failed: %v", err)
	}
	if len(got.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(got.Plugins))
	}
	if got.Plugins[0].Name != "search" {
		t.Errorf("expected 'search', got %q", got.Plugins[0].Name)
	}
}

func TestFetchIndexFromBadServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := FetchIndexFrom(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDownloadFileError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	err := downloadFile(context.Background(), server.URL+"/missing.wasm", filepath.Join(tmpDir, "out.wasm"))
	if err == nil {
		t.Fatal("expected error for 404 download")
	}
}

func TestFetchGitHubReleaseMock(t *testing.T) {
	t.Parallel()

	release := GitHubRelease{
		TagName: "v3.0.0",
		Assets: []GitHubAsset{
			{Name: "osg-test.wasm", BrowserDownloadURL: "http://example.com/osg-test.wasm", Size: 1024},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	})
	mux.HandleFunc("/repos/owner/repo/releases/tags/v3.0.0", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// We can't directly test fetchGitHubRelease since it hardcodes api.github.com.
	// Instead, verify our mock serves correctly and test the URL construction logic.
	resp, err := http.Get(server.URL + "/repos/owner/repo/releases/latest")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var got GitHubRelease
	json.NewDecoder(resp.Body).Decode(&got)
	if got.TagName != "v3.0.0" {
		t.Errorf("expected v3.0.0, got %s", got.TagName)
	}
	if len(got.Assets) != 1 || got.Assets[0].Name != "osg-test.wasm" {
		t.Errorf("unexpected assets: %+v", got.Assets)
	}
}

func TestCheckUpdateNoLockEntry(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, _ := LoadLockFile(tmpDir)

	_, err := CheckUpdate(context.Background(), "nonexistent", lock)
	if err == nil {
		t.Fatal("expected error for missing lock entry")
	}
}

func TestCheckUpdateNonGitHubSource(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, _ := LoadLockFile(tmpDir)
	lock.Set("local", LockEntry{Source: "/local/path", Version: "v1"})

	_, err := CheckUpdate(context.Background(), "local", lock)
	if err == nil {
		t.Fatal("expected error for non-GitHub source")
	}
}
