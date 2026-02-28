package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultTheme_CreatesFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := EnsureDefaultTheme(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify theme directory was created.
	themeDir := filepath.Join(dir, "default")
	info, err := os.Stat(themeDir)
	if err != nil {
		t.Fatalf("expected theme dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected theme dir to be a directory")
	}

	// Verify at least some expected files exist.
	for _, sub := range []string{
		"templates/page.html",
		"templates/index.html",
		"templates/section.html",
		"static/style.css",
	} {
		path := filepath.Join(themeDir, sub)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", sub, err)
		}
	}
}

func TestEnsureDefaultTheme_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// First call.
	if err := EnsureDefaultTheme(dir); err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Record mtime of a file.
	target := filepath.Join(dir, "default", "templates", "page.html")
	info1, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Second call — should not modify files (content is identical).
	if err := EnsureDefaultTheme(dir); err != nil {
		t.Fatalf("second call error: %v", err)
	}

	info2, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat after second call: %v", err)
	}

	// ModTime should be unchanged since content is identical.
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("expected mtime to be unchanged (idempotent), got %v vs %v",
			info1.ModTime(), info2.ModTime())
	}
}

func TestEnsureDefaultTheme_EmptyDir(t *testing.T) {
	t.Parallel()
	// Empty themesDir should be a no-op (installTheme returns nil).
	err := EnsureDefaultTheme("")
	if err != nil {
		t.Fatalf("unexpected error for empty dir: %v", err)
	}
}

func TestScaffoldTheme_CreatesTheme(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := ScaffoldTheme(dir, "my-theme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	themeDir := filepath.Join(dir, "my-theme")
	info, err := os.Stat(themeDir)
	if err != nil {
		t.Fatalf("expected theme dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected dir")
	}

	// Should have template files from embedded default.
	if _, err := os.Stat(filepath.Join(themeDir, "templates", "page.html")); err != nil {
		t.Errorf("expected page.html in scaffolded theme: %v", err)
	}
}

func TestScaffoldTheme_EmptyName(t *testing.T) {
	t.Parallel()
	err := ScaffoldTheme(t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestScaffoldTheme_DefaultNameCallsEnsure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Scaffold with name "default" should call EnsureDefaultTheme.
	err := ScaffoldTheme(dir, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created the default theme.
	if _, err := os.Stat(filepath.Join(dir, "default", "templates", "page.html")); err != nil {
		t.Errorf("expected default theme files: %v", err)
	}
}

func TestScaffoldTheme_AlreadyExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "existing"), 0o755)

	err := ScaffoldTheme(dir, "existing")
	if err == nil {
		t.Fatal("expected error for existing theme")
	}
}

func TestScaffoldTheme_EmptyThemesDir(t *testing.T) {
	t.Parallel()
	err := ScaffoldTheme("", "test")
	if err == nil {
		t.Fatal("expected error for empty themes dir")
	}
}

func TestInstallTheme_NoOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// First install.
	if err := installTheme(dir, "test", false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Write a custom file in the same location — should NOT be overwritten.
	customPath := filepath.Join(dir, "test", "templates", "page.html")
	custom := []byte("CUSTOM CONTENT")
	if err := os.WriteFile(customPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	// Second install with overwrite=false.
	if err := installTheme(dir, "test", false); err != nil {
		t.Fatalf("second install: %v", err)
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "CUSTOM CONTENT" {
		t.Error("expected custom content to be preserved with overwrite=false")
	}
}

func TestInstallTheme_Overwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// First install.
	if err := installTheme(dir, "test", false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Write a custom file in the same location.
	customPath := filepath.Join(dir, "test", "templates", "page.html")
	if err := os.WriteFile(customPath, []byte("CUSTOM"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second install with overwrite=true.
	if err := installTheme(dir, "test", true); err != nil {
		t.Fatalf("overwrite install: %v", err)
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "CUSTOM" {
		t.Error("expected custom content to be overwritten")
	}
}
