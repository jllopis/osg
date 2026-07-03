package plugin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// silentLogger returns a logger that discards output.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------------------------------------------------------------------------
// Emit / emitSingle
// ---------------------------------------------------------------------------

func TestEmit_NilPlugins(t *testing.T) {
	t.Parallel()
	// Manager with nil plugins slice (not just empty).
	m := &Manager{logger: silentLogger()}
	result := m.Emit(context.Background(), "test.event", map[string]any{"k": "v"})
	if result != nil {
		t.Errorf("expected nil result for nil plugins, got %v", result)
	}
}

func TestEmit_MultiplePlugins_AllFail(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	// Copy search.wasm as search2.wasm to get two plugins.
	src := filepath.Join(pluginsDir, "search.wasm")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read search.wasm: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "search2.wasm"), data, 0o644); err != nil {
		t.Fatalf("write search2.wasm: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search", "search2"}, 5, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.plugins))
	}

	// Send an event neither plugin handles.
	result := m.Emit(ctx, "unknown.event", map[string]any{"foo": "bar"})
	if result != nil {
		t.Errorf("expected nil for unhandled event with 2 plugins, got %v", result)
	}
}

func TestEmit_MultiplePlugins_BuildFinished(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	src := filepath.Join(pluginsDir, "search.wasm")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "search2.wasm"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	publicDir := filepath.Join(dir, "public")

	ctx := context.Background()
	m, err := LoadSandboxed(ctx, pluginsDir, publicDir, []string{"search", "search2"}, 0, silentLogger())
	if err != nil {
		t.Fatalf("LoadSandboxed: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.plugins))
	}

	pages := []any{
		map[string]any{
			"title":     "Coverage Page",
			"summary":   "Testing multi-plugin emit",
			"permalink": "/2025/01/01/cov/",
			"date":      "2025-01-01",
			"taxonomies": map[string]any{
				"tags": []any{"test"},
			},
		},
	}

	result := m.Emit(ctx, "build.finished", map[string]any{
		"config": map[string]any{"public_dir": publicDir},
		"site":   map[string]any{"pages": pages},
	})
	_ = result
}

func TestEmit_WithTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	publicDir := filepath.Join(dir, "public")

	ctx := context.Background()
	m, err := LoadSandboxed(ctx, pluginsDir, publicDir, []string{"search"}, 30, silentLogger())
	if err != nil {
		t.Fatalf("LoadSandboxed: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	result := m.Emit(ctx, "build.finished", map[string]any{
		"config": map[string]any{"public_dir": publicDir},
		"site":   map[string]any{"pages": []any{}},
	})
	_ = result
}

func TestEmit_NilLogger(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	result := m.Emit(ctx, "build.started", map[string]any{"k": "v"})
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// readPluginInfo
// ---------------------------------------------------------------------------

func TestReadPluginInfo_NoPluginInfoExport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(m.plugins))
	}

	info := m.plugins[0].info
	if info.Name != "search" {
		t.Errorf("expected name 'search', got %q", info.Name)
	}
	if info.Version != "" {
		t.Errorf("expected empty version, got %q", info.Version)
	}
}

// ---------------------------------------------------------------------------
// Call
// ---------------------------------------------------------------------------

func TestCall_SearchPlugin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	p := m.plugins[0]

	// Call with a valid event the plugin ignores.
	event := Event{Type: "build.started", Payload: map[string]any{"test": true}}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := p.Call(ctx, data)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	_ = resp

	// Call with build.finished.
	publicDir := filepath.Join(dir, "public")
	_ = os.MkdirAll(publicDir, 0o755)

	event2 := Event{Type: "build.finished", Payload: map[string]any{
		"config": map[string]any{"public_dir": publicDir},
		"site":   map[string]any{"pages": []any{}},
	}}
	data2, _ := json.Marshal(event2)

	resp2, err := p.Call(ctx, data2)
	if err != nil {
		t.Fatalf("Call build.finished: %v", err)
	}
	_ = resp2
}

// ---------------------------------------------------------------------------
// All tests that mutate githubAPIBase must run sequentially in this group.
// They are NOT parallel because they share a package-level variable.
// ---------------------------------------------------------------------------

func TestGitHubAPIDependentTests(t *testing.T) {
	// This parent test is NOT parallel. All subtests run sequentially.
	origBase := githubAPIBase
	t.Cleanup(func() { githubAPIBase = origBase })

	t.Run("FetchGitHubRelease_Latest", func(t *testing.T) {
		release := GitHubRelease{
			TagName: "v1.5.0",
			Assets: []GitHubAsset{
				{Name: "myplugin.wasm", BrowserDownloadURL: "http://example.com/myplugin.wasm", Size: 2048},
			},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		got, err := fetchGitHubRelease(context.Background(), "owner", "repo", "")
		if err != nil {
			t.Fatalf("fetchGitHubRelease latest: %v", err)
		}
		if got.TagName != "v1.5.0" {
			t.Errorf("expected v1.5.0, got %s", got.TagName)
		}
		if len(got.Assets) != 1 || got.Assets[0].Name != "myplugin.wasm" {
			t.Errorf("unexpected assets: %+v", got.Assets)
		}
	})

	t.Run("FetchGitHubRelease_SpecificTag", func(t *testing.T) {
		release := GitHubRelease{
			TagName: "v2.0.0",
			Assets:  []GitHubAsset{{Name: "plugin.wasm"}},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/tags/v2.0.0", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		got, err := fetchGitHubRelease(context.Background(), "owner", "repo", "v2.0.0")
		if err != nil {
			t.Fatalf("fetchGitHubRelease tag: %v", err)
		}
		if got.TagName != "v2.0.0" {
			t.Errorf("expected v2.0.0, got %s", got.TagName)
		}
	})

	t.Run("FetchGitHubRelease_LatestKeyword", func(t *testing.T) {
		release := GitHubRelease{TagName: "v9.0.0"}
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		got, err := fetchGitHubRelease(context.Background(), "o", "r", "latest")
		if err != nil {
			t.Fatalf("fetchGitHubRelease 'latest': %v", err)
		}
		if got.TagName != "v9.0.0" {
			t.Errorf("expected v9.0.0, got %s", got.TagName)
		}
	})

	t.Run("FetchGitHubRelease_NotFound_Latest", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := fetchGitHubRelease(context.Background(), "owner", "repo", "")
		if err == nil {
			t.Fatal("expected error for 404 latest")
		}
		if !strings.Contains(err.Error(), "no releases found") {
			t.Errorf("expected 'no releases found', got: %v", err)
		}
	})

	t.Run("FetchGitHubRelease_NotFound_Tag", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/tags/v999", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := fetchGitHubRelease(context.Background(), "owner", "repo", "v999")
		if err == nil {
			t.Fatal("expected error for 404 tag")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found', got: %v", err)
		}
	})

	t.Run("FetchGitHubRelease_ServerError", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := fetchGitHubRelease(context.Background(), "owner", "repo", "")
		if err == nil {
			t.Fatal("expected error for 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected status 500 in error, got: %v", err)
		}
	})

	t.Run("FetchGitHubRelease_BadJSON", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := fetchGitHubRelease(context.Background(), "owner", "repo", "")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "decode release") {
			t.Errorf("expected 'decode release' error, got: %v", err)
		}
	})

	t.Run("InstallFromGitHub_NoWasmAsset", func(t *testing.T) {
		release := GitHubRelease{
			TagName: "v1.0.0",
			Assets:  []GitHubAsset{{Name: "readme.md", BrowserDownloadURL: "http://example.com/readme.md"}},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := InstallFromGitHub(context.Background(), "owner", "repo", "", t.TempDir())
		if err == nil {
			t.Fatal("expected error for no wasm asset")
		}
		if !strings.Contains(err.Error(), "no .wasm asset") {
			t.Errorf("expected 'no .wasm asset', got: %v", err)
		}
	})

	t.Run("InstallFromGitHub_FullFlow", func(t *testing.T) {
		wasmContent := []byte("fake wasm binary for testing")

		mux := http.NewServeMux()
		mux.HandleFunc("/download/test-plugin.wasm", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(wasmContent)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		release := GitHubRelease{
			TagName: "v3.1.0",
			Assets: []GitHubAsset{
				{Name: "source.tar.gz", BrowserDownloadURL: server.URL + "/download/source.tar.gz"},
				{Name: "test-plugin.wasm", BrowserDownloadURL: server.URL + "/download/test-plugin.wasm", Size: len(wasmContent)},
			},
		}

		mux.HandleFunc("/repos/myowner/myrepo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		githubAPIBase = server.URL

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")

		name, err := InstallFromGitHub(context.Background(), "myowner", "myrepo", "", pluginsDir)
		if err != nil {
			t.Fatalf("InstallFromGitHub: %v", err)
		}
		if name != "test-plugin" {
			t.Errorf("expected 'test-plugin', got %q", name)
		}

		data, err := os.ReadFile(filepath.Join(pluginsDir, "test-plugin.wasm"))
		if err != nil {
			t.Fatalf("read downloaded wasm: %v", err)
		}
		if string(data) != string(wasmContent) {
			t.Errorf("content mismatch: got %q", string(data))
		}

		lock, err := LoadLockFile(tmpDir)
		if err != nil {
			t.Fatalf("LoadLockFile: %v", err)
		}
		entry, ok := lock.Get("test-plugin")
		if !ok {
			t.Fatal("expected test-plugin in lock file")
		}
		if entry.Version != "v3.1.0" {
			t.Errorf("expected version v3.1.0, got %s", entry.Version)
		}
		if entry.Source != "github.com/myowner/myrepo" {
			t.Errorf("expected source github.com/myowner/myrepo, got %s", entry.Source)
		}
	})

	t.Run("InstallFromGitHub_WithTag", func(t *testing.T) {
		wasmContent := []byte("tagged wasm")

		mux := http.NewServeMux()
		mux.HandleFunc("/download/tagged.wasm", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(wasmContent)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		release := GitHubRelease{
			TagName: "v2.0.0",
			Assets: []GitHubAsset{
				{Name: "tagged.wasm", BrowserDownloadURL: server.URL + "/download/tagged.wasm"},
			},
		}

		mux.HandleFunc("/repos/o/r/releases/tags/v2.0.0", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		githubAPIBase = server.URL

		pluginsDir := filepath.Join(t.TempDir(), "plugins")
		name, err := InstallFromGitHub(context.Background(), "o", "r", "v2.0.0", pluginsDir)
		if err != nil {
			t.Fatalf("InstallFromGitHub with tag: %v", err)
		}
		if name != "tagged" {
			t.Errorf("expected 'tagged', got %q", name)
		}
	})

	t.Run("InstallFromGitHub_FetchReleaseFails", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := InstallFromGitHub(context.Background(), "owner", "repo", "", t.TempDir())
		if err == nil {
			t.Fatal("expected error when release fetch fails")
		}
		if !strings.Contains(err.Error(), "fetch release") {
			t.Errorf("expected 'fetch release' error, got: %v", err)
		}
	})

	t.Run("InstallFromGitHub_DownloadFails", func(t *testing.T) {
		release := GitHubRelease{
			TagName: "v1.0.0",
			Assets: []GitHubAsset{
				{Name: "plugin.wasm", BrowserDownloadURL: "http://127.0.0.1:1/invalid-url"},
			},
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		_, err := InstallFromGitHub(context.Background(), "owner", "repo", "", t.TempDir())
		if err == nil {
			t.Fatal("expected error when download fails")
		}
		if !strings.Contains(err.Error(), "download") {
			t.Errorf("expected 'download' error, got: %v", err)
		}
	})

	t.Run("CheckUpdate_AlreadyUpToDate", func(t *testing.T) {
		release := GitHubRelease{TagName: "v1.0.0"}
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		lock, _ := LoadLockFile(t.TempDir())
		lock.Set("myplugin", LockEntry{Source: "github.com/owner/repo", Version: "v1.0.0"})

		newTag, err := CheckUpdate(context.Background(), "myplugin", lock)
		if err != nil {
			t.Fatalf("CheckUpdate: %v", err)
		}
		if newTag != "" {
			t.Errorf("expected empty tag (up-to-date), got %q", newTag)
		}
	})

	t.Run("CheckUpdate_UpdateAvailable", func(t *testing.T) {
		release := GitHubRelease{TagName: "v2.0.0"}
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		lock, _ := LoadLockFile(t.TempDir())
		lock.Set("myplugin", LockEntry{Source: "github.com/owner/repo", Version: "v1.0.0"})

		newTag, err := CheckUpdate(context.Background(), "myplugin", lock)
		if err != nil {
			t.Fatalf("CheckUpdate: %v", err)
		}
		if newTag != "v2.0.0" {
			t.Errorf("expected v2.0.0, got %q", newTag)
		}
	})

	t.Run("CheckUpdate_FetchFails", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		lock, _ := LoadLockFile(t.TempDir())
		lock.Set("myplugin", LockEntry{Source: "github.com/owner/repo", Version: "v1.0.0"})

		_, err := CheckUpdate(context.Background(), "myplugin", lock)
		if err == nil {
			t.Fatal("expected error when fetch fails")
		}
	})

	t.Run("UpdatePlugin_FullFlow", func(t *testing.T) {
		wasmContent := []byte("updated wasm content")

		mux := http.NewServeMux()
		mux.HandleFunc("/download/myplugin.wasm", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(wasmContent)
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		release := GitHubRelease{
			TagName: "v4.0.0",
			Assets: []GitHubAsset{
				{Name: "myplugin.wasm", BrowserDownloadURL: server.URL + "/download/myplugin.wasm"},
			},
		}

		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(release)
		})

		githubAPIBase = server.URL

		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0o755)

		lock, _ := LoadLockFile(tmpDir)
		lock.Set("myplugin", LockEntry{Source: "github.com/owner/repo", Version: "v3.0.0"})
		_ = lock.Save()

		// UpdatePlugin calls InstallFromGitHub, which writes to a new
		// LockFile instance on disk. The in-memory 'lock' is not updated,
		// so the returned version comes from the stale in-memory entry.
		_, err := UpdatePlugin(context.Background(), "myplugin", pluginsDir, lock)
		if err != nil {
			t.Fatalf("UpdatePlugin: %v", err)
		}

		// Verify the wasm file was downloaded correctly.
		data, err := os.ReadFile(filepath.Join(pluginsDir, "myplugin.wasm"))
		if err != nil {
			t.Fatalf("read wasm: %v", err)
		}
		if string(data) != string(wasmContent) {
			t.Errorf("content mismatch: got %q", string(data))
		}

		// The on-disk lock file should have the new version.
		diskLock, err := LoadLockFile(tmpDir)
		if err != nil {
			t.Fatalf("reload lock: %v", err)
		}
		entry, ok := diskLock.Get("myplugin")
		if !ok {
			t.Fatal("expected myplugin in lock file")
		}
		if entry.Version != "v4.0.0" {
			t.Errorf("expected v4.0.0 in on-disk lock, got %q", entry.Version)
		}
	})

	t.Run("FetchGitHubRelease_WithGitHubToken", func(t *testing.T) {
		// Set GITHUB_TOKEN to exercise the Authorization header path.
		t.Setenv("GITHUB_TOKEN", "test-token-123")

		var gotAuth string
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(GitHubRelease{TagName: "v5.0.0"})
		})

		server := httptest.NewServer(mux)
		defer server.Close()
		githubAPIBase = server.URL

		got, err := fetchGitHubRelease(context.Background(), "owner", "repo", "")
		if err != nil {
			t.Fatalf("fetchGitHubRelease with token: %v", err)
		}
		if got.TagName != "v5.0.0" {
			t.Errorf("expected v5.0.0, got %s", got.TagName)
		}
		if gotAuth != "Bearer test-token-123" {
			t.Errorf("expected Authorization 'Bearer test-token-123', got %q", gotAuth)
		}
	})

	t.Run("DownloadFile_WithGitHubToken", func(t *testing.T) {
		// Exercise the GITHUB_TOKEN path in downloadFile.
		t.Setenv("GITHUB_TOKEN", "dl-token-456")

		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte("content"))
		}))
		defer server.Close()

		dest := filepath.Join(t.TempDir(), "out.wasm")
		if _, err := downloadFile(context.Background(), server.URL+"/file.wasm", dest, ""); err != nil {
			t.Fatalf("downloadFile: %v", err)
		}
		if gotAuth != "Bearer dl-token-456" {
			t.Errorf("expected Authorization 'Bearer dl-token-456', got %q", gotAuth)
		}
	})
}

// ---------------------------------------------------------------------------
// InstallFromGitHub - tests that don't need mock API
// ---------------------------------------------------------------------------

func TestInstallFromGitHub_EmptyDir(t *testing.T) {
	t.Parallel()
	_, err := InstallFromGitHub(context.Background(), "owner", "repo", "", "")
	if err == nil || !strings.Contains(err.Error(), "plugins dir is not configured") {
		t.Errorf("expected 'plugins dir is not configured', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CheckUpdate - tests that don't need mock API
// ---------------------------------------------------------------------------

func TestCheckUpdate_NoLockEntry(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	_, err := CheckUpdate(context.Background(), "missing", lock)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not in lock file") {
		t.Errorf("expected 'not in lock file', got: %v", err)
	}
}

func TestCheckUpdate_NonGitHubSource(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	lock.Set("local", LockEntry{Source: "/local/path", Version: "v1"})

	_, err := CheckUpdate(context.Background(), "local", lock)
	if err == nil {
		t.Fatal("expected error for non-GitHub source")
	}
	if !strings.Contains(err.Error(), "not a GitHub repo") {
		t.Errorf("expected 'not a GitHub repo', got: %v", err)
	}
}

func TestCheckUpdate_BadSourceFormat(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	lock.Set("bad", LockEntry{Source: "github.com/noslash", Version: "v1"})

	_, err := CheckUpdate(context.Background(), "bad", lock)
	if err == nil {
		t.Fatal("expected error for bad source format")
	}
	if !strings.Contains(err.Error(), "unexpected source format") {
		t.Errorf("expected 'unexpected source format', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdatePlugin - tests that don't need mock API
// ---------------------------------------------------------------------------

func TestUpdatePlugin_NoLockEntry(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	_, err := UpdatePlugin(context.Background(), "missing", t.TempDir(), lock)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not in lock file") {
		t.Errorf("expected 'not in lock file', got: %v", err)
	}
}

func TestUpdatePlugin_NonGitHubSource(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	lock.Set("local", LockEntry{Source: "/local/path", Version: "v1"})

	_, err := UpdatePlugin(context.Background(), "local", t.TempDir(), lock)
	if err == nil {
		t.Fatal("expected error for non-GitHub source")
	}
	if !strings.Contains(err.Error(), "not a GitHub repo") {
		t.Errorf("expected 'not a GitHub repo', got: %v", err)
	}
}

func TestUpdatePlugin_BadSourceFormat(t *testing.T) {
	t.Parallel()
	lock, _ := LoadLockFile(t.TempDir())
	lock.Set("bad", LockEntry{Source: "github.com/noslash", Version: "v1"})

	_, err := UpdatePlugin(context.Background(), "bad", t.TempDir(), lock)
	if err == nil {
		t.Fatal("expected error for bad source format")
	}
	if !strings.Contains(err.Error(), "unexpected source format") {
		t.Errorf("expected 'unexpected source format', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LockFile - additional coverage
// ---------------------------------------------------------------------------

func TestLoadLockFile_CorruptJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lockDir := filepath.Join(tmpDir, ".osg")
	_ = os.MkdirAll(lockDir, 0o755)

	lockPath := filepath.Join(lockDir, lockFileName)
	_ = os.WriteFile(lockPath, []byte("not valid json{{{"), 0o644)

	lf, err := LoadLockFile(tmpDir)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "parse lock file") {
		t.Errorf("expected 'parse lock file' error, got: %v", err)
	}
	if lf == nil {
		t.Fatal("expected non-nil lock file even on error")
	}
}

func TestLoadLockFile_NullPluginsField(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lockDir := filepath.Join(tmpDir, ".osg")
	_ = os.MkdirAll(lockDir, 0o755)

	lockPath := filepath.Join(lockDir, lockFileName)
	_ = os.WriteFile(lockPath, []byte(`{"plugins": null}`), 0o644)

	lf, err := LoadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("LoadLockFile: %v", err)
	}
	if lf.Plugins == nil {
		t.Fatal("expected non-nil Plugins map even when JSON has null")
	}
}

func TestLockFile_SaveCreatesNestedDirs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c")
	lf := &LockFile{
		path:    filepath.Join(deepDir, ".osg", lockFileName),
		Plugins: map[string]LockEntry{"test": {Source: "s", Version: "v1"}},
	}

	if err := lf.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(lf.path)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if !strings.Contains(string(data), "test") {
		t.Error("saved lock file does not contain 'test'")
	}
}

func TestLockFile_RoundTripWithNestedDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	lock, err := LoadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("LoadLockFile: %v", err)
	}

	lock.Set("alpha", LockEntry{Source: "github.com/a/alpha", Version: "v1.0.0"})
	lock.Set("beta", LockEntry{Source: "github.com/b/beta", Version: "v2.0.0"})

	if err := lock.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	lock2, err := LoadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if len(lock2.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(lock2.Plugins))
	}

	e, ok := lock2.Get("alpha")
	if !ok || e.Version != "v1.0.0" || e.Source != "github.com/a/alpha" {
		t.Errorf("unexpected alpha entry: %+v (ok=%v)", e, ok)
	}

	e, ok = lock2.Get("beta")
	if !ok || e.Version != "v2.0.0" || e.Source != "github.com/b/beta" {
		t.Errorf("unexpected beta entry: %+v (ok=%v)", e, ok)
	}

	names := lock2.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("unexpected names: %v", names)
	}
}

// ---------------------------------------------------------------------------
// FetchIndexFrom - additional coverage
// ---------------------------------------------------------------------------

func TestFetchIndexFrom_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	_, err := FetchIndexFrom(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode index") {
		t.Errorf("expected 'decode index' error, got: %v", err)
	}
}

func TestFetchIndexFrom_ConnectionError(t *testing.T) {
	t.Parallel()

	_, err := FetchIndexFrom(context.Background(), "http://127.0.0.1:1/nonexistent")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
	if !strings.Contains(err.Error(), "fetch index") {
		t.Errorf("expected 'fetch index' error, got: %v", err)
	}
}

func TestFetchIndexFrom_MultiplePlugins(t *testing.T) {
	t.Parallel()

	index := PluginIndex{
		Plugins: []IndexEntry{
			{Name: "search", Description: "Full-text search", Author: "jllopis", Repo: "github.com/jllopis/osg-search"},
			{Name: "feed", Description: "RSS feed", Author: "jllopis", Repo: "github.com/jllopis/osg-feed"},
			{Name: "llmstxt", Description: "LLMs.txt", Author: "jllopis", Repo: "github.com/jllopis/osg-llmstxt"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(index)
	}))
	defer server.Close()

	got, err := FetchIndexFrom(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchIndexFrom: %v", err)
	}
	if len(got.Plugins) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(got.Plugins))
	}
}

// ---------------------------------------------------------------------------
// EnsureBundledPlugins - additional coverage
// ---------------------------------------------------------------------------

func TestEnsureBundledPlugins_AlreadyExists_NotOverwritten(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	_ = os.MkdirAll(pluginsDir, 0o755)

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("first extraction: %v", err)
	}

	wasmPath := filepath.Join(pluginsDir, "search.wasm")
	_ = os.WriteFile(wasmPath, []byte("custom"), 0o644)

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("second extraction: %v", err)
	}

	data, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "custom" {
		t.Errorf("file was overwritten; expected 'custom', got %d bytes", len(data))
	}
}

// ---------------------------------------------------------------------------
// downloadFile - additional coverage
// ---------------------------------------------------------------------------

func TestDownloadFile_Success(t *testing.T) {
	t.Parallel()

	content := []byte("download test content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "output.bin")
	if _, err := downloadFile(context.Background(), server.URL, dest, ""); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q", string(data))
	}
}

func TestDownloadFile_ConnectionRefused(t *testing.T) {
	t.Parallel()

	dest := filepath.Join(t.TempDir(), "output.bin")
	_, err := downloadFile(context.Background(), "http://127.0.0.1:1/unreachable", dest, "")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

// ---------------------------------------------------------------------------
// ParseGitHubRef - additional coverage
// ---------------------------------------------------------------------------

func TestParseGitHubRef_VariousFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref       string
		wantOwner string
		wantRepo  string
		wantTag   string
	}{
		{"github.com/user/repo", "user", "repo", ""},
		{"github.com/user/repo@v1.0.0", "user", "repo", "v1.0.0"},
		{"github.com/user/repo@latest", "user", "repo", "latest"},
		{"github.com/org-name/my-plugin.v2@v2.5.0", "org-name", "my-plugin.v2", "v2.5.0"},
		{"github.com/a/b", "a", "b", ""},
		{"github.com/a/b@some-branch", "a", "b", "some-branch"},
		{"", "", "", ""},
		{"gitlab.com/user/repo", "", "", ""},
		{"github.com/user", "", "", ""},
		{"github.com//repo", "", "", ""},
		{"not-a-url", "", "", ""},
	}
	for _, tc := range tests {
		owner, repo, tag := ParseGitHubRef(tc.ref)
		if owner != tc.wantOwner || repo != tc.wantRepo || tag != tc.wantTag {
			t.Errorf("ParseGitHubRef(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.ref, owner, repo, tag, tc.wantOwner, tc.wantRepo, tc.wantTag)
		}
	}
}

// ---------------------------------------------------------------------------
// EnsureOfficialPlugins - additional coverage
// ---------------------------------------------------------------------------

func TestEnsureOfficialPlugins_AllBundled(t *testing.T) {
	t.Parallel()
	logger := silentLogger()

	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	_ = os.MkdirAll(pluginsDir, 0o755)

	err := EnsureOfficialPlugins(context.Background(), pluginsDir, BundledPlugins, logger)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestEnsureOfficialPlugins_AllAlreadyInstalled(t *testing.T) {
	t.Parallel()
	logger := silentLogger()

	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	_ = os.MkdirAll(pluginsDir, 0o755)

	_ = os.WriteFile(filepath.Join(pluginsDir, "myplugin.wasm"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(pluginsDir, "other.wasm"), []byte("data"), 0o644)

	err := EnsureOfficialPlugins(context.Background(), pluginsDir, []string{"myplugin", "other"}, logger)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Emit - multi-plugin with timeout exercises goroutine timeout path
// ---------------------------------------------------------------------------

func TestEmit_MultiplePlugins_WithTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	src := filepath.Join(pluginsDir, "search.wasm")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "search2.wasm"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx := context.Background()
	// Load with timeout to exercise the multi-plugin goroutine timeout path.
	m, err := Load(ctx, pluginsDir, []string{"search", "search2"}, 30, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.plugins))
	}

	publicDir := filepath.Join(dir, "public")
	_ = os.MkdirAll(publicDir, 0o755)

	result := m.Emit(ctx, "build.finished", map[string]any{
		"config": map[string]any{"public_dir": publicDir},
		"site":   map[string]any{"pages": []any{}},
	})
	_ = result
}

// ---------------------------------------------------------------------------
// Emit - more edge cases
// ---------------------------------------------------------------------------

func TestEmit_SinglePlugin_WithNilResponse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	result := m.Emit(ctx, "nonexistent.event", map[string]any{"key": "value"})
	if result != nil {
		t.Errorf("expected nil for unhandled event, got %v", result)
	}
}

func TestEmit_SinglePlugin_NilPayload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, silentLogger())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	result := m.Emit(ctx, "build.started", nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// Load with timeout value
// ---------------------------------------------------------------------------

func TestLoad_WithTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 10, silentLogger())
	if err != nil {
		t.Fatalf("Load with timeout: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if m.timeout == 0 {
		t.Error("expected non-zero timeout")
	}
	if len(m.plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(m.plugins))
	}
}

// ---------------------------------------------------------------------------
// Metadata
// ---------------------------------------------------------------------------

func TestMetadata_MatchesPlugins(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	metas := m.Metadata()
	if len(metas) != len(m.plugins) {
		t.Fatalf("metadata count %d != plugin count %d", len(metas), len(m.plugins))
	}
	for i, meta := range metas {
		if meta.Name != m.plugins[i].info.Name {
			t.Errorf("metadata[%d].Name = %q, plugin[%d].info.Name = %q", i, meta.Name, i, m.plugins[i].info.Name)
		}
	}
}
