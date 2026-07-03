package build

import (
	"log/slog"
	"os"
	"strings"

	"osg/internal/config"
	"osg/internal/site"
)

type buildPlan struct {
	incremental    bool
	full           bool
	contentChanged bool
	changedFiles   map[string]bool
	removed        int
	removedFiles   []string
	prevOutputs    map[string]string
	reason         string

	// Smart incremental fields.
	templatesChanged bool            // at least one template file changed
	changedTemplates map[string]bool // set of changed template relative paths
	assetsOnly       bool            // only static/sass changed, skip HTML render

	// Dependency graph from previous build (for granular collection re-rendering).
	prevPageTemplates map[string]string   // source_path -> template_name
	prevSectionPages  map[string][]string // section_path -> [source_paths]
	prevPageOrder     []string            // newest-first source paths from previous build
	prevPageLinks     map[string][]string // source_path -> linked page paths from previous build

	// alsoRender is the neighbor-expansion set: pages whose own source did
	// not change but whose rendered prev/next, related, series or backlink
	// context references a changed page. Populated by renderPages.
	alsoRender map[string]bool
}

func (p buildPlan) shouldRenderPage(page *site.Page, outputPath string, templateName string) bool {
	if !p.incremental || p.full {
		return true
	}
	if page == nil || page.SourcePath == "" {
		return true
	}
	// Content changed for this specific page.
	if p.changedFiles != nil && p.changedFiles[page.SourcePath] {
		return true
	}
	// A neighboring page changed and this page's navigation shows it.
	if p.alsoRender != nil && p.alsoRender[page.SourcePath] {
		return true
	}
	// Template used by this page changed.
	if p.templatesChanged {
		if p.changedTemplates[templateName] {
			return true
		}
		// Shared partials affect all pages.
		if hasSharedPartialChanged(p.changedTemplates) {
			return true
		}
	}
	return outputMissing(outputPath)
}

// shouldRenderSection checks whether a section page needs re-rendering.
// It only returns true if a page within that section changed, or if the
// section/index template changed.
func (p buildPlan) shouldRenderSection(sectionPath, outputPath, templateName string) bool {
	if !p.incremental || p.full {
		return true
	}
	// Template changed for this section.
	if p.templatesChanged {
		if p.changedTemplates[templateName] {
			return true
		}
		if hasSharedPartialChanged(p.changedTemplates) {
			return true
		}
	}
	// Any page removed — we don't know from which section, so re-render all.
	if p.removed > 0 {
		return true
	}
	// Check if any page belonging to this section changed.
	if prevPages, ok := p.prevSectionPages[sectionPath]; ok {
		for _, src := range prevPages {
			if p.changedFiles[src] {
				return true
			}
		}
	} else {
		// No previous data for this section — could be new, must render.
		return true
	}
	return outputMissing(outputPath)
}

func (p buildPlan) shouldRenderCollection(outputPath string) bool {
	if !p.incremental || p.full || p.contentChanged {
		return true
	}
	// Template changes also require re-rendering collections (feeds, sitemap, etc.)
	if p.templatesChanged {
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

// hasSharedPartialChanged returns true if any template in partials/ changed.
// Partials can be included by any page template, so a partial change
// conservatively triggers a full re-render of all pages.
func hasSharedPartialChanged(changed map[string]bool) bool {
	for tpl := range changed {
		if strings.HasPrefix(tpl, "partials/") {
			return true
		}
	}
	return false
}

// diffTemplates compares per-file template hashes and returns the set of
// changed template names (relative paths like "page.html", "partials/head.html").
func diffTemplates(prev, current map[string]string) map[string]bool {
	changed := make(map[string]bool)
	if prev == nil {
		for k := range current {
			changed[k] = true
		}
		return changed
	}
	for k, hash := range current {
		if prevHash, ok := prev[k]; !ok || prevHash != hash {
			changed[k] = true
		}
	}
	// Templates removed in current (user deleted an override).
	for k := range prev {
		if _, ok := current[k]; !ok {
			changed[k] = true
		}
	}
	return changed
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

	// Config change → always full rebuild.
	if prev.ConfigHash != current.ConfigHash {
		plan.full = true
		plan.reason = "config changed"
		return plan, current
	}

	// Plugin change → always full rebuild (plugins can affect anything).
	if prev.PluginsHash != current.PluginsHash {
		plan.full = true
		plan.reason = "plugins changed"
		return plan, current
	}

	// Template change → smart rebuild (only affected pages).
	if prev.TemplatesHash != current.TemplatesHash {
		plan.templatesChanged = true
		plan.changedTemplates = diffTemplates(prev.Templates, current.Templates)
		if logger != nil {
			logger.Info("templates changed", "count", len(plan.changedTemplates))
		}
	}

	// Asset change → only reprocess assets if no other changes.
	assetsChanged := prev.AssetsHash != current.AssetsHash

	// Content diff.
	changed, removed := diffContent(prev.Content, current.Content)
	plan.changedFiles = changed
	plan.removedFiles = removed
	plan.removed = len(removed)
	plan.contentChanged = len(changed) > 0 || len(removed) > 0

	// Load dependency graph from previous build.
	plan.prevOutputs = prev.Outputs
	plan.prevPageTemplates = prev.PageTemplates
	plan.prevSectionPages = prev.SectionPages
	plan.prevPageOrder = prev.PageOrder
	plan.prevPageLinks = prev.PageLinks

	// Asset-only change: skip HTML rendering entirely.
	if assetsChanged && !plan.contentChanged && !plan.templatesChanged {
		plan.assetsOnly = true
		if logger != nil {
			logger.Info("asset-only change, skipping HTML render")
		}
	}

	return plan, current
}
