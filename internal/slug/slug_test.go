package slug

import "testing"

func TestSlugify(t *testing.T) {
	input := "Hello, World!"
	if got := Slugify(input); got != "hello-world" {
		t.Fatalf("expected hello-world, got %s", got)
	}
}

func TestDeriveUsesSlug(t *testing.T) {
	fm := map[string]any{"slug": "Custom Slug"}
	if got := Derive(fm, "file.md"); got != "custom-slug" {
		t.Fatalf("expected custom-slug, got %s", got)
	}
}
