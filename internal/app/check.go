package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/site"
	"osg/internal/vault"
)

// CheckOptions carries flags specific to the check command.
type CheckOptions struct {
	Links       bool
	Images      bool
	Frontmatter bool
	JSON        bool
}

// CheckIssue represents a single validation issue found by osg check.
type CheckIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Category string `json:"category"` // "link", "image", "frontmatter"
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"` // source file
	Target   string `json:"target,omitempty"` // broken link/image target
	Detail   string `json:"detail,omitempty"` // extra info (size, etc.)
}

// CheckResult aggregates all issues found during a check run.
type CheckResult struct {
	Issues   []CheckIssue `json:"issues"`
	Errors   int          `json:"errors"`
	Warnings int          `json:"warnings"`
}

func RunCheck(ctx context.Context, opts CLIOptions, checkOpts CheckOptions) error {
	_ = ctx
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	// Default: run all checks.
	all := !checkOpts.Links && !checkOpts.Images && !checkOpts.Frontmatter

	result := &CheckResult{}

	// Build the site index (parse all content files).
	siteIndex, err := buildSiteIndex(cfg, logger)
	if err != nil {
		return fmt.Errorf("build site index: %w", err)
	}

	if all || checkOpts.Links {
		// Broken internal links in rendered HTML (requires site index).
		// Note: broken wikilinks in source markdown are checked by `osg doctor`.
		checkBrokenInternalLinks(cfg, siteIndex, result, logger)
	}

	if all || checkOpts.Images {
		checkBrokenImageRefs(cfg, siteIndex, result, logger)
		checkOrphanImages(cfg, siteIndex, result, logger)
	}

	if all || checkOpts.Frontmatter {
		checkMissingDates(siteIndex, result, logger)
		checkMissingTags(siteIndex, result, logger)
		checkDuplicateSlugs(siteIndex, result, logger)
		checkPermalinkCollisions(siteIndex, result, logger)
	}

	// Output.
	if checkOpts.JSON {
		return outputJSON(result, os.Stdout)
	}
	return outputText(result, logger)
}

// buildSiteIndex parses all content files and returns the populated site.
func buildSiteIndex(cfg config.Config, logger *slog.Logger) (*site.Site, error) {
	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		return nil, err
	}

	siteIndex := site.New()
	for _, fp := range files {
		page, section, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, fp)
		if err != nil {
			logger.Warn("failed to parse content", "path", fp, "error", err)
			continue
		}
		if page != nil {
			if page.Draft && !cfg.IncludeDrafts {
				continue
			}
			if page.Lang == "" {
				page.Lang = cfg.DefaultLanguage
			}
			siteIndex.AddPage(page)
		}
		if section != nil {
			siteIndex.AddSection(section)
		}
	}
	siteIndex.BuildHierarchy()
	return siteIndex, nil
}

// ── Link checks ────────────────────────────────────────────────────────

// hrefRe matches href attributes in rendered HTML.
var hrefRe = regexp.MustCompile(`href=["'](/[^"'#?]*)["'#?]?`)

// checkBrokenInternalLinks scans rendered HTML for internal links that don't
// correspond to any page or section path.
func checkBrokenInternalLinks(cfg config.Config, s *site.Site, result *CheckResult, logger *slog.Logger) {
	// Build set of known paths (pages + sections).
	known := make(map[string]bool)
	for _, p := range s.Pages {
		known[normalizeLinkPath(p.Path)] = true
	}
	for sp := range s.Sections {
		known[normalizeLinkPath(sp)] = true
	}
	// Add well-known generated paths.
	for _, p := range []string{"/", "/sitemap.xml", "/robots.txt", "/atom.xml", "/rss.xml", "/404.html",
		"/search/", "/archive/", "/llms.txt", "/llms-full.txt"} {
		known[normalizeLinkPath(p)] = true
	}
	// Add taxonomy term paths.
	for _, tax := range cfg.Taxonomies {
		known[normalizeLinkPath("/"+tax.Name+"/")] = true
	}

	broken := make(map[string][]string) // target -> source files
	for _, page := range s.Pages {
		matches := hrefRe.FindAllStringSubmatch(page.Content, -1)
		for _, m := range matches {
			href := m[1]
			normalized := normalizeLinkPath(href)
			if known[normalized] {
				continue
			}
			// Skip static asset paths (css, js, fonts, images).
			if isStaticAssetPath(href) {
				continue
			}
			// Skip taxonomy term pages (e.g. /tags/go/) — they are generated
			// dynamically and we can't enumerate them without running the full
			// taxonomy build.
			if isTaxonomyTermPath(href, cfg.Taxonomies) {
				continue
			}
			broken[href] = append(broken[href], page.SourcePath)
		}
	}

	targets := sortedKeys(broken)
	for _, target := range targets {
		sources := broken[target]
		for _, src := range sources {
			addIssue(result, "error", "link", "broken internal link",
				src, target, "")
		}
	}

	if len(targets) > 0 {
		checkInfo(logger, "broken internal links found", "count", len(targets))
	}
}

// ── Image checks ───────────────────────────────────────────────────────

// imgSrcRe matches src attributes in <img> tags in rendered HTML.
var imgSrcRe = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

// checkBrokenImageRefs scans rendered HTML for image references that point to
// files not present in static/, content/, or theme static/.
func checkBrokenImageRefs(cfg config.Config, s *site.Site, result *CheckResult, logger *slog.Logger) {
	// Build set of known static files (relative to public root).
	staticFiles := make(map[string]bool)
	collectStaticFiles(cfg.StaticDir, "", staticFiles)
	collectStaticFiles(filepath.Join(cfg.ThemesDir, cfg.Theme, "static"), "", staticFiles)
	// Content images are served from their content path.
	collectContentImages(cfg.ContentDir, staticFiles)

	brokenCount := 0
	for _, page := range s.Pages {
		matches := imgSrcRe.FindAllStringSubmatch(page.Content, -1)
		for _, m := range matches {
			src := m[1]
			// Skip external images.
			if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "data:") {
				continue
			}
			normalized := strings.TrimPrefix(src, "/")
			if staticFiles[normalized] {
				continue
			}
			// Check if the image exists relative to the page's content directory.
			if existsRelativeToPage(cfg.ContentDir, page.SourcePath, src) {
				continue
			}
			addIssue(result, "warning", "image", "broken image reference",
				page.SourcePath, src, "")
			brokenCount++
		}
	}

	if brokenCount > 0 {
		checkInfo(logger, "broken image references found", "count", brokenCount)
	}
}

// checkOrphanImages finds image files in the content directory that are not
// referenced by any page.
func checkOrphanImages(cfg config.Config, s *site.Site, result *CheckResult, logger *slog.Logger) {
	if !pathExists(cfg.ContentDir) {
		return
	}

	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".svg": true, ".webp": true, ".bmp": true, ".avif": true,
	}

	// Collect all image files in content dir.
	type imageFile struct {
		rel  string
		size int64
	}
	var images []imageFile
	_ = filepath.WalkDir(cfg.ContentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if !imageExts[ext] {
			return nil
		}
		rel, _ := filepath.Rel(cfg.ContentDir, path)
		info, infoErr := d.Info()
		size := int64(0)
		if infoErr == nil {
			size = info.Size()
		}
		images = append(images, imageFile{rel: rel, size: size})
		return nil
	})

	if len(images) == 0 {
		return
	}

	// Build set of all referenced images from raw content and rendered HTML.
	referenced := make(map[string]bool)
	for _, page := range s.Pages {
		// From rendered HTML.
		matches := imgSrcRe.FindAllStringSubmatch(page.Content, -1)
		for _, m := range matches {
			src := strings.TrimPrefix(m[1], "/")
			referenced[src] = true
			referenced[filepath.Base(src)] = true
		}
		// From raw markdown (image refs like ![alt](path)).
		addMarkdownImageRefs(page.RawContent, referenced)
		// From wikilinks like ![[image.png]].
		addWikilinkImageRefs(page.RawContent, referenced)
	}

	orphanCount := 0
	var totalSize int64
	for _, img := range images {
		name := filepath.Base(img.rel)
		if referenced[img.rel] || referenced[name] {
			continue
		}
		sizeMB := fmt.Sprintf("%.1f KB", float64(img.size)/1024)
		if img.size > 1024*1024 {
			sizeMB = fmt.Sprintf("%.1f MB", float64(img.size)/(1024*1024))
		}
		addIssue(result, "warning", "image", "orphan image (not referenced by any page)",
			img.rel, "", sizeMB)
		orphanCount++
		totalSize += img.size
	}

	if orphanCount > 0 {
		checkInfo(logger, "orphan images found", "count", orphanCount,
			"total_size_mb", fmt.Sprintf("%.1f", float64(totalSize)/(1024*1024)))
	}
}

// ── Frontmatter checks ────────────────────────────────────────────────

func checkMissingDates(s *site.Site, result *CheckResult, logger *slog.Logger) {
	count := 0
	for _, page := range s.Pages {
		if page.Date.IsZero() {
			addIssue(result, "warning", "frontmatter", "post without date (cannot sort chronologically)",
				page.SourcePath, "", "")
			count++
		}
	}
	if count > 0 {
		checkInfo(logger, "posts without dates", "count", count)
	}
}

func checkMissingTags(s *site.Site, result *CheckResult, logger *slog.Logger) {
	count := 0
	for _, page := range s.Pages {
		if page.Menu {
			continue // menu pages don't need tags
		}
		hasTags := false
		for _, terms := range page.Taxonomies {
			if len(terms) > 0 {
				hasTags = true
				break
			}
		}
		if !hasTags {
			addIssue(result, "warning", "frontmatter", "post without tags (uncategorized content)",
				page.SourcePath, "", "")
			count++
		}
	}
	if count > 0 {
		checkInfo(logger, "posts without tags", "count", count)
	}
}

func checkDuplicateSlugs(s *site.Site, result *CheckResult, logger *slog.Logger) {
	byPath := make(map[string][]*site.Page)
	for _, page := range s.Pages {
		np := normalizeLinkPath(page.Path)
		byPath[np] = append(byPath[np], page)
	}

	count := 0
	for p, pages := range byPath {
		if len(pages) < 2 {
			continue
		}
		sources := make([]string, len(pages))
		for i, pg := range pages {
			sources[i] = pg.SourcePath
		}
		for _, src := range sources {
			addIssue(result, "error", "frontmatter", "duplicate slug (URL collision)",
				src, p, "collides with: "+strings.Join(sources, ", "))
		}
		count++
	}
	if count > 0 {
		checkInfo(logger, "duplicate slug collisions", "count", count)
	}
}

func checkPermalinkCollisions(s *site.Site, result *CheckResult, logger *slog.Logger) {
	byPermalink := make(map[string][]*site.Page)
	for _, page := range s.Pages {
		if page.Permalink == "" || page.Permalink == page.Path {
			continue // only check explicit permalinks
		}
		np := normalizeLinkPath(page.Permalink)
		byPermalink[np] = append(byPermalink[np], page)
	}

	count := 0
	for pl, pages := range byPermalink {
		if len(pages) < 2 {
			continue
		}
		sources := make([]string, len(pages))
		for i, pg := range pages {
			sources[i] = pg.SourcePath
		}
		for _, src := range sources {
			addIssue(result, "error", "frontmatter", "permalink collision",
				src, pl, "collides with: "+strings.Join(sources, ", "))
		}
		count++
	}
	if count > 0 {
		checkInfo(logger, "permalink collisions", "count", count)
	}
}

// ── Output ─────────────────────────────────────────────────────────────

func outputJSON(result *CheckResult, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	if result.Errors > 0 {
		return fmt.Errorf("check found %d error(s)", result.Errors)
	}
	return nil
}

func outputText(result *CheckResult, logger *slog.Logger) error {
	if logger != nil {
		for _, issue := range result.Issues {
			args := []any{"category", issue.Category}
			if issue.Source != "" {
				args = append(args, "source", issue.Source)
			}
			if issue.Target != "" {
				args = append(args, "target", issue.Target)
			}
			if issue.Detail != "" {
				args = append(args, "detail", issue.Detail)
			}

			switch issue.Severity {
			case "error":
				logger.Error(issue.Message, args...)
			default:
				logger.Warn(issue.Message, args...)
			}
		}
	}

	checkInfo(logger, "check summary", "errors", result.Errors, "warnings", result.Warnings)
	if result.Errors > 0 {
		return fmt.Errorf("check found %d error(s)", result.Errors)
	}
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────

func addIssue(result *CheckResult, severity, category, message, source, target, detail string) {
	result.Issues = append(result.Issues, CheckIssue{
		Severity: severity,
		Category: category,
		Message:  message,
		Source:   source,
		Target:   target,
		Detail:   detail,
	})
	switch severity {
	case "error":
		result.Errors++
	default:
		result.Warnings++
	}
}

func normalizeLinkPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	// Remove query string and fragment.
	if idx := strings.IndexAny(p, "?#"); idx >= 0 {
		p = p[:idx]
	}
	// Parse URL-encoded paths.
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Normalize trailing slash: /foo -> /foo/  (unless it has an extension).
	if !strings.HasSuffix(p, "/") && filepath.Ext(p) == "" {
		p += "/"
	}
	return p
}

func isStaticAssetPath(href string) bool {
	ext := strings.ToLower(filepath.Ext(href))
	switch ext {
	case ".css", ".js", ".woff", ".woff2", ".ttf", ".eot", ".otf",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".avif",
		".xml", ".json", ".txt", ".wasm", ".map":
		return true
	}
	return false
}

func isTaxonomyTermPath(href string, taxonomies []config.TaxonomyConfig) bool {
	for _, tax := range taxonomies {
		prefix := "/" + tax.Name + "/"
		if strings.HasPrefix(href, prefix) && href != prefix {
			return true
		}
	}
	return false
}

func collectStaticFiles(dir string, prefix string, out map[string]bool) {
	if !pathExists(dir) {
		return
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if prefix != "" {
			rel = prefix + "/" + rel
		}
		out[filepath.ToSlash(rel)] = true
		return nil
	})
}

func collectContentImages(contentDir string, out map[string]bool) {
	if !pathExists(contentDir) {
		return
	}
	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".svg": true, ".webp": true, ".bmp": true, ".avif": true,
	}
	_ = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if imageExts[ext] {
			rel, _ := filepath.Rel(contentDir, path)
			out[filepath.ToSlash(rel)] = true
		}
		return nil
	})
}

func existsRelativeToPage(contentDir, sourcePath, imgSrc string) bool {
	if strings.HasPrefix(imgSrc, "/") {
		// Absolute path: check from content root.
		check := filepath.Join(contentDir, imgSrc)
		return pathExists(check)
	}
	// Relative path: check from the page's directory.
	pageDir := filepath.Dir(sourcePath)
	check := filepath.Join(pageDir, imgSrc)
	return pathExists(check)
}

var mdImageRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

func addMarkdownImageRefs(raw string, out map[string]bool) {
	matches := mdImageRe.FindAllStringSubmatch(raw, -1)
	for _, m := range matches {
		src := strings.TrimSpace(m[1])
		// Strip title if present: "path/img.png \"title\""
		if idx := strings.Index(src, " "); idx > 0 {
			src = src[:idx]
		}
		src = strings.TrimPrefix(src, "/")
		out[src] = true
		out[filepath.Base(src)] = true
	}
}

var wikilinkImageRe = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

func addWikilinkImageRefs(raw string, out map[string]bool) {
	matches := wikilinkImageRe.FindAllStringSubmatch(raw, -1)
	for _, m := range matches {
		ref := strings.TrimSpace(m[1])
		out[ref] = true
		out[filepath.Base(ref)] = true
	}
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
