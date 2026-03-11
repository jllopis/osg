package app

import (
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/site"
)

func TestNormalizeLinkPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/foo/bar/", "/foo/bar/"},
		{"/foo/bar", "/foo/bar/"},
		{"/", "/"},
		{"", "/"},
		{"/style.css", "/style.css"},
		{"/page?q=1", "/page/"},
		{"/page#anchor", "/page/"},
		{"/2024/my-post/", "/2024/my-post/"},
		{"foo", "/foo/"},
	}

	for _, tt := range tests {
		got := normalizeLinkPath(tt.input)
		if got != tt.want {
			t.Errorf("normalizeLinkPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsStaticAssetPath(t *testing.T) {
	for _, path := range []string{"/style.css", "/app.js", "/font.woff2", "/img.png", "/data.json"} {
		if !isStaticAssetPath(path) {
			t.Errorf("isStaticAssetPath(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"/about/", "/2024/my-post/", "/tags/go/"} {
		if isStaticAssetPath(path) {
			t.Errorf("isStaticAssetPath(%q) = true, want false", path)
		}
	}
}

func TestIsTaxonomyTermPath(t *testing.T) {
	taxonomies := []config.TaxonomyConfig{
		{Name: "tags"},
		{Name: "area"},
	}

	if !isTaxonomyTermPath("/tags/go/", taxonomies) {
		t.Error("expected /tags/go/ to be a taxonomy term path")
	}
	if !isTaxonomyTermPath("/area/devops/", taxonomies) {
		t.Error("expected /area/devops/ to be a taxonomy term path")
	}
	if isTaxonomyTermPath("/tags/", taxonomies) {
		t.Error("expected /tags/ (index) not to be a taxonomy term path")
	}
	if isTaxonomyTermPath("/about/", taxonomies) {
		t.Error("expected /about/ not to be a taxonomy term path")
	}
}

func TestCheckBrokenInternalLinks(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Post A",
		Path:       "/2024/post-a/",
		SourcePath: "content/2024/post-a.md",
		Content:    `<a href="/2024/post-b/">link</a> <a href="/nonexistent/">broken</a>`,
	})
	s.AddPage(&site.Page{
		Title:      "Post B",
		Path:       "/2024/post-b/",
		SourcePath: "content/2024/post-b.md",
		Content:    `<a href="/2024/post-a/">ok</a>`,
	})
	s.BuildHierarchy()

	cfg := config.Config{
		Taxonomies: []config.TaxonomyConfig{{Name: "tags"}},
	}
	result := &CheckResult{}
	checkBrokenInternalLinks(cfg, s, result, nil)

	if result.Errors != 1 {
		t.Fatalf("expected 1 error, got %d", result.Errors)
	}
	if result.Issues[0].Target != "/nonexistent/" {
		t.Errorf("expected target /nonexistent/, got %s", result.Issues[0].Target)
	}
}

func TestCheckBrokenInternalLinks_SkipsStaticAssets(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Page",
		Path:       "/page/",
		SourcePath: "content/page.md",
		Content:    `<a href="/style.css">css</a> <a href="/app.js">js</a>`,
	})
	s.BuildHierarchy()

	result := &CheckResult{}
	checkBrokenInternalLinks(config.Config{}, s, result, nil)

	if result.Errors != 0 {
		t.Errorf("expected 0 errors for static assets, got %d", result.Errors)
	}
}

func TestCheckBrokenInternalLinks_SkipsTaxonomyTerms(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Page",
		Path:       "/page/",
		SourcePath: "content/page.md",
		Content:    `<a href="/tags/go/">Go tag</a>`,
	})
	s.BuildHierarchy()

	cfg := config.Config{
		Taxonomies: []config.TaxonomyConfig{{Name: "tags"}},
	}
	result := &CheckResult{}
	checkBrokenInternalLinks(cfg, s, result, nil)

	if result.Errors != 0 {
		t.Errorf("expected 0 errors for taxonomy terms, got %d", result.Errors)
	}
}

func TestCheckMissingDates(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{Title: "No Date", SourcePath: "a.md"})
	s.AddPage(&site.Page{Title: "Has Date", SourcePath: "b.md",
		Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)})

	result := &CheckResult{}
	checkMissingDates(s, result, nil)

	if result.Warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", result.Warnings)
	}
}

func TestCheckMissingTags(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{Title: "No Tags", SourcePath: "a.md"})
	s.AddPage(&site.Page{Title: "Has Tags", SourcePath: "b.md",
		Taxonomies: map[string][]string{"tags": {"go"}}})
	s.AddPage(&site.Page{Title: "Menu Page", SourcePath: "c.md", Menu: true}) // should be skipped

	result := &CheckResult{}
	checkMissingTags(s, result, nil)

	if result.Warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", result.Warnings)
	}
}

func TestCheckDuplicateSlugs(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{Title: "A", Path: "/2024/dup/", SourcePath: "a.md"})
	s.AddPage(&site.Page{Title: "B", Path: "/2024/dup/", SourcePath: "b.md"})
	s.AddPage(&site.Page{Title: "C", Path: "/2024/unique/", SourcePath: "c.md"})

	result := &CheckResult{}
	checkDuplicateSlugs(s, result, nil)

	if result.Errors != 2 {
		t.Fatalf("expected 2 errors (one per colliding page), got %d", result.Errors)
	}
}

func TestCheckPermalinkCollisions(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{Title: "A", Path: "/a/", Permalink: "/custom/url/", SourcePath: "a.md"})
	s.AddPage(&site.Page{Title: "B", Path: "/b/", Permalink: "/custom/url/", SourcePath: "b.md"})

	result := &CheckResult{}
	checkPermalinkCollisions(s, result, nil)

	if result.Errors != 2 {
		t.Fatalf("expected 2 errors (one per colliding page), got %d", result.Errors)
	}
}

func TestCheckPermalinkCollisions_NoFalsePositive(t *testing.T) {
	s := site.New()
	// Pages whose permalink == path should be skipped.
	s.AddPage(&site.Page{Title: "A", Path: "/a/", Permalink: "/a/", SourcePath: "a.md"})
	s.AddPage(&site.Page{Title: "B", Path: "/b/", Permalink: "/b/", SourcePath: "b.md"})

	result := &CheckResult{}
	checkPermalinkCollisions(s, result, nil)

	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
}

func TestAddIssue(t *testing.T) {
	result := &CheckResult{}
	addIssue(result, "error", "link", "broken", "a.md", "/x/", "")
	addIssue(result, "warning", "frontmatter", "missing", "b.md", "", "")

	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if result.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", result.Warnings)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestAddMarkdownImageRefs(t *testing.T) {
	raw := `Some text ![alt](images/photo.png) and ![](other.jpg "title")`
	out := make(map[string]bool)
	addMarkdownImageRefs(raw, out)

	if !out["images/photo.png"] {
		t.Error("expected images/photo.png to be referenced")
	}
	if !out["photo.png"] {
		t.Error("expected photo.png (basename) to be referenced")
	}
	if !out["other.jpg"] {
		t.Error("expected other.jpg to be referenced")
	}
}

func TestAddWikilinkImageRefs(t *testing.T) {
	raw := `Some text ![[screenshot.png]] and ![[photo.jpg|alt text]]`
	out := make(map[string]bool)
	addWikilinkImageRefs(raw, out)

	if !out["screenshot.png"] {
		t.Error("expected screenshot.png to be referenced")
	}
	if !out["photo.jpg"] {
		t.Error("expected photo.jpg to be referenced")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string][]string{
		"c": {"1"},
		"a": {"2"},
		"b": {"3"},
	}
	keys := sortedKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("expected [a b c], got %v", keys)
	}
}

func TestCheckOrphanImages_EmptyContentDir(t *testing.T) {
	s := site.New()
	cfg := config.Config{ContentDir: "/nonexistent/path"}

	result := &CheckResult{}
	checkOrphanImages(cfg, s, result, nil)

	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues for nonexistent dir, got %d", len(result.Issues))
	}
}

func TestCheckBrokenImageRefs_SkipsExternalURLs(t *testing.T) {
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Page",
		Path:       "/page/",
		SourcePath: "content/page.md",
		Content:    `<img src="https://example.com/img.png"> <img src="data:image/png;base64,abc">`,
	})
	s.BuildHierarchy()

	cfg := config.Config{ContentDir: "/nonexistent"}
	result := &CheckResult{}
	checkBrokenImageRefs(cfg, s, result, nil)

	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues for external images, got %d", len(result.Issues))
	}
}
