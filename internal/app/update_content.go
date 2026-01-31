package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"osg/internal/config"
	"osg/internal/content"
	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/logging"
	"osg/internal/publish"
	"osg/internal/slug"
	"osg/internal/vault"
)

type UpdateStats struct {
	Total    int
	Parsed   int
	Exported int
	Skipped  int
	Drafts   int
	Errors   int
}

func RunUpdateContent(_ context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.ObsidianVaultBase != "" {
		cfg.VaultBase = opts.ObsidianVaultBase
	}
	if opts.Vault != "" {
		cfg.Vault = opts.Vault
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

	logger := logging.New(cfg.Logging, opts.Verbose)

	files, err := vault.ListMarkdownFiles(vaultPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no markdown files found", "vault", vaultPath)
		return nil
	}

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
		outputPath := content.BuildOutputPath(cfg.ContentDir, cfg.ContentLayout, datePath, slugValue)

		if prev, exists := seen[outputPath]; exists {
			logger.Error("output path collision", "path", outputPath, "first", prev, "second", path)
			stats.Errors++
			continue
		}
		seen[outputPath] = path

		if opts.DryRun {
			logger.Info("dry-run", "source", path, "dest", outputPath)
			stats.Exported++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			logger.Warn("failed to create output directory", "path", outputPath, "error", err)
			stats.Errors++
			continue
		}

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
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}
