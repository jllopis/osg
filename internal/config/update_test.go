package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpdatePluginsEnabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(DefaultConfigYAML()), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := UpdatePluginsEnabled(path, []string{"search.wasm", "feed", "search", "  "}); err != nil {
		t.Fatalf("update plugins: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	want := []string{"feed", "search"}
	if !reflect.DeepEqual(cfg.PluginsEnabled, want) {
		t.Fatalf("plugins_enabled mismatch: got %v want %v", cfg.PluginsEnabled, want)
	}
}
