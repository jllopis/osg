package build

import (
	"log/slog"
	"os"

	"osg/internal/config"
	"osg/internal/site"
)

type buildPlan struct {
	incremental    bool
	full           bool
	contentChanged bool
	changedFiles   map[string]bool
	removed        int
	reason         string
}

func (p buildPlan) shouldRenderPage(page *site.Page, outputPath string) bool {
	if !p.incremental || p.full {
		return true
	}
	if page == nil {
		return true
	}
	if page.SourcePath == "" {
		return true
	}
	if p.changedFiles != nil && p.changedFiles[page.SourcePath] {
		return true
	}
	return outputMissing(outputPath)
}

func (p buildPlan) shouldRenderCollection(outputPath string) bool {
	if !p.incremental || p.full || p.contentChanged {
		return true
	}
	return outputMissing(outputPath)
}

func outputMissing(path string) bool {
	if path == "" {
		return true
	}
	if _, err := os.Stat(path); err != nil {
		return true
	}
	return false
}

func buildPlanFromCache(cfg config.Config, files []string, logger *slog.Logger) (buildPlan, *buildCache) {
	plan := buildPlan{incremental: cfg.BuildIncremental}
	if !cfg.BuildIncremental {
		plan.full = true
		plan.reason = "incremental disabled"
		return plan, nil
	}

	cachePath := buildCachePath(cfg)
	current, err := buildCacheFrom(cfg, files)
	if err != nil {
		plan.full = true
		plan.reason = "cache snapshot failed"
		if logger != nil {
			logger.Warn("cache snapshot failed", "error", err)
		}
		return plan, nil
	}

	prev, err := loadBuildCache(cachePath)
	if err != nil {
		plan.full = true
		plan.reason = "cache load failed"
		if logger != nil {
			logger.Warn("cache load failed", "error", err)
		}
		return plan, current
	}

	if prev == nil {
		plan.full = true
		plan.reason = "no cache"
		return plan, current
	}
	if prev.Version != buildCacheVersion {
		plan.full = true
		plan.reason = "cache version"
		return plan, current
	}

	if prev.ConfigHash != current.ConfigHash {
		plan.full = true
		plan.reason = "config changed"
		return plan, current
	}
	if prev.TemplatesHash != current.TemplatesHash {
		plan.full = true
		plan.reason = "templates changed"
		return plan, current
	}
	if prev.AssetsHash != current.AssetsHash {
		plan.full = true
		plan.reason = "assets changed"
		return plan, current
	}
	if prev.PluginsHash != current.PluginsHash {
		plan.full = true
		plan.reason = "plugins changed"
		return plan, current
	}

	changed, removed := diffContent(prev.Content, current.Content)
	plan.changedFiles = changed
	plan.removed = removed
	plan.contentChanged = len(changed) > 0 || removed > 0

	return plan, current
}
