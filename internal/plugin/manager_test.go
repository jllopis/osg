package plugin

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// normalizePluginName
// ---------------------------------------------------------------------------

func TestNormalizePluginName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"search", "search"},
		{"search.wasm", "search"},
		{"search.WASM", "search"},
		{"  search  ", "search"},
		{"  search.wasm  ", "search"},
		{"", ""},
		{"  ", ""},
		{"my-plugin.wasm", "my-plugin"},
		{"plugin.name.wasm", "plugin.name"},
	}
	for _, tc := range cases {
		got := normalizePluginName(tc.input)
		if got != tc.want {
			t.Errorf("normalizePluginName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Merge / mergeMaps
// ---------------------------------------------------------------------------

func TestMerge_Shallow(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"a": 1, "b": 2}
	src := map[string]any{"b": 3, "c": 4}
	Merge(dst, src)
	if dst["a"] != 1 {
		t.Errorf("expected a=1, got %v", dst["a"])
	}
	if dst["b"] != 3 {
		t.Errorf("expected b=3, got %v", dst["b"])
	}
	if dst["c"] != 4 {
		t.Errorf("expected c=4, got %v", dst["c"])
	}
}

func TestMerge_DeepNested(t *testing.T) {
	t.Parallel()
	dst := map[string]any{
		"config": map[string]any{
			"title": "old",
			"keep":  true,
		},
	}
	src := map[string]any{
		"config": map[string]any{
			"title": "new",
			"extra": "added",
		},
	}
	Merge(dst, src)
	cfg := dst["config"].(map[string]any)
	if cfg["title"] != "new" {
		t.Errorf("expected title=new, got %v", cfg["title"])
	}
	if cfg["keep"] != true {
		t.Errorf("expected keep=true, got %v", cfg["keep"])
	}
	if cfg["extra"] != "added" {
		t.Errorf("expected extra=added, got %v", cfg["extra"])
	}
}

func TestMerge_NilSafety(t *testing.T) {
	t.Parallel()
	// Should not panic.
	Merge(nil, map[string]any{"a": 1})
	Merge(map[string]any{"a": 1}, nil)
	Merge(nil, nil)
}

func TestMerge_ScalarOverwritesMap(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"x": map[string]any{"nested": 1}}
	src := map[string]any{"x": "scalar"}
	Merge(dst, src)
	if dst["x"] != "scalar" {
		t.Errorf("expected x=scalar, got %v", dst["x"])
	}
}

// ---------------------------------------------------------------------------
// Load - edge cases (no WASM needed)
// ---------------------------------------------------------------------------

func TestLoad_EmptyDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, err := Load(ctx, "", nil, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(m.plugins))
	}
}

func TestLoad_NonexistentDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m, err := Load(ctx, "/nonexistent/path/to/plugins", []string{"search"}, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(m.plugins))
	}
}

func TestLoad_DirIsFile(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	ctx := context.Background()
	m, err := Load(ctx, f.Name(), []string{"search"}, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(m.plugins))
	}
}

func TestLoad_EmptyEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Place a dummy .wasm file.
	_ = os.WriteFile(filepath.Join(dir, "dummy.wasm"), []byte("not a wasm"), 0o644)

	ctx := context.Background()
	m, err := Load(ctx, dir, nil, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins (none enabled), got %d", len(m.plugins))
	}
	_ = m.Close(ctx)
}

func TestLoad_EnabledButMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Dir exists but is empty - "search" is enabled but not installed.
	ctx := context.Background()
	logger := slog.Default()
	m, err := Load(ctx, dir, []string{"search"}, 0, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins (missing .wasm), got %d", len(m.plugins))
	}
	_ = m.Close(ctx)
}

func TestLoad_InvalidWasm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Write invalid content as .wasm.
	_ = os.WriteFile(filepath.Join(dir, "bad.wasm"), []byte("not valid wasm"), 0o644)

	ctx := context.Background()
	logger := slog.Default()
	m, err := Load(ctx, dir, []string{"bad"}, 0, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid plugin is skipped with warning, not an error.
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins (invalid wasm skipped), got %d", len(m.plugins))
	}
	_ = m.Close(ctx)
}

func TestLoad_SkipsNonWasmFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)

	ctx := context.Background()
	m, err := Load(ctx, dir, []string{"readme"}, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(m.plugins))
	}
	_ = m.Close(ctx)
}

// ---------------------------------------------------------------------------
// Emit - edge cases (no WASM needed)
// ---------------------------------------------------------------------------

func TestEmit_NoPlugins(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	ctx := context.Background()
	result := m.Emit(ctx, "build.started", map[string]any{"foo": "bar"})
	if result != nil {
		t.Errorf("expected nil from Emit with no plugins, got %v", result)
	}
}

func TestClose_NilRuntime(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	if err := m.Close(context.Background()); err != nil {
		t.Errorf("expected no error closing nil runtime, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration tests using the bundled search.wasm
// ---------------------------------------------------------------------------

// setupSearchPlugin extracts the bundled search.wasm and loads it.
func setupSearchPlugin(t *testing.T) (*Manager, context.Context) {
	t.Helper()
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	logger := slog.Default()
	m, err := Load(ctx, pluginsDir, []string{"search"}, 0, logger)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.plugins) != 1 {
		t.Fatalf("expected 1 plugin loaded, got %d", len(m.plugins))
	}
	return m, ctx
}

func TestLoad_BundledSearch(t *testing.T) {
	t.Parallel()
	m, ctx := setupSearchPlugin(t)
	defer func() { _ = m.Close(ctx) }()

	if m.plugins[0].name != "search" {
		t.Errorf("expected plugin name 'search', got %q", m.plugins[0].name)
	}
}

func TestMetadata_FallbackToFilename(t *testing.T) {
	t.Parallel()
	m, ctx := setupSearchPlugin(t)
	defer func() { _ = m.Close(ctx) }()

	// The current search plugin does not export plugin_info,
	// so Metadata should return a fallback with name = filename.
	metas := m.Metadata()
	if len(metas) != 1 {
		t.Fatalf("expected 1 metadata entry, got %d", len(metas))
	}
	if metas[0].Name != "search" {
		t.Errorf("expected metadata name 'search', got %q", metas[0].Name)
	}
	// No version since plugin_info is not exported.
	if metas[0].Version != "" {
		t.Errorf("expected empty version for plugin without plugin_info, got %q", metas[0].Version)
	}
}

func TestMetadata_EmptyManager(t *testing.T) {
	t.Parallel()
	m := &Manager{}
	metas := m.Metadata()
	if len(metas) != 0 {
		t.Errorf("expected 0 metadata entries, got %d", len(metas))
	}
}

func TestEmit_SearchIgnoresNonFinished(t *testing.T) {
	t.Parallel()
	m, ctx := setupSearchPlugin(t)
	defer func() { _ = m.Close(ctx) }()

	// The search plugin only listens to build.finished.
	// Other events should return no overrides.
	result := m.Emit(ctx, "build.started", map[string]any{
		"config": map[string]any{"public_dir": t.TempDir()},
		"site":   map[string]any{"pages": []any{}},
	})
	if result != nil {
		t.Errorf("expected nil overrides for build.started, got %v", result)
	}
}

func TestEmit_SearchBuildFinished(t *testing.T) {
	t.Parallel()
	m, ctx := setupSearchPlugin(t)
	defer func() { _ = m.Close(ctx) }()

	publicDir := filepath.Join(t.TempDir(), "public")
	_ = os.MkdirAll(publicDir, 0o755)

	pages := []any{
		map[string]any{
			"title":     "Test Post",
			"summary":   "A test summary",
			"permalink": "/2025/01/01/test-post/",
			"date":      "2025-01-01",
			"taxonomies": map[string]any{
				"tags": []any{"go", "wasm"},
			},
		},
	}

	result := m.Emit(ctx, "build.finished", map[string]any{
		"config": map[string]any{"public_dir": publicDir},
		"site":   map[string]any{"pages": pages},
	})

	// search plugin returns 0 (no overrides), so result should be nil.
	if result != nil {
		t.Errorf("expected nil overrides, got %v", result)
	}

	// But it should have written search.json and search/index.html.
	searchJSON := filepath.Join(publicDir, "search.json")
	if _, err := os.Stat(searchJSON); err != nil {
		t.Errorf("search.json not created: %v", err)
	}

	searchHTML := filepath.Join(publicDir, "search", "index.html")
	if _, err := os.Stat(searchHTML); err != nil {
		t.Errorf("search/index.html not created: %v", err)
	}

	// Verify search.json contains our test page.
	data, err := os.ReadFile(searchJSON)
	if err != nil {
		t.Fatalf("read search.json: %v", err)
	}
	content := string(data)
	if !contains(content, "Test Post") {
		t.Error("search.json does not contain 'Test Post'")
	}
	if !contains(content, "A test summary") {
		t.Error("search.json does not contain 'A test summary'")
	}
}

func TestLoad_FiltersByEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	// Enable a different plugin name - search should NOT be loaded.
	m, err := Load(ctx, pluginsDir, []string{"other-plugin"}, 0, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 0 {
		t.Errorf("expected 0 plugins (search not enabled), got %d", len(m.plugins))
	}
}

func TestLoad_NormalizesEnabledNames(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	ctx := context.Background()
	// Pass with .wasm extension and whitespace - should still match.
	m, err := Load(ctx, pluginsDir, []string{"  search.wasm  "}, 0, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer func() { _ = m.Close(ctx) }()

	if len(m.plugins) != 1 {
		t.Errorf("expected 1 plugin (normalized name match), got %d", len(m.plugins))
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
