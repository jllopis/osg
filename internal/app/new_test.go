package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"osg/internal/config"
)

func TestRunNew_CreatesFile(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "My First Post"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	outputPath := filepath.Join(vaultPath, "My First Post.md")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)

	// Check frontmatter structure
	if !strings.Contains(content, "---") {
		t.Error("missing frontmatter delimiters")
	}
	if !strings.Contains(content, "title: My First Post") {
		t.Error("missing title in frontmatter")
	}
	if !strings.Contains(content, "created:") {
		t.Error("missing created date in frontmatter")
	}
	// Default is draft
	if !strings.Contains(content, `publish: draft`) {
		t.Error("expected osg.publish: draft for default")
	}
}

func TestRunNew_PublishFlag(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Published Post", Publish: true}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultPath, "Published Post.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "publish: true") {
		t.Error("expected osg.publish: true when --publish is set")
	}
}

func TestRunNew_WithTags(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{
		Title: "Tagged Post",
		Tags:  []string{"go", "testing"},
	}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultPath, "Tagged Post.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "- go") {
		t.Error("missing tag 'go' in frontmatter")
	}
	if !strings.Contains(content, "- testing") {
		t.Error("missing tag 'testing' in frontmatter")
	}
}

func TestRunNew_EmptyTitle(t *testing.T) {
	t.Parallel()

	cfgPath, _ := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: ""}

	err := RunNew(context.TODO(), opts, postOpts)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunNew_FileAlreadyExists(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)

	// Create the file first
	existing := filepath.Join(vaultPath, "Duplicate.md")
	if err := os.WriteFile(existing, []byte("# Existing"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Duplicate"}

	err := RunNew(context.TODO(), opts, postOpts)
	if err == nil {
		t.Fatal("expected error for existing file")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunNew_DryRun(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath, DryRun: true}
	postOpts := NewPostOptions{Title: "Dry Run Post"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew dry-run failed: %v", err)
	}

	outputPath := filepath.Join(vaultPath, "Dry Run Post.md")
	if _, err := os.Stat(outputPath); err == nil {
		t.Error("file should not exist in dry-run mode")
	}
}

func TestRunNew_NoVaultPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: ""`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "No Vault"}

	err := RunNew(context.TODO(), opts, postOpts)
	if err == nil {
		t.Fatal("expected error when vault_path is empty")
	}
	if !strings.Contains(err.Error(), "vault path") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunNew_VaultPathOverride(t *testing.T) {
	t.Parallel()

	cfgPath, _ := writeNewConfig(t)

	// Use a different vault path via CLI override
	altVault := t.TempDir()
	opts := CLIOptions{ConfigPath: cfgPath, VaultPath: altVault}
	postOpts := NewPostOptions{Title: "Override Vault"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	outputPath := filepath.Join(altVault, "Override Vault.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist in overridden vault: %v", err)
	}
}

func TestRunNew_UnicodeTitle(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Inferencia Bayesiana"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	outputPath := filepath.Join(vaultPath, "Inferencia Bayesiana.md")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "title: Inferencia Bayesiana") {
		t.Error("title not preserved in frontmatter")
	}
}

// --- New tests for expanded osg block ---

func TestRunNew_OsgBlockComments(t *testing.T) {
	t.Parallel()

	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Full Osg Block"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultPath, "Full Osg Block.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	content := string(data)

	// Check that all osg placeholder comments are present.
	expectedComments := []string{
		"# title:",
		"# image:",
		"# featured:",
		"# path:",
		"# permalink:",
		"# menu:",
		"# abstract:",
		"# author:",
	}
	for _, comment := range expectedComments {
		if !strings.Contains(content, comment) {
			t.Errorf("missing osg comment placeholder: %q", comment)
		}
	}

	// The active publish field should be uncommented.
	if !strings.Contains(content, "  publish: draft") {
		t.Error("publish field should be active (not commented)")
	}
}

func TestBuildFrontmatter_Draft(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 6, 10, 30, 0, 0, time.UTC)
	opts := NewPostOptions{Title: "Test Draft", Publish: false}

	fm := buildFrontmatter("Test Draft", now, opts)

	if !strings.Contains(fm, "title: Test Draft") {
		t.Error("missing title")
	}
	if !strings.Contains(fm, "created: 2026-03-06 10:30") {
		t.Error("missing or wrong created date")
	}
	if !strings.Contains(fm, "publish: draft") {
		t.Error("expected publish: draft")
	}
	if !strings.HasPrefix(fm, "---\n") {
		t.Error("should start with ---")
	}
	if !strings.Contains(fm, "---\n\n") {
		t.Error("should end with --- followed by blank line")
	}
}

func TestBuildFrontmatter_Published(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC)
	opts := NewPostOptions{Title: "Published", Publish: true}

	fm := buildFrontmatter("Published", now, opts)

	if !strings.Contains(fm, "publish: true") {
		t.Error("expected publish: true")
	}
}

func TestBuildFrontmatter_WithTags(t *testing.T) {
	t.Parallel()

	now := time.Now()
	opts := NewPostOptions{Title: "Tagged", Tags: []string{"go", "rust"}}

	fm := buildFrontmatter("Tagged", now, opts)

	if !strings.Contains(fm, "tags:\n  - go\n  - rust\n") {
		t.Error("tags not formatted correctly")
	}
}

func TestBuildFrontmatter_NoTags(t *testing.T) {
	t.Parallel()

	now := time.Now()
	opts := NewPostOptions{Title: "No Tags"}

	fm := buildFrontmatter("No Tags", now, opts)

	if strings.Contains(fm, "tags:") {
		t.Error("tags section should not appear when no tags given")
	}
}

func TestBuildFrontmatter_TitleWithSpecialChars(t *testing.T) {
	t.Parallel()

	now := time.Now()
	opts := NewPostOptions{Title: `Why "Rust" is: great`}

	fm := buildFrontmatter(`Why "Rust" is: great`, now, opts)

	if !strings.Contains(fm, `title: "Why \"Rust\" is: great"`) {
		t.Errorf("title with special chars not quoted properly, got:\n%s", fm)
	}
}

func TestYamlScalar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"hello world", "hello world"},
		{"has: colon", `"has: colon"`},
		{`has "quotes"`, `"has \"quotes\""`},
		{"true", `"true"`},
		{"false", `"false"`},
		{"null", `"null"`},
		{"", `""`},
		{"- starts with dash", `"- starts with dash"`},
		{"normal title", "normal title"},
	}

	for _, tt := range tests {
		got := yamlScalar(tt.input)
		if got != tt.want {
			t.Errorf("yamlScalar(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveEditor(t *testing.T) {
	t.Run("config takes precedence", func(t *testing.T) {
		cfg := config.Config{DefaultEditor: "nvim"}
		t.Setenv("EDITOR", "vim")
		got := resolveEditor(cfg)
		if got != "nvim" {
			t.Errorf("resolveEditor = %q; want \"nvim\"", got)
		}
	})

	t.Run("falls back to EDITOR", func(t *testing.T) {
		cfg := config.Config{}
		t.Setenv("EDITOR", "vim")
		got := resolveEditor(cfg)
		if got != "vim" {
			t.Errorf("resolveEditor = %q; want \"vim\"", got)
		}
	})

	t.Run("empty when nothing set", func(t *testing.T) {
		cfg := config.Config{}
		t.Setenv("EDITOR", "")
		got := resolveEditor(cfg)
		if got != "" {
			t.Errorf("resolveEditor = %q; want empty", got)
		}
	})
}

func TestRunNew_EditorAutoSkipsWhenNoEditor(t *testing.T) {
	// When EditorAuto=true and no editor is configured, should not error.
	cfgPath, vaultPath := writeNewConfig(t)
	t.Setenv("EDITOR", "")

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{
		Title:      "Auto Editor Skip",
		Editor:     true,
		EditorAuto: true,
	}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	// File should still be created.
	outputPath := filepath.Join(vaultPath, "Auto Editor Skip.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestConfigDefaultEditor(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "/tmp"
default_editor: "code --wait"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	if cfg.DefaultEditor != "code --wait" {
		t.Errorf("DefaultEditor = %q; want \"code --wait\"", cfg.DefaultEditor)
	}
}

func TestConfigDefaultEditorDefault(t *testing.T) {
	t.Parallel()

	// Default config should have empty DefaultEditor.
	cfg := config.Default()
	if cfg.DefaultEditor != "" {
		t.Errorf("Default().DefaultEditor = %q; want empty", cfg.DefaultEditor)
	}
}

func TestConfigSchemaHasDefaultEditor(t *testing.T) {
	t.Parallel()

	_, ok := config.FindField("default_editor")
	if !ok {
		t.Error("default_editor not found in ConfigSchema")
	}
}

func TestConfigSchemaHasNewNotesDir(t *testing.T) {
	t.Parallel()

	_, ok := config.FindField("new_notes_dir")
	if !ok {
		t.Error("new_notes_dir not found in ConfigSchema")
	}
}

// --- new_notes_dir tests ---

func TestRunNew_NewNotesDirFromConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	vaultPath := filepath.Join(root, "vault")
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "` + vaultPath + `"` + "\n" +
		`new_notes_dir: "02_Notes"` + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Note In Subdir"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	// File should be in vault/02_Notes/
	outputPath := filepath.Join(vaultPath, "02_Notes", "Note In Subdir.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist at %s: %v", outputPath, err)
	}

	// Should NOT exist at vault root.
	rootPath := filepath.Join(vaultPath, "Note In Subdir.md")
	if _, err := os.Stat(rootPath); err == nil {
		t.Error("file should not exist at vault root when new_notes_dir is set")
	}
}

func TestRunNew_NewNotesDirAutoCreate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	vaultPath := filepath.Join(root, "vault")
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "` + vaultPath + `"` + "\n" +
		`new_notes_dir: "deep/nested/notes"` + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Deep Note"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	outputPath := filepath.Join(vaultPath, "deep", "nested", "notes", "Deep Note.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist at %s (auto-created dir): %v", outputPath, err)
	}
}

func TestRunNew_NewNotesDirCLIOverride(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	vaultPath := filepath.Join(root, "vault")
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "` + vaultPath + `"` + "\n" +
		`new_notes_dir: "02_Notes"` + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "CLI Dir", NotesDir: "Drafts"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	// Should use CLI override, not config.
	outputPath := filepath.Join(vaultPath, "Drafts", "CLI Dir.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist at %s (CLI override): %v", outputPath, err)
	}

	// Should NOT be in config dir.
	configPath := filepath.Join(vaultPath, "02_Notes", "CLI Dir.md")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("file should not exist in config new_notes_dir when CLI --notes-dir overrides it")
	}
}

func TestRunNew_NewNotesDirEmpty(t *testing.T) {
	t.Parallel()

	// When new_notes_dir is empty, files go to vault root (default behavior).
	cfgPath, vaultPath := writeNewConfig(t)
	opts := CLIOptions{ConfigPath: cfgPath}
	postOpts := NewPostOptions{Title: "Root Note"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew failed: %v", err)
	}

	outputPath := filepath.Join(vaultPath, "Root Note.md")
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("file should exist at vault root: %v", err)
	}
}

func TestRunNew_NewNotesDirDryRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	vaultPath := filepath.Join(root, "vault")
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "` + vaultPath + `"` + "\n" +
		`new_notes_dir: "02_Notes"` + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := CLIOptions{ConfigPath: cfgPath, DryRun: true}
	postOpts := NewPostOptions{Title: "Dry Run Subdir"}

	if err := RunNew(context.TODO(), opts, postOpts); err != nil {
		t.Fatalf("RunNew dry-run failed: %v", err)
	}

	// File should NOT be created in dry-run.
	outputPath := filepath.Join(vaultPath, "02_Notes", "Dry Run Subdir.md")
	if _, err := os.Stat(outputPath); err == nil {
		t.Error("file should not exist in dry-run mode")
	}
}

func TestConfigNewNotesDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.yaml")
	cfgContent := `vault_path: "/tmp"
new_notes_dir: "02_Notes"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	if cfg.NewNotesDir != "02_Notes" {
		t.Errorf("NewNotesDir = %q; want \"02_Notes\"", cfg.NewNotesDir)
	}
}

func TestConfigNewNotesDirDefault(t *testing.T) {
	t.Parallel()

	// Default config should have empty NewNotesDir.
	cfg := config.Default()
	if cfg.NewNotesDir != "" {
		t.Errorf("Default().NewNotesDir = %q; want empty", cfg.NewNotesDir)
	}
}

// writeNewConfig creates a minimal config with a vault directory and returns
// the config path and vault path.
func writeNewConfig(t *testing.T) (cfgPath string, vaultPath string) {
	t.Helper()

	root := t.TempDir()
	vaultPath = filepath.Join(root, "vault")
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		t.Fatalf("mkdir vault: %v", err)
	}

	cfgContent := `vault_path: "` + vaultPath + `"`
	cfgPath = filepath.Join(root, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return cfgPath, vaultPath
}
