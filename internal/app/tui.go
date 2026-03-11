package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"osg/internal/build"
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

	// Three log sinks: general (for init/update/build/doctor), serve, api.
	generalSink := tui.NewLogSink("general", history)
	serveSink := tui.NewLogSink("serve", history)
	apiSink := tui.NewLogSink("api", history)

	withLog := func(sink *tui.LogSink) CLIOptions {
		clone := opts
		clone.LogWriter = sink
		return clone
	}

	actions := tui.Actions{
		Init: func(actionCtx context.Context) error {
			return RunInit(actionCtx, withLog(generalSink))
		},
		Update: func(actionCtx context.Context) error {
			return RunUpdateContent(actionCtx, withLog(generalSink))
		},
		Build: func(actionCtx context.Context) error {
			return RunBuild(actionCtx, withLog(generalSink))
		},
		Serve: func(actionCtx context.Context) error {
			return RunServe(actionCtx, withLog(serveSink))
		},
		ServeWithAPI: func(actionCtx context.Context) error {
			o := withLog(serveSink)
			o.ServeAPI = true
			return RunServe(actionCtx, o)
		},
		RunAPI: func(actionCtx context.Context) error {
			apiOpts := APIOptions{Listen: cfg.Interactions.Listen}
			return RunAPI(actionCtx, withLog(apiSink), apiOpts)
		},
		Check: func(actionCtx context.Context) error {
			return RunCheck(actionCtx, withLog(generalSink), CheckOptions{})
		},
		Doctor: func(actionCtx context.Context) error {
			return RunDoctor(actionCtx, withLog(generalSink))
		},
		ThemeInit: func(actionCtx context.Context, name string, parent string) error {
			return RunThemeInit(actionCtx, withLog(generalSink), name, parent)
		},
		ThemeList: func(actionCtx context.Context) error {
			return RunThemeList(actionCtx, withLog(generalSink), generalSink)
		},
		PluginEnable: func(actionCtx context.Context, name string) error {
			return RunPluginEnable(actionCtx, withLog(generalSink), name)
		},
		PluginDisable: func(actionCtx context.Context, name string) error {
			return RunPluginDisable(actionCtx, withLog(generalSink), name)
		},
		PluginToggle: func(actionCtx context.Context, name string) error {
			return RunPluginToggle(actionCtx, withLog(generalSink), name)
		},
		PluginInstall: func(actionCtx context.Context, path string, name string) error {
			return RunPluginInstall(actionCtx, withLog(generalSink), path, name)
		},
		PluginList: func(actionCtx context.Context) error {
			return RunPluginList(actionCtx, withLog(generalSink), generalSink)
		},
		PluginInit: func(actionCtx context.Context, name string, dir string, lang string) error {
			return RunPluginInit(actionCtx, withLog(generalSink), name, dir, lang)
		},
		PluginSearch: func(actionCtx context.Context, query string) error {
			return RunPluginSearch(actionCtx, withLog(generalSink), query, generalSink)
		},
		PluginUpdate: func(actionCtx context.Context, name string) error {
			return RunPluginUpdate(actionCtx, withLog(generalSink), name, generalSink)
		},
		NewPost: func(actionCtx context.Context, title string) error {
			postOpts := NewPostOptions{Title: title}
			return RunNew(actionCtx, withLog(generalSink), postOpts)
		},
		Stats: func() (string, error) {
			s, err := build.ComputeStats(cfg)
			if err != nil {
				return "", err
			}
			return build.FormatStats(s), nil
		},
		Version: VersionInfo,
	}

	options := tui.Options{
		ConfigPath:     opts.ConfigPath,
		VaultPath:      cfg.VaultPath,
		ContentDir:     cfg.ContentDir,
		PublicDir:      cfg.PublicDir,
		ServeAddr:      opts.ServeAddr,
		APIAddr:        cfg.Interactions.Listen,
		LogPath:        historyPath(history),
		SiteTitle:      cfg.SiteTitle,
		PrefixKey:      cfg.TUIPrefix,
		PrefixMs:       cfg.TUIPrefixMs,
		LogModifier:    cfg.TUILogModifier,
		Plugins:        listPlugins(cfg.PluginsDir),
		EnabledPlugins: cfg.PluginsEnabled,
		HasContent:     pathExists(cfg.ContentDir),
	}

	return tui.Run(ctx, actions, options, history, generalSink, serveSink, apiSink)
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
