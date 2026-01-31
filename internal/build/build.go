package build

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/render"
	"osg/internal/site"
	"osg/internal/taxonomy"
	"osg/internal/vault"
)

type Stats struct {
	Total    int
	Rendered int
	Skipped  int
	Errors   int
}

func Run(cfg config.Config, verbose bool) error {
	logger := logging.New(cfg.Logging, verbose)

	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no content files found", "content_dir", cfg.ContentDir)
		return nil
	}

	if err := os.MkdirAll(cfg.PublicDir, 0o755); err != nil {
		return fmt.Errorf("create public dir: %w", err)
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

	renderedSections, err := renderSections(renderer, cfg, siteView, siteIndex)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += renderedSections

	renderedPages, err := renderPages(renderer, cfg, siteView, siteIndex)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += renderedPages

	taxonomyRendered, err := renderTaxonomies(renderer, cfg, siteView, siteIndex, indices)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += taxonomyRendered

	sitemapEntries := collectSitemapEntries(cfg, siteIndex, indices)
	sitemapRendered, err := renderSitemap(renderer, cfg, siteView, sitemapEntries)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += sitemapRendered

	robotsRendered, err := renderRobots(renderer, cfg, siteView)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += robotsRendered

	notFoundRendered, err := renderNotFound(renderer, cfg, siteView)
	if err != nil {
		stats.Errors++
		return err
	}
	stats.Rendered += notFoundRendered

	logger.Info("build summary",
		"total", stats.Total,
		"rendered", stats.Rendered,
		"skipped", stats.Skipped,
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}

func renderPages(renderer *render.Renderer, cfg config.Config, siteView map[string]any, siteIndex *site.Site) (int, error) {
	rendered := 0
	for _, page := range siteIndex.Pages {
		templateName := page.Template
		if templateName == "" {
			templateName = "page.html"
		}

		outputPath := outputHTMLPath(cfg.PublicDir, page.Path)
		ctx := pageContext(cfg, siteView, page)
		if err := renderer.RenderToFile(templateName, ctx, outputPath); err != nil {
			return rendered, err
		}
		rendered++
	}
	return rendered, nil
}

func renderSections(renderer *render.Renderer, cfg config.Config, siteView map[string]any, siteIndex *site.Site) (int, error) {
	rendered := 0
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
		ctx := sectionContext(cfg, siteView, section)
		if err := renderer.RenderToFile(templateName, ctx, outputPath); err != nil {
			return rendered, err
		}
		rendered++
	}
	return rendered, nil
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

func pageContext(cfg config.Config, siteView map[string]any, page *site.Page) map[string]any {
	return map[string]any{
		"config":       cfg,
		"site":         siteView,
		"page":         page.View(),
		"current_path": page.Path,
		"current_url":  page.Permalink,
		"lang":         page.Lang,
	}
}

func sectionContext(cfg config.Config, siteView map[string]any, section *site.Section) map[string]any {
	return map[string]any{
		"config":       cfg,
		"site":         siteView,
		"section":      section.View(),
		"current_path": section.Path,
		"current_url":  section.Permalink,
		"lang":         "",
	}
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

func renderTaxonomies(renderer *render.Renderer, cfg config.Config, siteView map[string]any, siteIndex *site.Site, indices map[string]*taxonomy.Index) (int, error) {
	if len(cfg.Taxonomies) == 0 {
		return 0, nil
	}

	rendered := 0

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
		listCtx := taxonomyListContext(cfg, siteView, taxCfg, terms, listPath)
		if err := renderer.RenderToFile(listTemplate, listCtx, listOutput); err != nil {
			return rendered, err
		}
		rendered++

		singleTemplate := taxonomyTemplateName(renderer, taxCfg.Name, "single.html", "taxonomy_single.html")
		for _, term := range terms {
			paginators := taxonomy.BuildPaginator(term.Pages, taxCfg.PaginateBy, term.Path, taxCfg.PaginatePath)
			if len(paginators) == 0 {
				termPath := term.Path
				outputPath := outputHTMLPath(cfg.PublicDir, termPath)
				ctx := taxonomyTermContext(cfg, siteView, taxCfg, term, termPath, nil)
				if err := renderer.RenderToFile(singleTemplate, ctx, outputPath); err != nil {
					return rendered, err
				}
				rendered++
				renderedFeeds, err := renderTaxonomyFeeds(renderer, cfg, siteView, taxCfg, term)
				if err != nil {
					return rendered, err
				}
				rendered += renderedFeeds
				continue
			}

			for i, paginator := range paginators {
				pagePath := taxonomyPagePath(term.Path, taxCfg.PaginatePath, i)
				outputPath := outputHTMLPath(cfg.PublicDir, pagePath)
				ctx := taxonomyTermContext(cfg, siteView, taxCfg, term, pagePath, &paginator)
				if err := renderer.RenderToFile(singleTemplate, ctx, outputPath); err != nil {
					return rendered, err
				}
				rendered++
			}
			renderedFeeds, err := renderTaxonomyFeeds(renderer, cfg, siteView, taxCfg, term)
			if err != nil {
				return rendered, err
			}
			rendered += renderedFeeds
		}
	}

	return rendered, nil
}

func taxonomyTemplateName(renderer *render.Renderer, taxonomyName string, specific string, fallback string) string {
	specificName := path.Join(taxonomyName, specific)
	if renderer.HasTemplate(specificName) {
		return specificName
	}
	return fallback
}

func taxonomyListContext(cfg config.Config, siteView map[string]any, taxCfg config.TaxonomyConfig, terms []*taxonomy.Term, currentPath string) map[string]any {
	return map[string]any{
		"config":       cfg,
		"site":         siteView,
		"taxonomy":     taxonomy.ConfigView(taxCfg),
		"terms":        taxonomy.TermViews(terms),
		"current_path": currentPath,
		"current_url":  buildURL(cfg.BaseURL, currentPath),
		"lang":         "",
	}
}

func taxonomyTermContext(cfg config.Config, siteView map[string]any, taxCfg config.TaxonomyConfig, term *taxonomy.Term, currentPath string, paginator *taxonomy.Paginator) map[string]any {
	ctx := map[string]any{
		"config":       cfg,
		"site":         siteView,
		"taxonomy":     taxonomy.ConfigView(taxCfg),
		"term":         taxonomy.TermView(term),
		"current_path": currentPath,
		"current_url":  buildURL(cfg.BaseURL, currentPath),
		"lang":         "",
	}
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

func renderTaxonomyFeeds(renderer *render.Renderer, cfg config.Config, siteView map[string]any, taxCfg config.TaxonomyConfig, term *taxonomy.Term) (int, error) {
	if !taxCfg.Feed {
		return 0, nil
	}

	feedTemplates := []string{}
	if renderer.HasTemplate("atom.xml") {
		feedTemplates = append(feedTemplates, "atom.xml")
	}
	if renderer.HasTemplate("rss.xml") {
		feedTemplates = append(feedTemplates, "rss.xml")
	}
	if len(feedTemplates) == 0 {
		return 0, nil
	}

	lastUpdated := latestUpdated(term.Pages)
	rendered := 0

	for _, tmpl := range feedTemplates {
		filename := tmpl
		feedURL := buildURL(cfg.BaseURL, path.Join(term.Path, filename))
		outputPath := outputFilePath(cfg.PublicDir, term.Path, filename)
		ctx := feedContext(cfg, siteView, taxCfg, term, feedURL, lastUpdated)
		if err := renderer.RenderToFile(tmpl, ctx, outputPath); err != nil {
			return rendered, err
		}
		rendered++
	}

	return rendered, nil
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

func feedContext(cfg config.Config, siteView map[string]any, taxCfg config.TaxonomyConfig, term *taxonomy.Term, feedURL string, lastUpdated time.Time) map[string]any {
	return map[string]any{
		"config":       cfg,
		"site":         siteView,
		"taxonomy":     taxonomy.ConfigView(taxCfg),
		"term":         taxonomy.TermView(term),
		"pages":        feedPages(term.Pages),
		"feed_url":     feedURL,
		"last_updated": lastUpdated.Format(time.RFC3339),
		"lang":         "",
	}
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

func renderSitemap(renderer *render.Renderer, cfg config.Config, siteView map[string]any, entries []SitemapEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	if !renderer.HasTemplate("sitemap.xml") {
		return 0, nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Permalink < entries[j].Permalink
	})

	if len(entries) <= maxSitemapEntries {
		ctx := map[string]any{
			"config":  cfg,
			"site":    siteView,
			"entries": sitemapEntryViews(entries),
		}
		outputPath := filepath.Join(cfg.PublicDir, "sitemap.xml")
		if err := renderer.RenderToFile("sitemap.xml", ctx, outputPath); err != nil {
			return 0, err
		}
		return 1, nil
	}

	if !renderer.HasTemplate("split_sitemap_index.xml") {
		return 0, fmt.Errorf("split sitemap required but split_sitemap_index.xml template missing")
	}

	sitemaps := make([]map[string]any, 0)
	rendered := 0
	for i := 0; i < len(entries); i += maxSitemapEntries {
		end := i + maxSitemapEntries
		if end > len(entries) {
			end = len(entries)
		}
		chunk := entries[i:end]
		filename := fmt.Sprintf("sitemap_%d.xml", (i/maxSitemapEntries)+1)
		outputPath := filepath.Join(cfg.PublicDir, filename)
		ctx := map[string]any{
			"config":  cfg,
			"site":    siteView,
			"entries": sitemapEntryViews(chunk),
		}
		if err := renderer.RenderToFile("sitemap.xml", ctx, outputPath); err != nil {
			return rendered, err
		}
		rendered++

		loc := buildURL(cfg.BaseURL, "/"+filename)
		lastmod := chunk[len(chunk)-1].Updated.Format(time.RFC3339)
		sitemaps = append(sitemaps, map[string]any{
			"loc":     loc,
			"lastmod": lastmod,
		})
	}

	indexCtx := map[string]any{
		"config":   cfg,
		"site":     siteView,
		"sitemaps": sitemaps,
	}
	indexPath := filepath.Join(cfg.PublicDir, "sitemap.xml")
	if err := renderer.RenderToFile("split_sitemap_index.xml", indexCtx, indexPath); err != nil {
		return rendered, err
	}
	rendered++

	return rendered, nil
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

func renderRobots(renderer *render.Renderer, cfg config.Config, siteView map[string]any) (int, error) {
	if !renderer.HasTemplate("robots.txt") {
		return 0, nil
	}
	ctx := map[string]any{
		"config": cfg,
		"site":   siteView,
	}
	outputPath := filepath.Join(cfg.PublicDir, "robots.txt")
	if err := renderer.RenderToFile("robots.txt", ctx, outputPath); err != nil {
		return 0, err
	}
	return 1, nil
}

func renderNotFound(renderer *render.Renderer, cfg config.Config, siteView map[string]any) (int, error) {
	if !renderer.HasTemplate("404.html") {
		return 0, nil
	}
	ctx := map[string]any{
		"config": cfg,
		"site":   siteView,
		"lang":   "",
	}
	outputPath := filepath.Join(cfg.PublicDir, "404.html")
	if err := renderer.RenderToFile("404.html", ctx, outputPath); err != nil {
		return 0, err
	}
	return 1, nil
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
