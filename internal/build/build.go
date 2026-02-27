package build

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"osg/internal/assets"
	"osg/internal/config"
	"osg/internal/i18n"
	imgopt "osg/internal/image"
	"osg/internal/logging"
	"osg/internal/markdown"
	"osg/internal/placeholder"
	"osg/internal/plugin"
	"osg/internal/render"
	"osg/internal/site"
	"osg/internal/slug"
	"osg/internal/summary"
	"osg/internal/taxonomy"
	"osg/internal/theme"
	"osg/internal/vault"
)

type Stats struct {
	Total    int
	Rendered int
	Skipped  int
	Cached   int
	Errors   int
}

// BuildOptions carries per-invocation settings that are not part of the
// persistent Config.  Zero value is safe (all features enabled, no force).
type BuildOptions struct {
	// SkipAI disables LLM-based summary generation.  When true, AI
	// summaries are not called and pages fall back to the "auto" strategy.
	// Used during serve to avoid costly API calls on every rebuild.
	SkipAI bool
	// ForceAISummaries bypasses the AI summary cache and regenerates all
	// summaries even when cached results exist.
	ForceAISummaries bool
	// Progress is an optional user-facing progress indicator (spinner).
	// When non-nil, long operations like AI summary generation update it.
	Progress logging.Progress
}

func Run(ctx context.Context, cfg config.Config, opts BuildOptions, verbose bool, logWriter io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := logging.NewWithWriter(cfg.Logging, verbose, logWriter)

	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no content files found", "content_dir", cfg.ContentDir)
		return nil
	}

	plan, cacheToSave := buildPlanFromCache(cfg, files, logger)
	if plan.incremental {
		if plan.full {
			logger.Info("build incremental", "mode", "full", "reason", plan.reason)
		} else {
			logger.Info("build incremental", "mode", "partial", "changed", len(plan.changedFiles), "removed", plan.removed)
		}
	}

	if cfg.CleanPublic && (plan.full || plan.removed > 0) {
		if plan.full {
			if err := os.RemoveAll(cfg.PublicDir); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("clean public dir: %w", err)
			}
			logger.Info("public cleaned", "dir", cfg.PublicDir)
		} else if plan.removed > 0 {
			removedCount := cleanupRemovedOutputs(cfg.PublicDir, plan.removedFiles, plan.prevOutputs, logger)
			if removedCount > 0 {
				logger.Info("public cleaned (partial)", "removed", removedCount)
			}
		}
	}

	if err := os.MkdirAll(cfg.PublicDir, 0o755); err != nil {
		return fmt.Errorf("create public dir: %w", err)
	}

	if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
		return fmt.Errorf("ensure default theme: %w", err)
	}

	// Load i18n translations: theme dir first, then user dir (user overrides).
	i18nBundle := i18n.New(cfg.DefaultLanguage)
	themeI18nDir := themeI18nDir(cfg)
	if themeI18nDir != "" {
		if err := i18nBundle.LoadDir(themeI18nDir); err != nil {
			return fmt.Errorf("load theme translations: %w", err)
		}
	}
	userI18nDir := filepath.Join("i18n")
	if err := i18nBundle.LoadDir(userI18nDir); err != nil {
		return fmt.Errorf("load user translations: %w", err)
	}

	if err := assets.Prepare(cfg, logger); err != nil {
		return err
	}

	// Load plugins early so they are available for all pipeline phases.
	if err := plugin.EnsureBundledPlugins(cfg.PluginsDir); err != nil {
		logger.Warn("failed to extract bundled plugins", "error", err)
	}
	plugins, err := plugin.Load(ctx, cfg.PluginsDir, cfg.PluginsEnabled, cfg.PluginTimeout, logger)
	if err != nil {
		logger.Warn("plugins disabled", "error", err)
		plugins = nil
	}
	if plugins != nil {
		defer func() {
			if err := plugins.Close(ctx); err != nil {
				logger.Warn("failed to close plugins", "error", err)
			}
		}()
	}

	siteIndex := site.New()
	stats := Stats{Total: len(files)}

	for _, filePath := range files {
		page, section, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, filePath)
		if err != nil {
			logger.Warn("failed to parse content", "path", filePath, "error", err)
			stats.Errors++
			continue
		}

		if page != nil {
			if page.Draft && !cfg.IncludeDrafts {
				stats.Skipped++
				continue
			}
			siteIndex.AddPage(page)
		}

		if section != nil {
			siteIndex.AddSection(section)
		}
	}

	siteIndex.BuildHierarchy()

	// content.transform: let plugins modify Markdown before rendering.
	applyContentTransform(ctx, plugins, cfg, siteIndex, logger)

	// Fill empty Page.Summary fields using the configured strategy.
	// Returns source paths of pages whose summary changed, so the
	// incremental build plan can force re-rendering those pages (and
	// their containing collections).
	summaryChanged := fillSummaries(ctx, cfg, opts, siteIndex, logger)
	if len(summaryChanged) > 0 {
		if plan.changedFiles == nil {
			plan.changedFiles = make(map[string]bool)
		}
		for _, sp := range summaryChanged {
			plan.changedFiles[sp] = true
		}
		plan.contentChanged = true
	}

	// Generate placeholder SVGs for pages without an image.
	if err := generatePlaceholders(siteIndex, cfg.PublicDir, logger); err != nil {
		return fmt.Errorf("generate placeholders: %w", err)
	}

	// Optimize images: generate responsive variants (resized JPEG + WebP).
	var imageResults map[string]*imgopt.Result
	if cfg.ImageOptimization {
		opts := imgopt.Options{
			Quality: cfg.ImageQuality,
			Widths:  cfg.ImageWidths,
			WebP:    true,
		}
		if opts.Quality <= 0 || opts.Quality > 100 {
			opts.Quality = 80
		}
		if len(opts.Widths) == 0 {
			opts.Widths = []int{640, 1200}
		}
		var err error
		imageResults, err = imgopt.Optimize(cfg.PublicDir, opts, logger)
		if err != nil {
			logger.Warn("image optimization failed", "error", err)
			// Non-fatal: continue with no optimized variants.
		}
	}

	// image.process: let plugins transform images via WASI filesystem.
	emitImageProcess(ctx, plugins, cfg, imageResults, logger)

	indices := taxonomy.Build(cfg.Taxonomies, siteIndex.Pages, cfg.BaseURL)
	siteView := siteIndex.View()
	baseCtx := baseContext(cfg, siteView, indices, siteIndex.MenuPages())

	// config.validate: plugins can validate config and return errors/warnings.
	if err := emitConfigValidate(ctx, plugins, cfg, logger); err != nil {
		return err
	}

	if plugins != nil {
		startPayload := cloneMap(baseCtx)
		startPayload["stats"] = buildStatsView(stats, siteIndex)
		if overrides := plugins.Emit(ctx, "build.started", startPayload); overrides != nil {
			plugin.Merge(baseCtx, overrides)
		}
	}

	renderer, err := render.New(cfg.TemplatesDir, themeTemplatesDir(cfg), render.Context{
		BaseURL:         cfg.BaseURL,
		ContentDir:      cfg.ContentDir,
		StaticDir:       cfg.StaticDir,
		PublicDir:       cfg.PublicDir,
		DefaultLanguage: cfg.DefaultLanguage,
		Site:            siteIndex,
		Taxonomies:      indices,
		ImageResults:    imageResults,
		I18n:            i18nBundle,
	})
	if err != nil {
		return err
	}

	renderedSections, cachedSections, err := renderSections(ctx, renderer, cfg, baseCtx, siteIndex, plugins, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += renderedSections
	stats.Cached += cachedSections

	renderedPages, cachedPages, err := renderPages(ctx, renderer, cfg, baseCtx, siteIndex, indices, plugins, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += renderedPages
	stats.Cached += cachedPages

	taxonomyRendered, taxonomyCached, err := renderTaxonomies(ctx, renderer, cfg, baseCtx, siteIndex, indices, plugins, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += taxonomyRendered
	stats.Cached += taxonomyCached

	siteFeedRendered, siteFeedCached, err := renderSiteFeed(renderer, cfg, baseCtx, siteIndex, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += siteFeedRendered
	stats.Cached += siteFeedCached

	sitemapEntries := collectSitemapEntries(cfg, siteIndex, indices)
	sitemapRendered, sitemapCached, err := renderSitemap(renderer, cfg, baseCtx, sitemapEntries, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += sitemapRendered
	stats.Cached += sitemapCached

	robotsRendered, robotsCached, err := renderRobots(renderer, cfg, baseCtx, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += robotsRendered
	stats.Cached += robotsCached

	notFoundRendered, notFoundCached, err := renderNotFound(renderer, cfg, baseCtx, plan)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += notFoundRendered
	stats.Cached += notFoundCached

	// Post-render: minify HTML, CSS, JS, JSON, SVG, XML files in public/.
	if cfg.Minify {
		minified, err := minifyDir(cfg.PublicDir, logger)
		if err != nil {
			logger.Warn("minification walk failed", "error", err)
		} else {
			logger.Info("minified output files", "count", minified)
		}
	}

	if plugins != nil {
		finishPayload := cloneMap(baseCtx)
		finishPayload["stats"] = buildStatsView(stats, siteIndex)
		_ = plugins.Emit(ctx, "build.finished", finishPayload)
	}

	logger.Info("build summary",
		"total", stats.Total,
		"rendered", stats.Rendered,
		"skipped", stats.Skipped,
		"cached", stats.Cached,
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	if cacheToSave != nil && stats.Errors == 0 {
		cacheToSave.Outputs = buildOutputsIndex(siteIndex, cfg.PublicDir)
		if err := saveBuildCache(buildCachePath(cfg), cacheToSave); err != nil {
			logger.Warn("cache write failed", "error", err)
		}
	}

	// after.build: emitted after all output is finalized (post build.finished,
	// post cache save). Plugins can use this for deploy, notifications, etc.
	if plugins != nil {
		afterPayload := cloneMap(baseCtx)
		afterPayload["stats"] = buildStatsView(stats, siteIndex)
		afterPayload["public_dir"] = cfg.PublicDir
		_ = plugins.Emit(ctx, "after.build", afterPayload)
	}

	return nil
}

func buildOutputsIndex(siteIndex *site.Site, publicDir string) map[string]string {
	if siteIndex == nil {
		return nil
	}
	outputs := map[string]string{}
	for _, page := range siteIndex.Pages {
		if page.SourcePath == "" {
			continue
		}
		outputs[page.SourcePath] = outputHTMLPath(publicDir, page.Path)
	}
	for _, section := range siteIndex.Sections {
		if section == nil || section.SourcePath == "" {
			continue
		}
		outputs[section.SourcePath] = outputHTMLPath(publicDir, section.Path)
	}
	return outputs
}

func cleanupRemovedOutputs(publicDir string, removed []string, outputs map[string]string, logger *slog.Logger) int {
	if len(removed) == 0 || len(outputs) == 0 {
		return 0
	}
	count := 0
	for _, source := range removed {
		output, ok := outputs[source]
		if !ok || output == "" {
			continue
		}
		if !strings.HasPrefix(filepath.Clean(output), filepath.Clean(publicDir)) {
			if logger != nil {
				logger.Warn("skip output cleanup outside public dir", "output", output)
			}
			continue
		}
		if err := os.Remove(output); err == nil {
			count++
			removeEmptyParents(publicDir, output)
		}
	}
	return count
}

func removeEmptyParents(root string, leaf string) {
	root = filepath.Clean(root)
	dir := filepath.Dir(leaf)
	for {
		if filepath.Clean(dir) == root || dir == "." || dir == string(filepath.Separator) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func renderPages(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, indices map[string]*taxonomy.Index, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
	// Build chronological navigation index (excluding menu pages).
	postPages := make([]*site.Page, 0, len(siteIndex.Pages))
	for _, p := range siteIndex.Pages {
		if !p.Menu {
			postPages = append(postPages, p)
		}
	}
	pagePos := make(map[*site.Page]int, len(postPages))
	for i, p := range postPages {
		pagePos[p] = i
	}

	rendered := 0
	cached := 0
	for _, page := range siteIndex.Pages {
		templateName := page.Template
		if templateName == "" {
			templateName = "page.html"
		}

		outputPath := outputHTMLPath(cfg.PublicDir, page.Path)
		if !plan.shouldRenderPage(page, outputPath) {
			cached++
			continue
		}
		renderCtx := pageContext(baseCtx, page)

		// Prev/next chronological navigation (sorted newest-first).
		if idx, ok := pagePos[page]; ok {
			if idx > 0 {
				renderCtx["next_page"] = postPages[idx-1].View() // newer
			}
			if idx < len(postPages)-1 {
				renderCtx["prev_page"] = postPages[idx+1].View() // older
			}
		}

		// Table of Contents from rendered HTML headings.
		if pageView, ok := renderCtx["page"].(map[string]any); ok {
			if content, ok := pageView["content"].(template.HTML); ok {
				if tocEntries := markdown.ExtractTOC(string(content)); len(tocEntries) > 0 {
					renderCtx["toc"] = markdown.TOCView(tocEntries)
				}
			}
		}

		// Related posts by shared taxonomy terms.
		if related := relatedPages(page, indices, 3); len(related) > 0 {
			views := make([]map[string]any, 0, len(related))
			for _, rp := range related {
				views = append(views, rp.View())
			}
			renderCtx["related_posts"] = views
		}

		renderCtx = applyPluginOverrides(ctx, plugins, "page.render", renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}
	return rendered, cached, nil
}

// relatedPages returns up to limit pages that share taxonomy terms with page,
// scored by how many terms they share (descending), then by date (newest first).
func relatedPages(page *site.Page, indices map[string]*taxonomy.Index, limit int) []*site.Page {
	if len(page.Taxonomies) == 0 || len(indices) == 0 {
		return nil
	}

	scores := map[*site.Page]int{}
	for kind, terms := range page.Taxonomies {
		index, ok := indices[kind]
		if !ok {
			continue
		}
		for _, termName := range terms {
			termSlug := slug.Slugify(strings.TrimSpace(termName))
			if termSlug == "" {
				continue
			}
			term, ok := index.Terms[termSlug]
			if !ok {
				continue
			}
			for _, p := range term.Pages {
				if p != page && !p.Menu {
					scores[p]++
				}
			}
		}
	}

	if len(scores) == 0 {
		return nil
	}

	candidates := make([]*site.Page, 0, len(scores))
	for p := range scores {
		candidates = append(candidates, p)
	}
	sort.Slice(candidates, func(i, j int) bool {
		si, sj := scores[candidates[i]], scores[candidates[j]]
		if si != sj {
			return si > sj
		}
		return candidates[i].Date.After(candidates[j].Date)
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func renderSections(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
	rendered := 0
	cached := 0
	for _, section := range siteIndex.Sections {
		templateName := section.Template
		if templateName == "" {
			if section.IsRoot {
				templateName = "index.html"
			} else {
				templateName = "section.html"
			}
		}

		outputPath := outputHTMLPath(cfg.PublicDir, section.Path)
		if !plan.shouldRenderCollection(outputPath) {
			cached++
			continue
		}
		renderCtx := sectionContext(baseCtx, section)
		renderCtx = applyPluginOverrides(ctx, plugins, "section.render", renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}
	return rendered, cached, nil
}

func outputHTMLPath(publicDir string, sitePath string) string {
	rel := strings.TrimPrefix(sitePath, "/")
	if rel == "" {
		return filepath.Join(publicDir, "index.html")
	}
	if strings.HasSuffix(rel, "/") {
		return filepath.Join(publicDir, rel, "index.html")
	}
	return filepath.Join(publicDir, rel, "index.html")
}

func pageContext(baseCtx map[string]any, page *site.Page) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["page"] = page.View()
	ctx["current_path"] = page.Path
	ctx["current_url"] = page.Permalink
	ctx["lang"] = page.Lang
	if page.Lang == "" {
		ctx["lang"] = defaultLangFromCtx(baseCtx)
	}
	return ctx
}

func sectionContext(baseCtx map[string]any, section *site.Section) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["section"] = section.View()
	ctx["current_path"] = section.Path
	ctx["current_url"] = section.Permalink
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	return ctx
}

func baseContext(cfg config.Config, siteView map[string]any, indices map[string]*taxonomy.Index, menuPages []*site.Page) map[string]any {
	ctx := map[string]any{
		"config": configView(cfg),
		"site":   siteView,
	}
	if len(indices) > 0 {
		ctx["taxonomies"] = taxonomiesView(indices)
	}
	if len(menuPages) > 0 {
		views := make([]map[string]any, 0, len(menuPages))
		for _, p := range menuPages {
			views = append(views, p.View())
		}
		ctx["menu_pages"] = views
	}
	return ctx
}

func configView(cfg config.Config) map[string]any {
	taxonomies := make([]map[string]any, 0, len(cfg.Taxonomies))
	for _, taxCfg := range cfg.Taxonomies {
		taxonomies = append(taxonomies, taxonomy.ConfigView(taxCfg))
	}

	absPublicDir, _ := filepath.Abs(cfg.PublicDir)

	return map[string]any{
		"base_url":           cfg.BaseURL,
		"site_title":         cfg.SiteTitle,
		"site_description":   cfg.SiteDescription,
		"theme":              cfg.Theme,
		"color_scheme":       cfg.ColorScheme,
		"default_language":   cfg.DefaultLanguage,
		"vault_path":         cfg.VaultPath,
		"content_dir":        cfg.ContentDir,
		"public_dir":         absPublicDir,
		"templates_dir":      cfg.TemplatesDir,
		"static_dir":         cfg.StaticDir,
		"themes_dir":         cfg.ThemesDir,
		"plugins_dir":        cfg.PluginsDir,
		"plugins_enabled":    cfg.PluginsEnabled,
		"sass_dir":           cfg.SassDir,
		"content_layout":     cfg.ContentLayout,
		"include_drafts":     cfg.IncludeDrafts,
		"compile_sass":       cfg.CompileSass,
		"tui_prefix":         cfg.TUIPrefix,
		"tui_prefix_ms":      cfg.TUIPrefixMs,
		"serve_watch":        cfg.ServeWatch,
		"serve_live_reload":  cfg.ServeReload,
		"serve_debounce_ms":  cfg.ServeDebounce,
		"build_incremental":  cfg.BuildIncremental,
		"build_cache_dir":    cfg.BuildCacheDir,
		"doctor_profile":     cfg.DoctorProfile,
		"summary_strategy":   cfg.SummaryStrategy,
		"site_feed":          cfg.SiteFeed,
		"site_feed_limit":    cfg.SiteFeedLimit,
		"image_optimization": cfg.ImageOptimization,
		"image_quality":      cfg.ImageQuality,
		"image_widths":       cfg.ImageWidths,
		"lightbox":           cfg.Lightbox,
		"minify":             cfg.Minify,
		"logging": map[string]any{
			"level":  cfg.Logging.Level,
			"format": cfg.Logging.Format,
		},
		"taxonomies": taxonomies,
	}
}

func taxonomiesView(indices map[string]*taxonomy.Index) map[string]any {
	out := map[string]any{}
	for name, index := range indices {
		out[name] = map[string]any{
			"taxonomy": taxonomy.ConfigView(index.Config),
			"terms":    taxonomy.TermViews(index.TermsSorted()),
		}
	}
	return out
}

func buildStatsView(stats Stats, siteIndex *site.Site) map[string]any {
	return map[string]any{
		"total":    stats.Total,
		"rendered": stats.Rendered,
		"skipped":  stats.Skipped,
		"cached":   stats.Cached,
		"errors":   stats.Errors,
		"pages":    len(siteIndex.Pages),
		"sections": len(siteIndex.Sections),
	}
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func applyPluginOverrides(ctx context.Context, plugins *plugin.Manager, event string, payload map[string]any) map[string]any {
	if plugins == nil {
		return payload
	}
	overrides := plugins.Emit(ctx, event, payload)
	if overrides == nil {
		return payload
	}
	plugin.Merge(payload, overrides)
	return payload
}

func themeTemplatesDir(cfg config.Config) string {
	if strings.TrimSpace(cfg.ThemesDir) == "" {
		return ""
	}
	if strings.TrimSpace(cfg.Theme) == "" {
		return ""
	}
	return path.Join(cfg.ThemesDir, cfg.Theme, "templates")
}

func themeI18nDir(cfg config.Config) string {
	if strings.TrimSpace(cfg.ThemesDir) == "" {
		return ""
	}
	if strings.TrimSpace(cfg.Theme) == "" {
		return ""
	}
	return path.Join(cfg.ThemesDir, cfg.Theme, "i18n")
}

// defaultLangFromCtx extracts the default language from a base context.
// It reads config.default_language, falling back to "es".
func defaultLangFromCtx(baseCtx map[string]any) string {
	cfgMap, ok := baseCtx["config"].(map[string]any)
	if !ok {
		return "es"
	}
	lang, ok := cfgMap["default_language"].(string)
	if !ok || lang == "" {
		return "es"
	}
	return lang
}

func renderTaxonomies(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, indices map[string]*taxonomy.Index, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
	if len(cfg.Taxonomies) == 0 {
		return 0, 0, nil
	}

	rendered := 0
	cached := 0

	for _, taxCfg := range cfg.Taxonomies {
		if !taxCfg.Render {
			continue
		}

		index, ok := indices[taxCfg.Name]
		if !ok {
			continue
		}

		terms := index.TermsSorted()
		listTemplate := taxonomyTemplateName(renderer, taxCfg.Name, "list.html", "taxonomy_list.html")
		listPath := ensureTrailingSlash(path.Join("/", taxCfg.Name))
		listOutput := outputHTMLPath(cfg.PublicDir, listPath)
		if !plan.shouldRenderCollection(listOutput) {
			cached++
		} else {
			listCtx := taxonomyListContext(baseCtx, cfg, taxCfg, terms, listPath)
			listCtx = applyPluginOverrides(ctx, plugins, "taxonomy.list.render", listCtx)
			if err := renderer.RenderToFile(listTemplate, listCtx, listOutput); err != nil {
				return rendered, cached, err
			}
			rendered++
		}

		singleTemplate := taxonomyTemplateName(renderer, taxCfg.Name, "single.html", "taxonomy_single.html")
		for _, term := range terms {
			paginators := taxonomy.BuildPaginator(term.Pages, taxCfg.PaginateBy, term.Path, taxCfg.PaginatePath)
			if len(paginators) == 0 {
				termPath := term.Path
				outputPath := outputHTMLPath(cfg.PublicDir, termPath)
				if !plan.shouldRenderCollection(outputPath) {
					cached++
				} else {
					termCtx := taxonomyTermContext(baseCtx, cfg, taxCfg, term, termPath, nil)
					termCtx = applyPluginOverrides(ctx, plugins, "taxonomy.term.render", termCtx)
					if err := renderer.RenderToFile(singleTemplate, termCtx, outputPath); err != nil {
						return rendered, cached, err
					}
					rendered++
				}
				renderedFeeds, cachedFeeds, err := renderTaxonomyFeeds(renderer, cfg, baseCtx, taxCfg, term, plan)
				if err != nil {
					return rendered, cached, err
				}
				rendered += renderedFeeds
				cached += cachedFeeds
				continue
			}

			for i, paginator := range paginators {
				pagePath := taxonomyPagePath(term.Path, taxCfg.PaginatePath, i)
				outputPath := outputHTMLPath(cfg.PublicDir, pagePath)
				if !plan.shouldRenderCollection(outputPath) {
					cached++
					continue
				}
				termCtx := taxonomyTermContext(baseCtx, cfg, taxCfg, term, pagePath, &paginator)
				termCtx = applyPluginOverrides(ctx, plugins, "taxonomy.term.render", termCtx)
				if err := renderer.RenderToFile(singleTemplate, termCtx, outputPath); err != nil {
					return rendered, cached, err
				}
				rendered++
			}
			renderedFeeds, cachedFeeds, err := renderTaxonomyFeeds(renderer, cfg, baseCtx, taxCfg, term, plan)
			if err != nil {
				return rendered, cached, err
			}
			rendered += renderedFeeds
			cached += cachedFeeds
		}
	}

	return rendered, cached, nil
}

func taxonomyTemplateName(renderer *render.Renderer, taxonomyName string, specific string, fallback string) string {
	specificName := path.Join(taxonomyName, specific)
	if renderer.HasTemplate(specificName) {
		return specificName
	}
	return fallback
}

func taxonomyListContext(baseCtx map[string]any, cfg config.Config, taxCfg config.TaxonomyConfig, terms []*taxonomy.Term, currentPath string) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["taxonomy"] = taxonomy.ConfigView(taxCfg)
	ctx["terms"] = taxonomy.TermViews(terms)
	ctx["current_path"] = currentPath
	ctx["current_url"] = buildURL(cfg.BaseURL, currentPath)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	return ctx
}

func taxonomyTermContext(baseCtx map[string]any, cfg config.Config, taxCfg config.TaxonomyConfig, term *taxonomy.Term, currentPath string, paginator *taxonomy.Paginator) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["taxonomy"] = taxonomy.ConfigView(taxCfg)
	ctx["term"] = taxonomy.TermView(term)
	ctx["current_path"] = currentPath
	ctx["current_url"] = buildURL(cfg.BaseURL, currentPath)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	if paginator != nil {
		ctx["paginator"] = taxonomy.PaginatorView(*paginator)
	}
	return ctx
}

func taxonomyPagePath(termPath string, paginatePath string, index int) string {
	if index == 0 {
		return ensureTrailingSlash(termPath)
	}
	if paginatePath == "" {
		paginatePath = "page"
	}
	return ensureTrailingSlash(path.Join(termPath, paginatePath, strconv.Itoa(index+1)))
}

func buildURL(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + path
}

func ensureTrailingSlash(input string) string {
	if strings.HasSuffix(input, "/") {
		return input
	}
	return input + "/"
}

func renderTaxonomyFeeds(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, taxCfg config.TaxonomyConfig, term *taxonomy.Term, plan buildPlan) (int, int, error) {
	if !taxCfg.Feed {
		return 0, 0, nil
	}

	feedTemplates := []string{}
	if renderer.HasTemplate("atom.xml") {
		feedTemplates = append(feedTemplates, "atom.xml")
	}
	if renderer.HasTemplate("rss.xml") {
		feedTemplates = append(feedTemplates, "rss.xml")
	}
	if len(feedTemplates) == 0 {
		return 0, 0, nil
	}

	lastUpdated := latestUpdated(term.Pages)
	rendered := 0
	cached := 0

	for _, tmpl := range feedTemplates {
		filename := tmpl
		feedURL := buildURL(cfg.BaseURL, path.Join(term.Path, filename))
		outputPath := outputFilePath(cfg.PublicDir, term.Path, filename)
		if !plan.shouldRenderCollection(outputPath) {
			cached++
			continue
		}
		ctx := feedContext(baseCtx, cfg, taxCfg, term, feedURL, lastUpdated)
		if err := renderer.RenderToFile(tmpl, ctx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}

	return rendered, cached, nil
}

func latestUpdated(pages []*site.Page) time.Time {
	if len(pages) == 0 {
		return time.Now()
	}
	latest := pages[0].Date
	for _, page := range pages[1:] {
		if page.Date.After(latest) {
			latest = page.Date
		}
	}
	return latest
}

func feedContext(baseCtx map[string]any, cfg config.Config, taxCfg config.TaxonomyConfig, term *taxonomy.Term, feedURL string, lastUpdated time.Time) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["taxonomy"] = taxonomy.ConfigView(taxCfg)
	ctx["term"] = taxonomy.TermView(term)
	ctx["pages"] = feedPages(term.Pages)
	ctx["feed_url"] = feedURL
	ctx["last_updated"] = lastUpdated.Format(time.RFC3339)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	return ctx
}

func feedPages(pages []*site.Page) []map[string]any {
	out := make([]map[string]any, 0, len(pages))
	for _, page := range pages {
		out = append(out, map[string]any{
			"title":     page.Title,
			"permalink": page.Permalink,
			"date":      page.Date.Format(time.RFC3339),
			"summary":   page.Summary,
			"content":   page.Content,
			"image":     page.Image,
			"path":      page.Path,
		})
	}
	return out
}

// renderSiteFeed generates site-wide atom.xml and rss.xml at the root of public/.
func renderSiteFeed(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, plan buildPlan) (int, int, error) {
	if !cfg.SiteFeed {
		return 0, 0, nil
	}

	feedTemplates := []string{}
	if renderer.HasTemplate("atom.xml") {
		feedTemplates = append(feedTemplates, "atom.xml")
	}
	if renderer.HasTemplate("rss.xml") {
		feedTemplates = append(feedTemplates, "rss.xml")
	}
	if len(feedTemplates) == 0 {
		return 0, 0, nil
	}

	// Pages are already sorted by date descending in BuildHierarchy().
	pages := siteIndex.Pages
	if cfg.SiteFeedLimit > 0 && len(pages) > cfg.SiteFeedLimit {
		pages = pages[:cfg.SiteFeedLimit]
	}

	lastUpdated := latestUpdated(pages)
	rendered := 0
	cached := 0

	for _, tmpl := range feedTemplates {
		filename := tmpl
		feedURL := buildURL(cfg.BaseURL, "/"+filename)
		outputPath := outputFilePath(cfg.PublicDir, "", filename)
		if !plan.shouldRenderCollection(outputPath) {
			cached++
			continue
		}
		ctx := siteFeedContext(baseCtx, cfg, pages, feedURL, lastUpdated)
		if err := renderer.RenderToFile(tmpl, ctx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}

	return rendered, cached, nil
}

// siteFeedContext builds the template context for the site-wide feed.
// It uses feed_title and feed_description instead of taxonomy/term,
// and the templates detect feed_title to pick the right title.
func siteFeedContext(baseCtx map[string]any, cfg config.Config, pages []*site.Page, feedURL string, lastUpdated time.Time) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["feed_title"] = cfg.SiteTitle
	ctx["feed_description"] = cfg.SiteDescription
	ctx["pages"] = feedPages(pages)
	ctx["feed_url"] = feedURL
	ctx["last_updated"] = lastUpdated.Format(time.RFC3339)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	return ctx
}

func outputFilePath(publicDir string, sitePath string, filename string) string {
	rel := strings.TrimPrefix(sitePath, "/")
	if rel == "" {
		return filepath.Join(publicDir, filename)
	}
	return filepath.Join(publicDir, rel, filename)
}

type SitemapEntry struct {
	Permalink string
	Updated   time.Time
	Extra     map[string]any
}

const maxSitemapEntries = 50000

func renderSitemap(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, entries []SitemapEntry, plan buildPlan) (int, int, error) {
	if len(entries) == 0 {
		return 0, 0, nil
	}
	if !renderer.HasTemplate("sitemap.xml") {
		return 0, 0, nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Permalink < entries[j].Permalink
	})

	if len(entries) <= maxSitemapEntries {
		ctx := cloneMap(baseCtx)
		ctx["entries"] = sitemapEntryViews(entries)
		outputPath := filepath.Join(cfg.PublicDir, "sitemap.xml")
		if !plan.shouldRenderCollection(outputPath) {
			return 0, 1, nil
		}
		if err := renderer.RenderToFile("sitemap.xml", ctx, outputPath); err != nil {
			return 0, 0, err
		}
		return 1, 0, nil
	}

	if !renderer.HasTemplate("split_sitemap_index.xml") {
		return 0, 0, fmt.Errorf("split sitemap required but split_sitemap_index.xml template missing")
	}

	sitemaps := make([]map[string]any, 0)
	rendered := 0
	cached := 0
	for i := 0; i < len(entries); i += maxSitemapEntries {
		end := i + maxSitemapEntries
		if end > len(entries) {
			end = len(entries)
		}
		chunk := entries[i:end]
		filename := fmt.Sprintf("sitemap_%d.xml", (i/maxSitemapEntries)+1)
		outputPath := filepath.Join(cfg.PublicDir, filename)
		loc := buildURL(cfg.BaseURL, "/"+filename)
		lastmod := chunk[len(chunk)-1].Updated.Format(time.RFC3339)
		sitemaps = append(sitemaps, map[string]any{
			"loc":     loc,
			"lastmod": lastmod,
		})
		if !plan.shouldRenderCollection(outputPath) {
			cached++
			continue
		}
		ctx := cloneMap(baseCtx)
		ctx["entries"] = sitemapEntryViews(chunk)
		if err := renderer.RenderToFile("sitemap.xml", ctx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}

	indexCtx := cloneMap(baseCtx)
	indexCtx["sitemaps"] = sitemaps
	indexPath := filepath.Join(cfg.PublicDir, "sitemap.xml")
	if !plan.shouldRenderCollection(indexPath) {
		return rendered, cached + 1, nil
	}
	if err := renderer.RenderToFile("split_sitemap_index.xml", indexCtx, indexPath); err != nil {
		return rendered, cached, err
	}
	rendered++

	return rendered, cached, nil
}

func sitemapEntryViews(entries []SitemapEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, map[string]any{
			"permalink": entry.Permalink,
			"updated":   entry.Updated.Format(time.RFC3339),
			"extra":     entry.Extra,
		})
	}
	return out
}

func renderRobots(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, plan buildPlan) (int, int, error) {
	if !renderer.HasTemplate("robots.txt") {
		return 0, 0, nil
	}
	ctx := cloneMap(baseCtx)
	outputPath := filepath.Join(cfg.PublicDir, "robots.txt")
	if !plan.shouldRenderCollection(outputPath) {
		return 0, 1, nil
	}
	if err := renderer.RenderToFile("robots.txt", ctx, outputPath); err != nil {
		return 0, 0, err
	}
	return 1, 0, nil
}

func renderNotFound(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, plan buildPlan) (int, int, error) {
	if !renderer.HasTemplate("404.html") {
		return 0, 0, nil
	}
	ctx := cloneMap(baseCtx)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	outputPath := filepath.Join(cfg.PublicDir, "404.html")
	if !plan.shouldRenderCollection(outputPath) {
		return 0, 1, nil
	}
	if err := renderer.RenderToFile("404.html", ctx, outputPath); err != nil {
		return 0, 0, err
	}
	return 1, 0, nil
}

func collectSitemapEntries(cfg config.Config, siteIndex *site.Site, indices map[string]*taxonomy.Index) []SitemapEntry {
	entries := map[string]SitemapEntry{}

	addEntry := func(permalink string, updated time.Time) {
		if strings.TrimSpace(permalink) == "" {
			return
		}
		existing, ok := entries[permalink]
		if !ok || updated.After(existing.Updated) {
			entries[permalink] = SitemapEntry{
				Permalink: permalink,
				Updated:   updated,
			}
		}
	}

	for _, page := range siteIndex.Pages {
		addEntry(page.Permalink, page.Date)
	}

	for _, section := range siteIndex.Sections {
		addEntry(section.Permalink, sectionUpdated(section))
	}

	for _, taxCfg := range cfg.Taxonomies {
		if !taxCfg.Render {
			continue
		}
		index, ok := indices[taxCfg.Name]
		if !ok {
			continue
		}

		listPath := ensureTrailingSlash(path.Join("/", taxCfg.Name))
		addEntry(buildURL(cfg.BaseURL, listPath), taxonomyIndexUpdated(index))

		for _, term := range index.TermsSorted() {
			paginators := taxonomy.BuildPaginator(term.Pages, taxCfg.PaginateBy, term.Path, taxCfg.PaginatePath)
			if len(paginators) == 0 {
				addEntry(term.Permalink, latestUpdated(term.Pages))
				continue
			}
			for i := range paginators {
				pagePath := taxonomyPagePath(term.Path, taxCfg.PaginatePath, i)
				addEntry(buildURL(cfg.BaseURL, pagePath), latestUpdated(term.Pages))
			}
		}
	}

	out := make([]SitemapEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}
	return out
}

func taxonomyIndexUpdated(index *taxonomy.Index) time.Time {
	latest := time.Time{}
	for _, term := range index.Terms {
		updated := latestUpdated(term.Pages)
		if updated.After(latest) {
			latest = updated
		}
	}
	if latest.IsZero() {
		return time.Now()
	}
	return latest
}

func sectionUpdated(section *site.Section) time.Time {
	latest := time.Time{}
	for _, page := range section.Pages {
		if page.Date.After(latest) {
			latest = page.Date
		}
	}
	for _, subsection := range section.Subsections {
		updated := sectionUpdated(subsection)
		if updated.After(latest) {
			latest = updated
		}
	}
	if latest.IsZero() {
		return time.Now()
	}
	return latest
}

// fillSummaries populates Page.Summary for any page that lacks one,
// using the configured summary strategy (auto/manual/ai).
// It returns the source paths of pages whose Summary was set, so the
// caller can mark them as changed for incremental rebuild.
func fillSummaries(ctx context.Context, cfg config.Config, opts BuildOptions, siteIndex *site.Site, logger *slog.Logger) []string {
	strategy := cfg.SummaryStrategy

	// For "ai", create a KairosProvider with the configured LLM backend.
	// If SkipAI is set (e.g. during serve), honour cached AI summaries
	// without making new LLM calls; uncached pages fall back to auto.
	if strings.EqualFold(strategy, "ai") {
		if opts.SkipAI {
			cachePath := aiCachePath(cfg)
			cache := loadAICache(cachePath, logger)
			return fillFromCacheOrAuto(ctx, siteIndex, cache, logger)
		}

		aiCfg := summary.AIConfig{
			Provider:     cfg.AI.Provider,
			Model:        cfg.AI.Model,
			APIKey:       cfg.AI.APIKey,
			BaseURL:      cfg.AI.BaseURL,
			SystemPrompt: cfg.AI.SystemPrompt,
			Language:     cfg.DefaultLanguage,
		}
		kp, err := summary.NewKairosProvider(ctx, aiCfg)
		if err != nil {
			logger.Warn("failed to create AI summary provider, falling back to auto", "error", err)
			return fillWithProvider(ctx, siteIndex, summary.ExtractProvider{}, cfg.SummaryStrategy, logger)
		}

		timeout := time.Duration(cfg.AI.Timeout) * time.Second
		concurrency := cfg.AI.Concurrency
		if concurrency <= 0 {
			concurrency = 3
		}

		// Load AI summary cache.
		cachePath := aiCachePath(cfg)
		cache := loadAICache(cachePath, logger)
		cache.provider = cfg.AI.Provider
		cache.model = cfg.AI.Model

		affected := fillWithAI(ctx, siteIndex, kp, timeout, concurrency, cache, opts.ForceAISummaries, logger, opts.Progress)

		// Save updated cache.
		if err := saveAICache(cachePath, cache, logger); err != nil {
			logger.Warn("failed to save AI summary cache", "error", err)
		}
		return affected
	}

	// Non-AI strategies: manual or auto.
	provider := summary.NewProvider(strategy)
	return fillWithProvider(ctx, siteIndex, provider, strategy, logger)
}

// fillWithProvider runs the simple sequential summary fill used by
// auto and manual strategies.  It returns the source paths of pages
// whose Summary was set.
func fillWithProvider(ctx context.Context, siteIndex *site.Site, provider summary.Provider, strategy string, logger *slog.Logger) []string {
	// NoopProvider means "manual" — nothing to do.
	if _, noop := provider.(summary.NoopProvider); noop {
		return nil
	}

	var affected []string
	for _, page := range siteIndex.Pages {
		if page.Summary != "" {
			continue // already has a frontmatter summary
		}
		if page.Menu {
			continue // standalone/menu pages are navigation items, not posts
		}
		text, err := provider.Summarize(ctx, page.Title, page.RawContent)
		if err != nil {
			logger.Warn("summary generation failed", "title", page.Title, "path", page.Path, "error", err)
			continue
		}
		if text != "" {
			page.Summary = text
			affected = append(affected, page.SourcePath)
		}
	}
	if len(affected) > 0 {
		logger.Info("summaries generated", "count", len(affected), "strategy", strategy)
	}
	return affected
}

// fillFromCacheOrAuto is used during serve mode when the AI strategy is
// configured but LLM calls are skipped.  It loads cached AI summaries for
// pages that have them and falls back to auto-extraction for the rest.
// This avoids discarding previously generated AI summaries on every serve
// rebuild while still keeping serve fast (no LLM calls).
func fillFromCacheOrAuto(ctx context.Context, siteIndex *site.Site, cache *AICache, logger *slog.Logger) []string {
	auto := summary.ExtractProvider{}
	var affected []string
	fromCache := 0
	fromAuto := 0

	for _, page := range siteIndex.Pages {
		if page.Summary != "" {
			continue
		}
		if page.Menu {
			continue // standalone/menu pages are navigation items, not posts
		}

		hash := contentHash(page.RawContent)
		if entry, ok := cache.Lookup(hash); ok {
			page.Summary = entry.Summary
			affected = append(affected, page.SourcePath)
			fromCache++
			continue
		}

		text, err := auto.Summarize(ctx, page.Title, page.RawContent)
		if err != nil {
			logger.Warn("auto summary failed", "title", page.Title, "path", page.Path, "error", err)
			continue
		}
		if text != "" {
			page.Summary = text
			affected = append(affected, page.SourcePath)
			fromAuto++
		}
	}

	if fromCache > 0 || fromAuto > 0 {
		logger.Info("summaries filled (serve mode)", "from_cache", fromCache, "from_auto", fromAuto)
	}
	return affected
}

// fillWithAI runs concurrent LLM-based summary generation with bounded
// parallelism and per-request timeouts.  It checks the cache before calling
// the LLM and stores new results back into the cache.
// It returns the source paths of pages whose Summary was set (from cache
// or from the LLM).
func fillWithAI(ctx context.Context, siteIndex *site.Site, provider summary.Provider, timeout time.Duration, concurrency int, cache *AICache, force bool, logger *slog.Logger, progress logging.Progress) []string {
	// Collect pages that need summaries.
	type pageJob struct {
		page        *site.Page
		contentHash string
	}
	var jobs []pageJob
	var affected []string
	for _, page := range siteIndex.Pages {
		if page.Summary != "" {
			continue // already has a frontmatter summary
		}
		if page.Menu {
			continue // standalone/menu pages are navigation items, not posts
		}
		hash := contentHash(page.RawContent)

		// Check cache (unless force is set).
		if !force {
			if entry, ok := cache.Lookup(hash); ok {
				page.Summary = entry.Summary
				affected = append(affected, page.SourcePath)
				continue
			}
		}

		jobs = append(jobs, pageJob{page: page, contentHash: hash})
	}

	if len(affected) > 0 {
		logger.Info("AI summaries from cache", "count", len(affected))
	}
	if len(jobs) == 0 {
		return affected
	}

	// Log the pages that will be sent to the LLM so the user knows what's happening.
	for _, job := range jobs {
		logger.Info("queued for AI summary", "title", job.page.Title, "path", job.page.Path)
	}
	logger.Info("generating AI summaries", "pages", len(jobs), "concurrency", concurrency)

	// Start a user-visible spinner if a progress indicator is available.
	// Use reflection to check for nil interface or nil pointer inside interface.
	if progress != nil && !isNilInterface(progress) {
		progress.Start(fmt.Sprintf("Generating AI summaries (0/%d)…", len(jobs)))
	}

	// Bounded parallelism via a semaphore channel.
	sem := make(chan struct{}, concurrency)
	type result struct {
		idx     int
		summary string
		err     error
	}
	results := make(chan result, len(jobs))

	for i, job := range jobs {
		sem <- struct{}{} // acquire
		go func(idx int, p *site.Page) {
			defer func() { <-sem }() // release

			reqCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			text, err := provider.Summarize(reqCtx, p.Title, p.RawContent)
			results <- result{idx: idx, summary: text, err: err}
		}(i, job.page)
	}

	// Collect results.
	filled := 0
	errors := 0
	processed := 0
	for range len(jobs) {
		r := <-results
		job := jobs[r.idx]
		processed++
		if r.err != nil {
			logger.Warn("AI summary failed", "title", job.page.Title, "path", job.page.Path, "error", r.err)
			errors++
		} else if r.summary != "" {
			job.page.Summary = r.summary
			cache.Store(job.contentHash, r.summary)
			logger.Info("AI summary generated", "title", job.page.Title)
			affected = append(affected, job.page.SourcePath)
			filled++
		}
		if progress != nil && !isNilInterface(progress) {
			progress.Update(fmt.Sprintf("Generating AI summaries (%d/%d)…", processed, len(jobs)))
		}
	}

	if progress != nil && !isNilInterface(progress) {
		progress.Stop()
	}

	logger.Info("AI summaries complete", "filled", filled, "errors", errors)
	return affected
}

// generatePlaceholders writes a deterministic SVG placeholder image for
// every page that has no image set, and updates page.Image to point to
// the generated file's public URL path.
func generatePlaceholders(siteIndex *site.Site, publicDir string, logger *slog.Logger) error {
	imgDir := filepath.Join(publicDir, "img")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return fmt.Errorf("create img dir: %w", err)
	}

	generated := 0
	for _, page := range siteIndex.Pages {
		if page.Image != "" {
			continue
		}
		filename := placeholder.Filename(page.Title)
		outPath := filepath.Join(imgDir, filename)

		// Skip if already written (deterministic, won't change).
		if _, err := os.Stat(outPath); err == nil {
			page.Image = "/img/" + filename
			continue
		}

		svg := placeholder.Generate(page.Title)
		if err := os.WriteFile(outPath, []byte(svg), 0o644); err != nil {
			return fmt.Errorf("write placeholder %s: %w", filename, err)
		}
		page.Image = "/img/" + filename
		generated++
	}

	if generated > 0 {
		logger.Info("placeholders generated", "count", generated)
	}
	return nil
}

// isNilInterface checks if an interface value is nil or contains a nil pointer.
func isNilInterface(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}
