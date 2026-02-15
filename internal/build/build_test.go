package build

import (
	"path/filepath"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/site"
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
