package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"osg/internal/config"
	"osg/internal/importer"
)

// RunImportWordpress imports posts from a WordPress WXR export file.
func RunImportWordpress(_ context.Context, opts CLIOptions, xmlPath string, dryRun bool) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("import: loading config: %w", err)
	}

	contentDir := cfg.ContentDir
	if opts.OsgContentDir != "" {
		contentDir = opts.OsgContentDir
	}

	posts, err := importer.ParseWordpress(xmlPath)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("Would import %d posts from WordPress:\n", len(posts))
		for _, p := range posts {
			fmt.Printf("  %s → %s\n", p.Title, p.OutputPath())
		}
		return nil
	}

	written := 0
	for _, p := range posts {
		outPath := filepath.Join(contentDir, p.OutputPath())
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(p.ToMarkdown()), 0o644); err != nil {
			return err
		}
		written++
	}

	fmt.Printf("Imported %d posts from WordPress into %s\n", written, contentDir)
	return nil
}

// RunImportHugo imports posts from a Hugo content directory.
func RunImportHugo(_ context.Context, opts CLIOptions, hugoDir string, dryRun bool) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("import: loading config: %w", err)
	}

	contentDir := cfg.ContentDir
	if opts.OsgContentDir != "" {
		contentDir = opts.OsgContentDir
	}

	posts, err := importer.ParseHugo(hugoDir)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("Would import %d posts from Hugo:\n", len(posts))
		for _, p := range posts {
			fmt.Printf("  %s → %s\n", p.Title, p.OutputPath())
		}
		return nil
	}

	written := 0
	for _, p := range posts {
		outPath := filepath.Join(contentDir, p.OutputPath())
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(p.ToMarkdown()), 0o644); err != nil {
			return err
		}
		written++
	}

	fmt.Printf("Imported %d posts from Hugo into %s\n", written, contentDir)
	return nil
}
