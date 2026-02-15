package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"osg/internal/config"
	"osg/internal/content"
	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/logging"
	"osg/internal/publish"
	"osg/internal/slug"
	"osg/internal/vault"
	"osg/internal/wikilink"
)

type UpdateStats struct {
	Total    int
	Parsed   int
	Exported int
	Skipped  int
	Drafts   int
	Errors   int
	Images   int
}

func RunUpdateContent(_ context.Context, opts CLIOptions) error {
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
	if opts.IncludeDrafts != nil {
		cfg.IncludeDrafts = *opts.IncludeDrafts
	}

	vaultPath, err := config.ResolveVaultPath(cfg)
	if err != nil {
		return err
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	files, err := vault.ListMarkdownFiles(vaultPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no markdown files found", "vault", vaultPath)
		return nil
	}

	// Build image index for resolving osg.image and wikilink references
	imageIndex, err := vault.BuildImageIndex(vaultPath)
	if err != nil {
		logger.Warn("failed to build image index, image resolution disabled", "error", err)
		imageIndex = vault.ImageIndex{}
	}
	logger.Info("image index built", "entries", len(imageIndex))

	stats := UpdateStats{Total: len(files)}
	seen := map[string]string{}

	for _, path := range files {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			logger.Warn("failed to read file", "path", path, "error", readErr)
			stats.Errors++
			continue
		}

		fm, body, _, fmErr := frontmatter.SplitFrontmatter(data)
		if fmErr != nil {
			logger.Warn("failed to parse frontmatter", "path", path, "error", fmErr)
			stats.Errors++
			continue
		}
		stats.Parsed++

		pub, isDraft := publish.ShouldPublish(fm)
		if !pub {
			stats.Skipped++
			continue
		}
		if isDraft && !cfg.IncludeDrafts {
			stats.Drafts++
			continue
		}

		info, statErr := os.Stat(path)
		if statErr != nil {
			logger.Warn("failed to stat file", "path", path, "error", statErr)
			stats.Errors++
			continue
		}

		dateValue := date.Derive(fm, info)
		slugValue := slug.Derive(fm, info.Name())

		dateISO := date.FormatISO(dateValue)
		datePath := date.FormatPath(dateValue)

		outputFM := content.NormalizeFrontmatter(fm, slugValue, dateISO, isDraft, info.Name())

		// If osg.path is set, use it as the output path instead of the default content_layout.
		// This allows standalone pages (e.g. "about") to have custom URL paths.
		osgPath := publish.GetOSGString(fm, "path")
		var outputPath string
		if osgPath != "" {
			outputPath = filepath.Join(cfg.ContentDir, filepath.Clean(osgPath), "index.md")
		} else {
			outputPath = content.BuildOutputPath(cfg.ContentDir, cfg.ContentLayout, datePath, slugValue)
		}

		if prev, exists := seen[outputPath]; exists {
			logger.Error("output path collision", "path", outputPath, "first", prev, "second", path)
			stats.Errors++
			continue
		}
		seen[outputPath] = path

		outputDir := filepath.Dir(outputPath)

		if opts.DryRun {
			logger.Info("dry-run", "source", path, "dest", outputPath)
			stats.Exported++
			continue
		}

		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.Warn("failed to create output directory", "path", outputPath, "error", err)
			stats.Errors++
			continue
		}

		// Compute the URL path prefix for this post (e.g. "/2025/09/16/logos/")
		// so images can be referenced with absolute paths that work from any page.
		pageURLDir := ""
		if relDir, relErr := filepath.Rel(cfg.ContentDir, outputDir); relErr == nil {
			pageURLDir = "/" + filepath.ToSlash(relDir) + "/"
		}

		// Resolve the frontmatter image (from osg.image or top-level image/cover/banner).
		// Try to find it in the vault, copy it, and set an absolute URL path.
		// If it can't be resolved and isn't an external URL, remove it so the
		// placeholder generator can create one during build.
		if imgRef, ok := outputFM["image"].(string); ok && imgRef != "" {
			if isExternalURL(imgRef) {
				// External URL — leave as-is
				logger.Info("image is external URL", "url", imgRef, "note", path)
			} else if strings.HasPrefix(imgRef, "/") {
				// Already absolute path — leave as-is
			} else if srcPath, found := imageIndex.Resolve(imgRef); found {
				destName := filepath.Base(srcPath)
				destPath := filepath.Join(outputDir, destName)
				if cpErr := copyFile(srcPath, destPath); cpErr != nil {
					logger.Warn("failed to copy image", "src", srcPath, "dest", destPath, "error", cpErr)
					delete(outputFM, "image") // let placeholder handle it
				} else {
					outputFM["image"] = pageURLDir + destName
					stats.Images++
					logger.Info("copied image", "src", srcPath, "dest", destPath, "url", outputFM["image"])
				}
			} else {
				logger.Warn("image not found in vault, will use placeholder", "ref", imgRef, "note", path)
				delete(outputFM, "image") // let placeholder handle it
			}
		}

		// Rewrite wikilink images in body and copy referenced images
		body = wikilink.RewriteImageLinks(body, func(ref string) (string, bool) {
			srcPath, found := imageIndex.Resolve(ref)
			if !found {
				logger.Warn("wikilink image not found in vault", "ref", ref, "note", path)
				return "", false
			}

			destName := filepath.Base(srcPath)
			destPath := filepath.Join(outputDir, destName)
			if cpErr := copyFile(srcPath, destPath); cpErr != nil {
				logger.Warn("failed to copy wikilink image", "src", srcPath, "dest", destPath, "error", cpErr)
				return "", false
			}

			stats.Images++
			logger.Info("copied wikilink image", "src", srcPath, "dest", destPath)
			return destName, true
		})

		outputBytes, err := content.RenderMarkdown(outputFM, body)
		if err != nil {
			logger.Warn("failed to render output", "path", path, "error", err)
			stats.Errors++
			continue
		}

		if err := os.WriteFile(outputPath, outputBytes, 0o644); err != nil {
			logger.Warn("failed to write output", "path", outputPath, "error", err)
			stats.Errors++
			continue
		}

		logger.Info("exported", "source", path, "dest", outputPath)
		stats.Exported++
	}

	logger.Info("update-content summary",
		"total", stats.Total,
		"parsed", stats.Parsed,
		"exported", stats.Exported,
		"skipped", stats.Skipped,
		"drafts", stats.Drafts,
		"images", stats.Images,
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}

// isExternalURL returns true if the string looks like an external URL.
func isExternalURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
