package ui

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"osg/internal/build"
	"osg/internal/config"
	"osg/internal/plugin"
	"osg/internal/site"
	"osg/internal/vault"
)

// PageView is a flattened, template-friendly view of a site.Page.
type PageView struct {
	Title   string
	Path    string
	Section string
	Date    time.Time
	Draft   bool
}

// MonthlyBar adds a precomputed Width (0..100) on top of build.MonthlyStats
// so templates don't have to compute it.
type MonthlyBar struct {
	Month string
	Count int
	Width int // percentage (0..100)
}

// PluginEntry is the unified view used by the /plugins page: every name
// found either in plugins_dir or in plugins_enabled, with metadata if it
// loaded successfully.
type PluginEntry struct {
	Name     string
	Enabled  bool               // present in cfg.PluginsEnabled
	Loaded   bool               // metadata is non-zero (load succeeded)
	Metadata *plugin.PluginMeta // nil if not loaded
}

// State is the snapshot of OSG state surfaced to the UI templates.
type State struct {
	Stats      *build.SiteStats
	Plugins    []PluginEntry
	Pages      []PageView
	Monthly    []MonthlyBar
	ContentDir string
}

// Collect gathers stats, pages and plugin metadata. Errors are logged but
// not returned: a partial UI is more useful than no UI at all.
func Collect(ctx context.Context, cfg config.Config, logger *slog.Logger) State {
	st := State{ContentDir: cfg.ContentDir}

	stats, err := build.ComputeStats(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("compute stats failed", "error", err)
		}
		stats = &build.SiteStats{}
	}
	st.Stats = stats
	st.Monthly = bars(stats.Monthly)

	st.Pages = collectPages(cfg, logger)

	st.Plugins = collectPlugins(ctx, cfg, logger)
	return st
}

// collectPlugins returns the unified plugin list (loaded + on-disk +
// enabled-but-missing) so the UI can show a complete picture.
func collectPlugins(ctx context.Context, cfg config.Config, logger *slog.Logger) []PluginEntry {
	enabledSet := make(map[string]bool, len(cfg.PluginsEnabled))
	for _, name := range cfg.PluginsEnabled {
		enabledSet[strings.TrimSuffix(strings.TrimSpace(name), ".wasm")] = true
	}

	// Discover .wasm files on disk so disabled plugins appear too.
	available := map[string]bool{}
	if cfg.PluginsDir != "" {
		entries, err := os.ReadDir(cfg.PluginsDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".wasm") {
					continue
				}
				available[strings.TrimSuffix(filepath.Base(e.Name()), filepath.Ext(e.Name()))] = true
			}
		}
	}

	// Load enabled plugins to capture metadata.
	loadedMeta := map[string]plugin.PluginMeta{}
	mgr, err := plugin.Load(ctx, cfg.PluginsDir, cfg.PluginsEnabled, cfg.PluginTimeout, logger)
	if err != nil {
		if logger != nil {
			logger.Warn("plugin load failed", "error", err)
		}
	} else {
		for _, m := range mgr.Metadata() {
			loadedMeta[m.Name] = m
		}
		_ = mgr.Close(ctx)
	}

	// Union of names across all sources.
	names := map[string]bool{}
	for n := range enabledSet {
		names[n] = true
	}
	for n := range available {
		names[n] = true
	}
	for n := range loadedMeta {
		names[n] = true
	}

	out := make([]PluginEntry, 0, len(names))
	for name := range names {
		entry := PluginEntry{
			Name:    name,
			Enabled: enabledSet[name],
		}
		if m, ok := loadedMeta[name]; ok {
			meta := m
			entry.Metadata = &meta
			entry.Loaded = true
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func collectPages(cfg config.Config, logger *slog.Logger) []PageView {
	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		if logger != nil {
			logger.Warn("list content failed", "error", err)
		}
		return nil
	}
	out := make([]PageView, 0, len(files))
	for _, fp := range files {
		page, _, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, fp)
		if err != nil || page == nil {
			continue
		}
		out = append(out, PageView{
			Title:   page.Title,
			Path:    page.Path,
			Section: sectionOf(page.Path),
			Date:    page.Date,
			Draft:   page.Draft,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date.Equal(out[j].Date) {
			return out[i].Title < out[j].Title
		}
		return out[i].Date.After(out[j].Date)
	})
	return out
}

func sectionOf(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) <= 1 {
		return "(root)"
	}
	return parts[0]
}

func bars(monthly []build.MonthlyStats) []MonthlyBar {
	if len(monthly) == 0 {
		return nil
	}
	maxCount := 0
	for _, m := range monthly {
		if m.Count > maxCount {
			maxCount = m.Count
		}
	}
	if maxCount <= 0 {
		maxCount = 1
	}
	out := make([]MonthlyBar, len(monthly))
	for i, m := range monthly {
		out[i] = MonthlyBar{
			Month: m.Month,
			Count: m.Count,
			Width: (m.Count * 100) / maxCount,
		}
	}
	return out
}
