package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildImageIndex(t *testing.T) {
	// Create a temp vault structure
	root := t.TempDir()

	// Create directories
	attachDir := filepath.Join(root, "Attachments")
	notesDir := filepath.Join(root, "Notes")
	hiddenDir := filepath.Join(root, ".obsidian")

	for _, dir := range []string{attachDir, notesDir, hiddenDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create image files
	images := map[string]string{
		filepath.Join(attachDir, "hero.png"):    "png",
		filepath.Join(attachDir, "photo.jpg"):   "jpg",
		filepath.Join(notesDir, "diagram.svg"):  "svg",
		filepath.Join(hiddenDir, "hidden.png"):  "hidden", // should be skipped
		filepath.Join(root, "readme.md"):        "md",     // not an image
		filepath.Join(root, ".secret.png"):      "hidden", // dot-prefixed
		filepath.Join(attachDir, "video.mp4"):   "video",  // not an image ext
		filepath.Join(attachDir, "banner.webp"): "webp",
	}
	for path := range images {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx, err := BuildImageIndex(root)
	if err != nil {
		t.Fatalf("BuildImageIndex: %v", err)
	}

	// Should find these by basename
	for _, name := range []string{"hero.png", "photo.jpg", "diagram.svg", "banner.webp"} {
		if _, ok := idx[name]; !ok {
			t.Errorf("expected %s in index", name)
		}
	}

	// Should find by vault-relative path
	if _, ok := idx["Attachments/hero.png"]; !ok {
		t.Error("expected Attachments/hero.png in index")
	}

	// Should NOT contain hidden files or non-images
	for _, name := range []string{"hidden.png", ".secret.png", "readme.md", "video.mp4"} {
		if _, ok := idx[name]; ok {
			t.Errorf("did not expect %s in index", name)
		}
	}
}

func TestImageIndex_Resolve(t *testing.T) {
	idx := ImageIndex{
		"hero.png":             "/vault/Attachments/hero.png",
		"Attachments/hero.png": "/vault/Attachments/hero.png",
		"photo.jpg":            "/vault/Notes/photo.jpg",
	}

	tests := []struct {
		ref    string
		expect string
		ok     bool
	}{
		{"hero.png", "/vault/Attachments/hero.png", true},
		{"Attachments/hero.png", "/vault/Attachments/hero.png", true},
		{"photo.jpg", "/vault/Notes/photo.jpg", true},
		{"missing.png", "", false},
		{"", "", false},
		{"  hero.png  ", "/vault/Attachments/hero.png", true},
	}

	for _, tc := range tests {
		got, ok := idx.Resolve(tc.ref)
		if ok != tc.ok || got != tc.expect {
			t.Errorf("Resolve(%q): got (%q, %v), want (%q, %v)", tc.ref, got, ok, tc.expect, tc.ok)
		}
	}
}

func TestListMarkdownFiles(t *testing.T) {
	root := t.TempDir()

	// Create directories
	notesDir := filepath.Join(root, "Notes")
	hiddenDir := filepath.Join(root, ".obsidian")
	subDir := filepath.Join(notesDir, "Sub")
	for _, dir := range []string{notesDir, hiddenDir, subDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create files
	files := map[string]bool{
		filepath.Join(notesDir, "post.md"):    true,
		filepath.Join(notesDir, "draft.md"):   true,
		filepath.Join(subDir, "nested.md"):    true,
		filepath.Join(notesDir, "image.png"):  false, // not markdown
		filepath.Join(hiddenDir, "hidden.md"): false, // hidden dir
		filepath.Join(root, ".secret.md"):     false, // dot-prefixed file
		filepath.Join(notesDir, "README.MD"):  true,  // uppercase .MD
		filepath.Join(notesDir, "data.json"):  false, // not markdown
	}
	for path := range files {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := ListMarkdownFiles(root)
	if err != nil {
		t.Fatalf("ListMarkdownFiles: %v", err)
	}

	// Count expected markdown files
	expected := 0
	for _, isMarkdown := range files {
		if isMarkdown {
			expected++
		}
	}

	if len(result) != expected {
		t.Fatalf("expected %d markdown files, got %d: %v", expected, len(result), result)
	}

	// Verify no hidden or non-md files snuck in
	for _, f := range result {
		base := filepath.Base(f)
		if base[0] == '.' {
			t.Errorf("dot-prefixed file should not be listed: %s", f)
		}
		ext := filepath.Ext(base)
		if ext != ".md" && ext != ".MD" {
			t.Errorf("non-markdown file should not be listed: %s", f)
		}
	}
}

func TestListMarkdownFilesEmptyPath(t *testing.T) {
	_, err := ListMarkdownFiles("")
	if err == nil {
		t.Fatal("expected error for empty vault path")
	}
}

func TestListMarkdownFilesEmptyDir(t *testing.T) {
	root := t.TempDir()
	result, err := ListMarkdownFiles(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 files in empty dir, got %d", len(result))
	}
}

func TestBuildImageIndexEmptyPath(t *testing.T) {
	_, err := BuildImageIndex("")
	if err == nil {
		t.Fatal("expected error for empty vault path")
	}
}
