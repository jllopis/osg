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
	for path, _ := range images {
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
