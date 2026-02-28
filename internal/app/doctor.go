package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/theme"
)

type doctorCounters struct {
	warn  int
	error int
}

func RunDoctor(ctx context.Context, opts CLIOptions) error {
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
	counts := doctorCounters{}

	profile := normalizeProfile(cfg.DoctorProfile)
	checkInfo(logger, "doctor starting", "profile", profile)

	// ── Category: Config ────────────────────────────────────────────
	checkInfo(logger, "checking config")

	if strings.TrimSpace(cfg.BaseURL) == "" {
		if profile == "prod" {
			checkError(logger, &counts, "base_url is empty",
				"fix", "set base_url in config.yaml to your site's public URL (e.g. https://example.com)")
		} else {
			checkWarn(logger, &counts, "base_url is empty",
				"fix", "set base_url before deploying to production")
		}
	} else if err := validateBaseURL(cfg.BaseURL); err != nil {
		if profile == "prod" {
			checkError(logger, &counts, "base_url invalid",
				"error", err.Error(),
				"fix", "base_url must include scheme and host (e.g. https://example.com)")
		} else {
			checkWarn(logger, &counts, "base_url invalid",
				"error", err.Error(),
				"fix", "base_url should include scheme and host (e.g. https://example.com)")
		}
	}

	if strings.TrimSpace(cfg.SummaryStrategy) != "" {
		switch cfg.SummaryStrategy {
		case "auto", "manual", "ai":
			// valid
		default:
			checkWarn(logger, &counts, "unknown summary_strategy",
				"value", cfg.SummaryStrategy,
				"fix", "valid values: auto, manual, ai")
		}
	}

	// ── Category: Paths ─────────────────────────────────────────────
	checkInfo(logger, "checking paths")

	checkPath(logger, &counts, "vault_path", cfg.VaultPath, profile == "prod")
	checkPath(logger, &counts, "content_dir", cfg.ContentDir, profile == "prod")
	checkPath(logger, &counts, "public_dir", cfg.PublicDir, profile == "prod")
	checkPath(logger, &counts, "templates_dir", cfg.TemplatesDir, profile == "prod")
	checkPath(logger, &counts, "static_dir", cfg.StaticDir, profile == "prod")
	checkPath(logger, &counts, "themes_dir", cfg.ThemesDir, profile == "prod")
	checkPath(logger, &counts, "plugins_dir", cfg.PluginsDir, false)
	checkPath(logger, &counts, "sass_dir", cfg.SassDir, cfg.CompileSass && profile == "prod")

	// ── Category: Theme ─────────────────────────────────────────────
	checkInfo(logger, "checking theme")

	themePath := filepath.Join(cfg.ThemesDir, cfg.Theme)
	if strings.TrimSpace(cfg.Theme) == "" {
		if profile == "prod" {
			checkError(logger, &counts, "theme is empty",
				"fix", "set theme in config.yaml (e.g. theme: default)")
		} else {
			checkWarn(logger, &counts, "theme is empty",
				"fix", "set theme in config.yaml (e.g. theme: default)")
		}
	} else if !pathExists(themePath) {
		if profile == "prod" {
			checkError(logger, &counts, "theme not found",
				"theme", cfg.Theme, "path", themePath,
				"fix", "run 'osg build' to sync the default theme, or install a theme to "+themePath)
		} else {
			checkWarn(logger, &counts, "theme not found",
				"theme", cfg.Theme, "path", themePath,
				"fix", "run 'osg build' to sync the default theme")
		}
	}

	checkThemeTemplates(logger, &counts, cfg, profile)
	checkRequiredTemplates(logger, &counts, cfg, profile)
	checkThemeMeta(logger, &counts, cfg)

	// ── Category: Taxonomies ────────────────────────────────────────
	checkInfo(logger, "checking taxonomies")
	checkTaxonomies(logger, &counts, cfg.Taxonomies, profile)

	// ── Category: Plugins ───────────────────────────────────────────
	checkInfo(logger, "checking plugins")
	checkPlugins(logger, &counts, cfg.PluginsDir, cfg.PluginsEnabled)

	// ── Category: Build & Serve ─────────────────────────────────────
	checkInfo(logger, "checking build & serve config")
	checkServeConfig(logger, &counts, cfg)
	checkSass(logger, &counts, cfg, profile)

	// ── Category: Content health ────────────────────────────────────
	checkInfo(logger, "checking content health")
	checkEmptySections(logger, &counts, cfg.ContentDir)
	checkBrokenWikilinks(logger, &counts, cfg.ContentDir)

	// ── Category: Assets ────────────────────────────────────────────
	checkInfo(logger, "checking assets")
	checkLargeImages(logger, &counts, cfg, profile)

	// ── Summary ─────────────────────────────────────────────────────
	logger.Info("doctor summary", "warnings", counts.warn, "errors", counts.error)
	if counts.error > 0 {
		return fmt.Errorf("doctor found %d error(s)", counts.error)
	}
	return nil
}

func checkInfo(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

func checkWarn(logger *slog.Logger, counts *doctorCounters, msg string, args ...any) {
	counts.warn++
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

func checkError(logger *slog.Logger, counts *doctorCounters, msg string, args ...any) {
	counts.error++
	if logger != nil {
		logger.Error(msg, args...)
	}
}

func checkPath(logger *slog.Logger, counts *doctorCounters, label string, value string, required bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			checkWarn(logger, counts, fmt.Sprintf("%s is empty", label),
				"fix", fmt.Sprintf("set %s in config.yaml", label))
		}
		return
	}
	if !pathExists(value) {
		if required {
			checkError(logger, counts, fmt.Sprintf("%s does not exist", label),
				"path", value,
				"fix", fmt.Sprintf("create the directory or update %s in config.yaml", label))
		} else {
			checkWarn(logger, counts, fmt.Sprintf("%s does not exist", label),
				"path", value,
				"fix", fmt.Sprintf("create the directory or remove %s from config.yaml", label))
		}
	}
}

func checkTaxonomies(logger *slog.Logger, counts *doctorCounters, taxonomies []config.TaxonomyConfig, profile string) {
	if len(taxonomies) == 0 {
		checkWarn(logger, counts, "no taxonomies configured",
			"fix", "add taxonomies to config.yaml (e.g. tags, area) to enable content grouping")
		return
	}

	seen := map[string]int{}
	for _, tax := range taxonomies {
		name := strings.TrimSpace(tax.Name)
		if name == "" {
			checkWarn(logger, counts, "taxonomy name is empty",
				"fix", "every taxonomy entry must have a non-empty 'name' field")
			continue
		}
		seen[name]++
		if tax.PaginateBy <= 0 {
			checkWarn(logger, counts, "taxonomy paginate_by should be > 0",
				"name", name,
				"fix", fmt.Sprintf("set paginate_by to a positive number for taxonomy '%s'", name))
		}
		if strings.TrimSpace(tax.PaginatePath) == "" {
			checkWarn(logger, counts, "taxonomy paginate_path is empty",
				"name", name,
				"fix", fmt.Sprintf("set paginate_path (e.g. 'page') for taxonomy '%s'", name))
		}
	}
	dups := []string{}
	for name, count := range seen {
		if count > 1 {
			dups = append(dups, name)
		}
	}
	sort.Strings(dups)
	if len(dups) > 0 {
		if profile == "prod" {
			checkError(logger, counts, "duplicate taxonomies",
				"names", dups,
				"fix", "remove duplicate taxonomy entries from config.yaml")
		} else {
			checkWarn(logger, counts, "duplicate taxonomies",
				"names", dups,
				"fix", "remove duplicate taxonomy entries from config.yaml")
		}
	}
}

func checkPlugins(logger *slog.Logger, counts *doctorCounters, dir string, enabled []string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		if len(enabled) > 0 {
			checkWarn(logger, counts, "plugins_enabled set but plugins_dir is empty",
				"fix", "set plugins_dir in config.yaml or remove plugins_enabled entries")
		}
		return
	}
	if !pathExists(dir) {
		if len(enabled) > 0 {
			checkWarn(logger, counts, "plugins_enabled set but plugins_dir missing",
				"path", dir,
				"fix", fmt.Sprintf("create directory %s or run 'osg plugin install'", dir))
		}
		return
	}

	for _, name := range enabled {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		path := filepath.Join(dir, trimmed+".wasm")
		if !pathExists(path) {
			checkWarn(logger, counts, "plugin enabled but not installed",
				"plugin", trimmed, "path", path,
				"fix", fmt.Sprintf("run 'osg plugin install %s' or remove from plugins_enabled", trimmed))
		}
	}
}

func checkThemeTemplates(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	if strings.TrimSpace(cfg.Theme) == "" {
		return
	}
	templatesDir := filepath.Join(cfg.ThemesDir, cfg.Theme, "templates")
	if !pathExists(templatesDir) {
		if profile == "prod" {
			checkError(logger, counts, "theme templates not found",
				"path", templatesDir,
				"fix", "run 'osg build' to sync the default theme templates")
		} else {
			checkWarn(logger, counts, "theme templates not found",
				"path", templatesDir,
				"fix", "run 'osg build' to sync the default theme templates")
		}
	}
}

// checkRequiredTemplates verifies that the essential templates exist
// in the theme or user templates directory.
func checkRequiredTemplates(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	if strings.TrimSpace(cfg.Theme) == "" {
		return
	}
	themeDir := filepath.Join(cfg.ThemesDir, cfg.Theme, "templates")
	userDir := cfg.TemplatesDir

	required := []string{"index.html", "page.html", "section.html"}
	optional := []string{"404.html", "sitemap.xml", "robots.txt", "atom.xml", "rss.xml"}

	for _, tmpl := range required {
		inTheme := pathExists(filepath.Join(themeDir, tmpl))
		inUser := pathExists(filepath.Join(userDir, tmpl))
		if !inTheme && !inUser {
			if profile == "prod" {
				checkError(logger, counts, "required template missing",
					"template", tmpl,
					"fix", fmt.Sprintf("create %s in %s or %s", tmpl, themeDir, userDir))
			} else {
				checkWarn(logger, counts, "required template missing",
					"template", tmpl,
					"fix", fmt.Sprintf("create %s in %s or %s (built-in fallback will be used)", tmpl, themeDir, userDir))
			}
		}
	}

	for _, tmpl := range optional {
		inTheme := pathExists(filepath.Join(themeDir, tmpl))
		inUser := pathExists(filepath.Join(userDir, tmpl))
		if !inTheme && !inUser {
			checkInfo(logger, "optional template not found (built-in fallback used)", "template", tmpl)
		}
	}
}

// checkEmptySections scans the content directory for subdirectories
// that contain no .md files.
func checkEmptySections(logger *slog.Logger, counts *doctorCounters, contentDir string) {
	if !pathExists(contentDir) {
		return
	}

	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sectionPath := filepath.Join(contentDir, e.Name())
		hasMarkdown := false
		_ = filepath.WalkDir(sectionPath, func(_ string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
				hasMarkdown = true
				return filepath.SkipAll
			}
			return nil
		})
		if !hasMarkdown {
			checkWarn(logger, counts, "empty section (no .md files)",
				"section", e.Name(), "path", sectionPath,
				"fix", "add content files or remove the empty directory")
		}
	}
}

// wikilinkRe matches [[target]] and [[target|display]] wikilinks.
var wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// checkBrokenWikilinks scans content .md files for wikilinks and reports
// targets that don't match any content file slug or title.
func checkBrokenWikilinks(logger *slog.Logger, counts *doctorCounters, contentDir string) {
	if !pathExists(contentDir) {
		return
	}

	// Build a set of known slugs and filenames (without .md).
	known := map[string]bool{}
	_ = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			base := strings.TrimSuffix(d.Name(), ".md")
			known[strings.ToLower(base)] = true
		}
		return nil
	})

	if len(known) == 0 {
		return
	}

	// Image extensions to skip (image wikilinks, not page links).
	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".svg": true, ".webp": true, ".bmp": true, ".avif": true,
	}

	broken := map[string][]string{} // target -> list of source files
	_ = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		matches := wikilinkRe.FindAllSubmatch(data, -1)
		for _, m := range matches {
			target := strings.TrimSpace(string(m[1]))
			// Skip image links.
			ext := strings.ToLower(filepath.Ext(target))
			if imageExts[ext] {
				continue
			}
			// Skip links with path separators (these are file paths, not page links).
			if strings.Contains(target, "/") {
				continue
			}
			// Strip any heading anchor (e.g. [[page#section]]).
			if idx := strings.Index(target, "#"); idx >= 0 {
				target = target[:idx]
			}
			normalized := strings.ToLower(strings.TrimSpace(target))
			if normalized == "" {
				continue
			}
			if !known[normalized] {
				rel, _ := filepath.Rel(contentDir, path)
				broken[target] = append(broken[target], rel)
			}
		}
		return nil
	})

	// Report broken links, sorted for deterministic output.
	targets := make([]string, 0, len(broken))
	for t := range broken {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	for _, target := range targets {
		sources := broken[target]
		checkWarn(logger, counts, "broken wikilink",
			"target", target, "referenced_by", strings.Join(sources, ", "),
			"fix", fmt.Sprintf("create a page matching '%s' or remove the wikilink", target))
	}
}

// largeImageThreshold is the size in bytes above which an image is flagged.
const largeImageThreshold = 1 * 1024 * 1024 // 1 MiB

// checkLargeImages scans the static dir, theme static dir, and content images
// for files exceeding the size threshold.
func checkLargeImages(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".bmp": true, ".avif": true, ".svg": true,
	}

	dirs := []string{cfg.StaticDir}
	themeStatic := filepath.Join(cfg.ThemesDir, cfg.Theme, "static")
	if pathExists(themeStatic) {
		dirs = append(dirs, themeStatic)
	}
	// Also scan content directory for copied images.
	if pathExists(cfg.ContentDir) {
		dirs = append(dirs, cfg.ContentDir)
	}

	for _, dir := range dirs {
		if !pathExists(dir) {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !imageExts[ext] {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.Size() > largeImageThreshold {
				sizeMB := float64(info.Size()) / (1024 * 1024)
				if profile == "prod" {
					checkError(logger, counts, "large image file",
						"path", path, "size_mb", fmt.Sprintf("%.1f", sizeMB),
						"fix", "optimize this image (compress or resize) before deploying")
				} else {
					checkWarn(logger, counts, "large image file",
						"path", path, "size_mb", fmt.Sprintf("%.1f", sizeMB),
						"fix", "consider compressing or resizing this image")
				}
			}
			return nil
		})
	}
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func normalizeProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile == "" {
		return "dev"
	}
	if profile != "dev" && profile != "prod" {
		return "dev"
	}
	return profile
}

func validateBaseURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("base_url should include scheme and host")
	}
	return nil
}

func checkServeConfig(logger *slog.Logger, counts *doctorCounters, cfg config.Config) {
	if cfg.ServeReload && !cfg.ServeWatch {
		checkWarn(logger, counts, "serve_live_reload enabled but serve_watch is false",
			"fix", "enable serve_watch or disable serve_live_reload in config.yaml")
	}
	if cfg.ServeWatch && cfg.ServeDebounce <= 0 {
		checkWarn(logger, counts, "serve_debounce_ms should be > 0",
			"fix", "set serve_debounce_ms to a positive value (e.g. 300)")
	}
}

// checkThemeMeta validates the active theme's theme.yaml and its parent chain.
func checkThemeMeta(logger *slog.Logger, counts *doctorCounters, cfg config.Config) {
	if strings.TrimSpace(cfg.Theme) == "" || strings.TrimSpace(cfg.ThemesDir) == "" {
		return
	}
	themeDir := filepath.Join(cfg.ThemesDir, cfg.Theme)
	if !pathExists(themeDir) {
		return // Already reported by checkThemeTemplates.
	}

	meta, err := theme.LoadMeta(themeDir)
	if err != nil {
		checkWarn(logger, counts, "theme.yaml parse error",
			"theme", cfg.Theme, "error", err.Error(),
			"fix", "fix or remove theme.yaml in "+themeDir)
		return
	}

	metaPath := filepath.Join(themeDir, "theme.yaml")
	if !pathExists(metaPath) {
		checkInfo(logger, "theme.yaml not found (optional, recommended for metadata)",
			"theme", cfg.Theme,
			"fix", "create theme.yaml with name, description, author fields")
	}

	// Validate parent chain.
	if meta.Parent != "" {
		_, err := theme.ResolveChain(cfg.ThemesDir, cfg.Theme)
		if err != nil {
			checkWarn(logger, counts, "theme parent chain error",
				"theme", cfg.Theme, "error", err.Error(),
				"fix", "check parent field in theme.yaml and ensure the parent theme is installed")
		}
	}
}

func checkSass(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	if !cfg.CompileSass {
		return
	}
	if _, err := exec.LookPath("sass"); err != nil {
		if profile == "prod" {
			checkError(logger, counts, "sass binary not found in PATH",
				"fix", "install sass (npm install -g sass) or disable compile_sass in config.yaml")
		} else {
			checkWarn(logger, counts, "sass binary not found in PATH",
				"fix", "install sass (npm install -g sass) or disable compile_sass")
		}
	}
}
