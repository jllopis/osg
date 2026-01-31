package build

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/render"
	"osg/internal/site"
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

	renderer, err := render.New(cfg.TemplatesDir, themeTemplatesDir(cfg))
	if err != nil {
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
	siteView := siteIndex.View()

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
