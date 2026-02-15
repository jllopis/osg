package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/tui"
)

func RunTUI(ctx context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}
	if opts.IncludeDrafts != nil {
		cfg.IncludeDrafts = *opts.IncludeDrafts
	}

	history, err := tui.NewHistory()
	if err != nil {
		history = nil
	}
	if history != nil {
		defer history.Close()
	}

	logSink := tui.NewLogSink(history)

	withLog := func() CLIOptions {
		clone := opts
		clone.LogWriter = logSink
		return clone
	}

	actions := tui.Actions{
		Init: func(actionCtx context.Context) error {
			return RunInit(actionCtx, withLog())
		},
		Update: func(actionCtx context.Context) error {
			return RunUpdateContent(actionCtx, withLog())
		},
		Build: func(actionCtx context.Context) error {
			return RunBuild(actionCtx, withLog())
		},
		Serve: func(actionCtx context.Context) error {
			return RunServe(actionCtx, withLog())
		},
		Doctor: func(actionCtx context.Context) error {
			return RunDoctor(actionCtx, withLog())
		},
		ThemeInit: func(actionCtx context.Context, name string) error {
			return RunThemeInit(actionCtx, withLog(), name)
		},
		PluginEnable: func(actionCtx context.Context, name string) error {
			return RunPluginEnable(actionCtx, withLog(), name)
		},
		PluginDisable: func(actionCtx context.Context, name string) error {
			return RunPluginDisable(actionCtx, withLog(), name)
		},
		PluginToggle: func(actionCtx context.Context, name string) error {
			return RunPluginToggle(actionCtx, withLog(), name)
		},
		PluginInstall: func(actionCtx context.Context, path string, name string) error {
			return RunPluginInstall(actionCtx, withLog(), path, name)
		},
		PluginList: func(actionCtx context.Context) error {
			return RunPluginList(actionCtx, withLog(), logSink)
		},
		PluginInit: func(actionCtx context.Context, name string, dir string) error {
			return RunPluginInit(actionCtx, withLog(), name, dir)
		},
		NewPost: func(actionCtx context.Context, title string) error {
			postOpts := NewPostOptions{Title: title}
			return RunNew(actionCtx, withLog(), postOpts)
		},
		Version: func() string {
			return VersionInfo()
		},
	}

	options := tui.Options{
		ConfigPath:     opts.ConfigPath,
		VaultPath:      cfg.VaultPath,
		ContentDir:     cfg.ContentDir,
		PublicDir:      cfg.PublicDir,
		ServeAddr:      opts.ServeAddr,
		LogPath:        historyPath(history),
		SiteTitle:      cfg.SiteTitle,
		PrefixKey:      cfg.TUIPrefix,
		PrefixMs:       cfg.TUIPrefixMs,
		Plugins:        listPlugins(cfg.PluginsDir),
		EnabledPlugins: cfg.PluginsEnabled,
		HasContent:     pathExists(cfg.ContentDir),
	}

	return tui.Run(ctx, actions, options, logSink, history)
}

func listPlugins(dir string) []string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	plugins := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".wasm") {
			continue
		}
		plugins = append(plugins, strings.TrimSuffix(name, filepath.Ext(name)))
	}

	sort.Strings(plugins)
	return plugins
}

func historyPath(history *tui.History) string {
	if history == nil {
		return ""
	}
	return history.Path()
}
