package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"osg/internal/config"
	"osg/internal/theme"
)

func RunInit(_ context.Context, opts CLIOptions) error {
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg := config.Default()
	if _, err := os.Stat(configPath); err == nil {
		loaded, err := config.Load(configPath)
		if err != nil {
			return err
		}
		cfg = loaded
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}

	if opts.IncludeDrafts != nil {
		cfg.IncludeDrafts = *opts.IncludeDrafts
	}

	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}

	dirs := []string{
		cfg.ContentDir,
		cfg.PluginsDir,
		cfg.SassDir,
		cfg.StaticDir,
		cfg.TemplatesDir,
		cfg.ThemesDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
		return fmt.Errorf("init theme: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil && filepath.Dir(configPath) != "." {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(config.DefaultConfigYAML()), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
