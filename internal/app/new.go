package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/date"
	"osg/internal/logging"
	"osg/internal/slug"
)

// NewPostOptions holds the parameters for creating a new post.
type NewPostOptions struct {
	Title      string
	Tags       []string
	Publish    bool   // true = osg.publish: true, false = osg.publish: "draft"
	Editor     bool   // open in editor after creation
	EditorAuto bool   // true when Editor was auto-detected (not explicit --editor)
	NotesDir   string // CLI override for new_notes_dir config
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

	content := buildFrontmatter(title, now, postOpts)

	// Resolve destination directory: CLI --notes-dir > config new_notes_dir > vault root.
	baseDir := vaultPath
	notesDir := postOpts.NotesDir
	if notesDir == "" {
		notesDir = cfg.NewNotesDir
	}
	if notesDir != "" {
		baseDir = filepath.Join(vaultPath, notesDir)
	}

	// File path: baseDir/{Title}.md
	// Use the original title as filename (Obsidian convention).
	filename := title + ".md"
	outputPath := filepath.Join(baseDir, filename)

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

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	logger.Info("created new post",
		"path", outputPath,
		"title", title,
		"slug", slugValue,
		"date", dateISO,
	)
	fmt.Printf("Created: %s\n", outputPath)

	// Open in editor if requested.
	if postOpts.Editor {
		// In auto-detect mode, only open if an editor is actually configured.
		if postOpts.EditorAuto && resolveEditor(cfg) == "" {
			// No editor configured; silently skip.
		} else if err := openEditor(cfg, outputPath); err != nil {
			// Non-fatal: the file was created successfully.
			logger.Warn("could not open editor", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: could not open editor: %v\n", err)
		}
	}

	return nil
}

// buildFrontmatter generates the YAML frontmatter with all osg fields.
// Active fields are set; inactive ones appear as YAML comments so the
// user can uncomment them as needed.
func buildFrontmatter(title string, now time.Time, postOpts NewPostOptions) string {
	var buf strings.Builder
	buf.WriteString("---\n")
	buf.WriteString("title: " + yamlScalar(title) + "\n")
	buf.WriteString("created: " + now.Format("2006-01-02 15:04") + "\n")

	if len(postOpts.Tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range postOpts.Tags {
			buf.WriteString("  - " + tag + "\n")
		}
	}

	// The osg block with all recognised fields.
	buf.WriteString("osg:\n")
	if postOpts.Publish {
		buf.WriteString("  publish: true\n")
	} else {
		buf.WriteString("  publish: draft\n")
	}
	// Remaining osg fields as commented placeholders.
	buf.WriteString("  # title: \"\"          # Override page title (highest precedence)\n")
	buf.WriteString("  # image: \"\"          # Featured/hero image path\n")
	buf.WriteString("  # featured: false    # Mark as featured post\n")
	buf.WriteString("  # path: \"\"           # Custom output path override\n")
	buf.WriteString("  # permalink: \"\"      # URL pattern ({date}, {year}, {month}, {day}, {slug}, {title})\n")
	buf.WriteString("  # menu: false        # Add to navigation menu\n")
	buf.WriteString("  # abstract: \"\"       # Summary/excerpt override\n")
	buf.WriteString("  # author: \"\"         # Author override\n")

	buf.WriteString("---\n\n")
	return buf.String()
}

// yamlScalar returns a YAML-safe scalar value for a string. If the
// string contains characters that need quoting, it is double-quoted.
func yamlScalar(s string) string {
	// Characters that require quoting in YAML scalars.
	if strings.ContainsAny(s, ":{}[]|>&*!#%@`\"'\\,\n\r\t") ||
		strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "? ") ||
		s == "" || s == "true" || s == "false" || s == "null" {
		// Use double-quoting with backslash escapes.
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

// resolveEditor returns the editor command to use. Priority:
// 1. config.DefaultEditor
// 2. $EDITOR environment variable
// Returns empty string if no editor is configured.
func resolveEditor(cfg config.Config) string {
	if cfg.DefaultEditor != "" {
		return cfg.DefaultEditor
	}
	return os.Getenv("EDITOR")
}

// openEditor launches the resolved editor with the given file path.
// It connects stdin/stdout/stderr so the editor is interactive.
func openEditor(cfg config.Config, filePath string) error {
	editor := resolveEditor(cfg)
	if editor == "" {
		return fmt.Errorf("no editor configured (set default_editor in config or $EDITOR env var)")
	}

	// Split the editor command in case it includes arguments
	// (e.g. "code --wait" or "vim -u NONE").
	parts := strings.Fields(editor)
	args := make([]string, 0, len(parts))
	args = append(args, parts[1:]...)
	args = append(args, filePath)

	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
