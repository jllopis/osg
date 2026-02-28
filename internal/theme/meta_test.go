package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMeta_Valid(t *testing.T) {
	dir := t.TempDir()
	data := `name: my-theme
description: A test theme
author: Test Author
version: "2.0"
parent: default
min_osg_version: "0.2"
`
	if err := os.WriteFile(filepath.Join(dir, "theme.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := LoadMeta(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "my-theme" {
		t.Errorf("name = %q, want my-theme", meta.Name)
	}
	if meta.Description != "A test theme" {
		t.Errorf("description = %q, want 'A test theme'", meta.Description)
	}
	if meta.Author != "Test Author" {
		t.Errorf("author = %q, want 'Test Author'", meta.Author)
	}
	if meta.Version != "2.0" {
		t.Errorf("version = %q, want '2.0'", meta.Version)
	}
	if meta.Parent != "default" {
		t.Errorf("parent = %q, want 'default'", meta.Parent)
	}
	if meta.MinOSGVersion != "0.2" {
		t.Errorf("min_osg_version = %q, want '0.2'", meta.MinOSGVersion)
	}
}

func TestLoadMeta_NoFile(t *testing.T) {
	dir := t.TempDir()
	meta, err := LoadMeta(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return name = directory basename.
	if meta.Name != filepath.Base(dir) {
		t.Errorf("name = %q, want %q", meta.Name, filepath.Base(dir))
	}
	if meta.Parent != "" {
		t.Errorf("parent should be empty, got %q", meta.Parent)
	}
}

func TestLoadMeta_EmptyName(t *testing.T) {
	dir := t.TempDir()
	data := `description: No name field
`
	if err := os.WriteFile(filepath.Join(dir, "theme.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := LoadMeta(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Name should fall back to directory basename.
	if meta.Name != filepath.Base(dir) {
		t.Errorf("name = %q, want %q", meta.Name, filepath.Base(dir))
	}
}

func TestLoadMeta_EmptyDir(t *testing.T) {
	_, err := LoadMeta("")
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestLoadMeta_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	// YAML with a mapping value where a scalar is expected triggers a parse error.
	if err := os.WriteFile(filepath.Join(dir, "theme.yaml"), []byte("name:\n  - nested: true\n  invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadMeta(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestResolveChain_Single(t *testing.T) {
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "my-theme"), 0o755)

	chain, err := ResolveChain(base, "my-theme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 1 {
		t.Fatalf("chain length = %d, want 1", len(chain))
	}
	if chain[0] != filepath.Join(base, "my-theme") {
		t.Errorf("chain[0] = %q, want %q", chain[0], filepath.Join(base, "my-theme"))
	}
}

func TestResolveChain_WithParent(t *testing.T) {
	base := t.TempDir()
	// Create parent theme.
	parentDir := filepath.Join(base, "parent")
	_ = os.MkdirAll(parentDir, 0o755)
	_ = os.WriteFile(filepath.Join(parentDir, "theme.yaml"), []byte("name: parent\n"), 0o644)

	// Create child theme.
	childDir := filepath.Join(base, "child")
	_ = os.MkdirAll(childDir, 0o755)
	_ = os.WriteFile(filepath.Join(childDir, "theme.yaml"), []byte("name: child\nparent: parent\n"), 0o644)

	chain, err := ResolveChain(base, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("chain length = %d, want 2", len(chain))
	}
	if chain[0] != childDir {
		t.Errorf("chain[0] = %q, want child dir", chain[0])
	}
	if chain[1] != parentDir {
		t.Errorf("chain[1] = %q, want parent dir", chain[1])
	}
}

func TestResolveChain_ThreeLevel(t *testing.T) {
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "grandparent"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "grandparent", "theme.yaml"), []byte("name: grandparent\n"), 0o644)

	_ = os.MkdirAll(filepath.Join(base, "parent"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "parent", "theme.yaml"), []byte("name: parent\nparent: grandparent\n"), 0o644)

	_ = os.MkdirAll(filepath.Join(base, "child"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "child", "theme.yaml"), []byte("name: child\nparent: parent\n"), 0o644)

	chain, err := ResolveChain(base, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("chain length = %d, want 3", len(chain))
	}
}

func TestResolveChain_Cycle(t *testing.T) {
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "a"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "a", "theme.yaml"), []byte("name: a\nparent: b\n"), 0o644)

	_ = os.MkdirAll(filepath.Join(base, "b"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "b", "theme.yaml"), []byte("name: b\nparent: a\n"), 0o644)

	_, err := ResolveChain(base, "a")
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestResolveChain_MissingParent(t *testing.T) {
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "child"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "child", "theme.yaml"), []byte("name: child\nparent: nonexistent\n"), 0o644)

	_, err := ResolveChain(base, "child")
	if err == nil {
		t.Fatal("expected error for missing parent")
	}
}

func TestResolveChain_EmptyInputs(t *testing.T) {
	chain, err := ResolveChain("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chain != nil {
		t.Errorf("expected nil chain, got %v", chain)
	}
}

func TestListThemes(t *testing.T) {
	base := t.TempDir()
	// Create two themes.
	_ = os.MkdirAll(filepath.Join(base, "alpha"), 0o755)
	_ = os.WriteFile(filepath.Join(base, "alpha", "theme.yaml"), []byte("name: alpha\ndescription: First\n"), 0o644)

	_ = os.MkdirAll(filepath.Join(base, "beta"), 0o755)
	// No theme.yaml — should still appear with directory name.

	// Create a regular file (not a dir) — should be ignored.
	_ = os.WriteFile(filepath.Join(base, "not-a-theme.txt"), []byte("hello"), 0o644)

	themes, err := ListThemes(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(themes) != 2 {
		t.Fatalf("themes count = %d, want 2", len(themes))
	}
}

func TestListThemes_EmptyDir(t *testing.T) {
	themes, err := ListThemes("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if themes != nil {
		t.Errorf("expected nil, got %v", themes)
	}
}

func TestListThemes_NonexistentDir(t *testing.T) {
	themes, err := ListThemes("/nonexistent/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if themes != nil {
		t.Errorf("expected nil, got %v", themes)
	}
}

func TestWriteMeta(t *testing.T) {
	dir := t.TempDir()
	meta := ThemeMeta{
		Name:        "test-theme",
		Description: "A test",
		Parent:      "default",
	}
	if err := WriteMeta(dir, meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back.
	loaded, err := LoadMeta(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Name != "test-theme" {
		t.Errorf("name = %q, want 'test-theme'", loaded.Name)
	}
	if loaded.Parent != "default" {
		t.Errorf("parent = %q, want 'default'", loaded.Parent)
	}
}

func TestScaffoldChildTheme(t *testing.T) {
	base := t.TempDir()
	// Create parent theme.
	parentDir := filepath.Join(base, "default")
	_ = os.MkdirAll(parentDir, 0o755)
	_ = os.WriteFile(filepath.Join(parentDir, "theme.yaml"), []byte("name: default\n"), 0o644)

	err := ScaffoldChildTheme(base, "my-child", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify theme.yaml was created.
	childDir := filepath.Join(base, "my-child")
	meta, err := LoadMeta(childDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "my-child" {
		t.Errorf("name = %q, want 'my-child'", meta.Name)
	}
	if meta.Parent != "default" {
		t.Errorf("parent = %q, want 'default'", meta.Parent)
	}

	// Verify directory structure.
	for _, sub := range []string{"templates/partials", "static", "i18n"} {
		path := filepath.Join(childDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", sub, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}
}

func TestScaffoldChildTheme_MissingParent(t *testing.T) {
	base := t.TempDir()
	err := ScaffoldChildTheme(base, "child", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing parent")
	}
}

func TestScaffoldChildTheme_SelfParent(t *testing.T) {
	base := t.TempDir()
	err := ScaffoldChildTheme(base, "x", "x")
	if err == nil {
		t.Fatal("expected error for self-parent")
	}
}

func TestScaffoldChildTheme_AlreadyExists(t *testing.T) {
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "default"), 0o755)
	_ = os.MkdirAll(filepath.Join(base, "child"), 0o755)
	err := ScaffoldChildTheme(base, "child", "default")
	if err == nil {
		t.Fatal("expected error for existing theme")
	}
}
