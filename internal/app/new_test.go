package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
