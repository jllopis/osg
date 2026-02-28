package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStaleContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create two page directories with index.md
	fresh := filepath.Join(dir, "2026", "01", "01", "fresh-page")
	stale := filepath.Join(dir, "2026", "01", "01", "stale-page")

	for _, d := range []string{fresh, stale} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "index.md"), []byte("---\ntitle: test\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Add an image next to the stale page (should also be removed)
	if err := os.WriteFile(filepath.Join(stale, "photo.webp"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	// validPaths only contains the fresh page
	validPaths := map[string]string{
		filepath.Join(fresh, "index.md"): "/vault/fresh.md",
	}

	cleaned := removeStaleContent(dir, validPaths, nil)

	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	// Fresh page still exists
	if _, err := os.Stat(filepath.Join(fresh, "index.md")); err != nil {
		t.Errorf("fresh page should still exist: %v", err)
	}

	// Stale directory completely removed
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale directory should have been removed")
	}
}

func TestRemoveStaleContent_SkipsSectionIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a section _index.md (should never be removed)
	section := filepath.Join(dir, "2026")
	if err := os.MkdirAll(section, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(section, "_index.md"), []byte("---\ntitle: 2026\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	validPaths := map[string]string{} // nothing valid

	cleaned := removeStaleContent(dir, validPaths, nil)

	if cleaned != 0 {
		t.Errorf("expected 0 cleaned (only _index.md present), got %d", cleaned)
	}

	if _, err := os.Stat(filepath.Join(section, "_index.md")); err != nil {
		t.Errorf("section _index.md should still exist: %v", err)
	}
}

func TestRemoveStaleContent_EmptyContentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	validPaths := map[string]string{}

	cleaned := removeStaleContent(dir, validPaths, nil)

	if cleaned != 0 {
		t.Errorf("expected 0 cleaned on empty dir, got %d", cleaned)
	}
}

func TestRemoveStaleContent_NoStale(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	page := filepath.Join(dir, "2026", "01", "01", "my-post")
	if err := os.MkdirAll(page, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(page, "index.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	validPaths := map[string]string{
		filepath.Join(page, "index.md"): "/vault/post.md",
	}

	cleaned := removeStaleContent(dir, validPaths, nil)

	if cleaned != 0 {
		t.Errorf("expected 0 cleaned when all content is fresh, got %d", cleaned)
	}
}
