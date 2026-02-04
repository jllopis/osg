package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"osg/internal/assets"
	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/plugin"
	"osg/internal/render"
	"osg/internal/site"
	"osg/internal/taxonomy"
	"osg/internal/vault"
)

type Stats struct {
	Total    int
	Rendered int
	Skipped  int
	Cached   int
	Errors   int
}

func Run(ctx context.Context, cfg config.Config, verbose bool, logWriter io.Writer) error {
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

	if err := assets.Prepare(cfg, logger); err != nil {
		return err
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
	indices := taxonomy.Build(cfg.Taxonomies, siteIndex.Pages, cfg.BaseURL)
	siteView := siteIndex.View()
	baseCtx := baseContext(cfg, siteView, indices)

	plugins, err := plugin.Load(ctx, cfg.PluginsDir, cfg.PluginsEnabled, logger)
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

		startPayload := cloneMap(baseCtx)
		startPayload["stats"] = buildStatsView(stats, siteIndex)
		if overrides := plugins.Emit(ctx, "build.started", startPayload); overrides != nil {
			plugin.Merge(baseCtx, overrides)
		}
	}

	renderer, err := render.New(cfg.TemplatesDir, themeTemplatesDir(cfg), render.Context{
		BaseURL:    cfg.BaseURL,
		ContentDir: cfg.ContentDir,
		StaticDir:  cfg.StaticDir,
		PublicDir:  cfg.PublicDir,
		Site:       siteIndex,
		Taxonomies: indices,
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

	renderedPages, cachedPages, err := renderPages(ctx, renderer, cfg, baseCtx, siteIndex, plugins, plan)
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

func renderPages(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
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
		renderCtx = applyPluginOverrides(ctx, plugins, "page.render", renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}
	return rendered, cached, nil
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
	return ctx
}

func sectionContext(baseCtx map[string]any, section *site.Section) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["section"] = section.View()
	ctx["current_path"] = section.Path
	ctx["current_url"] = section.Permalink
	ctx["lang"] = ""
	return ctx
}

func baseContext(cfg config.Config, siteView map[string]any, indices map[string]*taxonomy.Index) map[string]any {
	ctx := map[string]any{
		"config": configView(cfg),
		"site":   siteView,
	}
	if len(indices) > 0 {
		ctx["taxonomies"] = taxonomiesView(indices)
	}
	return ctx
}

func configView(cfg config.Config) map[string]any {
	taxonomies := make([]map[string]any, 0, len(cfg.Taxonomies))
	for _, taxCfg := range cfg.Taxonomies {
		taxonomies = append(taxonomies, taxonomy.ConfigView(taxCfg))
	}

	return map[string]any{
		"base_url":          cfg.BaseURL,
		"theme":             cfg.Theme,
		"vault_path":        cfg.VaultPath,
		"content_dir":       cfg.ContentDir,
		"public_dir":        cfg.PublicDir,
		"templates_dir":     cfg.TemplatesDir,
		"static_dir":        cfg.StaticDir,
		"themes_dir":        cfg.ThemesDir,
		"plugins_dir":       cfg.PluginsDir,
		"plugins_enabled":   cfg.PluginsEnabled,
		"sass_dir":          cfg.SassDir,
		"content_layout":    cfg.ContentLayout,
		"include_drafts":    cfg.IncludeDrafts,
		"compile_sass":      cfg.CompileSass,
		"tui_prefix":        cfg.TUIPrefix,
		"tui_prefix_ms":     cfg.TUIPrefixMs,
		"serve_watch":       cfg.ServeWatch,
		"serve_live_reload": cfg.ServeReload,
		"serve_debounce_ms": cfg.ServeDebounce,
		"build_incremental": cfg.BuildIncremental,
		"build_cache_dir":   cfg.BuildCacheDir,
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
	ctx["lang"] = ""
	return ctx
}

func taxonomyTermContext(baseCtx map[string]any, cfg config.Config, taxCfg config.TaxonomyConfig, term *taxonomy.Term, currentPath string, paginator *taxonomy.Paginator) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["taxonomy"] = taxonomy.ConfigView(taxCfg)
	ctx["term"] = taxonomy.TermView(term)
	ctx["current_path"] = currentPath
	ctx["current_url"] = buildURL(cfg.BaseURL, currentPath)
	ctx["lang"] = ""
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
	ctx["lang"] = ""
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
			"path":      page.Path,
		})
	}
	return out
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
	ctx["lang"] = ""
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
