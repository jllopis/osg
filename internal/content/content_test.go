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

func TestNormalizeFrontmatter_TitleFallbackToFilename(t *testing.T) {
	fm := map[string]any{}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "My Post.md")
	if out["title"] != "My Post" {
		t.Fatalf("expected title from filename, got %v", out["title"])
	}
}

func TestNormalizeFrontmatter_Lang(t *testing.T) {
	fm := map[string]any{"title": "Test", "lang": "en"}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["lang"] != "en" {
		t.Fatalf("expected lang=en, got %v", out["lang"])
	}
}

func TestNormalizeFrontmatter_LanguageAlias(t *testing.T) {
	fm := map[string]any{"title": "Test", "language": "fr"}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["lang"] != "fr" {
		t.Fatalf("expected lang=fr from language alias, got %v", out["lang"])
	}
}

func TestNormalizeFrontmatter_Tags(t *testing.T) {
	fm := map[string]any{"title": "Test", "tags": []any{"go", "rust"}}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	tags, ok := out["tags"].([]string)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", out["tags"])
	}
}

func TestNormalizeFrontmatter_FeaturedStringTrue(t *testing.T) {
	fm := map[string]any{
		"title": "Test",
		"osg":   map[string]any{"featured": "true"},
	}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["featured"] != true {
		t.Fatalf("expected featured=true from string, got %v", out["featured"])
	}
}

func TestNormalizeFrontmatter_DraftFlag(t *testing.T) {
	fm := map[string]any{"title": "Test"}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", true, "test.md")
	if out["draft"] != true {
		t.Fatalf("expected draft=true, got %v", out["draft"])
	}
}

func TestNormalizeFrontmatter_Area(t *testing.T) {
	fm := map[string]any{"title": "Test", "area": "tech"}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["area"] != "tech" {
		t.Fatalf("expected area=tech, got %v", out["area"])
	}
}

func TestNormalizeFrontmatter_Template(t *testing.T) {
	fm := map[string]any{"title": "Test", "template": "custom.html"}
	out := NormalizeFrontmatter(fm, "test", "2025-01-01", false, "test.md")
	if out["template"] != "custom.html" {
		t.Fatalf("expected template=custom.html, got %v", out["template"])
	}
}

// --- toStringSlice ---

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want []string
	}{
		{"nil", nil, nil},
		{"string slice", []string{"a", "b"}, []string{"a", "b"}},
		{"string slice with blanks", []string{"a", "", " ", "b"}, []string{"a", "b"}},
		{"any slice", []any{"go", "rust"}, []string{"go", "rust"}},
		{"any slice mixed types", []any{"go", 42, "rust"}, []string{"go", "rust"}},
		{"any slice with blanks", []any{"go", "", " "}, []string{"go"}},
		{"single string", "solo", []string{"solo"}},
		{"empty string", "", nil},
		{"whitespace string", "  ", nil},
		{"int", 42, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toStringSlice(tc.val)
			if len(got) != len(tc.want) {
				t.Fatalf("toStringSlice(%v) = %v (len %d), want %v (len %d)", tc.val, got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("toStringSlice(%v)[%d] = %q, want %q", tc.val, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// --- compactStrings ---

func TestCompactStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"normal", []string{"a", "b"}, []string{"a", "b"}},
		{"with blanks", []string{"a", "", " ", "b"}, []string{"a", "b"}},
		{"all blanks", []string{"", " ", "  "}, []string{}},
		{"empty input", []string{}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compactStrings(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("compactStrings(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// --- RenderMarkdown ---

func TestRenderMarkdown(t *testing.T) {
	fm := map[string]any{"title": "Hello", "tags": []string{"go"}}
	body := []byte("\nThis is the body.\n")

	data, err := RenderMarkdown(fm, body)
	if err != nil {
		t.Fatalf("RenderMarkdown error: %v", err)
	}

	s := string(data)
	if s[:4] != "---\n" {
		t.Fatalf("expected YAML front delimiters, got %q", s[:10])
	}
	if !contains(s, "title: Hello") {
		t.Fatalf("expected title in output, got %q", s)
	}
	if !contains(s, "This is the body.") {
		t.Fatalf("expected body in output, got %q", s)
	}
}

func TestRenderMarkdownEmptyBody(t *testing.T) {
	fm := map[string]any{"title": "Empty"}
	data, err := RenderMarkdown(fm, nil)
	if err != nil {
		t.Fatalf("RenderMarkdown error: %v", err)
	}
	if !contains(string(data), "title: Empty") {
		t.Fatalf("expected title in output, got %q", string(data))
	}
}

// --- pickBool ---

func TestPickBool(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		key  string
		want bool
	}{
		{"bool true", map[string]any{"menu": true}, "menu", true},
		{"bool false", map[string]any{"menu": false}, "menu", false},
		{"string true", map[string]any{"menu": "true"}, "menu", true},
		{"string TRUE", map[string]any{"menu": "TRUE"}, "menu", true},
		{"string false", map[string]any{"menu": "false"}, "menu", false},
		{"missing key", map[string]any{"other": true}, "menu", false},
		{"nil map", nil, "menu", false},
		{"non-bool value", map[string]any{"menu": 42}, "menu", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickBool(tc.fm, tc.key); got != tc.want {
				t.Errorf("pickBool(%v, %q) = %v, want %v", tc.fm, tc.key, got, tc.want)
			}
		})
	}
}

// --- BuildOutputPath ---

func TestBuildOutputPathEmptyLayout(t *testing.T) {
	path := BuildOutputPath("content", "", "2025/01/02", "my-post")
	if path != "content/2025/01/02/my-post/index.md" {
		t.Fatalf("expected default layout, got %q", path)
	}
}

func TestBuildOutputPathCustomLayout(t *testing.T) {
	path := BuildOutputPath("out", "{slug}", "2025/01/02", "hello")
	if path != "out/hello/index.md" {
		t.Fatalf("expected slug-only layout, got %q", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
