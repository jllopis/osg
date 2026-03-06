package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// testInDir runs fn inside a temporary directory, restoring the original
// working directory when done.
func testInDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	fn()
}

func TestRunInit_CreatesDirsAndConfig(t *testing.T) {
	tmpDir := t.TempDir()

	testInDir(t, tmpDir, func() {
		opts := CLIOptions{
			ConfigPath: "config.yaml",
		}

		err := RunInit(context.Background(), opts)
		if err != nil {
			t.Fatalf("RunInit: %v", err)
		}

		// Config file should be created.
		if _, err := os.Stat("config.yaml"); err != nil {
			t.Errorf("config.yaml not created: %v", err)
		}

		// Standard directories should exist.
		for _, dir := range []string{"content", "plugins", "sass", "static", "templates", "themes"} {
			if _, err := os.Stat(dir); err != nil {
				t.Errorf("directory %s not created: %v", dir, err)
			}
		}
	})
}

func TestRunInit_ExtractsBundledPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	testInDir(t, tmpDir, func() {
		opts := CLIOptions{
			ConfigPath: "config.yaml",
		}

		err := RunInit(context.Background(), opts)
		if err != nil {
			t.Fatalf("RunInit: %v", err)
		}

		// search.wasm should be extracted into plugins/ directory.
		wasmPath := filepath.Join("plugins", "search.wasm")
		info, err := os.Stat(wasmPath)
		if err != nil {
			t.Fatalf("search.wasm not found after init: %v", err)
		}
		if info.Size() == 0 {
			t.Error("search.wasm is empty")
		}
	})
}

func TestRunInit_ExistingConfigNotOverwritten(t *testing.T) {
	tmpDir := t.TempDir()

	testInDir(t, tmpDir, func() {
		// Pre-create a config file.
		content := []byte("site_title: My Site\n")
		if err := os.WriteFile("config.yaml", content, 0o644); err != nil {
			t.Fatal(err)
		}

		opts := CLIOptions{
			ConfigPath: "config.yaml",
		}

		err := RunInit(context.Background(), opts)
		if err != nil {
			t.Fatalf("RunInit: %v", err)
		}

		// Config should not be overwritten.
		data, err := os.ReadFile("config.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(content) {
			t.Errorf("config was overwritten: got %q", string(data))
		}
	})
}
