package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBundledPlugins_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	// search.wasm should be extracted.
	wasmPath := filepath.Join(pluginsDir, "search.wasm")
	info, err := os.Stat(wasmPath)
	if err != nil {
		t.Fatalf("search.wasm not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("search.wasm is empty")
	}
}

func TestEnsureBundledPlugins_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Place a custom search.wasm before extraction.
	customContent := []byte("custom plugin")
	wasmPath := filepath.Join(pluginsDir, "search.wasm")
	if err := os.WriteFile(wasmPath, customContent, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureBundledPlugins(pluginsDir); err != nil {
		t.Fatalf("EnsureBundledPlugins: %v", err)
	}

	// Custom content should be preserved.
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom plugin" {
		t.Errorf("expected custom content, got %d bytes", len(data))
	}
}

func TestEnsureBundledPlugins_EmptyDir(t *testing.T) {
	if err := EnsureBundledPlugins(""); err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
}

func TestBundledPlugins_ListContainsSearch(t *testing.T) {
	found := false
	for _, name := range BundledPlugins {
		if name == "search" {
			found = true
			break
		}
	}
	if !found {
		t.Error("BundledPlugins does not contain 'search'")
	}
}
