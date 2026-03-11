package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/i18n"
	"osg/internal/render"
	"osg/internal/site"
	"osg/internal/summary"
	"osg/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// cleanupRemovedOutputs
// ---------------------------------------------------------------------------

func TestCleanupRemovedOutputs_EmptyInputs(t *testing.T) {
	t.Run("empty removed list", func(t *testing.T) {
		count := cleanupRemovedOutputs("/public", nil, map[string]string{"a": "b"}, nil)
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	t.Run("empty outputs map", func(t *testing.T) {
		count := cleanupRemovedOutputs("/public", []string{"a.md"}, nil, nil)
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	t.Run("both empty", func(t *testing.T) {
		count := cleanupRemovedOutputs("/public", nil, nil, nil)
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})
}

func TestCleanupRemovedOutputs_RemovesFiles(t *testing.T) {
	publicDir := t.TempDir()
	// Create a nested output file.
	outDir := filepath.Join(publicDir, "blog", "hello")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(outFile, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed := []string{"content/blog/hello.md"}
	outputs := map[string]string{
		"content/blog/hello.md": outFile,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	count := cleanupRemovedOutputs(publicDir, removed, outputs, logger)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
	// File should be gone.
	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		t.Error("output file should have been removed")
	}
}

func TestCleanupRemovedOutputs_SkipsOutsidePublicDir(t *testing.T) {
	publicDir := t.TempDir()
	otherDir := t.TempDir()
	otherFile := filepath.Join(otherDir, "index.html")
	if err := os.WriteFile(otherFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed := []string{"outside.md"}
	outputs := map[string]string{
		"outside.md": otherFile,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	count := cleanupRemovedOutputs(publicDir, removed, outputs, logger)
	if count != 0 {
		t.Errorf("expected 0 (outside public dir), got %d", count)
	}
	// File should still exist.
	if _, err := os.Stat(otherFile); err != nil {
		t.Error("file outside public dir should not have been removed")
	}
}

func TestCleanupRemovedOutputs_MissingOutputKey(t *testing.T) {
	publicDir := t.TempDir()
	removed := []string{"unknown.md"}
	outputs := map[string]string{
		"other.md": filepath.Join(publicDir, "other", "index.html"),
	}
	count := cleanupRemovedOutputs(publicDir, removed, outputs, nil)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCleanupRemovedOutputs_EmptyOutputValue(t *testing.T) {
	publicDir := t.TempDir()
	removed := []string{"a.md"}
	outputs := map[string]string{"a.md": ""}
	count := cleanupRemovedOutputs(publicDir, removed, outputs, nil)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// removeEmptyParents
// ---------------------------------------------------------------------------

func TestRemoveEmptyParents(t *testing.T) {
	t.Run("removes empty parent directories up to root", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "a", "b", "c")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		leaf := filepath.Join(nested, "index.html")
		if err := os.WriteFile(leaf, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		// Remove the leaf file first (simulates os.Remove in cleanupRemovedOutputs).
		_ = os.Remove(leaf)
		removeEmptyParents(root, leaf)

		// c, b, a should all be removed since they were empty.
		for _, dir := range []string{
			filepath.Join(root, "a", "b", "c"),
			filepath.Join(root, "a", "b"),
			filepath.Join(root, "a"),
		} {
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Errorf("directory %s should have been removed", dir)
			}
		}
		// root should still exist.
		if _, err := os.Stat(root); err != nil {
			t.Error("root should still exist")
		}
	})

	t.Run("stops when directory is not empty", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		// Put a file in "a" so it's not empty after "b" is removed.
		keepFile := filepath.Join(root, "a", "keep.txt")
		if err := os.WriteFile(keepFile, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
		leaf := filepath.Join(nested, "index.html")
		if err := os.WriteFile(leaf, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(leaf)
		removeEmptyParents(root, leaf)

		// "b" should be removed (empty), "a" should remain (has keep.txt).
		if _, err := os.Stat(filepath.Join(root, "a", "b")); !os.IsNotExist(err) {
			t.Error("directory 'b' should have been removed")
		}
		if _, err := os.Stat(filepath.Join(root, "a")); err != nil {
			t.Error("directory 'a' should still exist (has keep.txt)")
		}
	})

	t.Run("root equals leaf parent does nothing", func(t *testing.T) {
		root := t.TempDir()
		leaf := filepath.Join(root, "index.html")
		// This should not panic or remove root.
		removeEmptyParents(root, leaf)
		if _, err := os.Stat(root); err != nil {
			t.Error("root should still exist")
		}
	})
}

// ---------------------------------------------------------------------------
// baseContext — nav_taxonomy branch
// ---------------------------------------------------------------------------

func TestBaseContext_NavTaxonomy(t *testing.T) {
	cfg := config.Config{
		BaseURL:     "http://example.com",
		NavTaxonomy: "categories",
	}
	siteView := map[string]any{"pages": []any{}}

	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "categories", Render: true},
		Terms: map[string]*taxonomy.Term{
			"tech": {
				Name:      "tech",
				Slug:      "tech",
				Path:      "/categories/tech/",
				Permalink: "http://example.com/categories/tech/",
			},
		},
	}
	indices := map[string]*taxonomy.Index{"categories": idx}

	ctx := baseContext(cfg, siteView, indices, nil)

	navTerms, ok := ctx["nav_terms"].([]map[string]any)
	if !ok {
		t.Fatal("nav_terms should be []map[string]any")
	}
	if len(navTerms) != 1 {
		t.Fatalf("expected 1 nav term, got %d", len(navTerms))
	}
	if navTerms[0]["name"] != "tech" {
		t.Errorf("nav_terms[0].name = %v, want tech", navTerms[0]["name"])
	}
	if navTerms[0]["permalink"] != "http://example.com/categories/tech/" {
		t.Errorf("nav_terms[0].permalink = %v", navTerms[0]["permalink"])
	}
}

func TestBaseContext_NavTaxonomy_Missing(t *testing.T) {
	cfg := config.Config{
		BaseURL:     "http://example.com",
		NavTaxonomy: "nonexistent",
	}
	siteView := map[string]any{}
	indices := map[string]*taxonomy.Index{
		"tags": {Config: config.TaxonomyConfig{Name: "tags"}, Terms: map[string]*taxonomy.Term{}},
	}
	ctx := baseContext(cfg, siteView, indices, nil)
	if _, exists := ctx["nav_terms"]; exists {
		t.Error("nav_terms should not be present when nav_taxonomy does not match any index")
	}
}

func TestBaseContext_NavTaxonomy_Empty(t *testing.T) {
	cfg := config.Config{BaseURL: "http://example.com", NavTaxonomy: ""}
	siteView := map[string]any{}
	ctx := baseContext(cfg, siteView, nil, nil)
	if _, exists := ctx["nav_terms"]; exists {
		t.Error("nav_terms should not be present when nav_taxonomy is empty")
	}
}

func TestBaseContext_CurrentYear(t *testing.T) {
	cfg := config.Config{}
	siteView := map[string]any{}
	ctx := baseContext(cfg, siteView, nil, nil)
	year, ok := ctx["current_year"].(int)
	if !ok {
		t.Fatal("current_year should be an int")
	}
	if year != time.Now().Year() {
		t.Errorf("current_year = %d, want %d", year, time.Now().Year())
	}
}

// ---------------------------------------------------------------------------
// configView — comprehensive field check
// ---------------------------------------------------------------------------

func TestConfigView_AllFields(t *testing.T) {
	cfg := config.Config{
		BaseURL:           "https://site.com",
		SiteTitle:         "Title",
		SiteDescription:   "Desc",
		Theme:             "mytheme",
		Logo:              "/logo.png",
		Favicon:           "/favicon.ico",
		ColorScheme:       "dark",
		DefaultLanguage:   "en",
		VaultPath:         "/vault",
		ContentDir:        "content",
		PublicDir:         "public",
		TemplatesDir:      "templates",
		StaticDir:         "static",
		ThemesDir:         "themes",
		PluginsDir:        "plugins",
		PluginsEnabled:    []string{"a", "b"},
		SassDir:           "sass",
		ContentLayout:     "date",
		IncludeDrafts:     true,
		CompileSass:       true,
		TUIPrefix:         ">",
		TUIPrefixMs:       100,
		ServeWatch:        true,
		ServeReload:       true,
		ServeDebounce:     200,
		BuildIncremental:  true,
		BuildCacheDir:     ".cache",
		DoctorProfile:     "strict",
		SummaryStrategy:   "auto",
		SiteFeed:          true,
		SiteFeedLimit:     20,
		ImageOptimization: true,
		ImageQuality:      85,
		ImageWidths:       []int{640, 1200},
		Lightbox:          true,
		Sharing:           true,
		Minify:            true,
		NavTaxonomy:       "tags",
		Social:            map[string]string{"twitter": "@me"},
		Copyright:         "Copyright {year}",
		Logging:           config.LoggingConfig{Level: "info", Format: "json"},
		Taxonomies:        []config.TaxonomyConfig{{Name: "tags", Render: true}},
		Interactions: config.InteractionsConfig{
			Enabled: true,
			APIURL:  "http://api.example.com",
			Comments: config.CommentsConfig{
				Enabled: true,
				Providers: []config.AuthProviderConfig{
					{Provider: "github"},
				},
			},
		},
	}

	view := configView(cfg)

	checks := map[string]any{
		"base_url":           "https://site.com",
		"site_title":         "Title",
		"site_description":   "Desc",
		"theme":              "mytheme",
		"logo":               "/logo.png",
		"favicon":            "/favicon.ico",
		"color_scheme":       "dark",
		"default_language":   "en",
		"vault_path":         "/vault",
		"content_dir":        "content",
		"templates_dir":      "templates",
		"static_dir":         "static",
		"themes_dir":         "themes",
		"plugins_dir":        "plugins",
		"sass_dir":           "sass",
		"content_layout":     "date",
		"include_drafts":     true,
		"compile_sass":       true,
		"tui_prefix":         ">",
		"tui_prefix_ms":      100,
		"serve_watch":        true,
		"serve_live_reload":  true,
		"serve_debounce_ms":  200,
		"build_incremental":  true,
		"build_cache_dir":    ".cache",
		"doctor_profile":     "strict",
		"summary_strategy":   "auto",
		"site_feed":          true,
		"site_feed_limit":    20,
		"image_optimization": true,
		"image_quality":      85,
		"lightbox":           true,
		"sharing":            true,
		"minify":             true,
		"nav_taxonomy":       "tags",
	}

	for key, want := range checks {
		got, exists := view[key]
		if !exists {
			t.Errorf("key %q missing from configView", key)
			continue
		}
		if got != want {
			t.Errorf("configView[%q] = %v (%T), want %v (%T)", key, got, got, want, want)
		}
	}

	// Copyright should have {year} replaced.
	copyright, ok := view["copyright"].(string)
	if !ok {
		t.Fatal("copyright should be a string")
	}
	if copyright == "Copyright {year}" {
		t.Error("copyright should have {year} replaced with actual year")
	}

	// multilingual should be false (no Languages set).
	if view["multilingual"] != false {
		t.Errorf("multilingual = %v, want false", view["multilingual"])
	}

	// interactions fields.
	if view["interactions_enabled"] != true {
		t.Errorf("interactions_enabled = %v, want true", view["interactions_enabled"])
	}
	if view["interactions_api_url"] != "http://api.example.com" {
		t.Errorf("interactions_api_url = %v", view["interactions_api_url"])
	}
	if view["comments_enabled"] != true {
		t.Errorf("comments_enabled = %v, want true", view["comments_enabled"])
	}

	// taxonomies list.
	taxList, ok := view["taxonomies"].([]map[string]any)
	if !ok {
		t.Fatal("taxonomies should be []map[string]any")
	}
	if len(taxList) != 1 {
		t.Errorf("taxonomies len = %d, want 1", len(taxList))
	}

	// languages.
	langs, ok := view["languages"].([]map[string]any)
	if !ok {
		t.Fatal("languages should be []map[string]any")
	}
	if len(langs) != 1 { // just default language
		t.Errorf("languages len = %d, want 1", len(langs))
	}

	// public_dir should be absolute.
	pubDir, ok := view["public_dir"].(string)
	if !ok || pubDir == "" {
		t.Error("public_dir should be a non-empty string")
	}
}

// ---------------------------------------------------------------------------
// commentsProvidersView
// ---------------------------------------------------------------------------

func TestCommentsProvidersView_Empty(t *testing.T) {
	ccfg := config.CommentsConfig{}
	providers := commentsProvidersView(ccfg)
	if len(providers) != 0 {
		t.Errorf("expected empty, got %d providers", len(providers))
	}
}

func TestCommentsProvidersView_Github(t *testing.T) {
	ccfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github"},
		},
	}
	providers := commentsProvidersView(ccfg)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0]["provider"] != "github" {
		t.Errorf("provider = %v, want github", providers[0]["provider"])
	}
	if providers[0]["label"] != "GitHub" {
		t.Errorf("label = %v, want GitHub", providers[0]["label"])
	}
}

func TestCommentsProvidersView_Google(t *testing.T) {
	ccfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "google"},
		},
	}
	providers := commentsProvidersView(ccfg)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0]["label"] != "Google" {
		t.Errorf("label = %v, want Google", providers[0]["label"])
	}
}

func TestCommentsProvidersView_Unknown(t *testing.T) {
	ccfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "myauth"},
		},
	}
	providers := commentsProvidersView(ccfg)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	// Unknown providers use the raw provider name as the label.
	if providers[0]["label"] != "myauth" {
		t.Errorf("label = %v, want myauth", providers[0]["label"])
	}
}

func TestCommentsProvidersView_Multiple(t *testing.T) {
	ccfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github"},
			{Provider: "google"},
			{Provider: "custom"},
		},
	}
	providers := commentsProvidersView(ccfg)
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}
	labels := []string{"GitHub", "Google", "custom"}
	for i, want := range labels {
		if providers[i]["label"] != want {
			t.Errorf("providers[%d].label = %v, want %v", i, providers[i]["label"], want)
		}
	}
}

// ---------------------------------------------------------------------------
// applyPluginOverrides
// ---------------------------------------------------------------------------

func TestApplyPluginOverrides_NilPlugins(t *testing.T) {
	payload := map[string]any{"key": "value"}
	result := applyPluginOverrides(context.Background(), nil, "test.event", payload)
	if result["key"] != "value" {
		t.Errorf("expected payload unchanged, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// generatePlaceholders
// ---------------------------------------------------------------------------

func TestGeneratePlaceholders_PageWithImage(t *testing.T) {
	publicDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Has Image", Image: "/img/existing.jpg"})

	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Page with image should keep its original image.
	if siteIndex.Pages[0].Image != "/img/existing.jpg" {
		t.Errorf("image = %q, want /img/existing.jpg", siteIndex.Pages[0].Image)
	}
}

func TestGeneratePlaceholders_PageWithoutImage(t *testing.T) {
	publicDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "No Image"})

	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Page without image should get a placeholder.
	if siteIndex.Pages[0].Image == "" {
		t.Error("page without image should get a placeholder image path")
	}

	// The placeholder file should exist on disk.
	imgDir := filepath.Join(publicDir, "img")
	entries, err := os.ReadDir(imgDir)
	if err != nil {
		t.Fatalf("failed to read img dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 placeholder file, got %d", len(entries))
	}
}

func TestGeneratePlaceholders_SkipsExistingFile(t *testing.T) {
	publicDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "Test Page"})

	// First call generates the placeholder.
	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("first call: %v", err)
	}
	firstImage := siteIndex.Pages[0].Image

	// Reset and call again -- should skip generation but still set Image.
	siteIndex.Pages[0].Image = ""
	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if siteIndex.Pages[0].Image != firstImage {
		t.Errorf("second call image = %q, want %q", siteIndex.Pages[0].Image, firstImage)
	}
}

func TestGeneratePlaceholders_EmptySite(t *testing.T) {
	publicDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteIndex := site.New()

	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// fillSummaries -- non-AI strategies
// ---------------------------------------------------------------------------

func TestFillSummaries_AutoStrategy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Post",
		SourcePath: "post.md",
		RawContent: "This is the first sentence. And another sentence for good measure.",
	})

	cfg := config.Config{SummaryStrategy: "auto"}
	opts := BuildOptions{SkipAI: false}

	affected := fillSummaries(context.Background(), cfg, opts, siteIndex, logger)

	if len(affected) != 1 {
		t.Fatalf("expected 1 affected, got %d", len(affected))
	}
	if siteIndex.Pages[0].Summary == "" {
		t.Error("page summary should have been filled")
	}
}

func TestFillSummaries_ManualStrategy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Post",
		SourcePath: "post.md",
		RawContent: "Content that should not be summarized.",
	})

	cfg := config.Config{SummaryStrategy: "manual"}
	opts := BuildOptions{}

	affected := fillSummaries(context.Background(), cfg, opts, siteIndex, logger)

	if affected != nil {
		t.Errorf("manual strategy should return nil, got %v", affected)
	}
	if siteIndex.Pages[0].Summary != "" {
		t.Error("manual strategy should not set summary")
	}
}

func TestFillSummaries_EmptyStrategy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Post",
		SourcePath: "post.md",
		RawContent: "Some content here.",
	})

	cfg := config.Config{SummaryStrategy: ""}
	opts := BuildOptions{}

	// Should use the default provider from summary.NewProvider("").
	_ = fillSummaries(context.Background(), cfg, opts, siteIndex, logger)
	// No crash is success.
}

// ---------------------------------------------------------------------------
// fillWithProvider -- additional edge cases
// ---------------------------------------------------------------------------

func TestFillWithProvider_EmptySite(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteIndex := site.New()

	affected := fillWithProvider(context.Background(), siteIndex, summary.ExtractProvider{}, "auto", logger)

	if len(affected) != 0 {
		t.Errorf("expected 0 affected for empty site, got %d", len(affected))
	}
}

func TestFillWithProvider_AllHaveSummaries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "A", SourcePath: "a.md", Summary: "already"})
	siteIndex.AddPage(&site.Page{Title: "B", SourcePath: "b.md", Summary: "set"})

	affected := fillWithProvider(context.Background(), siteIndex, summary.ExtractProvider{}, "auto", logger)

	if len(affected) != 0 {
		t.Errorf("expected 0 affected, got %d", len(affected))
	}
}

// ---------------------------------------------------------------------------
// buildPlan.shouldRenderPage -- additional edge cases
// ---------------------------------------------------------------------------

func TestShouldRenderPage_NilChangedFiles(t *testing.T) {
	plan := buildPlan{
		incremental:  true,
		full:         false,
		changedFiles: nil,
	}
	page := &site.Page{SourcePath: "a.md"}
	// nil changedFiles map means the file is not in changedFiles,
	// so it falls through to outputMissing.
	got := plan.shouldRenderPage(page, "/nonexistent/path.html")
	if !got {
		t.Error("expected true for missing output even with nil changedFiles")
	}
}

// ---------------------------------------------------------------------------
// hashConfig -- stability and sensitivity
// ---------------------------------------------------------------------------

func TestHashConfig_Stability(t *testing.T) {
	cfg := config.Config{
		BaseURL:    "http://example.com",
		Theme:      "default",
		ContentDir: "content",
		PublicDir:  "public",
	}
	h1, err := hashConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hashConfig should be deterministic")
	}
}

func TestHashConfig_DifferentTaxonomies(t *testing.T) {
	cfg1 := config.Config{
		BaseURL: "http://example.com",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags"},
		},
	}
	cfg2 := config.Config{
		BaseURL: "http://example.com",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "categories"},
		},
	}
	h1, err := hashConfig(cfg1)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashConfig(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("configs with different taxonomies should have different hashes")
	}
}

func TestHashConfig_PluginsEnabled(t *testing.T) {
	cfg1 := config.Config{PluginsEnabled: []string{"a"}}
	cfg2 := config.Config{PluginsEnabled: []string{"b"}}
	h1, err := hashConfig(cfg1)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashConfig(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("configs with different plugins should have different hashes")
	}
}

// ---------------------------------------------------------------------------
// hashContent
// ---------------------------------------------------------------------------

func TestHashContent(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.md")
	f2 := filepath.Join(dir, "b.md")
	if err := os.WriteFile(f1, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	stamps, err := hashContent([]string{f1, f2})
	if err != nil {
		t.Fatal(err)
	}
	if len(stamps) != 2 {
		t.Errorf("expected 2 stamps, got %d", len(stamps))
	}
	if stamps[f1].Size != 5 {
		t.Errorf("f1 size = %d, want 5", stamps[f1].Size)
	}
	if stamps[f2].Size != 5 {
		t.Errorf("f2 size = %d, want 5", stamps[f2].Size)
	}
}

func TestHashContent_EmptyList(t *testing.T) {
	stamps, err := hashContent(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stamps) != 0 {
		t.Errorf("expected 0 stamps, got %d", len(stamps))
	}
}

func TestHashContent_NonexistentFile(t *testing.T) {
	_, err := hashContent([]string{"/nonexistent/file.md"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// hashDir
// ---------------------------------------------------------------------------

func TestHashDir_Empty(t *testing.T) {
	h, err := hashDir("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("expected empty string for empty root, got %q", h)
	}
}

func TestHashDir_Whitespace(t *testing.T) {
	h, err := hashDir("   ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("expected empty string for whitespace root, got %q", h)
	}
}

func TestHashDir_NonexistentDir(t *testing.T) {
	h, err := hashDir("/nonexistent/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("expected empty string for nonexistent dir, got %q", h)
	}
}

func TestHashDir_WithFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := hashDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Error("expected non-empty hash for dir with files")
	}
	if len(h) != 64 {
		t.Errorf("hash length = %d, want 64", len(h))
	}
}

func TestHashDir_Deterministic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := hashDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := hashDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hashDir should be deterministic")
	}
}

func TestHashDir_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("visible"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Hash with filter that accepts all files.
	h1, err := hashDir(dir, includeAllFiles)
	if err != nil {
		t.Fatal(err)
	}

	// Remove hidden dir -- hash should be the same since it was skipped.
	_ = os.RemoveAll(hiddenDir)
	h2, err := hashDir(dir, includeAllFiles)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hidden directories should be skipped")
	}
}

func TestHashDir_WithFilter(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "page.html"), []byte("<html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Hash with only template files.
	hTemplates, err := hashDir(dir, isTemplateFile)
	if err != nil {
		t.Fatal(err)
	}

	// Hash with all files.
	hAll, err := hashDir(dir, includeAllFiles)
	if err != nil {
		t.Fatal(err)
	}

	// Should differ because isTemplateFile excludes CSS.
	if hTemplates == hAll {
		t.Error("hashes should differ when filter excludes files")
	}
}

func TestHashDir_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "just-a-file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pointing to a regular file (not a dir) should return empty.
	h, err := hashDir(filePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("expected empty hash for a file (not dir), got %q", h)
	}
}

// ---------------------------------------------------------------------------
// loadBuildCache / saveBuildCache
// ---------------------------------------------------------------------------

func TestLoadBuildCache_EmptyPath(t *testing.T) {
	cache, err := loadBuildCache("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != nil {
		t.Error("expected nil cache for empty path")
	}
}

func TestLoadBuildCache_WhitespacePath(t *testing.T) {
	cache, err := loadBuildCache("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != nil {
		t.Error("expected nil cache for whitespace path")
	}
}

func TestLoadBuildCache_NonexistentFile(t *testing.T) {
	cache, err := loadBuildCache("/nonexistent/build.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != nil {
		t.Error("expected nil cache for nonexistent file")
	}
}

func TestSaveBuildCache_NilCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build.json")
	if err := saveBuildCache(path, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File should not be created.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("nil cache should not create a file")
	}
}

func TestSaveBuildCache_EmptyPath(t *testing.T) {
	cache := &buildCache{Version: buildCacheVersion}
	if err := saveBuildCache("", cache); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveLoadBuildCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build.json")

	original := &buildCache{
		Version:       buildCacheVersion,
		ConfigHash:    "abc123",
		TemplatesHash: "def456",
		AssetsHash:    "ghi789",
		PluginsHash:   "jkl012",
		Content: map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
		},
		Outputs: map[string]string{
			"a.md": "public/a/index.html",
		},
		GeneratedAt: "2025-01-01T00:00:00Z",
	}

	if err := saveBuildCache(path, original); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := loadBuildCache(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded cache should not be nil")
	}
	if loaded.Version != original.Version {
		t.Errorf("version = %d, want %d", loaded.Version, original.Version)
	}
	if loaded.ConfigHash != original.ConfigHash {
		t.Errorf("config_hash = %q, want %q", loaded.ConfigHash, original.ConfigHash)
	}
	if loaded.TemplatesHash != original.TemplatesHash {
		t.Errorf("templates_hash = %q", loaded.TemplatesHash)
	}
	if loaded.AssetsHash != original.AssetsHash {
		t.Errorf("assets_hash = %q", loaded.AssetsHash)
	}
	if loaded.PluginsHash != original.PluginsHash {
		t.Errorf("plugins_hash = %q", loaded.PluginsHash)
	}
	stamp, ok := loaded.Content["a.md"]
	if !ok || stamp.ModTime != 100 || stamp.Size != 10 {
		t.Errorf("content stamp = %+v", stamp)
	}
	if loaded.Outputs["a.md"] != "public/a/index.html" {
		t.Errorf("outputs = %v", loaded.Outputs)
	}
}

func TestLoadBuildCache_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build.json")
	if err := os.WriteFile(path, []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadBuildCache(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveBuildCache_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "build.json")
	cache := &buildCache{Version: buildCacheVersion}
	if err := saveBuildCache(path, cache); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

// ---------------------------------------------------------------------------
// buildCacheFrom
// ---------------------------------------------------------------------------

func TestBuildCacheFrom(t *testing.T) {
	dir := t.TempDir()

	// Create the directories that buildCacheFrom will hash.
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f1, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		ContentDir:   contentDir,
		PublicDir:    filepath.Join(dir, "public"),
		TemplatesDir: filepath.Join(dir, "templates"),
		StaticDir:    filepath.Join(dir, "static"),
		ThemesDir:    filepath.Join(dir, "themes"),
		PluginsDir:   filepath.Join(dir, "plugins"),
		SassDir:      filepath.Join(dir, "sass"),
		Theme:        "default",
	}

	cache, err := buildCacheFrom(cfg, []string{f1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("cache should not be nil")
	}
	if cache.Version != buildCacheVersion {
		t.Errorf("version = %d, want %d", cache.Version, buildCacheVersion)
	}
	if cache.ConfigHash == "" {
		t.Error("config hash should not be empty")
	}
	if cache.GeneratedAt == "" {
		t.Error("generated_at should not be empty")
	}
	stamp, ok := cache.Content[f1]
	if !ok {
		t.Fatal("content should contain the file")
	}
	if stamp.Size != 7 { // "# Hello" = 7 bytes
		t.Errorf("stamp.Size = %d, want 7", stamp.Size)
	}
}

func TestBuildCacheFrom_EmptyFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		ContentDir:   filepath.Join(dir, "content"),
		PublicDir:    filepath.Join(dir, "public"),
		TemplatesDir: filepath.Join(dir, "templates"),
		StaticDir:    filepath.Join(dir, "static"),
		ThemesDir:    filepath.Join(dir, "themes"),
		PluginsDir:   filepath.Join(dir, "plugins"),
		SassDir:      filepath.Join(dir, "sass"),
		Theme:        "default",
	}

	cache, err := buildCacheFrom(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("cache should not be nil")
	}
	if len(cache.Content) != 0 {
		t.Errorf("expected 0 content entries, got %d", len(cache.Content))
	}
}

// ---------------------------------------------------------------------------
// languagesView
// ---------------------------------------------------------------------------

func TestLanguagesView_DefaultOnly(t *testing.T) {
	cfg := config.Config{DefaultLanguage: "es"}
	langs := languagesView(cfg)
	if len(langs) != 1 {
		t.Fatalf("expected 1 language, got %d", len(langs))
	}
	if langs[0]["code"] != "es" {
		t.Errorf("code = %v, want es", langs[0]["code"])
	}
	if langs[0]["default"] != true {
		t.Error("first language should be marked as default")
	}
}

func TestLanguagesView_WithSecondary(t *testing.T) {
	cfg := config.Config{
		DefaultLanguage: "es",
		Languages: []config.LanguageConfig{
			{Code: "en", Label: "English"},
			{Code: "fr", Label: "Francais"},
		},
	}
	langs := languagesView(cfg)
	if len(langs) != 3 {
		t.Fatalf("expected 3 languages, got %d", len(langs))
	}
	if langs[0]["default"] != true {
		t.Error("first should be default")
	}
	if langs[1]["code"] != "en" || langs[1]["default"] != false {
		t.Errorf("langs[1] = %v", langs[1])
	}
	if langs[2]["code"] != "fr" || langs[2]["label"] != "Francais" {
		t.Errorf("langs[2] = %v", langs[2])
	}
}

// ---------------------------------------------------------------------------
// buildStatsView -- additional checks
// ---------------------------------------------------------------------------

func TestBuildStatsView_ZeroStats(t *testing.T) {
	stats := Stats{}
	siteIndex := site.New()
	view := buildStatsView(stats, siteIndex)
	if view["total"] != 0 {
		t.Errorf("total = %v", view["total"])
	}
	if view["pages"] != 0 {
		t.Errorf("pages = %v", view["pages"])
	}
	if view["sections"] != 1 { // root section is always present
		t.Errorf("sections = %v, want 1 (root)", view["sections"])
	}
}

// ---------------------------------------------------------------------------
// buildOutputsIndex -- additional edge cases
// ---------------------------------------------------------------------------

func TestBuildOutputsIndex_EmptySite(t *testing.T) {
	s := &site.Site{
		Pages:    []*site.Page{},
		Sections: map[string]*site.Section{},
	}
	out := buildOutputsIndex(s, "public")
	if len(out) != 0 {
		t.Errorf("expected 0 entries, got %d", len(out))
	}
}

func TestBuildOutputsIndex_NilSection(t *testing.T) {
	s := &site.Site{
		Pages: []*site.Page{},
		Sections: map[string]*site.Section{
			"nil": nil,
		},
	}
	// Should not panic.
	out := buildOutputsIndex(s, "public")
	if len(out) != 0 {
		t.Errorf("expected 0 entries, got %d", len(out))
	}
}

// ---------------------------------------------------------------------------
// taxonomyPagePath -- additional cases
// ---------------------------------------------------------------------------

func TestTaxonomyPagePath_CustomPaginatePath(t *testing.T) {
	got := taxonomyPagePath("/tags/go", "pagina", 2)
	want := "/tags/go/pagina/3/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// latestUpdated -- additional
// ---------------------------------------------------------------------------

func TestLatestUpdated_AllSameDate(t *testing.T) {
	d := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	pages := []*site.Page{{Date: d}, {Date: d}, {Date: d}}
	got := latestUpdated(pages)
	if !got.Equal(d) {
		t.Errorf("got %v, want %v", got, d)
	}
}

// ---------------------------------------------------------------------------
// sectionUpdated -- additional
// ---------------------------------------------------------------------------

func TestSectionUpdated_OnlySubsections(t *testing.T) {
	deep := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := &site.Section{
		Subsections: []*site.Section{
			{Pages: []*site.Page{{Date: deep}}},
		},
	}
	got := sectionUpdated(s)
	if !got.Equal(deep) {
		t.Errorf("got %v, want %v", got, deep)
	}
}

// ---------------------------------------------------------------------------
// taxonomyIndexUpdated -- additional
// ---------------------------------------------------------------------------

func TestTaxonomyIndexUpdated_SingleTerm(t *testing.T) {
	d := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	idx := &taxonomy.Index{
		Terms: map[string]*taxonomy.Term{
			"go": {Pages: []*site.Page{{Date: d}}},
		},
	}
	got := taxonomyIndexUpdated(idx)
	if !got.Equal(d) {
		t.Errorf("got %v, want %v", got, d)
	}
}

// ---------------------------------------------------------------------------
// hashPlugins
// ---------------------------------------------------------------------------

func TestHashPlugins_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	h1, err := hashPlugins(config.Config{PluginsDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Empty dir should produce a consistent hash.
	h2, err := hashPlugins(config.Config{PluginsDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hashPlugins should be deterministic for empty dir")
	}
}

func TestHashPlugins_WithWasmFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.wasm"), []byte("fake-wasm"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Also add a non-wasm file that should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := hashPlugins(config.Config{PluginsDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Error("expected non-empty hash for dir with wasm files")
	}
}

func TestHashPlugins_NonexistentDir(t *testing.T) {
	h, err := hashPlugins(config.Config{PluginsDir: "/nonexistent/dir"})
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("expected empty hash for nonexistent dir, got %q", h)
	}
}

// ---------------------------------------------------------------------------
// hashAssets
// ---------------------------------------------------------------------------

func TestHashAssets_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		StaticDir: filepath.Join(dir, "static"),
		ThemesDir: filepath.Join(dir, "themes"),
		SassDir:   filepath.Join(dir, "sass"),
		Theme:     "default",
	}
	h, err := hashAssets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// All dirs nonexistent but hashStrings still produces a hash.
	if h == "" {
		t.Error("expected non-empty hash (hashStrings of empty values)")
	}
}

// ---------------------------------------------------------------------------
// hashTemplates
// ---------------------------------------------------------------------------

func TestHashTemplates(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		TemplatesDir: filepath.Join(dir, "templates"),
		ThemesDir:    filepath.Join(dir, "themes"),
		Theme:        "default",
	}
	h, err := hashTemplates(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Should succeed even with empty/nonexistent dirs (builtins always exist).
	if h == "" {
		t.Error("expected non-empty hash (at least builtins)")
	}
}

// ---------------------------------------------------------------------------
// feedPages -- additional edge cases
// ---------------------------------------------------------------------------

func TestFeedPages_AllDrafts(t *testing.T) {
	pages := []*site.Page{
		{Title: "D1", Draft: true},
		{Title: "D2", Draft: true},
	}
	views := feedPages(pages)
	if len(views) != 0 {
		t.Errorf("expected 0 views for all-draft pages, got %d", len(views))
	}
}

// ---------------------------------------------------------------------------
// configView -- comments_enabled logic
// ---------------------------------------------------------------------------

func TestConfigView_CommentsEnabledFalseWhenNoProviders(t *testing.T) {
	cfg := config.Config{
		Interactions: config.InteractionsConfig{
			Comments: config.CommentsConfig{
				Enabled:   true,
				Providers: nil, // no providers
			},
		},
	}
	view := configView(cfg)
	if view["comments_enabled"] != false {
		t.Error("comments_enabled should be false when no providers")
	}
}

func TestConfigView_CommentsEnabledFalseWhenDisabled(t *testing.T) {
	cfg := config.Config{
		Interactions: config.InteractionsConfig{
			Comments: config.CommentsConfig{
				Enabled: false,
				Providers: []config.AuthProviderConfig{
					{Provider: "github"},
				},
			},
		},
	}
	view := configView(cfg)
	if view["comments_enabled"] != false {
		t.Error("comments_enabled should be false when Enabled is false")
	}
}

// ---------------------------------------------------------------------------
// siteFeedContext -- additional checks
// ---------------------------------------------------------------------------

func TestSiteFeedContext_EmptyPages(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "es"},
	}
	cfg := config.Config{SiteTitle: "T", SiteDescription: "D"}
	now := time.Now()

	ctx := siteFeedContext(base, cfg, nil, "http://example.com/atom.xml", now)
	pages, ok := ctx["pages"].([]map[string]any)
	if !ok {
		t.Fatal("pages should be []map[string]any")
	}
	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

// ---------------------------------------------------------------------------
// collectSitemapEntries -- hreflang alternates
// ---------------------------------------------------------------------------

func TestCollectSitemapEntries_HreflangAlternates(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	page := &site.Page{
		Permalink: "http://example.com/hello/",
		Date:      now,
		Lang:      "es",
		Translations: []site.Translation{
			{Lang: "en", Permalink: "http://example.com/en/hello/"},
		},
	}
	cfg := config.Config{BaseURL: "http://example.com"}
	s := &site.Site{
		Pages: []*site.Page{page},
	}
	entries := collectSitemapEntries(cfg, s, nil)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Extra == nil {
		t.Fatal("extra should contain alternates")
	}
	alts, ok := entries[0].Extra["alternates"].([]map[string]string)
	if !ok {
		t.Fatal("alternates should be []map[string]string")
	}
	if len(alts) != 2 {
		t.Errorf("expected 2 alternates, got %d", len(alts))
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- basic scenarios
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_IncrementalDisabled(t *testing.T) {
	cfg := config.Config{BuildIncremental: false}
	plan, cache := buildPlanFromCache(cfg, nil, nil)
	if !plan.full {
		t.Error("expected full build when incremental is disabled")
	}
	if plan.reason != "incremental disabled" {
		t.Errorf("reason = %q, want 'incremental disabled'", plan.reason)
	}
	if cache != nil {
		t.Error("expected nil cache when incremental is disabled")
	}
}

// ---------------------------------------------------------------------------
// diffContent -- additional edge cases
// ---------------------------------------------------------------------------

func TestDiffContent_BothNil(t *testing.T) {
	changed, removed := diffContent(nil, nil)
	if len(changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(changed))
	}
	if len(removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(removed))
	}
}

func TestDiffContent_EmptyBoth(t *testing.T) {
	changed, removed := diffContent(map[string]fileStamp{}, map[string]fileStamp{})
	if len(changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(changed))
	}
	if len(removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(removed))
	}
}

func TestDiffContent_SizeChanged(t *testing.T) {
	prev := map[string]fileStamp{
		"a.md": {ModTime: 100, Size: 10},
	}
	current := map[string]fileStamp{
		"a.md": {ModTime: 100, Size: 20}, // same time, different size
	}
	changed, _ := diffContent(prev, current)
	if !changed["a.md"] {
		t.Error("expected a.md to be changed (size differs)")
	}
}

// ---------------------------------------------------------------------------
// pageContext -- additional edge case: base ctx not mutated
// ---------------------------------------------------------------------------

func TestPageContext_DoesNotMutateBase(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
		"site":   map[string]any{},
	}
	page := &site.Page{Title: "T", Path: "/t/", Permalink: "http://x/t/"}
	_ = pageContext(base, page)

	if _, exists := base["page"]; exists {
		t.Error("pageContext must not mutate base context")
	}
	if _, exists := base["current_path"]; exists {
		t.Error("pageContext must not mutate base context")
	}
}

// ---------------------------------------------------------------------------
// sectionContext -- additional edge case: base ctx not mutated
// ---------------------------------------------------------------------------

func TestSectionContext_DoesNotMutateBase(t *testing.T) {
	base := map[string]any{
		"config": map[string]any{"default_language": "en"},
	}
	section := &site.Section{Title: "S", Path: "/s/", Permalink: "http://x/s/"}
	_ = sectionContext(base, section)

	if _, exists := base["section"]; exists {
		t.Error("sectionContext must not mutate base context")
	}
}

// ---------------------------------------------------------------------------
// isNilInterface -- additional typed nil variants
// ---------------------------------------------------------------------------

func TestIsNilInterface_TypedNilVariants(t *testing.T) {
	var nilSlice []int
	if !isNilInterface(nilSlice) {
		t.Error("expected true for typed nil slice")
	}

	nonNilSlice := []int{1}
	if isNilInterface(nonNilSlice) {
		t.Error("expected false for non-nil slice")
	}

	var nilCh chan string
	if !isNilInterface(nilCh) {
		t.Error("expected true for typed nil channel")
	}
}

// ---------------------------------------------------------------------------
// minifyDir -- test with CSS file (complements existing minify_test.go)
// ---------------------------------------------------------------------------

func TestMinifyDir_WithCSSAndJS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body {\n  margin: 0;\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("var x = 1 ;"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	count, err := minifyDir(dir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 minified files, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// buildURL -- additional edge case
// ---------------------------------------------------------------------------

func TestBuildURL_RootWithTrailingSlash(t *testing.T) {
	got := buildURL("https://example.com/", "/")
	if got != "https://example.com/" {
		t.Errorf("got %q, want 'https://example.com/'", got)
	}
}

// ---------------------------------------------------------------------------
// ensureTrailingSlash -- additional
// ---------------------------------------------------------------------------

func TestEnsureTrailingSlash_Nested(t *testing.T) {
	got := ensureTrailingSlash("/a/b/c")
	if got != "/a/b/c/" {
		t.Errorf("got %q, want '/a/b/c/'", got)
	}
}

// ---------------------------------------------------------------------------
// configView -- multilingual true
// ---------------------------------------------------------------------------

func TestConfigView_MultilingualTrue(t *testing.T) {
	cfg := config.Config{
		DefaultLanguage: "es",
		Languages: []config.LanguageConfig{
			{Code: "en", Label: "English"},
		},
	}
	view := configView(cfg)
	if view["multilingual"] != true {
		t.Errorf("multilingual = %v, want true", view["multilingual"])
	}
	langs, ok := view["languages"].([]map[string]any)
	if !ok {
		t.Fatal("languages should be []map[string]any")
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

// ---------------------------------------------------------------------------
// hashConfig -- empty config
// ---------------------------------------------------------------------------

func TestHashConfig_EmptyConfig(t *testing.T) {
	cfg := config.Config{}
	h, err := hashConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Error("hash should not be empty even for zero config")
	}
	if len(h) != 64 {
		t.Errorf("hash length = %d, want 64", len(h))
	}
}

// ---------------------------------------------------------------------------
// hashConfig -- all fields populated
// ---------------------------------------------------------------------------

func TestHashConfig_AllFieldsSensitivity(t *testing.T) {
	base := config.Config{
		BaseURL:        "http://example.com",
		Theme:          "default",
		ContentDir:     "content",
		PublicDir:      "public",
		TemplatesDir:   "templates",
		StaticDir:      "static",
		ThemesDir:      "themes",
		PluginsDir:     "plugins",
		SassDir:        "sass",
		IncludeDrafts:  false,
		CompileSass:    false,
		CleanPublic:    false,
		PluginsEnabled: []string{"a"},
		Taxonomies:     []config.TaxonomyConfig{{Name: "tags"}},
	}

	// Change each field and verify hash changes.
	fieldChanges := []struct {
		name   string
		mutate func(c *config.Config)
	}{
		{"BaseURL", func(c *config.Config) { c.BaseURL = "http://other.com" }},
		{"Theme", func(c *config.Config) { c.Theme = "custom" }},
		{"ContentDir", func(c *config.Config) { c.ContentDir = "docs" }},
		{"PublicDir", func(c *config.Config) { c.PublicDir = "dist" }},
		{"IncludeDrafts", func(c *config.Config) { c.IncludeDrafts = true }},
		{"CompileSass", func(c *config.Config) { c.CompileSass = true }},
		{"CleanPublic", func(c *config.Config) { c.CleanPublic = true }},
	}

	baseHash, err := hashConfig(base)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range fieldChanges {
		t.Run(tc.name, func(t *testing.T) {
			modified := base
			tc.mutate(&modified)
			h, err := hashConfig(modified)
			if err != nil {
				t.Fatal(err)
			}
			if h == baseHash {
				t.Errorf("changing %s should produce a different hash", tc.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache with incremental + no cache dir
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_IncrementalNoCacheDir(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    "",
		ContentDir:       contentDir,
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	plan, cache := buildPlanFromCache(cfg, []string{f}, logger)
	// With empty BuildCacheDir, loadBuildCache returns nil -> full rebuild
	if !plan.full {
		t.Error("expected full build when no cache exists")
	}
	// Cache snapshot should still succeed.
	if cache == nil {
		t.Error("cache snapshot should not be nil")
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache with valid cache roundtrip
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_CacheHit(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run: builds cache, no previous cache exists -> full build.
	plan1, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan1.full {
		t.Error("first run should be full (no previous cache)")
	}

	// Save cache from first run.
	cache1.Outputs = map[string]string{f: filepath.Join(dir, "public", "hello", "index.html")}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	// Second run with same files: should be incremental, not full.
	plan2, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if plan2.full {
		t.Errorf("second run should not be full (cache hit), reason=%q", plan2.reason)
	}
	if plan2.contentChanged {
		t.Error("content should not be changed when files are the same")
	}
}

// ---------------------------------------------------------------------------
// hashAssets with actual files
// ---------------------------------------------------------------------------

func TestHashAssets_WithFiles(t *testing.T) {
	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		StaticDir: staticDir,
		ThemesDir: filepath.Join(dir, "themes"),
		SassDir:   filepath.Join(dir, "sass"),
		Theme:     "default",
	}
	h1, err := hashAssets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == "" {
		t.Error("expected non-empty hash")
	}

	// Changing the file should change the hash (on next call after modtime changes).
	// Since modtime may not change in test, just verify we get a consistent hash.
	h2, err := hashAssets(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
}

// ---------------------------------------------------------------------------
// hashTemplates with actual template file
// ---------------------------------------------------------------------------

func TestHashTemplates_WithUserTemplate(t *testing.T) {
	dir := t.TempDir()
	templDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templDir, "custom.html"), []byte("<h1>Custom</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		TemplatesDir: templDir,
		ThemesDir:    filepath.Join(dir, "themes"),
		Theme:        "default",
	}

	h1, err := hashTemplates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Without the user template dir, hash should differ.
	cfg2 := config.Config{
		TemplatesDir: filepath.Join(dir, "no-such-templates"),
		ThemesDir:    filepath.Join(dir, "themes"),
		Theme:        "default",
	}
	h2, err := hashTemplates(cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("adding a user template should change the templates hash")
	}
}

// ---------------------------------------------------------------------------
// generatePlaceholders -- multiple pages mixed
// ---------------------------------------------------------------------------

func TestGeneratePlaceholders_MixedPages(t *testing.T) {
	publicDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{Title: "With Image", Image: "/img/hero.jpg"})
	siteIndex.AddPage(&site.Page{Title: "No Image A"})
	siteIndex.AddPage(&site.Page{Title: "No Image B"})

	if err := generatePlaceholders(siteIndex, publicDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First page keeps its image.
	if siteIndex.Pages[0].Image != "/img/hero.jpg" {
		t.Errorf("page 0 image = %q, want /img/hero.jpg", siteIndex.Pages[0].Image)
	}
	// Other pages get placeholders.
	if siteIndex.Pages[1].Image == "" {
		t.Error("page 1 should have a placeholder image")
	}
	if siteIndex.Pages[2].Image == "" {
		t.Error("page 2 should have a placeholder image")
	}
	// Both placeholders should be set (verified above); they may or may not
	// differ depending on title hashing, so no further assertion needed.
}

// ---------------------------------------------------------------------------
// fillSummaries -- "ai" strategy with SkipAI
// ---------------------------------------------------------------------------

func TestFillSummaries_AIStrategyWithSkipAI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Post",
		SourcePath: "post.md",
		RawContent: "This is a test post. It has some content that can be extracted.",
	})

	cfg := config.Config{
		SummaryStrategy: "ai",
		BuildCacheDir:   t.TempDir(),
	}
	opts := BuildOptions{SkipAI: true}

	affected := fillSummaries(context.Background(), cfg, opts, siteIndex, logger)

	// With SkipAI, it should fall back to cache + auto extraction.
	// Since no cache exists, auto-extract is used.
	if len(affected) != 1 {
		t.Fatalf("expected 1 affected, got %d", len(affected))
	}
	if siteIndex.Pages[0].Summary == "" {
		t.Error("page summary should have been filled via auto-extract fallback")
	}
}

// ---------------------------------------------------------------------------
// fillSummaries -- "ai" with invalid provider (falls back to auto)
// ---------------------------------------------------------------------------

func TestFillSummaries_AIStrategyInvalidProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Post",
		SourcePath: "post.md",
		RawContent: "This is content for a page. Enough words to extract a summary.",
	})

	cfg := config.Config{
		SummaryStrategy: "ai",
		AI: config.AIConfig{
			Provider: "nonexistent-provider",
		},
		BuildCacheDir: t.TempDir(),
	}
	opts := BuildOptions{SkipAI: false}

	affected := fillSummaries(context.Background(), cfg, opts, siteIndex, logger)

	// With invalid provider, NewKairosProvider fails, falls back to auto ExtractProvider.
	if len(affected) != 1 {
		t.Fatalf("expected 1 affected (fallback to auto), got %d", len(affected))
	}
	if siteIndex.Pages[0].Summary == "" {
		t.Error("page summary should have been filled via auto fallback")
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- cache version mismatch
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_CacheVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// Save a cache with wrong version.
	oldCache := &buildCache{
		Version:    999, // wrong version
		ConfigHash: "abc",
	}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, oldCache); err != nil {
		t.Fatalf("save: %v", err)
	}

	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build on version mismatch")
	}
	if plan.reason != "cache version" {
		t.Errorf("reason = %q, want 'cache version'", plan.reason)
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- config changed
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_ConfigChanged(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run to establish cache.
	_, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	cache1.Outputs = map[string]string{}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Change config and run again.
	cfg.BaseURL = "http://changed.com"
	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build when config changed")
	}
	if plan.reason != "config changed" {
		t.Errorf("reason = %q, want 'config changed'", plan.reason)
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- templates changed
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_TemplatesChanged(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	templatesDir := filepath.Join(dir, "templates")

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     templatesDir,
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run.
	_, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	cache1.Outputs = map[string]string{}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Add a template file.
	if err := os.WriteFile(filepath.Join(templatesDir, "new.html"), []byte("<p>new</p>"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build when templates changed")
	}
	if plan.reason != "templates changed" {
		t.Errorf("reason = %q, want 'templates changed'", plan.reason)
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- assets changed
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_AssetsChanged(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	staticDir := filepath.Join(dir, "static")

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        staticDir,
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run.
	_, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	cache1.Outputs = map[string]string{}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Add a static file.
	if err := os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build when assets changed")
	}
	if plan.reason != "assets changed" {
		t.Errorf("reason = %q, want 'assets changed'", plan.reason)
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- plugins changed
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_PluginsChanged(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	pluginsDir := filepath.Join(dir, "plugins")

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       pluginsDir,
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run.
	_, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	cache1.Outputs = map[string]string{}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Add a WASM plugin.
	if err := os.WriteFile(filepath.Join(pluginsDir, "new.wasm"), []byte("wasm"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build when plugins changed")
	}
	if plan.reason != "plugins changed" {
		t.Errorf("reason = %q, want 'plugins changed'", plan.reason)
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- content changed (file modified)
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_ContentChanged(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run.
	_, cache1 := buildPlanFromCache(cfg, []string{f}, logger)
	cache1.Outputs = map[string]string{f: filepath.Join(dir, "public", "hello", "index.html")}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(f, []byte("# Hello World -- Updated!"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, _ := buildPlanFromCache(cfg, []string{f}, logger)
	if plan.full {
		t.Error("content change should not trigger full rebuild")
	}
	if !plan.contentChanged {
		t.Error("contentChanged should be true")
	}
	if !plan.changedFiles[f] {
		t.Error("the modified file should be in changedFiles")
	}
}

// ---------------------------------------------------------------------------
// buildPlanFromCache -- content removed
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_ContentRemoved(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f1 := filepath.Join(contentDir, "a.md")
	f2 := filepath.Join(contentDir, "b.md")
	if err := os.WriteFile(f1, []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("# B"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	// First run with both files.
	_, cache1 := buildPlanFromCache(cfg, []string{f1, f2}, logger)
	cache1.Outputs = map[string]string{
		f1: filepath.Join(dir, "public", "a", "index.html"),
		f2: filepath.Join(dir, "public", "b", "index.html"),
	}
	cachePath := buildCachePath(cfg)
	if err := saveBuildCache(cachePath, cache1); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Second run without f2 (simulating removal).
	plan, _ := buildPlanFromCache(cfg, []string{f1}, logger)
	if plan.full {
		t.Error("removal should not trigger full rebuild")
	}
	if !plan.contentChanged {
		t.Error("contentChanged should be true when a file is removed")
	}
	if plan.removed != 1 {
		t.Errorf("removed = %d, want 1", plan.removed)
	}
	if len(plan.removedFiles) != 1 {
		t.Errorf("removedFiles len = %d, want 1", len(plan.removedFiles))
	}
}

// ---------------------------------------------------------------------------
// minifyFile -- direct test
// ---------------------------------------------------------------------------

func TestMinifyFile_HTML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.html")
	if err := os.WriteFile(path, []byte("<html>  <body>  <p>  Hello  </p>  </body>  </html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newMinifier()
	if err := minifyFile(m, path, "text/html"); err != nil {
		t.Fatalf("minifyFile error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if len(data) >= 50 { // original is 50 bytes
		t.Error("minified HTML should be smaller")
	}
}

func TestMinifyFile_SVG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.svg")
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <rect x="10" y="10" width="80" height="80" fill="blue" />
</svg>`
	if err := os.WriteFile(path, []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newMinifier()
	if err := minifyFile(m, path, "image/svg+xml"); err != nil {
		t.Fatalf("minifyFile error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if len(data) >= len(svg) {
		t.Error("minified SVG should be smaller")
	}
}

func TestMinifyFile_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	if err := os.WriteFile(path, []byte(`{  "key"  :  "value"  }`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newMinifier()
	if err := minifyFile(m, path, "application/json"); err != nil {
		t.Fatalf("minifyFile error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != `{"key":"value"}` {
		t.Errorf("got %q", string(data))
	}
}

func TestMinifyFile_NonexistentFile(t *testing.T) {
	m := newMinifier()
	err := minifyFile(m, "/nonexistent/file.html", "text/html")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// minifyDir -- SVG file
// ---------------------------------------------------------------------------

func TestMinifyDir_SVGFile(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <rect x="10" y="10" width="80" height="80" fill="red" />
</svg>`
	if err := os.WriteFile(filepath.Join(dir, "icon.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	count, err := minifyDir(dir, logger)
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 minified, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// minifyDir -- nested directories
// ---------------------------------------------------------------------------

func TestMinifyDir_NestedDirs(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.html"), []byte("<html> <body></body> </html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.css"), []byte("body {\n  margin: 0;\n}"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	count, err := minifyDir(dir, logger)
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 minified, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// collectSitemapEntries -- taxonomy with pagination
// ---------------------------------------------------------------------------

func TestCollectSitemapEntries_TaxonomyPagination(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create many pages to trigger pagination.
	var pages []*site.Page
	for i := 0; i < 5; i++ {
		pages = append(pages, &site.Page{
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Permalink: "http://example.com/p/",
		})
	}

	cfg := config.Config{
		BaseURL: "http://example.com",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: true, PaginateBy: 2, PaginatePath: "page"},
		},
	}
	s := &site.Site{Pages: []*site.Page{}}
	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags", Render: true, PaginateBy: 2, PaginatePath: "page"},
		Terms: map[string]*taxonomy.Term{
			"go": {
				Name:      "go",
				Slug:      "go",
				Path:      "/tags/go/",
				Permalink: "http://example.com/tags/go/",
				Pages:     pages,
			},
		},
	}
	indices := map[string]*taxonomy.Index{"tags": idx}
	entries := collectSitemapEntries(cfg, s, indices)
	// Should have:
	//   /tags/ (list) = 1
	//   /tags/go/ (page 1) + /tags/go/page/2/ (page 2) + /tags/go/page/3/ (page 3) = 3
	// Total = 4
	if len(entries) < 4 {
		t.Errorf("expected at least 4 entries (list + paginated term pages), got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// configView -- copyright year replacement
// ---------------------------------------------------------------------------

func TestConfigView_CopyrightYearReplacement(t *testing.T) {
	cfg := config.Config{
		Copyright: "Copyright {year} My Site",
	}
	view := configView(cfg)
	copyright := view["copyright"].(string)
	expected := "Copyright " + time.Now().Format("2006") + " My Site"
	if copyright != expected {
		t.Errorf("copyright = %q, want %q", copyright, expected)
	}
}

func TestConfigView_CopyrightNoPlaceholder(t *testing.T) {
	cfg := config.Config{
		Copyright: "All rights reserved",
	}
	view := configView(cfg)
	if view["copyright"] != "All rights reserved" {
		t.Errorf("copyright = %v", view["copyright"])
	}
}

// ---------------------------------------------------------------------------
// configView -- plugins_enabled slice
// ---------------------------------------------------------------------------

func TestConfigView_PluginsEnabled(t *testing.T) {
	cfg := config.Config{
		PluginsEnabled: []string{"llmstxt", "mermaid"},
	}
	view := configView(cfg)
	enabled, ok := view["plugins_enabled"].([]string)
	if !ok {
		t.Fatal("plugins_enabled should be []string")
	}
	if len(enabled) != 2 {
		t.Errorf("plugins_enabled len = %d, want 2", len(enabled))
	}
}

// ---------------------------------------------------------------------------
// configView -- image_widths
// ---------------------------------------------------------------------------

func TestConfigView_ImageWidths(t *testing.T) {
	cfg := config.Config{
		ImageWidths: []int{320, 640, 1200},
	}
	view := configView(cfg)
	widths, ok := view["image_widths"].([]int)
	if !ok {
		t.Fatal("image_widths should be []int")
	}
	if len(widths) != 3 {
		t.Errorf("image_widths len = %d, want 3", len(widths))
	}
}

// ---------------------------------------------------------------------------
// configView -- social map
// ---------------------------------------------------------------------------

func TestConfigView_Social(t *testing.T) {
	cfg := config.Config{
		Social: map[string]string{
			"twitter":  "@user",
			"mastodon": "@user@masto.social",
		},
	}
	view := configView(cfg)
	social, ok := view["social"].(map[string]string)
	if !ok {
		t.Fatal("social should be map[string]string")
	}
	if social["twitter"] != "@user" {
		t.Errorf("social.twitter = %v", social["twitter"])
	}
}

// ---------------------------------------------------------------------------
// timing tests (already 100% but these add complementary coverage)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// buildPlanFromCache -- invalid cache JSON
// ---------------------------------------------------------------------------

func TestBuildPlanFromCache_InvalidCacheJSON(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(contentDir, "hello.md")
	if err := os.WriteFile(f, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write invalid JSON to cache file.
	cachePath := filepath.Join(cacheDir, "build.json")
	if err := os.WriteFile(cachePath, []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		BuildIncremental: true,
		BuildCacheDir:    cacheDir,
		ContentDir:       contentDir,
		PublicDir:        filepath.Join(dir, "public"),
		TemplatesDir:     filepath.Join(dir, "templates"),
		StaticDir:        filepath.Join(dir, "static"),
		ThemesDir:        filepath.Join(dir, "themes"),
		PluginsDir:       filepath.Join(dir, "plugins"),
		SassDir:          filepath.Join(dir, "sass"),
		Theme:            "default",
	}

	plan, cache := buildPlanFromCache(cfg, []string{f}, logger)
	if !plan.full {
		t.Error("expected full build when cache is invalid JSON")
	}
	if plan.reason != "cache load failed" {
		t.Errorf("reason = %q, want 'cache load failed'", plan.reason)
	}
	// Cache snapshot should still succeed.
	if cache == nil {
		t.Error("cache snapshot should not be nil even when load fails")
	}
}

// ---------------------------------------------------------------------------
// newMinifier -- sanity check
// ---------------------------------------------------------------------------

func TestNewMinifier(t *testing.T) {
	m := newMinifier()
	if m == nil {
		t.Fatal("newMinifier should not return nil")
	}
}

// ---------------------------------------------------------------------------
// mimeByExt coverage
// ---------------------------------------------------------------------------

func TestMimeByExt(t *testing.T) {
	expected := map[string]string{
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".svg":  "image/svg+xml",
		".xml":  "text/xml",
	}
	for ext, want := range expected {
		got, ok := mimeByExt[ext]
		if !ok {
			t.Errorf("mimeByExt missing extension %q", ext)
		}
		if got != want {
			t.Errorf("mimeByExt[%q] = %q, want %q", ext, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// saveBuildCache -- whitespace path
// ---------------------------------------------------------------------------

func TestSaveBuildCache_WhitespacePath(t *testing.T) {
	cache := &buildCache{Version: buildCacheVersion}
	if err := saveBuildCache("   ", cache); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// helper: testRenderer creates a minimal *render.Renderer using only builtin
// templates (no user/theme dirs).
// ---------------------------------------------------------------------------

func testRenderer(t *testing.T) *render.Renderer {
	t.Helper()
	s := site.New()
	b := i18n.New("es")
	r, err := render.New("", "", render.Context{
		BaseURL:         "http://example.com",
		ContentDir:      "content",
		StaticDir:       "static",
		PublicDir:       "public",
		DefaultLanguage: "es",
		Site:            s,
		I18n:            b,
	})
	if err != nil {
		t.Fatalf("render.New: %v", err)
	}
	return r
}

// ---------------------------------------------------------------------------
// renderNotFound
// ---------------------------------------------------------------------------

func TestRenderNotFound_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderNotFound(renderer, cfg, baseCtx, plan)
	if err != nil {
		t.Fatalf("renderNotFound: %v", err)
	}
	if rendered != 1 {
		t.Errorf("rendered = %d, want 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	// Verify the output file was created.
	outputPath := filepath.Join(dir, "404.html")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("404.html was not created")
	}
}

func TestRenderNotFound_CachedPlan(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	// Pre-create the output so outputMissing returns false.
	if err := os.WriteFile(filepath.Join(dir, "404.html"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := buildPlan{incremental: true, full: false, contentChanged: false}

	rendered, cached, err := renderNotFound(renderer, cfg, baseCtx, plan)
	if err != nil {
		t.Fatalf("renderNotFound: %v", err)
	}
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0", rendered)
	}
	if cached != 1 {
		t.Errorf("cached = %d, want 1", cached)
	}
}

// ---------------------------------------------------------------------------
// renderRobots
// ---------------------------------------------------------------------------

func TestRenderRobots_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderRobots(renderer, cfg, baseCtx, plan)
	if err != nil {
		t.Fatalf("renderRobots: %v", err)
	}
	if rendered != 1 {
		t.Errorf("rendered = %d, want 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	outputPath := filepath.Join(dir, "robots.txt")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("robots.txt was not created")
	}
}

func TestRenderRobots_CachedPlan(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	// Pre-create the output so outputMissing returns false.
	if err := os.WriteFile(filepath.Join(dir, "robots.txt"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := buildPlan{incremental: true, full: false, contentChanged: false}

	rendered, cached, err := renderRobots(renderer, cfg, baseCtx, plan)
	if err != nil {
		t.Fatalf("renderRobots: %v", err)
	}
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0", rendered)
	}
	if cached != 1 {
		t.Errorf("cached = %d, want 1", cached)
	}
}

// ---------------------------------------------------------------------------
// renderSiteFeed
// ---------------------------------------------------------------------------

func TestRenderSiteFeed_Disabled(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
		SiteFeed:  false,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSiteFeed(context.Background(), renderer, cfg, baseCtx, site.New(), nil, plan)
	if err != nil {
		t.Fatalf("renderSiteFeed: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 when SiteFeed is false, got %d/%d", rendered, cached)
	}
}

func TestRenderSiteFeed_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	s.AddPage(&site.Page{
		Title:     "Post 1",
		Path:      "/2025/01/post-1/",
		Permalink: "http://example.com/2025/01/post-1/",
		Date:      now,
		Content:   "<p>Hello</p>",
	})
	cfg := config.Config{
		BaseURL:         "http://example.com",
		SiteTitle:       "Test Site",
		SiteDescription: "A test",
		PublicDir:       dir,
		SiteFeed:        true,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSiteFeed(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSiteFeed: %v", err)
	}
	// Should render both atom.xml and rss.xml.
	if rendered < 1 {
		t.Errorf("rendered = %d, want >= 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

func TestRenderSiteFeed_WithLimit(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	for i := 0; i < 5; i++ {
		s.AddPage(&site.Page{
			Title:     "Post",
			Path:      "/p/",
			Permalink: "http://example.com/p/",
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Content:   "<p>Hello</p>",
		})
	}
	cfg := config.Config{
		BaseURL:         "http://example.com",
		SiteTitle:       "Test",
		PublicDir:       dir,
		SiteFeed:        true,
		SiteFeedLimit:   2,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, _, err := renderSiteFeed(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSiteFeed: %v", err)
	}
	if rendered < 1 {
		t.Errorf("rendered = %d, want >= 1", rendered)
	}
}

// ---------------------------------------------------------------------------
// renderSitemap
// ---------------------------------------------------------------------------

func TestRenderSitemap_EmptyEntries(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSitemap(context.Background(), renderer, cfg, baseCtx, nil, nil, plan)
	if err != nil {
		t.Fatalf("renderSitemap: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 for empty entries, got %d/%d", rendered, cached)
	}
}

func TestRenderSitemap_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	entries := []SitemapEntry{
		{Permalink: "http://example.com/a/", Updated: time.Now()},
		{Permalink: "http://example.com/b/", Updated: time.Now()},
	}

	rendered, cached, err := renderSitemap(context.Background(), renderer, cfg, baseCtx, entries, nil, plan)
	if err != nil {
		t.Fatalf("renderSitemap: %v", err)
	}
	if rendered != 1 {
		t.Errorf("rendered = %d, want 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	outputPath := filepath.Join(dir, "sitemap.xml")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("sitemap.xml was not created")
	}
}

func TestRenderSitemap_CachedPlan(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	// Pre-create the output so outputMissing returns false.
	if err := os.WriteFile(filepath.Join(dir, "sitemap.xml"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := buildPlan{incremental: true, full: false, contentChanged: false}

	entries := []SitemapEntry{
		{Permalink: "http://example.com/a/", Updated: time.Now()},
	}

	rendered, cached, err := renderSitemap(context.Background(), renderer, cfg, baseCtx, entries, nil, plan)
	if err != nil {
		t.Fatalf("renderSitemap: %v", err)
	}
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0", rendered)
	}
	if cached != 1 {
		t.Errorf("cached = %d, want 1", cached)
	}
}

// ---------------------------------------------------------------------------
// renderSections
// ---------------------------------------------------------------------------

func TestRenderSections_EmptySections(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	s := site.New()
	// site.New() has no sections at all.
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	// site.New() has 0 sections - but pages may auto-create one from BuildHierarchy.
	// Just verify no error occurred. Some sections might be 0 or non-0 depending on site.New.
	_ = rendered
	_ = cached
}

func TestRenderSections_WithSections(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	s := site.New()
	s.AddSection(&site.Section{
		Title:  "Blog",
		Path:   "/blog/",
		IsRoot: false,
		Pages:  []*site.Page{},
	})
	s.AddSection(&site.Section{
		Title:  "Home",
		Path:   "/",
		IsRoot: true,
		Pages:  []*site.Page{},
	})
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	if rendered != 2 {
		t.Errorf("rendered = %d, want 2", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	// Check output files exist.
	if _, err := os.Stat(filepath.Join(dir, "blog", "index.html")); os.IsNotExist(err) {
		t.Error("blog/index.html was not created")
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); os.IsNotExist(err) {
		t.Error("index.html was not created")
	}
}

func TestRenderSections_IncrementalPlan(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	s := site.New()
	s.AddSection(&site.Section{
		Title:  "Blog",
		Path:   "/blog/",
		IsRoot: false,
		Pages:  []*site.Page{},
	})
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	// Incremental plan with no content changes but output missing -> still renders.
	plan := buildPlan{incremental: true, full: false, contentChanged: false}

	rendered, cached, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	// Output is missing so it should render.
	if rendered+cached != len(s.Sections) {
		t.Errorf("rendered(%d)+cached(%d) != sections(%d)", rendered, cached, len(s.Sections))
	}
}

// ---------------------------------------------------------------------------
// renderPages
// ---------------------------------------------------------------------------

func TestRenderPages_EmptyPages(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	s := site.New()
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderPages(context.Background(), renderer, cfg, baseCtx, s, map[string]*taxonomy.Index{}, nil, plan)
	if err != nil {
		t.Fatalf("renderPages: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 for empty pages, got %d/%d", rendered, cached)
	}
}

func TestRenderPages_WithPages(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Post 1",
		Slug:       "post-1",
		Path:       "/2025/01/post-1/",
		Permalink:  "http://example.com/2025/01/post-1/",
		Date:       now,
		Content:    "<p>Hello</p>",
		RawContent: "Hello",
		Lang:       "es",
	})
	s.AddPage(&site.Page{
		Title:      "Post 2",
		Slug:       "post-2",
		Path:       "/2025/02/post-2/",
		Permalink:  "http://example.com/2025/02/post-2/",
		Date:       now.Add(-time.Hour),
		Content:    "<p>World</p>",
		RawContent: "World",
		Lang:       "es",
	})
	s.BuildHierarchy()
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderPages(context.Background(), renderer, cfg, baseCtx, s, map[string]*taxonomy.Index{}, nil, plan)
	if err != nil {
		t.Fatalf("renderPages: %v", err)
	}
	if rendered != 2 {
		t.Errorf("rendered = %d, want 2", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

func TestRenderPages_CachedPlan(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "Post",
		Slug:       "post",
		Path:       "/2025/01/post/",
		Permalink:  "http://example.com/2025/01/post/",
		Date:       now,
		Content:    "<p>Hello</p>",
		RawContent: "Hello",
		Lang:       "es",
		SourcePath: "/content/post.md",
	})
	s.BuildHierarchy()
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	// Pre-create the output file so outputMissing returns false.
	outDir := filepath.Join(dir, "2025", "01", "post")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := buildPlan{incremental: true, full: false, contentChanged: false}

	rendered, cached, err := renderPages(context.Background(), renderer, cfg, baseCtx, s, map[string]*taxonomy.Index{}, nil, plan)
	if err != nil {
		t.Fatalf("renderPages: %v", err)
	}
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0", rendered)
	}
	if cached != 1 {
		t.Errorf("cached = %d, want 1", cached)
	}
}

func TestRenderPages_MenuPageSkipped(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	s := site.New()
	s.AddPage(&site.Page{
		Title:      "About",
		Slug:       "about",
		Path:       "/about/",
		Permalink:  "http://example.com/about/",
		Date:       now,
		Content:    "<p>About</p>",
		RawContent: "About",
		Lang:       "es",
		Menu:       true,
	})
	s.AddPage(&site.Page{
		Title:      "Post",
		Slug:       "post",
		Path:       "/2025/01/post/",
		Permalink:  "http://example.com/2025/01/post/",
		Date:       now,
		Content:    "<p>Post</p>",
		RawContent: "Post",
		Lang:       "es",
	})
	s.BuildHierarchy()
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, _, err := renderPages(context.Background(), renderer, cfg, baseCtx, s, map[string]*taxonomy.Index{}, nil, plan)
	if err != nil {
		t.Fatalf("renderPages: %v", err)
	}
	// Both pages should be rendered (menu pages are rendered, just not in chronological nav).
	if rendered != 2 {
		t.Errorf("rendered = %d, want 2", rendered)
	}
}

// ---------------------------------------------------------------------------
// taxonomyTemplateName
// ---------------------------------------------------------------------------

func TestTaxonomyTemplateName_Fallback(t *testing.T) {
	renderer := testRenderer(t)
	// No custom taxonomy template; should fall back.
	got := taxonomyTemplateName(renderer, "tags", "list.html", "taxonomy_list.html")
	if got != "taxonomy_list.html" {
		t.Errorf("got %q, want %q", got, "taxonomy_list.html")
	}
}

func TestTaxonomyTemplateName_FallbackSingle(t *testing.T) {
	renderer := testRenderer(t)
	got := taxonomyTemplateName(renderer, "categories", "single.html", "taxonomy_single.html")
	if got != "taxonomy_single.html" {
		t.Errorf("got %q, want %q", got, "taxonomy_single.html")
	}
}

// ---------------------------------------------------------------------------
// renderTaxonomies
// ---------------------------------------------------------------------------

func TestRenderTaxonomies_EmptyTaxonomies(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	s := site.New()
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, map[string]*taxonomy.Index{}, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 for empty taxonomies, got %d/%d", rendered, cached)
	}
}

func TestRenderTaxonomies_WithTerms(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: true},
		},
	}
	page := &site.Page{
		Title:     "Post",
		Path:      "/p/",
		Permalink: "http://example.com/p/",
		Date:      now,
		Content:   "<p>Post</p>",
	}
	s := site.New()
	s.AddPage(page)

	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags", Render: true},
		Terms: map[string]*taxonomy.Term{
			"go": {
				Name:      "Go",
				Slug:      "go",
				Path:      "/tags/go/",
				Permalink: "http://example.com/tags/go/",
				Pages:     []*site.Page{page},
			},
		},
	}
	indices := map[string]*taxonomy.Index{"tags": idx}
	baseCtx := baseContext(cfg, s.View(), indices, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, indices, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	// Should render: list page + 1 term page = 2
	if rendered < 2 {
		t.Errorf("rendered = %d, want >= 2", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

func TestRenderTaxonomies_RenderDisabled(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: false},
		},
	}
	s := site.New()
	indices := map[string]*taxonomy.Index{}
	baseCtx := baseContext(cfg, s.View(), indices, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, indices, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 when render=false, got %d/%d", rendered, cached)
	}
}

func TestRenderTaxonomies_MissingIndex(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: true},
		},
	}
	s := site.New()
	indices := map[string]*taxonomy.Index{} // no "tags" index
	baseCtx := baseContext(cfg, s.View(), indices, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, indices, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 when index is missing, got %d/%d", rendered, cached)
	}
}

// ---------------------------------------------------------------------------
// renderTaxonomyFeeds
// ---------------------------------------------------------------------------

func TestRenderTaxonomyFeeds_Disabled(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:   "http://example.com",
		PublicDir: dir,
	}
	taxCfg := config.TaxonomyConfig{Name: "tags", Feed: false}
	term := &taxonomy.Term{
		Name: "Go", Slug: "go", Path: "/tags/go/",
		Pages: []*site.Page{},
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomyFeeds(renderer, cfg, baseCtx, taxCfg, term, plan)
	if err != nil {
		t.Fatalf("renderTaxonomyFeeds: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 when feed=false, got %d/%d", rendered, cached)
	}
}

func TestRenderTaxonomyFeeds_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
	}
	taxCfg := config.TaxonomyConfig{Name: "tags", Feed: true}
	term := &taxonomy.Term{
		Name:      "Go",
		Slug:      "go",
		Path:      "/tags/go/",
		Permalink: "http://example.com/tags/go/",
		Pages: []*site.Page{
			{Title: "Post", Path: "/p/", Permalink: "http://example.com/p/", Date: now, Content: "<p>Hi</p>"},
		},
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomyFeeds(renderer, cfg, baseCtx, taxCfg, term, plan)
	if err != nil {
		t.Fatalf("renderTaxonomyFeeds: %v", err)
	}
	// Should render at least atom.xml and rss.xml for the term.
	if rendered < 1 {
		t.Errorf("rendered = %d, want >= 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

// ---------------------------------------------------------------------------
// renderTaxonomies with pagination
// ---------------------------------------------------------------------------

func TestRenderTaxonomies_WithPagination(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: true, PaginateBy: 2, PaginatePath: "page"},
		},
	}
	var pages []*site.Page
	for i := 0; i < 5; i++ {
		pages = append(pages, &site.Page{
			Title:     "Post",
			Path:      "/p/",
			Permalink: "http://example.com/p/",
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Content:   "<p>Post</p>",
		})
	}
	s := site.New()
	for _, p := range pages {
		s.AddPage(p)
	}

	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags", Render: true, PaginateBy: 2, PaginatePath: "page"},
		Terms: map[string]*taxonomy.Term{
			"go": {
				Name:      "Go",
				Slug:      "go",
				Path:      "/tags/go/",
				Permalink: "http://example.com/tags/go/",
				Pages:     pages,
			},
		},
	}
	indices := map[string]*taxonomy.Index{"tags": idx}
	baseCtx := baseContext(cfg, s.View(), indices, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, indices, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	// List page + paginated term pages (3 pages for 5 items / 2 per page)
	if rendered < 4 {
		t.Errorf("rendered = %d, want >= 4", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

// ---------------------------------------------------------------------------
// renderTaxonomies with feeds
// ---------------------------------------------------------------------------

func TestRenderTaxonomies_WithFeeds(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		DefaultLanguage: "es",
		Taxonomies: []config.TaxonomyConfig{
			{Name: "tags", Render: true, Feed: true},
		},
	}
	page := &site.Page{
		Title:     "Post",
		Path:      "/p/",
		Permalink: "http://example.com/p/",
		Date:      now,
		Content:   "<p>Post</p>",
	}
	s := site.New()
	s.AddPage(page)

	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags", Render: true, Feed: true},
		Terms: map[string]*taxonomy.Term{
			"go": {
				Name:      "Go",
				Slug:      "go",
				Path:      "/tags/go/",
				Permalink: "http://example.com/tags/go/",
				Pages:     []*site.Page{page},
			},
		},
	}
	indices := map[string]*taxonomy.Index{"tags": idx}
	baseCtx := baseContext(cfg, s.View(), indices, nil)
	plan := buildPlan{full: true}

	rendered, _, err := renderTaxonomies(context.Background(), renderer, cfg, baseCtx, s, indices, nil, plan)
	if err != nil {
		t.Fatalf("renderTaxonomies: %v", err)
	}
	// list + term + atom + rss = at least 4
	if rendered < 4 {
		t.Errorf("rendered = %d, want >= 4", rendered)
	}
}

// ---------------------------------------------------------------------------
// saveBuildCache -- successful write
// ---------------------------------------------------------------------------

func TestSaveBuildCache_Successful(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "sub", "build.json")
	cache := &buildCache{
		Version:       buildCacheVersion,
		ConfigHash:    "abc123",
		TemplatesHash: "def456",
		GeneratedAt:   time.Now().Format(time.RFC3339Nano),
		Content: map[string]fileStamp{
			"/tmp/a.md": {ModTime: 1234, Size: 100},
		},
	}
	if err := saveBuildCache(cachePath, cache); err != nil {
		t.Fatalf("saveBuildCache: %v", err)
	}

	// Verify the file was created and is valid JSON.
	loaded, err := loadBuildCache(cachePath)
	if err != nil {
		t.Fatalf("loadBuildCache: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded cache should not be nil")
	}
	if loaded.Version != buildCacheVersion {
		t.Errorf("version = %d, want %d", loaded.Version, buildCacheVersion)
	}
	if loaded.ConfigHash != "abc123" {
		t.Errorf("config_hash = %q", loaded.ConfigHash)
	}
}

// ---------------------------------------------------------------------------
// hashDir -- file (not dir) as root
// ---------------------------------------------------------------------------

func TestHashDir_FileAsRoot(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := hashDir(f, includeAllFiles)
	if err != nil {
		t.Fatalf("hashDir: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash for file root, got %q", hash)
	}
}

// ---------------------------------------------------------------------------
// loadBuildCache -- permission error (non-readable file)
// ---------------------------------------------------------------------------

func TestLoadBuildCache_ReadError(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "build.json")
	if err := os.WriteFile(cachePath, []byte(`{"version":2}`), 0o000); err != nil {
		t.Fatal(err)
	}
	_, err := loadBuildCache(cachePath)
	if err == nil {
		// On some platforms this might succeed (e.g., running as root).
		// Just skip in that case.
		t.Skip("read permission test not effective on this platform")
	}
}

// ---------------------------------------------------------------------------
// buildCacheFrom -- with real temp directories
// ---------------------------------------------------------------------------

func TestBuildCacheFrom_WithRealDirs(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	templatesDir := filepath.Join(dir, "templates")
	staticDir := filepath.Join(dir, "static")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}

	f := filepath.Join(contentDir, "post.md")
	if err := os.WriteFile(f, []byte("# Post"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "page.html"), []byte("<p>{{.}}</p>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		BaseURL:      "http://example.com",
		ContentDir:   contentDir,
		TemplatesDir: templatesDir,
		StaticDir:    staticDir,
		PublicDir:    filepath.Join(dir, "public"),
		ThemesDir:    filepath.Join(dir, "themes"),
		PluginsDir:   filepath.Join(dir, "plugins"),
		SassDir:      filepath.Join(dir, "sass"),
		Theme:        "default",
	}

	cache, err := buildCacheFrom(cfg, []string{f})
	if err != nil {
		t.Fatalf("buildCacheFrom: %v", err)
	}
	if cache == nil {
		t.Fatal("cache should not be nil")
	}
	if cache.Version != buildCacheVersion {
		t.Errorf("version = %d, want %d", cache.Version, buildCacheVersion)
	}
	if cache.ConfigHash == "" {
		t.Error("config hash should not be empty")
	}
	if cache.TemplatesHash == "" {
		t.Error("templates hash should not be empty")
	}
	if len(cache.Content) != 1 {
		t.Errorf("content entries = %d, want 1", len(cache.Content))
	}
}

// --- renderSectionFeeds tests ---

func TestRenderSectionFeeds_Disabled(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	cfg := config.Config{
		BaseURL:      "http://example.com",
		PublicDir:    dir,
		SectionFeeds: false,
	}
	baseCtx := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSectionFeeds(context.Background(), renderer, cfg, baseCtx, site.New(), nil, plan)
	if err != nil {
		t.Fatalf("renderSectionFeeds: %v", err)
	}
	if rendered != 0 || cached != 0 {
		t.Errorf("expected 0/0 when SectionFeeds is false, got %d/%d", rendered, cached)
	}
}

func TestRenderSectionFeeds_SkipsRoot(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	// Add a page to the root section only.
	s.AddPage(&site.Page{
		Title:     "Root Post",
		Path:      "/root-post/",
		Permalink: "http://example.com/root-post/",
		Date:      now,
		Content:   "<p>Root</p>",
	})
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		SectionFeeds:    true,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSectionFeeds(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSectionFeeds: %v", err)
	}
	// Root section is skipped, so nothing should be rendered.
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0 (root should be skipped)", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
}

func TestRenderSectionFeeds_FullBuild(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	s.AddPage(&site.Page{
		Title:     "Blog Post",
		Slug:      "blog-post",
		Path:      "/blog/blog-post/",
		Permalink: "http://example.com/blog/blog-post/",
		Date:      now,
		Content:   "<p>Blog content</p>",
	})
	s.BuildHierarchy()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		SiteTitle:       "Test Site",
		SiteDescription: "A test",
		PublicDir:       dir,
		SectionFeeds:    true,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSectionFeeds(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSectionFeeds: %v", err)
	}
	// Should render at least atom.xml for the blog section.
	if rendered < 1 {
		t.Errorf("rendered = %d, want >= 1", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	// Verify files exist.
	atomPath := filepath.Join(dir, "blog", "atom.xml")
	if _, err := os.Stat(atomPath); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", atomPath)
	}
}

func TestRenderSectionFeeds_SkipsDraftOnlySection(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	s.AddPage(&site.Page{
		Title:     "Draft Post",
		Slug:      "draft-post",
		Path:      "/blog/draft-post/",
		Permalink: "http://example.com/blog/draft-post/",
		Date:      now,
		Content:   "<p>Draft</p>",
		Draft:     true,
	})
	s.BuildHierarchy()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		SectionFeeds:    true,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, _, err := renderSectionFeeds(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSectionFeeds: %v", err)
	}
	// All pages are drafts, so feedPages filters them all -> no output.
	if rendered != 0 {
		t.Errorf("rendered = %d, want 0 (all drafts)", rendered)
	}
}

func TestSectionFeedContext(t *testing.T) {
	cfg := config.Config{
		BaseURL:         "http://example.com",
		SiteTitle:       "Test",
		SiteDescription: "A test site",
		DefaultLanguage: "es",
	}
	base := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	section := &site.Section{
		Title: "Blog",
		Slug:  "blog",
		Path:  "/blog/",
	}
	now := time.Now()
	pages := []map[string]any{
		{"title": "Post 1", "permalink": "http://example.com/blog/p1/"},
	}

	ctx := sectionFeedContext(base, cfg, section, pages, "http://example.com/blog/atom.xml", now)

	if ctx["feed_title"] != "Blog" {
		t.Errorf("feed_title = %v, want Blog", ctx["feed_title"])
	}
	if ctx["feed_description"] != "A test site" {
		t.Errorf("feed_description = %v, want A test site", ctx["feed_description"])
	}
	if ctx["feed_url"] != "http://example.com/blog/atom.xml" {
		t.Errorf("feed_url = %v, want http://example.com/blog/atom.xml", ctx["feed_url"])
	}
	if ctx["lang"] != "es" {
		t.Errorf("lang = %v, want es", ctx["lang"])
	}
	if pp, ok := ctx["pages"].([]map[string]any); !ok || len(pp) != 1 {
		t.Errorf("pages = %v, want 1 page", ctx["pages"])
	}
}

// --- renderPaginatedIndex tests ---

func TestRenderPaginatedIndex_NoPagination(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	// Only 2 pages, PostsPerPage=10 -> no pagination
	for i := 0; i < 2; i++ {
		s.AddPage(&site.Page{
			Title:     fmt.Sprintf("Post %d", i),
			Path:      fmt.Sprintf("/post-%d/", i),
			Permalink: fmt.Sprintf("http://example.com/post-%d/", i),
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Content:   "<p>Hello</p>",
		})
	}
	s.BuildHierarchy()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		PostsPerPage:    10,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	// Should render index.html without pagination pages.
	if rendered != 1 {
		t.Errorf("rendered = %d, want 1 (single index)", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}
	// Only /index.html should exist, not /page/2/.
	page2 := filepath.Join(dir, "page", "2", "index.html")
	if _, err := os.Stat(page2); !os.IsNotExist(err) {
		t.Errorf("unexpected %s when no pagination needed", page2)
	}
}

func TestRenderPaginatedIndex_WithPagination(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	// 5 pages, PostsPerPage=2 -> 3 pagination pages
	for i := 0; i < 5; i++ {
		s.AddPage(&site.Page{
			Title:     fmt.Sprintf("Post %d", i),
			Path:      fmt.Sprintf("/post-%d/", i),
			Permalink: fmt.Sprintf("http://example.com/post-%d/", i),
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Content:   "<p>Hello</p>",
		})
	}
	s.BuildHierarchy()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		SiteTitle:       "Test",
		PublicDir:       dir,
		PostsPerPage:    2,
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, cached, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	// 3 paginated index pages (page 1, 2, 3).
	if rendered < 3 {
		t.Errorf("rendered = %d, want >= 3 (3 pagination pages)", rendered)
	}
	if cached != 0 {
		t.Errorf("cached = %d, want 0", cached)
	}

	// Verify /index.html exists.
	indexPath := filepath.Join(dir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("expected /index.html to exist")
	}
	// Verify /page/2/index.html exists.
	page2 := filepath.Join(dir, "page", "2", "index.html")
	if _, err := os.Stat(page2); os.IsNotExist(err) {
		t.Error("expected /page/2/index.html to exist")
	}
	// Verify /page/3/index.html exists.
	page3 := filepath.Join(dir, "page", "3", "index.html")
	if _, err := os.Stat(page3); os.IsNotExist(err) {
		t.Error("expected /page/3/index.html to exist")
	}
}

func TestRenderPaginatedIndex_DisabledWhenZero(t *testing.T) {
	renderer := testRenderer(t)
	dir := t.TempDir()
	now := time.Now()
	s := site.New()
	for i := 0; i < 5; i++ {
		s.AddPage(&site.Page{
			Title:     fmt.Sprintf("Post %d", i),
			Path:      fmt.Sprintf("/post-%d/", i),
			Permalink: fmt.Sprintf("http://example.com/post-%d/", i),
			Date:      now.Add(time.Duration(-i) * time.Hour),
			Content:   "<p>Hello</p>",
		})
	}
	s.BuildHierarchy()
	cfg := config.Config{
		BaseURL:         "http://example.com",
		PublicDir:       dir,
		PostsPerPage:    0, // disabled
		DefaultLanguage: "es",
	}
	baseCtx := baseContext(cfg, s.View(), map[string]*taxonomy.Index{}, nil)
	plan := buildPlan{full: true}

	rendered, _, err := renderSections(context.Background(), renderer, cfg, baseCtx, s, nil, plan)
	if err != nil {
		t.Fatalf("renderSections: %v", err)
	}
	// Should render a single index page (no pagination).
	if rendered != 1 {
		t.Errorf("rendered = %d, want 1 (no pagination when PostsPerPage=0)", rendered)
	}
}

func TestPaginatedSectionContext(t *testing.T) {
	cfg := config.Config{DefaultLanguage: "es"}
	base := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	section := &site.Section{
		Title:  "Home",
		Path:   "/",
		IsRoot: true,
		Pages: []*site.Page{
			{Title: "P1", Path: "/p1/", Permalink: "http://example.com/p1/"},
			{Title: "P2", Path: "/p2/", Permalink: "http://example.com/p2/"},
		},
	}
	paginator := taxonomy.Paginator{
		PaginateBy:   2,
		BaseURL:      "/",
		NumberPagers: 3,
		CurrentIndex: 1,
		TotalPages:   3,
		First:        "/",
		Last:         "/page/3/",
		Next:         "/page/2/",
		Pages:        section.Pages,
	}

	ctx := paginatedSectionContext(base, section, paginator)

	// Should have a paginator.
	pag, ok := ctx["paginator"].(map[string]any)
	if !ok {
		t.Fatal("expected paginator in context")
	}
	if pag["current_index"] != 1 {
		t.Errorf("current_index = %v, want 1", pag["current_index"])
	}
	if pag["total_pages"] != 3 {
		t.Errorf("total_pages = %v, want 3", pag["total_pages"])
	}
	if pag["next"] != "/page/2/" {
		t.Errorf("next = %v, want /page/2/", pag["next"])
	}

	// Section pages should be the paginated subset.
	sec, ok := ctx["section"].(map[string]any)
	if !ok {
		t.Fatal("expected section in context")
	}
	pages, ok := sec["pages"].([]map[string]any)
	if !ok {
		t.Fatal("expected pages as []map[string]any")
	}
	if len(pages) != 2 {
		t.Errorf("pages = %d, want 2", len(pages))
	}
}

func TestSectionFeedContext_EmptyTitle(t *testing.T) {
	cfg := config.Config{DefaultLanguage: "en"}
	base := baseContext(cfg, site.New().View(), map[string]*taxonomy.Index{}, nil)
	section := &site.Section{Slug: "notes", Path: "/notes/"}
	ctx := sectionFeedContext(base, cfg, section, nil, "http://example.com/notes/atom.xml", time.Now())
	if ctx["feed_title"] != "notes" {
		t.Errorf("feed_title = %v, want notes (fallback to slug)", ctx["feed_title"])
	}
}
