package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/site"
	"osg/internal/summary"
	"osg/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// outputHTMLPath
// ---------------------------------------------------------------------------

func TestOutputHTMLPath(t *testing.T) {
	tests := []struct {
		name      string
		publicDir string
		sitePath  string
		want      string
	}{
		{
			name:      "empty site path",
			publicDir: "public",
			sitePath:  "",
			want:      filepath.Join("public", "index.html"),
		},
		{
			name:      "root slash",
			publicDir: "public",
			sitePath:  "/",
			want:      filepath.Join("public", "index.html"),
		},
		{
			name:      "page path without trailing slash",
			publicDir: "public",
			sitePath:  "/blog/hello",
			want:      filepath.Join("public", "blog", "hello", "index.html"),
		},
		{
			name:      "page path with trailing slash",
			publicDir: "public",
			sitePath:  "/blog/hello/",
			want:      filepath.Join("public", "blog", "hello", "index.html"),
		},
		{
			name:      "nested path",
			publicDir: "/var/www",
			sitePath:  "/docs/guide/intro",
			want:      filepath.Join("/var/www", "docs", "guide", "intro", "index.html"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := outputHTMLPath(tc.publicDir, tc.sitePath)
			if got != tc.want {
				t.Errorf("outputHTMLPath(%q, %q) = %q; want %q", tc.publicDir, tc.sitePath, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// outputFilePath
// ---------------------------------------------------------------------------

func TestOutputFilePath(t *testing.T) {
	tests := []struct {
		name      string
		publicDir string
		sitePath  string
		filename  string
		want      string
	}{
		{
			name:      "root level file",
			publicDir: "public",
			sitePath:  "",
			filename:  "atom.xml",
			want:      filepath.Join("public", "atom.xml"),
		},
		{
			name:      "root slash path",
			publicDir: "public",
			sitePath:  "/",
			filename:  "rss.xml",
			want:      filepath.Join("public", "rss.xml"),
		},
		{
			name:      "nested path",
			publicDir: "public",
			sitePath:  "/tags/go",
			filename:  "atom.xml",
			want:      filepath.Join("public", "tags", "go", "atom.xml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := outputFilePath(tc.publicDir, tc.sitePath, tc.filename)
			if got != tc.want {
				t.Errorf("outputFilePath(%q, %q, %q) = %q; want %q", tc.publicDir, tc.sitePath, tc.filename, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildURL
// ---------------------------------------------------------------------------

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "empty base URL returns path",
			baseURL: "",
			path:    "/blog/",
			want:    "/blog/",
		},
		{
			name:    "whitespace-only base URL returns path",
			baseURL: "   ",
			path:    "/about/",
			want:    "/about/",
		},
		{
			name:    "base URL with trailing slash",
			baseURL: "https://example.com/",
			path:    "/blog/hello/",
			want:    "https://example.com/blog/hello/",
		},
		{
			name:    "base URL without trailing slash",
			baseURL: "https://example.com",
			path:    "/blog/hello/",
			want:    "https://example.com/blog/hello/",
		},
		{
			name:    "root path",
			baseURL: "https://example.com",
			path:    "/",
			want:    "https://example.com/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildURL(tc.baseURL, tc.path)
			if got != tc.want {
				t.Errorf("buildURL(%q, %q) = %q; want %q", tc.baseURL, tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ensureTrailingSlash
// ---------------------------------------------------------------------------

func TestEnsureTrailingSlash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/blog", "/blog/"},
		{"/blog/", "/blog/"},
		{"", "/"},
		{"/", "/"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ensureTrailingSlash(tc.input)
			if got != tc.want {
				t.Errorf("ensureTrailingSlash(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// taxonomyPagePath
// ---------------------------------------------------------------------------

func TestTaxonomyPagePath(t *testing.T) {
	tests := []struct {
		name         string
		termPath     string
		paginatePath string
		index        int
		want         string
	}{
		{
			name:         "index 0 returns term path with trailing slash",
			termPath:     "/tags/go",
			paginatePath: "page",
			index:        0,
			want:         "/tags/go/",
		},
		{
			name:         "index 1 returns page 2",
			termPath:     "/tags/go",
			paginatePath: "page",
			index:        1,
			want:         "/tags/go/page/2/",
		},
		{
			name:         "index 2 returns page 3",
			termPath:     "/tags/go",
			paginatePath: "page",
			index:        2,
			want:         "/tags/go/page/3/",
		},
		{
			name:         "empty paginate path defaults to page",
			termPath:     "/categories/tech",
			paginatePath: "",
			index:        1,
			want:         "/categories/tech/page/2/",
		},
		{
			name:         "custom paginate path",
			termPath:     "/tags/go",
			paginatePath: "p",
			index:        3,
			want:         "/tags/go/p/4/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := taxonomyPagePath(tc.termPath, tc.paginatePath, tc.index)
			if got != tc.want {
				t.Errorf("taxonomyPagePath(%q, %q, %d) = %q; want %q", tc.termPath, tc.paginatePath, tc.index, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// cloneMap
// ---------------------------------------------------------------------------

func TestCloneMap(t *testing.T) {
	t.Run("shallow copy", func(t *testing.T) {
		original := map[string]any{"a": 1, "b": "hello"}
		cloned := cloneMap(original)
		if len(cloned) != len(original) {
			t.Fatalf("length mismatch: got %d, want %d", len(cloned), len(original))
		}
		for key, value := range original {
			if cloned[key] != value {
				t.Errorf("cloned[%q] = %v; want %v", key, cloned[key], value)
			}
		}
		// Mutation of clone should not affect original.
		cloned["c"] = "new"
		if _, ok := original["c"]; ok {
			t.Error("mutation of clone affected original")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		cloned := cloneMap(nil)
		if cloned == nil {
			t.Error("cloneMap(nil) returned nil; want empty map")
		}
		if len(cloned) != 0 {
			t.Errorf("cloneMap(nil) returned map with %d entries; want 0", len(cloned))
		}
	})
}

// ---------------------------------------------------------------------------
// configView
// ---------------------------------------------------------------------------

func TestConfigView(t *testing.T) {
	cfg := config.Config{
		BaseURL:     "https://example.com",
		SiteTitle:   "My Site",
		Theme:       "default",
		ColorScheme: "auto",
	}

	view := configView(cfg)
	if view["base_url"] != "https://example.com" {
		t.Errorf("base_url = %v; want %q", view["base_url"], "https://example.com")
	}
	if view["site_title"] != "My Site" {
		t.Errorf("site_title = %v; want %q", view["site_title"], "My Site")
	}
	if view["theme"] != "default" {
		t.Errorf("theme = %v; want %q", view["theme"], "default")
	}
	if view["color_scheme"] != "auto" {
		t.Errorf("color_scheme = %v; want %q", view["color_scheme"], "auto")
	}
	// Check that logging sub-map exists.
	logging, ok := view["logging"].(map[string]any)
	if !ok {
		t.Fatal("logging key missing or wrong type")
	}
	if logging["level"] != cfg.Logging.Level {
		t.Errorf("logging.level = %v; want %v", logging["level"], cfg.Logging.Level)
	}
	// Check image optimization keys.
	if _, ok := view["image_optimization"]; !ok {
		t.Error("image_optimization key missing from configView")
	}
	if _, ok := view["image_quality"]; !ok {
		t.Error("image_quality key missing from configView")
	}
	if _, ok := view["image_widths"]; !ok {
		t.Error("image_widths key missing from configView")
	}
}

// ---------------------------------------------------------------------------
// latestUpdated
// ---------------------------------------------------------------------------

func TestLatestUpdated(t *testing.T) {
	t.Run("returns latest date", func(t *testing.T) {
		d1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		d2 := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		d3 := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
		pages := []*site.Page{
			{Date: d1},
			{Date: d2},
			{Date: d3},
		}
		got := latestUpdated(pages)
		if !got.Equal(d2) {
			t.Errorf("latestUpdated returned %v; want %v", got, d2)
		}
	})

	t.Run("empty pages returns approximately now", func(t *testing.T) {
		before := time.Now().Add(-time.Second)
		got := latestUpdated(nil)
		after := time.Now().Add(time.Second)
		if got.Before(before) || got.After(after) {
			t.Errorf("latestUpdated(nil) = %v; expected near time.Now()", got)
		}
	})

	t.Run("single page", func(t *testing.T) {
		d := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
		pages := []*site.Page{{Date: d}}
		got := latestUpdated(pages)
		if !got.Equal(d) {
			t.Errorf("latestUpdated returned %v; want %v", got, d)
		}
	})
}

// ---------------------------------------------------------------------------
// sitemapEntryViews
// ---------------------------------------------------------------------------

func TestSitemapEntryViews(t *testing.T) {
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	entries := []SitemapEntry{
		{Permalink: "https://example.com/a/", Updated: ts, Extra: map[string]any{"priority": "0.8"}},
		{Permalink: "https://example.com/b/", Updated: ts.Add(24 * time.Hour)},
	}

	views := sitemapEntryViews(entries)
	if len(views) != 2 {
		t.Fatalf("got %d views; want 2", len(views))
	}
	if views[0]["permalink"] != "https://example.com/a/" {
		t.Errorf("views[0].permalink = %v", views[0]["permalink"])
	}
	if views[0]["updated"] != ts.Format(time.RFC3339) {
		t.Errorf("views[0].updated = %v; want %v", views[0]["updated"], ts.Format(time.RFC3339))
	}
	extra, ok := views[0]["extra"].(map[string]any)
	if !ok {
		t.Fatal("views[0].extra missing or wrong type")
	}
	if extra["priority"] != "0.8" {
		t.Errorf("views[0].extra.priority = %v; want %q", extra["priority"], "0.8")
	}
	// Second entry has no Extra set (nil map).
	if extra2, ok := views[1]["extra"].(map[string]any); ok && len(extra2) > 0 {
		t.Errorf("views[1].extra = %v; want nil or empty", views[1]["extra"])
	}
}

// ---------------------------------------------------------------------------
// feedPages
// ---------------------------------------------------------------------------

func TestFeedPages(t *testing.T) {
	ts := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	pages := []*site.Page{
		{
			Title:     "Hello",
			Permalink: "https://example.com/hello/",
			Date:      ts,
			Summary:   "A summary",
			Content:   "<p>Hello world</p>",
			Image:     "/img/hello.jpg",
			Path:      "/hello/",
		},
	}

	views := feedPages(pages)
	if len(views) != 1 {
		t.Fatalf("got %d views; want 1", len(views))
	}
	v := views[0]
	if v["title"] != "Hello" {
		t.Errorf("title = %v", v["title"])
	}
	if v["permalink"] != "https://example.com/hello/" {
		t.Errorf("permalink = %v", v["permalink"])
	}
	if v["date"] != ts.Format(time.RFC3339) {
		t.Errorf("date = %v; want %v", v["date"], ts.Format(time.RFC3339))
	}
	if v["summary"] != "A summary" {
		t.Errorf("summary = %v", v["summary"])
	}
	if v["image"] != "/img/hello.jpg" {
		t.Errorf("image = %v", v["image"])
	}

	t.Run("empty pages", func(t *testing.T) {
		views := feedPages(nil)
		if len(views) != 0 {
			t.Errorf("feedPages(nil) returned %d views; want 0", len(views))
		}
	})
}

// ---------------------------------------------------------------------------
// buildStatsView
// ---------------------------------------------------------------------------

func TestBuildStatsView(t *testing.T) {
	stats := Stats{Total: 10, Rendered: 7, Skipped: 1, Cached: 2, Errors: 0}
	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "A", Path: "/a/"})
	siteIndex.AddPage(&site.Page{Title: "B", Path: "/b/"})
	siteIndex.AddSection(&site.Section{Path: "/", IsRoot: true})

	view := buildStatsView(stats, siteIndex)
	if view["total"] != 10 {
		t.Errorf("total = %v; want 10", view["total"])
	}
	if view["rendered"] != 7 {
		t.Errorf("rendered = %v; want 7", view["rendered"])
	}
	if view["pages"] != 2 {
		t.Errorf("pages = %v; want 2", view["pages"])
	}
	if view["sections"] != 1 {
		t.Errorf("sections = %v; want 1", view["sections"])
	}
}

// ---------------------------------------------------------------------------
// themeTemplatesDir
// ---------------------------------------------------------------------------

func TestThemeTemplatesDir(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{
			name: "both set",
			cfg:  config.Config{ThemesDir: "themes", Theme: "default"},
			want: "themes/default/templates",
		},
		{
			name: "empty themes dir",
			cfg:  config.Config{ThemesDir: "", Theme: "default"},
			want: "",
		},
		{
			name: "whitespace themes dir",
			cfg:  config.Config{ThemesDir: "  ", Theme: "default"},
			want: "",
		},
		{
			name: "empty theme",
			cfg:  config.Config{ThemesDir: "themes", Theme: ""},
			want: "",
		},
		{
			name: "whitespace theme",
			cfg:  config.Config{ThemesDir: "themes", Theme: "   "},
			want: "",
		},
		{
			name: "both empty",
			cfg:  config.Config{ThemesDir: "", Theme: ""},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := themeTemplatesDir(tc.cfg)
			if got != tc.want {
				t.Errorf("themeTemplatesDir(%+v) = %q; want %q", tc.cfg, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// fillWithProvider — returns affected source paths
// ---------------------------------------------------------------------------

func TestFillWithProvider_ReturnsAffectedPaths(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("auto fills empty summaries and returns paths", func(t *testing.T) {
		siteIndex := site.New()
		siteIndex.AddPage(&site.Page{Title: "A", SourcePath: "a.md", RawContent: "Hello world. This is a test."})
		siteIndex.AddPage(&site.Page{Title: "B", SourcePath: "b.md", Summary: "already set"})
		siteIndex.AddPage(&site.Page{Title: "C", SourcePath: "c.md", RawContent: "Another page. More content here."})

		affected := fillWithProvider(context.Background(), siteIndex, summary.ExtractProvider{}, "auto", logger)

		if len(affected) != 2 {
			t.Fatalf("expected 2 affected paths, got %d: %v", len(affected), affected)
		}

		// Page B already had a summary — should not appear.
		for _, sp := range affected {
			if sp == "b.md" {
				t.Error("page B was already summarized; should not be in affected list")
			}
		}

		// Verify summaries were set.
		if siteIndex.Pages[0].Summary == "" {
			t.Error("page A summary should have been set")
		}
		if siteIndex.Pages[2].Summary == "" {
			t.Error("page C summary should have been set")
		}
	})

	t.Run("noop returns nil", func(t *testing.T) {
		siteIndex := site.New()
		siteIndex.AddPage(&site.Page{Title: "A", SourcePath: "a.md", RawContent: "Some content."})

		affected := fillWithProvider(context.Background(), siteIndex, summary.NoopProvider{}, "manual", logger)
		if affected != nil {
			t.Errorf("expected nil for noop, got %v", affected)
		}
	})
}

// ---------------------------------------------------------------------------
// fillWithAI — returns affected source paths for cache hits and LLM results
// ---------------------------------------------------------------------------

// fakeProvider is a test double for summary.Provider.
type fakeProvider struct {
	results map[string]string // title -> summary
}

func (f fakeProvider) Summarize(_ context.Context, title string, _ string) (string, error) {
	if s, ok := f.results[title]; ok {
		return s, nil
	}
	return "", fmt.Errorf("quota exhausted")
}

func TestFillWithAI_ReturnsAffectedFromCacheAndLLM(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Cached", SourcePath: "cached.md", RawContent: "cached body"})
	siteIndex.AddPage(&site.Page{Title: "New", SourcePath: "new.md", RawContent: "new body"})
	siteIndex.AddPage(&site.Page{Title: "Fail", SourcePath: "fail.md", RawContent: "fail body"})

	cache := newAICache("gemini", "test-model")
	cachedHash := contentHash("cached body")
	cache.Store(cachedHash, "cached summary")

	provider := fakeProvider{results: map[string]string{
		"New": "new summary",
		// "Fail" intentionally missing -> provider returns error
	}}

	affected := fillWithAI(context.Background(), siteIndex, provider, 10*time.Second, 2, cache, false, logger, nil)

	// "Cached" -> cache hit, should be affected.
	// "New"    -> LLM success, should be affected.
	// "Fail"   -> LLM error, should NOT be affected.
	if len(affected) != 2 {
		t.Fatalf("expected 2 affected paths, got %d: %v", len(affected), affected)
	}

	has := map[string]bool{}
	for _, sp := range affected {
		has[sp] = true
	}
	if !has["cached.md"] {
		t.Error("cached.md should be in affected list (cache hit)")
	}
	if !has["new.md"] {
		t.Error("new.md should be in affected list (LLM result)")
	}
	if has["fail.md"] {
		t.Error("fail.md should NOT be in affected list (LLM error)")
	}

	// Verify summaries were set.
	if siteIndex.Pages[0].Summary != "cached summary" {
		t.Errorf("Cached page summary = %q; want %q", siteIndex.Pages[0].Summary, "cached summary")
	}
	if siteIndex.Pages[1].Summary != "new summary" {
		t.Errorf("New page summary = %q; want %q", siteIndex.Pages[1].Summary, "new summary")
	}
	if siteIndex.Pages[2].Summary != "" {
		t.Errorf("Fail page summary = %q; want empty", siteIndex.Pages[2].Summary)
	}
}

// ---------------------------------------------------------------------------
// fillWithProvider — skips menu pages
// ---------------------------------------------------------------------------

func TestFillWithProvider_SkipsMenuPages(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Post", SourcePath: "post.md", RawContent: "Post content here."})
	siteIndex.AddPage(&site.Page{Title: "About", SourcePath: "about.md", RawContent: "About page content.", Menu: true})

	affected := fillWithProvider(context.Background(), siteIndex, summary.ExtractProvider{}, "auto", logger)

	if len(affected) != 1 {
		t.Fatalf("expected 1 affected path, got %d: %v", len(affected), affected)
	}
	if affected[0] != "post.md" {
		t.Errorf("affected[0] = %q; want %q", affected[0], "post.md")
	}
	if siteIndex.Pages[1].Summary != "" {
		t.Errorf("menu page should not have a summary, got %q", siteIndex.Pages[1].Summary)
	}
}

// ---------------------------------------------------------------------------
// fillWithAI — skips menu pages
// ---------------------------------------------------------------------------

func TestFillWithAI_SkipsMenuPages(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Post", SourcePath: "post.md", RawContent: "Post body"})
	siteIndex.AddPage(&site.Page{Title: "About", SourcePath: "about.md", RawContent: "About body", Menu: true})

	cache := newAICache("gemini", "test-model")
	provider := fakeProvider{results: map[string]string{
		"Post":  "Post summary",
		"About": "About summary",
	}}

	affected := fillWithAI(context.Background(), siteIndex, provider, 10*time.Second, 2, cache, false, logger, nil)

	if len(affected) != 1 {
		t.Fatalf("expected 1 affected path, got %d: %v", len(affected), affected)
	}
	if affected[0] != "post.md" {
		t.Errorf("affected[0] = %q; want %q", affected[0], "post.md")
	}
	if siteIndex.Pages[0].Summary != "Post summary" {
		t.Errorf("post summary = %q; want %q", siteIndex.Pages[0].Summary, "Post summary")
	}
	if siteIndex.Pages[1].Summary != "" {
		t.Errorf("menu page should not have a summary, got %q", siteIndex.Pages[1].Summary)
	}
}

// ---------------------------------------------------------------------------
// fillFromCacheOrAuto — serve mode: cached AI + auto fallback
// ---------------------------------------------------------------------------

func TestFillFromCacheOrAuto_UsesCachedAndFallsBack(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Cached", SourcePath: "cached.md", RawContent: "cached body"})
	siteIndex.AddPage(&site.Page{Title: "Uncached", SourcePath: "uncached.md", RawContent: "Uncached body. This is a test page."})
	siteIndex.AddPage(&site.Page{Title: "HasSummary", SourcePath: "has.md", RawContent: "body", Summary: "already set"})
	siteIndex.AddPage(&site.Page{Title: "MenuPage", SourcePath: "menu.md", RawContent: "menu body", Menu: true})

	cache := newAICache("gemini", "test-model")
	cache.Store(contentHash("cached body"), "AI cached summary")

	affected := fillFromCacheOrAuto(context.Background(), siteIndex, cache, logger)

	// "Cached"     -> cache hit
	// "Uncached"   -> auto-extracted
	// "HasSummary" -> skipped (already has summary)
	// "MenuPage"   -> skipped (menu page)
	if len(affected) != 2 {
		t.Fatalf("expected 2 affected paths, got %d: %v", len(affected), affected)
	}

	has := map[string]bool{}
	for _, sp := range affected {
		has[sp] = true
	}
	if !has["cached.md"] {
		t.Error("cached.md should be in affected list (cache hit)")
	}
	if !has["uncached.md"] {
		t.Error("uncached.md should be in affected list (auto fallback)")
	}

	// Verify the cached page got the AI summary, not auto-extracted.
	if siteIndex.Pages[0].Summary != "AI cached summary" {
		t.Errorf("cached page summary = %q; want %q", siteIndex.Pages[0].Summary, "AI cached summary")
	}
	// Verify the uncached page got an auto-extracted summary.
	if siteIndex.Pages[1].Summary == "" {
		t.Error("uncached page should have an auto-extracted summary")
	}
	if siteIndex.Pages[1].Summary == "AI cached summary" {
		t.Error("uncached page should NOT have the AI cached summary")
	}
	// Verify existing summary was not overwritten.
	if siteIndex.Pages[2].Summary != "already set" {
		t.Errorf("page with existing summary = %q; want %q", siteIndex.Pages[2].Summary, "already set")
	}
	// Verify menu page was skipped.
	if siteIndex.Pages[3].Summary != "" {
		t.Errorf("menu page should not have a summary, got %q", siteIndex.Pages[3].Summary)
	}
}

func TestFillFromCacheOrAuto_EmptyCache(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "A", SourcePath: "a.md", RawContent: "Some content. More sentences."})

	cache := newAICache("", "")

	affected := fillFromCacheOrAuto(context.Background(), siteIndex, cache, logger)

	if len(affected) != 1 {
		t.Fatalf("expected 1 affected path, got %d: %v", len(affected), affected)
	}
	if siteIndex.Pages[0].Summary == "" {
		t.Error("page should have an auto-extracted summary when cache is empty")
	}
}

func TestFillWithAI_AffectedPagesMarkChangedInPlan(t *testing.T) {
	// Simulate the integration: fillSummaries returns affected paths,
	// and Run() adds them to plan.changedFiles.
	plan := buildPlan{
		incremental: true,
		full:        false,
		changedFiles: map[string]bool{
			"already-changed.md": true,
		},
		contentChanged: false,
	}

	summaryChanged := []string{"page-a.md", "page-b.md"}

	// This mirrors the logic in Run().
	if len(summaryChanged) > 0 {
		if plan.changedFiles == nil {
			plan.changedFiles = make(map[string]bool)
		}
		for _, sp := range summaryChanged {
			plan.changedFiles[sp] = true
		}
		plan.contentChanged = true
	}

	if !plan.changedFiles["page-a.md"] {
		t.Error("page-a.md should be in changedFiles")
	}
	if !plan.changedFiles["page-b.md"] {
		t.Error("page-b.md should be in changedFiles")
	}
	if !plan.changedFiles["already-changed.md"] {
		t.Error("already-changed.md should still be in changedFiles")
	}
	if !plan.contentChanged {
		t.Error("contentChanged should be true")
	}

	// Verify shouldRenderPage returns true for affected pages.
	pageA := &site.Page{SourcePath: "page-a.md"}
	if !plan.shouldRenderPage(pageA, "/some/existing/output.html") {
		t.Error("shouldRenderPage should return true for page-a.md (in changedFiles)")
	}

	// An unaffected page should not be in changedFiles.
	if plan.changedFiles["unchanged.md"] {
		t.Error("unchanged.md should not be in changedFiles")
	}
}

// ---------------------------------------------------------------------------
// isNilInterface
// ---------------------------------------------------------------------------

func TestIsNilInterface(t *testing.T) {
	var nilPtr *site.Page
	var nilMap map[string]any
	var nilSlice []string
	var nilFunc func()
	var nilChan chan int

	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"untyped nil", nil, true},
		{"nil pointer", nilPtr, true},
		{"nil map", nilMap, true},
		{"nil slice", nilSlice, true},
		{"nil func", nilFunc, true},
		{"nil chan", nilChan, true},
		{"non-nil pointer", &site.Page{}, false},
		{"non-nil map", map[string]any{"a": 1}, false},
		{"non-nil slice", []string{"a"}, false},
		{"string value", "hello", false},
		{"int value", 42, false},
		{"bool value", false, false},
		{"struct value", site.Page{Title: "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNilInterface(tt.val)
			if got != tt.want {
				t.Errorf("isNilInterface(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// defaultLangFromCtx
// ---------------------------------------------------------------------------

func TestDefaultLangFromCtx(t *testing.T) {
	tests := []struct {
		name string
		ctx  map[string]any
		want string
	}{
		{"nil map", nil, "es"},
		{"empty map", map[string]any{}, "es"},
		{"no config key", map[string]any{"foo": "bar"}, "es"},
		{"config is not a map", map[string]any{"config": "string"}, "es"},
		{"config has no default_language", map[string]any{
			"config": map[string]any{"theme": "default"},
		}, "es"},
		{"default_language is empty string", map[string]any{
			"config": map[string]any{"default_language": ""},
		}, "es"},
		{"default_language is int", map[string]any{
			"config": map[string]any{"default_language": 42},
		}, "es"},
		{"default_language is en", map[string]any{
			"config": map[string]any{"default_language": "en"},
		}, "en"},
		{"default_language is fr", map[string]any{
			"config": map[string]any{"default_language": "fr"},
		}, "fr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultLangFromCtx(tt.ctx)
			if got != tt.want {
				t.Errorf("defaultLangFromCtx() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// themeI18nDir
// ---------------------------------------------------------------------------

func TestThemeI18nDir(t *testing.T) {
	tests := []struct {
		name      string
		themesDir string
		theme     string
		want      string
	}{
		{"both set", "themes", "default", "themes/default/i18n"},
		{"empty themes dir", "", "default", ""},
		{"whitespace themes dir", "  ", "default", ""},
		{"empty theme", "themes", "", ""},
		{"whitespace theme", "themes", "  ", ""},
		{"both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{ThemesDir: tt.themesDir, Theme: tt.theme}
			got := themeI18nDir(cfg)
			if got != tt.want {
				t.Errorf("themeI18nDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// aiCachePath
// ---------------------------------------------------------------------------

func TestAICachePath(t *testing.T) {
	tests := []struct {
		name          string
		buildCacheDir string
		want          string
	}{
		{"custom dir", "/tmp/cache", filepath.Join("/tmp/cache", "ai-summaries.json")},
		{"empty uses default", "", filepath.Join(".osg/cache", "ai-summaries.json")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{BuildCacheDir: tt.buildCacheDir}
			got := aiCachePath(cfg)
			if got != tt.want {
				t.Errorf("aiCachePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hashConfig
// ---------------------------------------------------------------------------

func TestHashConfig(t *testing.T) {
	cfg1 := config.Config{BaseURL: "http://example.com", Theme: "default"}
	cfg2 := config.Config{BaseURL: "http://example.com", Theme: "default"}
	cfg3 := config.Config{BaseURL: "http://other.com", Theme: "default"}

	h1, err := hashConfig(cfg1)
	if err != nil {
		t.Fatalf("hashConfig(cfg1) error: %v", err)
	}
	h2, err := hashConfig(cfg2)
	if err != nil {
		t.Fatalf("hashConfig(cfg2) error: %v", err)
	}
	h3, err := hashConfig(cfg3)
	if err != nil {
		t.Fatalf("hashConfig(cfg3) error: %v", err)
	}

	if h1 != h2 {
		t.Error("identical configs should produce the same hash")
	}
	if h1 == h3 {
		t.Error("different configs should produce different hashes")
	}
	if h1 == "" {
		t.Error("hash should not be empty")
	}
}

// ---------------------------------------------------------------------------
// pageContext
// ---------------------------------------------------------------------------

func TestPageContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
		"site":   map[string]any{},
	}
	page := &site.Page{
		Title:     "Hello",
		Path:      "/blog/hello/",
		Permalink: "http://example.com/blog/hello/",
		Lang:      "fr",
	}

	ctx := pageContext(base, page)

	if ctx["current_path"] != "/blog/hello/" {
		t.Errorf("current_path = %v, want /blog/hello/", ctx["current_path"])
	}
	if ctx["current_url"] != "http://example.com/blog/hello/" {
		t.Errorf("current_url = %v, want page permalink", ctx["current_url"])
	}
	if ctx["lang"] != "fr" {
		t.Errorf("lang = %v, want fr (from page)", ctx["lang"])
	}

	// page without Lang should fall back to config default_language
	page2 := &site.Page{Path: "/x/", Permalink: "http://example.com/x/"}
	ctx2 := pageContext(base, page2)
	if ctx2["lang"] != "en" {
		t.Errorf("lang = %v, want en (fallback)", ctx2["lang"])
	}

	// base context should not be mutated
	if _, exists := base["page"]; exists {
		t.Error("pageContext should not mutate base context")
	}
}

// ---------------------------------------------------------------------------
// sectionContext
// ---------------------------------------------------------------------------

func TestSectionContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "de"},
		"site":   map[string]any{},
	}
	section := &site.Section{
		Title:     "Blog",
		Path:      "/blog/",
		Permalink: "http://example.com/blog/",
	}

	ctx := sectionContext(base, section)

	if ctx["current_path"] != "/blog/" {
		t.Errorf("current_path = %v, want /blog/", ctx["current_path"])
	}
	if ctx["current_url"] != "http://example.com/blog/" {
		t.Errorf("current_url = %v", ctx["current_url"])
	}
	if ctx["lang"] != "de" {
		t.Errorf("lang = %v, want de", ctx["lang"])
	}
	if ctx["section"] == nil {
		t.Error("section key should be present")
	}
	if _, exists := base["section"]; exists {
		t.Error("sectionContext should not mutate base context")
	}
}

// ---------------------------------------------------------------------------
// baseContext
// ---------------------------------------------------------------------------

func TestBaseContext(t *testing.T) {
	cfg := config.Config{
		BaseURL:   "http://example.com",
		SiteTitle: "My Site",
	}
	siteView := map[string]any{"pages": []any{}}

	t.Run("no taxonomies no menu", func(t *testing.T) {
		ctx := baseContext(cfg, siteView, nil, nil)
		if ctx["config"] == nil {
			t.Error("config key should be present")
		}
		if ctx["site"] == nil {
			t.Error("site key should be present")
		}
		if _, exists := ctx["taxonomies"]; exists {
			t.Error("taxonomies should not be present when indices is nil")
		}
		if _, exists := ctx["menu_pages"]; exists {
			t.Error("menu_pages should not be present when empty")
		}
	})

	t.Run("with taxonomies", func(t *testing.T) {
		idx := &taxonomy.Index{
			Config: config.TaxonomyConfig{Name: "tags", Render: true},
			Terms:  map[string]*taxonomy.Term{},
		}
		indices := map[string]*taxonomy.Index{"tags": idx}
		ctx := baseContext(cfg, siteView, indices, nil)
		if ctx["taxonomies"] == nil {
			t.Error("taxonomies key should be present when indices provided")
		}
	})

	t.Run("with menu pages", func(t *testing.T) {
		menuPages := []*site.Page{
			{Title: "About", Path: "/about/"},
		}
		ctx := baseContext(cfg, siteView, nil, menuPages)
		mp, ok := ctx["menu_pages"].([]map[string]any)
		if !ok {
			t.Fatal("menu_pages should be a []map[string]any")
		}
		if len(mp) != 1 {
			t.Errorf("menu_pages len = %d, want 1", len(mp))
		}
	})
}

// ---------------------------------------------------------------------------
// buildOutputsIndex
// ---------------------------------------------------------------------------

func TestBuildOutputsIndex(t *testing.T) {
	t.Run("nil site returns nil", func(t *testing.T) {
		got := buildOutputsIndex(nil, "public")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("maps pages and sections", func(t *testing.T) {
		s := &site.Site{
			Pages: []*site.Page{
				{SourcePath: "content/blog/hello.md", Path: "/blog/hello/"},
				{SourcePath: "", Path: "/no-source/"},
			},
			Sections: map[string]*site.Section{
				"blog":  {SourcePath: "content/blog/_index.md", Path: "/blog/"},
				"empty": {SourcePath: "", Path: "/empty/"},
			},
		}
		out := buildOutputsIndex(s, "public")
		if len(out) != 2 {
			t.Errorf("expected 2 entries, got %d", len(out))
		}
		if _, ok := out["content/blog/hello.md"]; !ok {
			t.Error("missing page entry")
		}
		if _, ok := out["content/blog/_index.md"]; !ok {
			t.Error("missing section entry")
		}
	})
}

// ---------------------------------------------------------------------------
// taxonomiesView
// ---------------------------------------------------------------------------

func TestTaxonomiesView(t *testing.T) {
	t.Run("empty indices", func(t *testing.T) {
		out := taxonomiesView(map[string]*taxonomy.Index{})
		if len(out) != 0 {
			t.Errorf("expected empty map, got %d entries", len(out))
		}
	})

	t.Run("non-empty", func(t *testing.T) {
		idx := &taxonomy.Index{
			Config: config.TaxonomyConfig{Name: "tags"},
			Terms: map[string]*taxonomy.Term{
				"go": {
					Name: "go",
					Slug: "go",
					Path: "/tags/go/",
				},
			},
		}
		out := taxonomiesView(map[string]*taxonomy.Index{"tags": idx})
		if out["tags"] == nil {
			t.Error("tags key should be present")
		}
		tv, ok := out["tags"].(map[string]any)
		if !ok {
			t.Fatal("tags value should be a map[string]any")
		}
		if tv["taxonomy"] == nil {
			t.Error("taxonomy key should be present")
		}
		if tv["terms"] == nil {
			t.Error("terms key should be present")
		}
	})
}

// ---------------------------------------------------------------------------
// taxonomyListContext
// ---------------------------------------------------------------------------

func TestTaxonomyListContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
	}
	taxCfg := config.TaxonomyConfig{Name: "tags"}
	cfg := config.Config{BaseURL: "http://example.com"}
	terms := []*taxonomy.Term{
		{Name: "go", Slug: "go", Path: "/tags/go/"},
	}

	ctx := taxonomyListContext(base, cfg, taxCfg, terms, "/tags/")

	if ctx["current_path"] != "/tags/" {
		t.Errorf("current_path = %v", ctx["current_path"])
	}
	if ctx["current_url"] != "http://example.com/tags/" {
		t.Errorf("current_url = %v", ctx["current_url"])
	}
	if ctx["lang"] != "en" {
		t.Errorf("lang = %v", ctx["lang"])
	}
	if ctx["taxonomy"] == nil {
		t.Error("taxonomy should be present")
	}
	if ctx["terms"] == nil {
		t.Error("terms should be present")
	}
}

// ---------------------------------------------------------------------------
// taxonomyTermContext
// ---------------------------------------------------------------------------

func TestTaxonomyTermContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
	}
	taxCfg := config.TaxonomyConfig{Name: "tags"}
	cfg := config.Config{BaseURL: "http://example.com"}
	term := &taxonomy.Term{Name: "go", Slug: "go", Path: "/tags/go/"}

	t.Run("without paginator", func(t *testing.T) {
		ctx := taxonomyTermContext(base, cfg, taxCfg, term, "/tags/go/", nil)
		if ctx["term"] == nil {
			t.Error("term should be present")
		}
		if _, ok := ctx["paginator"]; ok {
			t.Error("paginator should not be present when nil")
		}
		if ctx["lang"] != "en" {
			t.Errorf("lang = %v", ctx["lang"])
		}
	})

	t.Run("with paginator", func(t *testing.T) {
		pag := &taxonomy.Paginator{
			CurrentIndex: 0,
			TotalPages:   3,
		}
		ctx := taxonomyTermContext(base, cfg, taxCfg, term, "/tags/go/", pag)
		if ctx["paginator"] == nil {
			t.Error("paginator should be present")
		}
	})
}

// ---------------------------------------------------------------------------
// feedContext
// ---------------------------------------------------------------------------

func TestFeedContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
	}
	taxCfg := config.TaxonomyConfig{Name: "tags"}
	cfg := config.Config{BaseURL: "http://example.com"}
	now := time.Now()
	term := &taxonomy.Term{
		Name:  "go",
		Slug:  "go",
		Path:  "/tags/go/",
		Pages: []*site.Page{{Title: "Post", Date: now}},
	}

	ctx := feedContext(base, cfg, taxCfg, term, "http://example.com/tags/go/atom.xml", now)

	if ctx["feed_url"] != "http://example.com/tags/go/atom.xml" {
		t.Errorf("feed_url = %v", ctx["feed_url"])
	}
	if ctx["last_updated"] != now.Format(time.RFC3339) {
		t.Errorf("last_updated = %v", ctx["last_updated"])
	}
	if ctx["lang"] != "en" {
		t.Errorf("lang = %v", ctx["lang"])
	}
	if ctx["taxonomy"] == nil {
		t.Error("taxonomy should be present")
	}
	if ctx["term"] == nil {
		t.Error("term should be present")
	}
	pages, ok := ctx["pages"].([]map[string]any)
	if !ok || len(pages) != 1 {
		t.Error("pages should have 1 entry")
	}
}

// ---------------------------------------------------------------------------
// siteFeedContext
// ---------------------------------------------------------------------------

func TestSiteFeedContext(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
	}
	cfg := config.Config{
		SiteTitle:       "My Blog",
		SiteDescription: "A test blog",
		BaseURL:         "http://example.com",
	}
	now := time.Now()
	pages := []*site.Page{
		{Title: "Post 1", Date: now, Permalink: "http://example.com/post-1/"},
	}

	ctx := siteFeedContext(base, cfg, pages, "http://example.com/atom.xml", now)

	if ctx["feed_title"] != "My Blog" {
		t.Errorf("feed_title = %v", ctx["feed_title"])
	}
	if ctx["feed_description"] != "A test blog" {
		t.Errorf("feed_description = %v", ctx["feed_description"])
	}
	if ctx["feed_url"] != "http://example.com/atom.xml" {
		t.Errorf("feed_url = %v", ctx["feed_url"])
	}
	if ctx["lang"] != "en" {
		t.Errorf("lang = %v", ctx["lang"])
	}
	fp, ok := ctx["pages"].([]map[string]any)
	if !ok || len(fp) != 1 {
		t.Error("pages should have 1 entry")
	}
}

// ---------------------------------------------------------------------------
// sectionUpdated
// ---------------------------------------------------------------------------

func TestSectionUpdated(t *testing.T) {
	t.Run("empty section returns now-ish", func(t *testing.T) {
		s := &site.Section{}
		got := sectionUpdated(s)
		// Should return time.Now() for empty sections; just check it's recent
		if time.Since(got) > 2*time.Second {
			t.Errorf("expected recent time for empty section, got %v", got)
		}
	})

	t.Run("picks latest page date", func(t *testing.T) {
		old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		recent := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		s := &site.Section{
			Pages: []*site.Page{
				{Title: "Old", Date: old},
				{Title: "Recent", Date: recent},
			},
		}
		got := sectionUpdated(s)
		if !got.Equal(recent) {
			t.Errorf("got %v, want %v", got, recent)
		}
	})

	t.Run("recurses into subsections", func(t *testing.T) {
		old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		deep := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		s := &site.Section{
			Pages: []*site.Page{{Date: old}},
			Subsections: []*site.Section{
				{Pages: []*site.Page{{Date: deep}}},
			},
		}
		got := sectionUpdated(s)
		if !got.Equal(deep) {
			t.Errorf("got %v, want %v", got, deep)
		}
	})
}

// ---------------------------------------------------------------------------
// taxonomyIndexUpdated
// ---------------------------------------------------------------------------

func TestTaxonomyIndexUpdated(t *testing.T) {
	t.Run("empty index returns now-ish", func(t *testing.T) {
		idx := &taxonomy.Index{Terms: map[string]*taxonomy.Term{}}
		got := taxonomyIndexUpdated(idx)
		if time.Since(got) > 2*time.Second {
			t.Errorf("expected recent time for empty index, got %v", got)
		}
	})

	t.Run("picks latest across terms", func(t *testing.T) {
		d1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		d2 := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		idx := &taxonomy.Index{
			Terms: map[string]*taxonomy.Term{
				"go":   {Pages: []*site.Page{{Date: d1}}},
				"rust": {Pages: []*site.Page{{Date: d2}}},
			},
		}
		got := taxonomyIndexUpdated(idx)
		if !got.Equal(d2) {
			t.Errorf("got %v, want %v", got, d2)
		}
	})
}

// ---------------------------------------------------------------------------
// collectSitemapEntries
// ---------------------------------------------------------------------------

func TestCollectSitemapEntries(t *testing.T) {
	now := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	t.Run("pages and sections", func(t *testing.T) {
		cfg := config.Config{BaseURL: "http://example.com"}
		s := &site.Site{
			Pages: []*site.Page{
				{Permalink: "http://example.com/hello/", Date: now},
				{Permalink: "http://example.com/world/", Date: now},
			},
			Sections: map[string]*site.Section{
				"blog": {
					Permalink: "http://example.com/blog/",
					Pages:     []*site.Page{{Date: now}},
				},
			},
		}
		entries := collectSitemapEntries(cfg, s, nil)
		if len(entries) != 3 {
			t.Errorf("expected 3 entries, got %d", len(entries))
		}
	})

	t.Run("deduplicates by permalink", func(t *testing.T) {
		cfg := config.Config{BaseURL: "http://example.com"}
		s := &site.Site{
			Pages: []*site.Page{
				{Permalink: "http://example.com/same/", Date: now},
				{Permalink: "http://example.com/same/", Date: now},
			},
		}
		entries := collectSitemapEntries(cfg, s, nil)
		if len(entries) != 1 {
			t.Errorf("expected 1 deduplicated entry, got %d", len(entries))
		}
	})

	t.Run("skips empty permalinks", func(t *testing.T) {
		cfg := config.Config{BaseURL: "http://example.com"}
		s := &site.Site{
			Pages: []*site.Page{
				{Permalink: "", Date: now},
				{Permalink: "  ", Date: now},
			},
		}
		entries := collectSitemapEntries(cfg, s, nil)
		if len(entries) != 0 {
			t.Errorf("expected 0 entries for empty permalinks, got %d", len(entries))
		}
	})

	t.Run("includes taxonomy entries", func(t *testing.T) {
		cfg := config.Config{
			BaseURL: "http://example.com",
			Taxonomies: []config.TaxonomyConfig{
				{Name: "tags", Render: true},
			},
		}
		s := &site.Site{}
		idx := &taxonomy.Index{
			Config: config.TaxonomyConfig{Name: "tags", Render: true},
			Terms: map[string]*taxonomy.Term{
				"go": {
					Name:      "go",
					Slug:      "go",
					Path:      "/tags/go/",
					Permalink: "http://example.com/tags/go/",
					Pages:     []*site.Page{{Date: now}},
				},
			},
		}
		indices := map[string]*taxonomy.Index{"tags": idx}
		entries := collectSitemapEntries(cfg, s, indices)
		// Should have: /tags/ (list) + /tags/go/ (term) = 2
		if len(entries) < 2 {
			t.Errorf("expected at least 2 entries (list + term), got %d", len(entries))
		}
	})

	t.Run("skips non-render taxonomies", func(t *testing.T) {
		cfg := config.Config{
			BaseURL: "http://example.com",
			Taxonomies: []config.TaxonomyConfig{
				{Name: "tags", Render: false},
			},
		}
		s := &site.Site{}
		idx := &taxonomy.Index{
			Config: config.TaxonomyConfig{Name: "tags", Render: false},
			Terms: map[string]*taxonomy.Term{
				"go": {Name: "go", Pages: []*site.Page{{Date: now}}},
			},
		}
		indices := map[string]*taxonomy.Index{"tags": idx}
		entries := collectSitemapEntries(cfg, s, indices)
		if len(entries) != 0 {
			t.Errorf("expected 0 entries when render=false, got %d", len(entries))
		}
	})
}

// ---------------------------------------------------------------------------
// includeAllFiles (cache.go) — uses fakeDirEntry from cache_test.go
// ---------------------------------------------------------------------------

func TestIncludeAllFiles(t *testing.T) {
	// includeAllFiles returns !d.IsDir(), so files pass, dirs don't
	file := fakeDirEntry{name: "test.txt", isDir: false}
	dir := fakeDirEntry{name: "subdir", isDir: true}

	if !includeAllFiles("test.txt", file) {
		t.Error("includeAllFiles should return true for files")
	}
	if includeAllFiles("subdir", dir) {
		t.Error("includeAllFiles should return false for directories")
	}
}

// ---------------------------------------------------------------------------
// relatedPages
// ---------------------------------------------------------------------------

func TestRelatedPages_NoTaxonomies(t *testing.T) {
	page := &site.Page{Title: "No Tags"}
	result := relatedPages(page, nil, 3)
	if result != nil {
		t.Errorf("expected nil for page with no taxonomies, got %v", result)
	}
}

func TestRelatedPages_NoIndices(t *testing.T) {
	page := &site.Page{
		Title:      "Has Tags",
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	result := relatedPages(page, map[string]*taxonomy.Index{}, 3)
	if result != nil {
		t.Errorf("expected nil for empty indices, got %v", result)
	}
}

func TestRelatedPages_SharedTag(t *testing.T) {
	now := time.Now()
	pageA := &site.Page{
		Title:      "A",
		Path:       "/a/",
		Date:       now,
		Taxonomies: map[string][]string{"tags": {"go", "web"}},
	}
	pageB := &site.Page{
		Title:      "B",
		Path:       "/b/",
		Date:       now.Add(-1 * time.Hour),
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	pageC := &site.Page{
		Title:      "C",
		Path:       "/c/",
		Date:       now.Add(-2 * time.Hour),
		Taxonomies: map[string][]string{"tags": {"go", "web"}},
	}
	pageUnrelated := &site.Page{
		Title:      "Unrelated",
		Path:       "/unrelated/",
		Date:       now,
		Taxonomies: map[string][]string{"tags": {"python"}},
	}

	allPages := []*site.Page{pageA, pageB, pageC, pageUnrelated}
	indices := taxonomy.Build(
		[]config.TaxonomyConfig{{Name: "tags", Render: true}},
		allPages, "",
	)

	related := relatedPages(pageA, indices, 3)
	if len(related) != 2 {
		t.Fatalf("expected 2 related pages, got %d", len(related))
	}
	// pageC shares 2 tags with A (go+web), pageB shares 1 (go).
	// pageC should come first due to higher score.
	if related[0] != pageC {
		t.Errorf("expected pageC first (higher score), got %s", related[0].Title)
	}
	if related[1] != pageB {
		t.Errorf("expected pageB second, got %s", related[1].Title)
	}
}

func TestRelatedPages_ExcludesSelf(t *testing.T) {
	now := time.Now()
	pageA := &site.Page{
		Title:      "A",
		Path:       "/a/",
		Date:       now,
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	indices := taxonomy.Build(
		[]config.TaxonomyConfig{{Name: "tags", Render: true}},
		[]*site.Page{pageA}, "",
	)
	related := relatedPages(pageA, indices, 3)
	if len(related) != 0 {
		t.Errorf("should not include self, got %d related", len(related))
	}
}

func TestRelatedPages_ExcludesMenuPages(t *testing.T) {
	now := time.Now()
	pageA := &site.Page{
		Title:      "A",
		Path:       "/a/",
		Date:       now,
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	menuPage := &site.Page{
		Title:      "About",
		Path:       "/about/",
		Date:       now,
		Menu:       true,
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	indices := taxonomy.Build(
		[]config.TaxonomyConfig{{Name: "tags", Render: true}},
		[]*site.Page{pageA, menuPage}, "",
	)
	related := relatedPages(pageA, indices, 3)
	if len(related) != 0 {
		t.Errorf("should exclude menu pages, got %d related", len(related))
	}
}

func TestRelatedPages_RespectsLimit(t *testing.T) {
	now := time.Now()
	pageA := &site.Page{
		Title:      "A",
		Path:       "/a/",
		Date:       now,
		Taxonomies: map[string][]string{"tags": {"go"}},
	}
	pages := []*site.Page{pageA}
	for i := 0; i < 10; i++ {
		pages = append(pages, &site.Page{
			Title:      fmt.Sprintf("P%d", i),
			Path:       fmt.Sprintf("/p%d/", i),
			Date:       now.Add(time.Duration(-i) * time.Hour),
			Taxonomies: map[string][]string{"tags": {"go"}},
		})
	}
	indices := taxonomy.Build(
		[]config.TaxonomyConfig{{Name: "tags", Render: true}},
		pages, "",
	)
	related := relatedPages(pageA, indices, 3)
	if len(related) != 3 {
		t.Errorf("expected limit of 3, got %d", len(related))
	}
}
