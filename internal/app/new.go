package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/date"
	"osg/internal/logging"
	"osg/internal/slug"

	"gopkg.in/yaml.v3"
)

// NewPostOptions holds the parameters for creating a new post.
type NewPostOptions struct {
	Title   string
	Tags    []string
	Publish bool // true = osg.publish: true, false = osg.publish: "draft"
	Editor  bool // open in $EDITOR after creation
}

// RunNew creates a new Markdown note in the vault with osg frontmatter.
func RunNew(_ context.Context, opts CLIOptions, postOpts NewPostOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}

	vaultPath, err := config.ResolveVaultPath(cfg)
	if err != nil {
		return err
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	title := strings.TrimSpace(postOpts.Title)
	if title == "" {
		return fmt.Errorf("title is required")
	}

	now := time.Now()
	dateISO := date.FormatISO(now)
	slugValue := slug.Slugify(title)

	// Build the frontmatter in Obsidian-native format.
	fm := map[string]any{
		"title":   title,
		"created": now.Format("2006-01-02 15:04"),
	}

	if len(postOpts.Tags) > 0 {
		fm["tags"] = postOpts.Tags
	}

	// The osg block controls publishing behaviour.
	osgBlock := map[string]any{}
	if postOpts.Publish {
		osgBlock["publish"] = true
	} else {
		osgBlock["publish"] = "draft"
	}
	fm["osg"] = osgBlock

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf strings.Builder
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")

	// File path: vault_path/{Title}.md
	// Use the original title as filename (Obsidian convention).
	filename := title + ".md"
	outputPath := filepath.Join(vaultPath, filename)

	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file already exists: %s", outputPath)
	}

	if opts.DryRun {
		logger.Info("dry-run: would create",
			"path", outputPath,
			"title", title,
			"slug", slugValue,
			"date", dateISO,
		)
		fmt.Printf("Would create: %s\n", outputPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	logger.Info("created new post",
		"path", outputPath,
		"title", title,
		"slug", slugValue,
		"date", dateISO,
	)
	fmt.Printf("Created: %s\n", outputPath)

	return nil
}
