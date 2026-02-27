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

func RunPluginInstall(_ context.Context, opts CLIOptions, srcPath string, name string) error {
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
