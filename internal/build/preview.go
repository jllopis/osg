package build

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"

	"osg/internal/assets"
	"osg/internal/config"
	"osg/internal/i18n"
	"osg/internal/markdown"
	"osg/internal/render"
	"osg/internal/site"
	"osg/internal/taxonomy"
	"osg/internal/theme"
)

// PreviewBuild renders a single markdown file into a temporary directory and
// returns the path to that directory.  The caller is responsible for serving
// or cleaning up the directory.
func PreviewBuild(cfg config.Config, filePath string, logger *slog.Logger) (string, error) {
	tmpDir, err := os.MkdirTemp("", "osg-preview-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Override public dir to the temp directory.
	cfg.PublicDir = tmpDir

	// Ensure default theme is available.
	if err := theme.EnsureDefaultTheme(cfg.ThemesDir); err != nil {
		return tmpDir, fmt.Errorf("ensure default theme: %w", err)
	}

	// Resolve theme chain.
	themeChain, err := theme.ResolveChain(cfg.ThemesDir, cfg.Theme)
	if err != nil {
		return tmpDir, fmt.Errorf("resolve theme chain: %w", err)
	}

	// Load i18n.
	i18nBundle := i18n.New(cfg.DefaultLanguage)
	for idx := len(themeChain) - 1; idx >= 0; idx-- {
		dir := filepath.Join(themeChain[idx], "i18n")
		if err := i18nBundle.LoadDir(dir); err != nil {
			return tmpDir, fmt.Errorf("load theme translations: %w", err)
		}
	}
	if err := i18nBundle.LoadDir("i18n"); err != nil {
		return tmpDir, fmt.Errorf("load user translations: %w", err)
	}

	// Copy static assets so CSS/fonts work.
	if err := assets.PrepareWithChain(cfg, themeChain, logger); err != nil {
		return tmpDir, fmt.Errorf("prepare assets: %w", err)
	}

	// Parse the single file.
	page, _, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, filePath)
	if err != nil {
		return tmpDir, fmt.Errorf("parse file: %w", err)
	}
	if page == nil {
		return tmpDir, fmt.Errorf("file is a section index, not a page")
	}
	if page.Lang == "" {
		page.Lang = cfg.DefaultLanguage
	}

	// Build a minimal site with just this page.
	siteIndex := site.New()
	siteIndex.AddPage(page)
	siteIndex.BuildHierarchy()

	indices := taxonomy.Build(cfg.Taxonomies, siteIndex.Pages, cfg.BaseURL)
	taxonomy.FilterPageTaxonomies(cfg.Taxonomies, siteIndex.Pages)
	siteView := siteIndex.View()
	bCtx := baseContext(cfg, siteView, indices, nil)

	// Build template chain and renderer.
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
		I18n:            i18nBundle,
	})
	if err != nil {
		return tmpDir, fmt.Errorf("create renderer: %w", err)
	}

	// Build render context with TOC.
	renderCtx := pageContext(bCtx, page)
	if pageView, ok := renderCtx["page"].(map[string]any); ok {
		if content, ok := pageView["content"].(template.HTML); ok {
			if tocEntries := markdown.ExtractTOC(string(content)); len(tocEntries) > 0 {
				renderCtx["toc"] = markdown.TOCView(tocEntries)
			}
		}
	}

	// Render page.
	templateName := page.Template
	if templateName == "" {
		templateName = "page.html"
	}
	outputPath := outputHTMLPath(cfg.PublicDir, page.Path)
	if err := renderer.RenderToFile(templateName, renderCtx, outputPath); err != nil {
		return tmpDir, fmt.Errorf("render page: %w", err)
	}

	logger.Info("preview built", "page", page.Path, "output", tmpDir)
	return tmpDir, nil
}
