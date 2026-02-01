package app

import (
	"context"

	"osg/internal/config"
	"osg/internal/theme"
)

func RunThemeInit(_ context.Context, opts CLIOptions, name string) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	return theme.ScaffoldTheme(cfg.ThemesDir, name)
}
