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
	"regexp"
	"runtime"
	"runtime/pprof"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	"osg/internal/webhook"
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
	// Profile is an optional filesystem path for a CPU profile output.
	// When non-empty, a pprof CPU profile is written to this path.
	Profile string
	// DryRun builds to a temp directory and prints what would be generated.
	DryRun bool
	// TimingJSON, when non-empty, writes build timing as JSON to this path.
	TimingJSON string
}

func Run(ctx context.Context, cfg config.Config, opts BuildOptions, verbose bool, logWriter io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := logging.NewWithWriter(cfg.Logging, verbose, logWriter)

	// Dry-run: build to a temp directory, then print results.
	if opts.DryRun {
		return runDryRun(ctx, cfg, opts, verbose, logWriter, logger)
	}

	// CPU profiling (--profile flag).
	if opts.Profile != "" {
		f, err := os.Create(opts.Profile)
		if err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
		defer func() { _ = f.Close() }()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
		logger.Info("CPU profiling enabled", "output", opts.Profile)
	}

	buildStart := time.Now()
	timings := &BuildTimings{}

	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no content files found", "content_dir", cfg.ContentDir)
		return nil
	}

	// --- Stage: plan ---
	done := timings.stage("plan")
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
	done()

	// --- Stage: theme + i18n ---
	done = timings.stage("theme")
	if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
		return fmt.Errorf("ensure default theme: %w", err)
	}

	// Resolve theme inheritance chain (child -> parent -> grandparent).
	themeChain, err := theme.ResolveChain(cfg.ThemesDir, cfg.Theme)
	if err != nil {
		return fmt.Errorf("resolve theme chain: %w", err)
	}

	// Load i18n translations: ancestor themes first, then child, then user (last wins).
	i18nBundle := i18n.New(cfg.DefaultLanguage)
	for idx := len(themeChain) - 1; idx >= 0; idx-- {
		dir := filepath.Join(themeChain[idx], "i18n")
		if err := i18nBundle.LoadDir(dir); err != nil {
			return fmt.Errorf("load theme translations from %s: %w", dir, err)
		}
	}
	userI18nDir := "i18n"
	if err := i18nBundle.LoadDir(userI18nDir); err != nil {
		return fmt.Errorf("load user translations: %w", err)
	}
	done()

	// --- Stage: assets ---
	done = timings.stage("assets")
	if err := assets.PrepareWithChain(cfg, themeChain, logger); err != nil {
		return err
	}
	done()

	// --- Stage: plugins ---
	done = timings.stage("plugins")
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
	done()

	// --- Stage: parse ---
	done = timings.stage("parse")
	siteIndex := site.New()
	stats := Stats{Total: len(files)}

	// Parse content files in parallel.  Each ParseFile reads a single
	// file and renders its Markdown, producing independent Page/Section
	// values.  Results are collected into a channel and merged into the
	// (non-thread-safe) siteIndex sequentially.
	type parseResult struct {
		page    *site.Page
		section *site.Section
		err     error
		path    string
	}

	parseCh := make(chan parseResult, len(files))
	parseWorkers := runtime.NumCPU()
	if parseWorkers > len(files) {
		parseWorkers = len(files)
	}

	var parseWG sync.WaitGroup
	parseJobs := make(chan string, len(files))
	for range parseWorkers {
		parseWG.Add(1)
		go func() {
			defer parseWG.Done()
			for fp := range parseJobs {
				page, section, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, fp)
				parseCh <- parseResult{page: page, section: section, err: err, path: fp}
			}
		}()
	}
	for _, fp := range files {
		parseJobs <- fp
	}
	close(parseJobs)
	go func() {
		parseWG.Wait()
		close(parseCh)
	}()

	for res := range parseCh {
		if res.err != nil {
			logger.Warn("failed to parse content", "path", res.path, "error", res.err)
			stats.Errors++
			continue
		}

		if res.page != nil {
			if res.page.Draft && !cfg.IncludeDrafts {
				stats.Skipped++
				continue
			}
			// Default empty Lang to the site's default language.
			if res.page.Lang == "" {
				res.page.Lang = cfg.DefaultLanguage
			}
			siteIndex.AddPage(res.page)
		}

		if res.section != nil {
			siteIndex.AddSection(res.section)
		}
	}

	siteIndex.BuildHierarchy()

	// Link translations: pages with the same slug in different languages
	// get cross-references so templates can render hreflang alternates.
	if cfg.IsMultilingual() {
		siteIndex.LinkTranslations()
	}
	done()

	// --- Stage: transform ---
	done = timings.stage("transform")
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
	done()

	// --- Stage: images ---
	done = timings.stage("images")
	// Generate placeholder SVGs for pages without an image.
	if err := generatePlaceholders(siteIndex, cfg.PublicDir, logger); err != nil {
		return fmt.Errorf("generate placeholders: %w", err)
	}

	// Optimize images: generate responsive variants (resized JPEG + WebP).
	var imageResults map[string]*imgopt.Result
	if cfg.ImageOptimization {
		imgOpts := imgopt.Options{
			Quality: cfg.ImageQuality,
			Widths:  cfg.ImageWidths,
			WebP:    true,
			AVIF:    true,
		}
		if imgOpts.Quality <= 0 || imgOpts.Quality > 100 {
			imgOpts.Quality = 80
		}
		if len(imgOpts.Widths) == 0 {
			imgOpts.Widths = []int{640, 1200}
		}
		var err error
		imageResults, err = imgopt.Optimize(cfg.PublicDir, imgOpts, logger)
		if err != nil {
			logger.Warn("image optimization failed", "error", err)
			// Non-fatal: continue with no optimized variants.
		}
	}

	// image.process: let plugins transform images via WASI filesystem.
	emitImageProcess(ctx, plugins, cfg, imageResults, logger)
	done()

	// --- Stage: taxonomy ---
	done = timings.stage("taxonomy")
	indices := taxonomy.Build(cfg.Taxonomies, siteIndex.Pages, cfg.BaseURL)
	// Strip excluded terms from each page's Taxonomies so templates
	// ("Publicado en:", card pills, etc.) never display them.
	taxonomy.FilterPageTaxonomies(cfg.Taxonomies, siteIndex.Pages)
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
	done()

	// --- Stage: templates ---
	done = timings.stage("templates")
	// Build template directories from the theme chain.
	var templateChain []string
	for _, dir := range themeChain {
		templateChain = append(templateChain, filepath.Join(dir, "templates"))
	}
	renderer, err := render.NewWithChain(cfg.TemplatesDir, templateChain, render.Context{
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
	done()

	// --- Stage: render ---
	done = timings.stage("render")
	if plan.assetsOnly {
		logger.Info("asset-only change, skipping HTML render")
	} else {
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

		siteFeedRendered, siteFeedCached, err := renderSiteFeed(ctx, renderer, cfg, baseCtx, siteIndex, plugins, plan)
		if err != nil {
			stats.Errors++
			return err
		}
		stats.Rendered += siteFeedRendered
		stats.Cached += siteFeedCached

		sectionFeedRendered, sectionFeedCached, err := renderSectionFeeds(ctx, renderer, cfg, baseCtx, siteIndex, plugins, plan)
		if err != nil {
			stats.Errors++
			return err
		}
		stats.Rendered += sectionFeedRendered
		stats.Cached += sectionFeedCached

		sitemapEntries := collectSitemapEntries(cfg, siteIndex, indices)
		sitemapRendered, sitemapCached, err := renderSitemap(ctx, renderer, cfg, baseCtx, sitemapEntries, plugins, plan)
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

		bookmarksRendered, bookmarksCached, err := renderBookmarks(renderer, cfg, baseCtx, plan)
		if err != nil {
			stats.Errors++
			return err
		}
		stats.Rendered += bookmarksRendered
		stats.Cached += bookmarksCached
	}
	done()

	// --- First-party analytics script ---
	if cfg.Analytics {
		apiURL := cfg.Interactions.APIURL
		if err := GenerateAnalyticsScript(cfg.PublicDir, apiURL); err != nil {
			logger.Warn("analytics script generation failed", "error", err)
		}
	}

	// --- Stage: minify ---
	done = timings.stage("minify")
	// Post-render: minify HTML, CSS, JS, JSON, SVG, XML files in public/.
	if cfg.Minify {
		minified, err := minifyDir(cfg.PublicDir, logger)
		if err != nil {
			logger.Warn("minification walk failed", "error", err)
		} else {
			logger.Info("minified output files", "count", minified)
		}
	}
	done()

	if plugins != nil {
		finishPayload := cloneMap(baseCtx)
		finishPayload["stats"] = buildStatsView(stats, siteIndex)
		_ = plugins.Emit(ctx, "build.finished", finishPayload)
	}

	// Record total build time and log stage breakdown.
	timings.Total = time.Since(buildStart)
	timings.Log(logger)

	// Export timing as JSON if requested.
	if opts.TimingJSON != "" {
		if err := timings.WriteJSON(opts.TimingJSON); err != nil {
			logger.Warn("failed to write timing JSON", "error", err)
		}
	}

	// Append to build history.
	appendBuildHistory(timings, stats)

	logger.Info("build summary",
		"total", stats.Total,
		"rendered", stats.Rendered,
		"skipped", stats.Skipped,
		"cached", stats.Cached,
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		webhook.Dispatch(ctx, cfg, "build.failure", map[string]any{
			"stats": buildStatsView(stats, siteIndex),
		}, logger)
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	if cacheToSave != nil {
		cacheToSave.Outputs = buildOutputsIndex(siteIndex, cfg.PublicDir)
		cacheToSave.PageTemplates = buildPageTemplatesIndex(siteIndex)
		cacheToSave.SectionPages = buildSectionPagesIndex(siteIndex)
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

	webhook.Dispatch(ctx, cfg, "build.success", map[string]any{
		"stats": buildStatsView(stats, siteIndex),
	}, logger)

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

// buildPageTemplatesIndex records which template each page uses.
func buildPageTemplatesIndex(siteIndex *site.Site) map[string]string {
	if siteIndex == nil {
		return nil
	}
	idx := make(map[string]string, len(siteIndex.Pages))
	for _, page := range siteIndex.Pages {
		if page.SourcePath == "" {
			continue
		}
		tpl := page.Template
		if tpl == "" {
			tpl = "page.html"
		}
		idx[page.SourcePath] = tpl
	}
	return idx
}

// buildSectionPagesIndex records which pages belong to each section.
func buildSectionPagesIndex(siteIndex *site.Site) map[string][]string {
	if siteIndex == nil {
		return nil
	}
	idx := make(map[string][]string, len(siteIndex.Sections))
	for _, section := range siteIndex.Sections {
		if section == nil {
			continue
		}
		sources := make([]string, 0, len(section.Pages))
		for _, p := range section.Pages {
			if p.SourcePath != "" {
				sources = append(sources, p.SourcePath)
			}
		}
		idx[section.Path] = sources
	}
	return idx
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
	// Build page -> section mapping for breadcrumbs.
	pageSection := make(map[*site.Page]*site.Section)
	var walkSections func(s *site.Section)
	walkSections = func(s *site.Section) {
		for _, p := range s.Pages {
			pageSection[p] = s
		}
		for _, sub := range s.Subsections {
			walkSections(sub)
		}
	}
	if siteIndex.Root != nil {
		walkSections(siteIndex.Root)
	}

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

	// Build series index: series name -> sorted pages.
	seriesIndex := buildSeriesIndex(siteIndex.Pages)

	// Build backlink index: page path -> pages that link to it.
	backlinkIndex := buildBacklinkIndex(siteIndex.Pages)

	// Separate pages into cached vs to-render.
	type renderJob struct {
		page       *site.Page
		template   string
		outputPath string
		renderCtx  map[string]any
	}

	var jobs []renderJob
	cached := 0
	for _, page := range siteIndex.Pages {
		templateName := page.Template
		if templateName == "" {
			templateName = "page.html"
		}

		outputPath := outputHTMLPath(cfg.PublicDir, page.Path)
		if !plan.shouldRenderPage(page, outputPath, templateName) {
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

		// Breadcrumb section context (only for non-root sections).
		if sec, ok := pageSection[page]; ok && !sec.IsRoot {
			renderCtx["page_section"] = map[string]any{
				"title": sec.Title,
				"path":  sec.Path,
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

		// Series navigation (pages with same series name).
		if page.Series != "" {
			if seriesPages, ok := seriesIndex[page.Series]; ok && len(seriesPages) > 1 {
				views := make([]map[string]any, 0, len(seriesPages))
				for i, sp := range seriesPages {
					views = append(views, sp.View())
					if sp == page {
						if i > 0 {
							renderCtx["series_prev"] = seriesPages[i-1].View()
						}
						if i < len(seriesPages)-1 {
							renderCtx["series_next"] = seriesPages[i+1].View()
						}
					}
				}
				renderCtx["series_name"] = page.Series
				renderCtx["series_pages"] = views
			}
		}

		// Backlinks: pages that link to this page.
		if backlinks, ok := backlinkIndex[page.Path]; ok && len(backlinks) > 0 {
			views := make([]map[string]any, 0, len(backlinks))
			for _, bp := range backlinks {
				views = append(views, bp.View())
			}
			renderCtx["backlinks"] = views
		}

		renderCtx = applyPluginOverrides(ctx, plugins, "page.before_render", renderCtx)
		renderCtx = applyPluginOverrides(ctx, plugins, "page.render", renderCtx)
		preserveHTMLFields(renderCtx)
		jobs = append(jobs, renderJob{page: page, template: templateName, outputPath: outputPath, renderCtx: renderCtx})
	}

	if len(jobs) == 0 {
		return 0, cached, nil
	}

	// Render pages in parallel with a bounded worker pool.
	workers := runtime.NumCPU()
	if workers > len(jobs) {
		workers = len(jobs)
	}

	jobCh := make(chan renderJob, len(jobs))
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				if err := renderer.RenderToFile(j.template, j.renderCtx, j.outputPath); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
			}
		}()
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return 0, cached, err
	default:
	}

	return len(jobs), cached, nil
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

// buildSeriesIndex groups pages by series name, sorted by SeriesOrder then Date.
func buildSeriesIndex(pages []*site.Page) map[string][]*site.Page {
	index := map[string][]*site.Page{}
	for _, p := range pages {
		if p.Series != "" && !p.Menu {
			index[p.Series] = append(index[p.Series], p)
		}
	}
	for name := range index {
		sort.Slice(index[name], func(i, j int) bool {
			a, b := index[name][i], index[name][j]
			if a.SeriesOrder != b.SeriesOrder {
				return a.SeriesOrder < b.SeriesOrder
			}
			return a.Date.Before(b.Date) // oldest first within series
		})
	}
	return index
}

// hrefRe matches href attributes in rendered HTML links.
var hrefRe = regexp.MustCompile(`<a\s[^>]*href=["']([^"']+)["']`)

// buildBacklinkIndex scans rendered HTML content for internal links and
// builds a reverse index: target page path -> pages that link to it.
func buildBacklinkIndex(pages []*site.Page) map[string][]*site.Page {
	// Build a set of known page paths for quick lookup.
	pathSet := make(map[string]*site.Page, len(pages))
	for _, p := range pages {
		pathSet[p.Path] = p
		// Also index without trailing slash for flexible matching.
		trimmed := strings.TrimSuffix(p.Path, "/")
		if trimmed != "" {
			pathSet[trimmed] = p
		}
	}

	index := map[string][]*site.Page{}
	seen := map[string]map[*site.Page]bool{} // deduplicate

	for _, page := range pages {
		if page.Menu {
			continue
		}
		matches := hrefRe.FindAllStringSubmatch(page.Content, -1)
		for _, m := range matches {
			href := m[1]
			// Only consider internal links (relative paths starting with /).
			if !strings.HasPrefix(href, "/") {
				continue
			}
			// Strip fragment.
			if idx := strings.Index(href, "#"); idx >= 0 {
				href = href[:idx]
			}
			target, ok := pathSet[href]
			if !ok {
				continue
			}
			if target == page {
				continue // skip self-links
			}
			if seen[target.Path] == nil {
				seen[target.Path] = map[*site.Page]bool{}
			}
			if !seen[target.Path][page] {
				seen[target.Path][page] = true
				index[target.Path] = append(index[target.Path], page)
			}
		}
	}
	return index
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

		// Homepage pagination: split root section pages into multiple index pages.
		if section.IsRoot && cfg.PostsPerPage > 0 {
			r, c, err := renderPaginatedIndex(ctx, renderer, cfg, baseCtx, section, templateName, plugins, plan)
			if err != nil {
				return rendered, cached, err
			}
			rendered += r
			cached += c
			continue
		}

		outputPath := outputHTMLPath(cfg.PublicDir, section.Path)
		if !plan.shouldRenderSection(section.Path, outputPath, templateName) {
			cached++
			continue
		}
		renderCtx := sectionContext(baseCtx, section)
		renderCtx = applyPluginOverrides(ctx, plugins, "section.render", renderCtx)
		preserveHTMLFields(renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}
	return rendered, cached, nil
}

// renderPaginatedIndex generates paginated index pages for the homepage.
// Page 1 is written to /index.html, page 2 to /page/2/index.html, etc.
func renderPaginatedIndex(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, section *site.Section, templateName string, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
	paginators := taxonomy.BuildPaginator(section.Pages, cfg.PostsPerPage, "/", "page")
	rendered := 0
	cached := 0

	if len(paginators) == 0 {
		// Fewer pages than PostsPerPage — render the index without pagination.
		outputPath := outputHTMLPath(cfg.PublicDir, section.Path)
		if !plan.shouldRenderCollection(outputPath) {
			return 0, 1, nil
		}
		renderCtx := sectionContext(baseCtx, section)
		renderCtx = applyPluginOverrides(ctx, plugins, "section.render", renderCtx)
		preserveHTMLFields(renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return 0, 0, err
		}
		return 1, 0, nil
	}

	for i, paginator := range paginators {
		pagePath := "/"
		if i > 0 {
			pagePath = fmt.Sprintf("/page/%d/", i+1)
		}
		outputPath := outputHTMLPath(cfg.PublicDir, pagePath)
		if !plan.shouldRenderCollection(outputPath) {
			cached++
			continue
		}
		renderCtx := paginatedSectionContext(baseCtx, section, paginator)
		renderCtx = applyPluginOverrides(ctx, plugins, "section.render", renderCtx)
		preserveHTMLFields(renderCtx)
		if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
			return rendered, cached, err
		}
		rendered++
	}

	return rendered, cached, nil
}

// paginatedSectionContext builds a section context with paginated pages
// and pagination metadata for the template.
func paginatedSectionContext(baseCtx map[string]any, section *site.Section, paginator taxonomy.Paginator) map[string]any {
	ctx := cloneMap(baseCtx)

	// Build a section view with the paginated subset of pages.
	sectionView := section.View()
	paginatedPages := make([]map[string]any, 0, len(paginator.Pages))
	for _, page := range paginator.Pages {
		paginatedPages = append(paginatedPages, page.View())
	}
	sectionView["pages"] = paginatedPages

	ctx["section"] = sectionView
	ctx["current_path"] = section.Path
	ctx["current_url"] = section.Permalink
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	ctx["paginator"] = taxonomy.PaginatorView(paginator)

	return ctx
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
		"config":       configView(cfg),
		"site":         siteView,
		"current_year": time.Now().Year(),
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
	// If nav_taxonomy is set, expose its terms as nav_terms for header nav.
	if cfg.NavTaxonomy != "" {
		if index, ok := indices[cfg.NavTaxonomy]; ok {
			terms := index.TermsSorted()
			navTerms := make([]map[string]any, 0, len(terms))
			for _, t := range terms {
				navTerms = append(navTerms, map[string]any{
					"name":      t.Name,
					"permalink": t.Permalink,
				})
			}
			ctx["nav_terms"] = navTerms
		}
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
		"logo":               cfg.Logo,
		"favicon":            cfg.Favicon,
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
		"section_feeds":      cfg.SectionFeeds,
		"posts_per_page":     cfg.PostsPerPage,
		"image_optimization": cfg.ImageOptimization,
		"image_quality":      cfg.ImageQuality,
		"image_widths":       cfg.ImageWidths,
		"lightbox":           cfg.Lightbox,
		"sharing":            cfg.Sharing,
		"breadcrumbs":        cfg.Breadcrumbs,
		"math":               cfg.Math,
		"minify":             cfg.Minify,
		"nav_taxonomy":       cfg.NavTaxonomy,
		"multilingual":       cfg.IsMultilingual(),
		"languages":          languagesView(cfg),
		"author_bio":         cfg.AuthorBio,
		"author_avatar":      cfg.AuthorAvatar,
		"social":             cfg.Social,
		"copyright":          strings.ReplaceAll(cfg.Copyright, "{year}", fmt.Sprintf("%d", time.Now().Year())),
		"license":            template.HTML(inlineMarkdownLinks(cfg.License)),
		"llmstxt":            slices.Contains(cfg.PluginsEnabled, "llmstxt"),
		"logging": map[string]any{
			"level":  cfg.Logging.Level,
			"format": cfg.Logging.Format,
		},
		"taxonomies":           taxonomies,
		"interactions_enabled": cfg.Interactions.Enabled,
		"interactions_api_url": cfg.Interactions.APIURL,
		"comments_enabled":     cfg.Interactions.Comments.Enabled && len(cfg.Interactions.Comments.Providers) > 0,
		"comments_providers":   commentsProvidersView(cfg.Interactions.Comments),
		"analytics":            cfg.Analytics,
		"analytics_head":       template.HTML(cfg.HeadExtra),
		"analytics_body":       template.HTML(buildAnalyticsBody(cfg)),
	}
}

// reInlineLink matches markdown-style links: [text](url)
var reInlineLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// inlineMarkdownLinks converts [text](url) to <a> tags in a string.
// Used for the license config field so users can embed links naturally.
func inlineMarkdownLinks(s string) string {
	if s == "" {
		return s
	}
	return reInlineLink.ReplaceAllString(s, `<a href="$2" rel="noopener" target="_blank">$1</a>`)
}

// buildAnalyticsBody combines third-party provider snippets with body_extra.
// Provider scripts are injected before </body> as recommended by most providers.
func buildAnalyticsBody(cfg config.Config) string {
	var sb strings.Builder
	sb.WriteString(analyticsSnippets(cfg.AnalyticsProviders))
	if cfg.BodyExtra != "" {
		sb.WriteString(cfg.BodyExtra)
		if !strings.HasSuffix(cfg.BodyExtra, "\n") {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// commentsProvidersView builds the template-friendly slice of comment
// auth providers (provider name + display label).
func commentsProvidersView(ccfg config.CommentsConfig) []map[string]any {
	var providers []map[string]any
	for _, p := range ccfg.Providers {
		label := p.Provider
		switch p.Provider {
		case "github":
			label = "GitHub"
		case "google":
			label = "Google"
		}
		providers = append(providers, map[string]any{
			"provider": p.Provider,
			"label":    label,
		})
	}
	return providers
}

func languagesView(cfg config.Config) []map[string]any {
	var langs []map[string]any
	// Include the default language first.
	langs = append(langs, map[string]any{
		"code":    cfg.DefaultLanguage,
		"label":   cfg.LanguageLabel(cfg.DefaultLanguage),
		"default": true,
	})
	for _, l := range cfg.Languages {
		langs = append(langs, map[string]any{
			"code":    l.Code,
			"label":   l.Label,
			"default": false,
		})
	}
	return langs
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

// preserveHTMLFields ensures that content fields remain template.HTML after
// plugin deep-merge (which replaces them with plain strings from JSON).
// Handles both page and section contexts.
func preserveHTMLFields(renderCtx map[string]any) {
	for _, key := range []string{"page", "section"} {
		m, ok := renderCtx[key].(map[string]any)
		if !ok {
			continue
		}
		if content, ok := m["content"].(string); ok {
			m["content"] = template.HTML(content)
		}
	}
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
		if page.Draft {
			continue
		}
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
func renderSiteFeed(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
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
		feedCtx := siteFeedContext(baseCtx, cfg, pages, feedURL, lastUpdated)
		feedCtx = applyPluginOverrides(ctx, plugins, "feed.transform", feedCtx)
		if err := renderer.RenderToFile(tmpl, feedCtx, outputPath); err != nil {
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

// renderSectionFeeds generates atom.xml and rss.xml inside each non-root
// section directory (e.g. /blog/atom.xml). Sections with zero non-draft
// pages are skipped.
func renderSectionFeeds(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, siteIndex *site.Site, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
	if !cfg.SectionFeeds {
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

	rendered := 0
	cached := 0

	for _, section := range siteIndex.Sections {
		if section.IsRoot {
			continue // site feed already handles the root
		}
		pages := feedPages(section.Pages)
		if len(pages) == 0 {
			continue
		}

		lastUpdated := latestUpdated(section.Pages)

		for _, tmpl := range feedTemplates {
			feedURL := buildURL(cfg.BaseURL, section.Path+tmpl)
			outputPath := outputFilePath(cfg.PublicDir, section.Path, tmpl)
			if !plan.shouldRenderCollection(outputPath) {
				cached++
				continue
			}
			feedCtx := sectionFeedContext(baseCtx, cfg, section, pages, feedURL, lastUpdated)
			feedCtx = applyPluginOverrides(ctx, plugins, "feed.transform", feedCtx)
			if err := renderer.RenderToFile(tmpl, feedCtx, outputPath); err != nil {
				return rendered, cached, err
			}
			rendered++
		}
	}

	return rendered, cached, nil
}

// sectionFeedContext builds the template context for a per-section feed.
// Uses feed_title/feed_description like the site feed so the same
// atom.xml/rss.xml templates work unchanged.
func sectionFeedContext(baseCtx map[string]any, cfg config.Config, section *site.Section, pages []map[string]any, feedURL string, lastUpdated time.Time) map[string]any {
	ctx := cloneMap(baseCtx)
	ctx["feed_title"] = section.Title
	if section.Title == "" {
		ctx["feed_title"] = section.Slug
	}
	ctx["feed_description"] = cfg.SiteDescription
	ctx["pages"] = pages
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

func renderSitemap(ctx context.Context, renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, entries []SitemapEntry, plugins *plugin.Manager, plan buildPlan) (int, int, error) {
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
		smCtx := cloneMap(baseCtx)
		smCtx["entries"] = sitemapEntryViews(entries)
		smCtx = applyPluginOverrides(ctx, plugins, "sitemap.transform", smCtx)
		outputPath := filepath.Join(cfg.PublicDir, "sitemap.xml")
		if !plan.shouldRenderCollection(outputPath) {
			return 0, 1, nil
		}
		if err := renderer.RenderToFile("sitemap.xml", smCtx, outputPath); err != nil {
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

func renderBookmarks(renderer *render.Renderer, cfg config.Config, baseCtx map[string]any, plan buildPlan) (int, int, error) {
	if !renderer.HasTemplate("bookmarks.html") {
		return 0, 0, nil
	}
	ctx := cloneMap(baseCtx)
	ctx["lang"] = defaultLangFromCtx(baseCtx)
	ctx["current_path"] = "/bookmarks/"
	outputPath := filepath.Join(cfg.PublicDir, "bookmarks", "index.html")
	if !plan.shouldRenderCollection(outputPath) {
		return 0, 1, nil
	}
	if err := renderer.RenderToFile("bookmarks.html", ctx, outputPath); err != nil {
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
		if page.Draft {
			continue
		}
		entry := SitemapEntry{
			Permalink: page.Permalink,
			Updated:   page.Date,
		}
		// Include hreflang alternates for translated pages.
		if len(page.Translations) > 0 {
			alternates := []map[string]string{
				{"lang": page.Lang, "href": page.Permalink},
			}
			for _, t := range page.Translations {
				alternates = append(alternates, map[string]string{
					"lang": t.Lang, "href": t.Permalink,
				})
			}
			entry.Extra = map[string]any{"alternates": alternates}
		}
		if strings.TrimSpace(entry.Permalink) != "" {
			existing, ok := entries[entry.Permalink]
			if !ok || entry.Updated.After(existing.Updated) {
				entries[entry.Permalink] = entry
			}
		}
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

// runDryRun builds the site to a temp directory, then prints what would be
// generated as a table of file paths and sizes.
func runDryRun(ctx context.Context, cfg config.Config, opts BuildOptions, verbose bool, logWriter io.Writer, logger *slog.Logger) error {
	tmpDir, err := os.MkdirTemp("", "osg-dryrun-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	origPublicDir := cfg.PublicDir
	cfg.PublicDir = tmpDir

	// Run the actual build silently.
	dryOpts := opts
	dryOpts.DryRun = false
	if err := Run(ctx, cfg, dryOpts, false, io.Discard); err != nil {
		return fmt.Errorf("dry-run build: %w", err)
	}

	// Walk the output and collect files.
	type entry struct {
		path string
		size int64
	}
	var entries []entry
	var totalSize int64

	_ = filepath.Walk(tmpDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(tmpDir, p)
		entries = append(entries, entry{path: "/" + filepath.ToSlash(relPath), size: info.Size()})
		totalSize += info.Size()
		return nil
	})

	// Sort by path for deterministic output.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	// Print table.
	w := logWriter
	if w == nil {
		w = os.Stdout
	}
	_, _ = fmt.Fprintf(w, "\n  Dry run: %d files would be generated in %s\n\n", len(entries), origPublicDir)
	_, _ = fmt.Fprintf(w, "  %-60s %10s\n", "PATH", "SIZE")
	_, _ = fmt.Fprintf(w, "  %-60s %10s\n", strings.Repeat("─", 60), strings.Repeat("─", 10))
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "  %-60s %10s\n", e.path, humanSize(e.size))
	}
	_, _ = fmt.Fprintf(w, "  %-60s %10s\n", strings.Repeat("─", 60), strings.Repeat("─", 10))
	_, _ = fmt.Fprintf(w, "  %-60s %10s\n\n", "TOTAL", humanSize(totalSize))

	return nil
}

func humanSize(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
