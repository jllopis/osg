package ui

import (
	"context"
	"log/slog"
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

// State is the snapshot of OSG state surfaced to the UI templates.
type State struct {
	Stats      *build.SiteStats
	Plugins    []plugin.PluginMeta
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

	mgr, err := plugin.Load(ctx, cfg.PluginsDir, cfg.PluginsEnabled, cfg.PluginTimeout, logger)
	if err != nil {
		if logger != nil {
			logger.Warn("plugin load failed", "error", err)
		}
	} else {
		st.Plugins = mgr.Metadata()
		// Best-effort close to release the wazero runtime.
		_ = mgr.Close(ctx)
	}

	return st
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
