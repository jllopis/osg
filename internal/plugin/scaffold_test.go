package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldRust(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := ScaffoldRust(base, "My Plugin"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "My Plugin")
	files := []string{
		filepath.Join(root, "Cargo.toml"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "src", "lib.rs"),
		filepath.Join(root, "build.sh"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("expected file %s: %v", file, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read file %s: %v", file, err)
		}
		content := string(data)
		if strings.Contains(content, "{{name}}") || strings.Contains(content, "{{crate_name}}") {
			t.Fatalf("placeholders not replaced in %s", file)
		}
	}
}

func TestScaffoldRustContent(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "feed", "rust"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "feed")

	// Cargo.toml should have crate name
	data, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		t.Fatalf("read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(data), "osg-feed") {
		t.Error("Cargo.toml should contain crate name osg-feed")
	}

	// lib.rs should have plugin_info export
	data, err = os.ReadFile(filepath.Join(root, "src", "lib.rs"))
	if err != nil {
		t.Fatalf("read lib.rs: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "plugin_info") {
		t.Error("lib.rs should contain plugin_info export")
	}
	if !strings.Contains(content, "bytes_to_wasm") {
		t.Error("lib.rs should contain bytes_to_wasm helper")
	}
	if !strings.Contains(content, `"feed"`) {
		t.Error("lib.rs should contain plugin name 'feed'")
	}
}

func TestScaffoldGo(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "analytics", "go"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "analytics")
	files := []string{
		filepath.Join(root, "main.go"),
		filepath.Join(root, "go.mod"),
		filepath.Join(root, "build.sh"),
		filepath.Join(root, "README.md"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("expected file %s: %v", file, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read file %s: %v", file, err)
		}
		content := string(data)
		if strings.Contains(content, "{{name}}") || strings.Contains(content, "{{module_name}}") {
			t.Fatalf("placeholders not replaced in %s", file)
		}
	}
}

func TestScaffoldGoContent(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "sitemap", "go"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "sitemap")

	// go.mod should have correct module name
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "module osg-sitemap") {
		t.Errorf("go.mod should contain 'module osg-sitemap', got:\n%s", content)
	}

	// main.go should have plugin_info and wasmexport
	data, err = os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "plugin_info") {
		t.Error("main.go should contain plugin_info export")
	}
	if !strings.Contains(content, "//go:wasmexport") {
		t.Error("main.go should contain //go:wasmexport directive")
	}
	if !strings.Contains(content, `"sitemap"`) {
		t.Error("main.go should contain plugin name 'sitemap'")
	}
	if strings.Contains(content, "//go:build ignore") {
		t.Error("main.go should NOT contain //go:build ignore tag after scaffolding")
	}

	// build.sh should reference the plugin name
	data, err = os.ReadFile(filepath.Join(root, "build.sh"))
	if err != nil {
		t.Fatalf("read build.sh: %v", err)
	}
	if !strings.Contains(string(data), "sitemap.wasm") {
		t.Error("build.sh should reference sitemap.wasm")
	}
}

func TestScaffoldGoTmplExtensionStripped(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "test-ext", "go"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "test-ext")

	// go.mod should exist (not go.mod.tmpl)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal("go.mod should exist after scaffold")
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod.tmpl")); err == nil {
		t.Fatal("go.mod.tmpl should NOT exist — .tmpl extension should be stripped")
	}
}

func TestScaffoldTinyGoAlias(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "test-alias", "tinygo"); err != nil {
		t.Fatalf("scaffold with lang=tinygo failed: %v", err)
	}

	root := filepath.Join(base, "test-alias")
	if _, err := os.Stat(filepath.Join(root, "main.go")); err != nil {
		t.Fatal("tinygo alias should produce Go scaffold with main.go")
	}
}

func TestScaffoldDefaultLang(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := Scaffold(base, "default-lang", ""); err != nil {
		t.Fatalf("scaffold with empty lang failed: %v", err)
	}

	root := filepath.Join(base, "default-lang")
	// Default should be Rust
	if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err != nil {
		t.Fatal("empty lang should default to Rust (Cargo.toml expected)")
	}
}

func TestScaffoldUnsupportedLang(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	err := Scaffold(base, "bad-lang", "python")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("expected 'unsupported language' error, got: %v", err)
	}
}

func TestScaffoldAlreadyExists(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	target := filepath.Join(base, "existing")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	err := Scaffold(base, "existing", "rust")
	if err == nil {
		t.Fatal("expected error when plugin already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestScaffoldEmptyName(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	err := Scaffold(base, "", "rust")
	if err == nil {
		t.Fatal("expected error for empty plugin name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected 'name is required' error, got: %v", err)
	}
}

func TestToCrateName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feed", "osg-feed"},
		{"my plugin", "osg-my-plugin"},
		{"osg-search", "osg-search"},
		{"My_Plugin.v2", "osg-my-plugin-v2"},
	}
	for _, tc := range tests {
		got := toCrateName(tc.input)
		if got != tc.want {
			t.Errorf("toCrateName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestToModuleName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sitemap", "osg-sitemap"},
		{"my plugin", "osg-my-plugin"},
		{"osg-analytics", "osg-analytics"},
		{"My_Plugin.v2", "osg-my-plugin-v2"},
	}
	for _, tc := range tests {
		got := toModuleName(tc.input)
		if got != tc.want {
			t.Errorf("toModuleName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
