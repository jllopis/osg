package build

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"osg/internal/config"
	imgopt "osg/internal/image"
	"osg/internal/markdown"
	"osg/internal/plugin"
	"osg/internal/site"
)

// emitConfigValidate fires the config.validate event and checks
// the plugin response for errors.  If any plugin returns a non-empty
// "errors" list, the build is aborted with the first error message.
// Warnings are logged but do not stop the build.
func emitConfigValidate(ctx context.Context, plugins *plugin.Manager, cfg config.Config, logger *slog.Logger) error {
	if plugins == nil {
		return nil
	}

	payload := map[string]any{
		"config": configView(cfg),
	}
	result := plugins.Emit(ctx, "config.validate", payload)
	if result == nil {
		return nil
	}

	// Log warnings if present.
	if warnings, ok := result["warnings"]; ok {
		if list, ok := warnings.([]any); ok {
			for _, w := range list {
				logger.Warn("plugin config warning", "message", fmt.Sprint(w))
			}
		}
	}

	// Abort on errors.
	if errors, ok := result["errors"]; ok {
		if list, ok := errors.([]any); ok && len(list) > 0 {
			msgs := make([]string, 0, len(list))
			for _, e := range list {
				msgs = append(msgs, fmt.Sprint(e))
			}
			return fmt.Errorf("plugin config validation failed: %s", strings.Join(msgs, "; "))
		}
	}

	return nil
}

// applyContentTransform fires the content.transform event for each page,
// allowing plugins to modify the Markdown source before it is rendered to HTML.
// If a plugin returns a modified body_markdown, the page's RawContent and
// Content are updated accordingly.
func applyContentTransform(ctx context.Context, plugins *plugin.Manager, cfg config.Config, siteIndex *site.Site, logger *slog.Logger) {
	if plugins == nil || len(siteIndex.Pages) == 0 {
		return
	}

	cfgView := configView(cfg)
	for _, page := range siteIndex.Pages {
		payload := map[string]any{
			"config": cfgView,
			"page": map[string]any{
				"title":         page.Title,
				"slug":          page.Slug,
				"path":          page.Path,
				"permalink":     page.Permalink,
				"date":          page.Date,
				"body_markdown": page.RawContent,
				"summary":       page.Summary,
				"taxonomies":    page.Taxonomies,
				"extra":         page.Extra,
			},
		}

		overrides := plugins.Emit(ctx, "content.transform", payload)
		if overrides == nil {
			continue
		}

		pageOverrides, ok := overrides["page"].(map[string]any)
		if !ok {
			continue
		}

		newBody, ok := pageOverrides["body_markdown"].(string)
		if !ok || newBody == page.RawContent {
			continue
		}

		// Re-render the transformed Markdown to HTML.
		html, err := markdown.Render([]byte(newBody))
		if err != nil {
			logger.Warn("content.transform re-render failed",
				"page", page.SourcePath, "error", err)
			continue
		}

		page.RawContent = newBody
		page.Content = html

		// Recalculate word count and reading time.
		page.WordCount = len(strings.Fields(newBody))
		page.ReadingTime = page.WordCount / 200
		if page.ReadingTime < 1 {
			page.ReadingTime = 1
		}

		logger.Debug("content.transform applied", "page", page.SourcePath)
	}
}

// emitImageProcess fires the image.process event for each optimized image,
// allowing plugins to perform additional transformations via WASI filesystem.
func emitImageProcess(ctx context.Context, plugins *plugin.Manager, cfg config.Config, imageResults map[string]*imgopt.Result, logger *slog.Logger) {
	if plugins == nil || len(imageResults) == 0 {
		return
	}

	cfgView := configView(cfg)
	for srcPath, result := range imageResults {
		var variants []map[string]any
		for _, variantList := range result.Variants {
			for _, v := range variantList {
				variants = append(variants, map[string]any{
					"url_path": v.URLPath,
					"width":    v.Width,
					"format":   v.Format,
				})
			}
		}

		payload := map[string]any{
			"config": cfgView,
			"image": map[string]any{
				"src_path":       srcPath,
				"public_dir":     cfg.PublicDir,
				"original":       result.Original,
				"original_width": result.OriginalWidth,
				"variants":       variants,
			},
		}

		// Fire and forget: plugins do their work via WASI file I/O.
		_ = plugins.Emit(ctx, "image.process", payload)
	}
}
