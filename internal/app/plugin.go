package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/plugin"
)

type PluginInfo struct {
	Name    string
	Enabled bool
	Path    string
}

func RunPluginInstall(ctx context.Context, opts CLIOptions, srcPath string, name string) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(srcPath) == "" {
		return fmt.Errorf("plugin path is required")
	}
	if strings.TrimSpace(cfg.PluginsDir) == "" {
		return fmt.Errorf("plugins dir is not configured")
	}

	// GitHub repository reference: github.com/user/repo[@tag]
	if plugin.IsGitHubRef(srcPath) {
		owner, repo, tag := plugin.ParseGitHubRef(srcPath)
		installedName, err := plugin.InstallFromGitHub(ctx, owner, repo, tag, cfg.PluginsDir)
		if err != nil {
			return fmt.Errorf("install from GitHub: %w", err)
		}
		logger := logging.New(cfg.Logging, opts.Verbose)
		logger.Info("installed plugin from GitHub", "name", installedName, "source", srcPath)
		return nil
	}

	// Local file path.
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("plugin path is a directory")
	}

	if name == "" {
		name = strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name is empty")
	}

	if filepath.Ext(name) == "" {
		name += ".wasm"
	}

	if err := os.MkdirAll(cfg.PluginsDir, 0o755); err != nil {
		return err
	}

	dest := filepath.Join(cfg.PluginsDir, name)
	return copyFile(srcPath, dest)
}

func RunPluginEnable(_ context.Context, opts CLIOptions, name string) error {
	return updatePluginState(opts, name, true)
}

func RunPluginDisable(_ context.Context, opts CLIOptions, name string) error {
	return updatePluginState(opts, name, false)
}

func RunPluginToggle(_ context.Context, opts CLIOptions, name string) error {
	name = normalizePluginName(name)
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}
	enabled := map[string]bool{}
	for _, plugin := range cfg.PluginsEnabled {
		enabled[plugin] = true
	}
	_, on := enabled[name]
	return updatePluginState(opts, name, !on)
}

func RunPluginList(ctx context.Context, opts CLIOptions, out io.Writer) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}
	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, out)

	installed, err := listInstalledPlugins(cfg.PluginsDir)
	if err != nil {
		return err
	}
	enabled := map[string]bool{}
	for _, name := range cfg.PluginsEnabled {
		enabled[name] = true
	}

	if len(installed) == 0 {
		logger.Info("no plugins installed")
		return nil
	}

	// Load plugins to read optional metadata (plugin_info export).
	mgr, loadErr := plugin.Load(ctx, cfg.PluginsDir, cfg.PluginsEnabled, 0, nil)
	metaMap := map[string]plugin.PluginMeta{}
	if loadErr == nil && mgr != nil {
		defer mgr.Close(ctx)
		for _, m := range mgr.Metadata() {
			metaMap[m.Name] = m
		}
	}

	for _, p := range installed {
		state := "disabled"
		if enabled[p.Name] {
			state = "enabled"
		}
		attrs := []any{"name", p.Name, "state", state}
		if meta, ok := metaMap[p.Name]; ok {
			if meta.Version != "" {
				attrs = append(attrs, "version", meta.Version)
			}
			if meta.Description != "" {
				attrs = append(attrs, "description", meta.Description)
			}
		}
		attrs = append(attrs, "path", p.Path)
		logger.Info("plugin", attrs...)
	}
	return nil
}

func RunPluginSearch(ctx context.Context, opts CLIOptions, query string, out io.Writer) error {
	index, err := plugin.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetch plugin index: %w", err)
	}

	results := plugin.SearchIndex(index, query)
	if len(results) == 0 {
		fmt.Fprintln(out, "No plugins found.")
		return nil
	}

	for _, e := range results {
		line := e.Name
		if e.Version != "" {
			line += " (" + e.Version + ")"
		}
		if e.Description != "" {
			line += " — " + e.Description
		}
		fmt.Fprintln(out, line)
		if e.Repo != "" {
			fmt.Fprintf(out, "  install: osg plugin install %s\n", e.Repo)
		}
	}
	return nil
}

func RunPluginUpdate(ctx context.Context, opts CLIOptions, name string, out io.Writer) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	lock, err := plugin.LoadLockFile(".")
	if err != nil {
		return fmt.Errorf("load lock file: %w", err)
	}

	names := []string{name}
	if strings.TrimSpace(name) == "" {
		names = lock.Names()
	}

	if len(names) == 0 {
		fmt.Fprintln(out, "No plugins tracked in lock file. Install plugins from GitHub to enable updates.")
		return nil
	}

	for _, n := range names {
		newVersion, err := plugin.CheckUpdate(ctx, n, lock)
		if err != nil {
			fmt.Fprintf(out, "%s: %v\n", n, err)
			continue
		}
		if newVersion == "" {
			fmt.Fprintf(out, "%s: up to date\n", n)
			continue
		}

		entry, _ := lock.Get(n)
		fmt.Fprintf(out, "%s: %s -> %s, updating...\n", n, entry.Version, newVersion)

		_, err = plugin.UpdatePlugin(ctx, n, cfg.PluginsDir, lock)
		if err != nil {
			fmt.Fprintf(out, "%s: update failed: %v\n", n, err)
			continue
		}
		fmt.Fprintf(out, "%s: updated to %s\n", n, newVersion)
	}
	return nil
}

func RunPluginInit(_ context.Context, opts CLIOptions, name string, baseDir string, lang string) error {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "plugins_src"
	}
	return plugin.Scaffold(baseDir, name, lang)
}

func updatePluginState(opts CLIOptions, name string, enable bool) error {
	name = normalizePluginName(name)
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	enabled := map[string]bool{}
	for _, plugin := range cfg.PluginsEnabled {
		enabled[plugin] = true
	}

	if enable {
		enabled[name] = true
	} else {
		delete(enabled, name)
	}

	names := make([]string, 0, len(enabled))
	for plugin := range enabled {
		names = append(names, plugin)
	}
	sort.Strings(names)

	return config.UpdatePluginsEnabled(opts.ConfigPath, names)
}

func normalizePluginName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".wasm") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return strings.TrimSpace(name)
}

func listInstalledPlugins(dir string) ([]PluginInfo, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := []PluginInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".wasm") {
			continue
		}
		path := filepath.Join(dir, name)
		out = append(out, PluginInfo{
			Name: strings.TrimSuffix(name, filepath.Ext(name)),
			Path: path,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func copyFile(src string, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
