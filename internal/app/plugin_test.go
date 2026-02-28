package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

func TestPluginEnableDisable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(config.DefaultConfigYAML()), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: path}

	if err := RunPluginEnable(context.TODO(), opts, "search.wasm"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.PluginsEnabled) != 1 || cfg.PluginsEnabled[0] != "search" {
		t.Fatalf("unexpected plugins_enabled: %v", cfg.PluginsEnabled)
	}

	if err := RunPluginDisable(context.TODO(), opts, "search"); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}

	cfg, err = config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.PluginsEnabled) != 0 {
		t.Fatalf("expected empty plugins_enabled, got %v", cfg.PluginsEnabled)
	}
}
