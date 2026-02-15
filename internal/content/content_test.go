package content

import "testing"

func TestBuildOutputPath(t *testing.T) {
	path := BuildOutputPath("content", "{date}/{slug}", "2025/11/02", "my-post")
	if path != "content/2025/11/02/my-post/index.md" {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestNormalizeFrontmatter_OSGImage(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
		"osg": map[string]any{
			"image": "hero.jpg",
		},
		"image": "old.jpg",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["image"] != "hero.jpg" {
		t.Fatalf("expected osg.image to take precedence, got %v", out["image"])
	}
}

func TestNormalizeFrontmatter_FallbackImage(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
		"cover": "fallback.png",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["image"] != "fallback.png" {
		t.Fatalf("expected fallback to cover, got %v", out["image"])
	}
}

func TestNormalizeFrontmatter_OSGFeatured(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
		"osg": map[string]any{
			"featured": true,
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["featured"] != true {
		t.Fatalf("expected featured=true, got %v", out["featured"])
	}
}

func TestNormalizeFrontmatter_ExtraFeaturedFallback(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
		"extra": map[string]any{
			"featured": true,
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["featured"] != true {
		t.Fatalf("expected featured=true from extra fallback, got %v", out["featured"])
	}
}

func TestNormalizeFrontmatter_NoFeatured(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if _, ok := out["featured"]; ok {
		t.Fatalf("expected no featured key, got %v", out["featured"])
	}
}

func TestNormalizeFrontmatter_OSGMenu(t *testing.T) {
	fm := map[string]any{
		"title": "About",
		"osg": map[string]any{
			"menu": true,
		},
	}
	out := NormalizeFrontmatter(fm, "about", "2025-01-01", false, "about.md")
	if out["menu"] != true {
		t.Fatalf("expected menu=true, got %v", out["menu"])
	}
}

func TestNormalizeFrontmatter_OSGMenuFalse(t *testing.T) {
	fm := map[string]any{
		"title": "Regular Post",
		"osg": map[string]any{
			"menu": false,
		},
	}
	out := NormalizeFrontmatter(fm, "post", "2025-01-01", false, "post.md")
	if _, ok := out["menu"]; ok {
		t.Fatalf("expected no menu key when osg.menu=false, got %v", out["menu"])
	}
}

func TestNormalizeFrontmatter_NoMenuWithoutOSG(t *testing.T) {
	fm := map[string]any{
		"title": "Regular Post",
	}
	out := NormalizeFrontmatter(fm, "post", "2025-01-01", false, "post.md")
	if _, ok := out["menu"]; ok {
		t.Fatalf("expected no menu key without osg block, got %v", out["menu"])
	}
}
