package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"osg/internal/config"
	"osg/internal/theme"
)

func RunThemeInit(_ context.Context, opts CLIOptions, name string, parent string) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	parent = strings.TrimSpace(parent)
	if parent != "" {
		// Ensure the default theme exists so it can be used as parent.
		if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
			return err
		}
		return theme.ScaffoldChildTheme(cfg.ThemesDir, name, parent)
	}

	return theme.ScaffoldTheme(cfg.ThemesDir, name)
}

func RunThemeList(_ context.Context, opts CLIOptions, w io.Writer) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	// Ensure default theme is extracted so it appears in the list.
	if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
		return err
	}

	themes, err := theme.ListThemes(cfg.ThemesDir)
	if err != nil {
		return err
	}

	if len(themes) == 0 {
		fmt.Fprintln(w, "No themes found.")
		return nil
	}

	for _, t := range themes {
		active := ""
		if t.Name == cfg.Theme {
			active = " (active)"
		}
		line := fmt.Sprintf("  %s%s", t.Name, active)
		if t.Parent != "" {
			line += fmt.Sprintf("  [parent: %s]", t.Parent)
		}
		if t.Description != "" {
			line += fmt.Sprintf("  — %s", t.Description)
		}
		fmt.Fprintln(w, line)
	}
	return nil
}
