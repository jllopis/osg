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

func TestNormalizeFrontmatter_OSGAbstract(t *testing.T) {
	fm := map[string]any{
		"title":   "Test",
		"summary": "auto summary",
		"osg": map[string]any{
			"abstract": "Hand-written abstract from Obsidian.",
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["summary"] != "Hand-written abstract from Obsidian." {
		t.Fatalf("expected osg.abstract to take precedence, got %v", out["summary"])
	}
}

func TestNormalizeFrontmatter_OSGAbstractOverridesDescription(t *testing.T) {
	fm := map[string]any{
		"title":       "Test",
		"description": "meta description",
		"osg": map[string]any{
			"abstract": "The abstract wins.",
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["summary"] != "The abstract wins." {
		t.Fatalf("expected osg.abstract over description, got %v", out["summary"])
	}
}

func TestNormalizeFrontmatter_FallbackSummaryWithoutAbstract(t *testing.T) {
	fm := map[string]any{
		"title":   "Test",
		"summary": "fallback summary",
		"osg": map[string]any{
			"publish": true,
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["summary"] != "fallback summary" {
		t.Fatalf("expected fallback to summary when no abstract, got %v", out["summary"])
	}
}

func TestNormalizeFrontmatter_NoAbstractNoSummary(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if _, ok := out["summary"]; ok {
		t.Fatalf("expected no summary key, got %v", out["summary"])
	}
}

func TestNormalizeFrontmatter_OSGAuthor(t *testing.T) {
	fm := map[string]any{
		"title":  "Test",
		"author": "Top Level Author",
		"osg": map[string]any{
			"author": "Joan Llopis",
		},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["author"] != "Joan Llopis" {
		t.Fatalf("expected osg.author to take precedence, got %v", out["author"])
	}
}

func TestNormalizeFrontmatter_FallbackAuthor(t *testing.T) {
	fm := map[string]any{
		"title":  "Test",
		"author": "Fallback Author",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["author"] != "Fallback Author" {
		t.Fatalf("expected fallback to top-level author, got %v", out["author"])
	}
}

func TestNormalizeFrontmatter_NoAuthor(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if _, ok := out["author"]; ok {
		t.Fatalf("expected no author key, got %v", out["author"])
	}
}
