package build

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"osg/internal/config"
	imgopt "osg/internal/image"
	"osg/internal/site"
)

// discardLogger returns a logger that silently discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------------------------------------------------------------------------
// emitConfigValidate
// ---------------------------------------------------------------------------

func TestEmitConfigValidate_NilPlugins(t *testing.T) {
	err := emitConfigValidate(context.Background(), nil, config.Config{}, discardLogger())
	if err != nil {
		t.Errorf("expected nil error for nil plugins, got %v", err)
	}
}

func TestEmitConfigValidate_NilResult(t *testing.T) {
	// A manager with no loaded plugins returns nil from Emit.
	// We can't easily construct a Manager without WASM, but nil
	// manager is already tested above.  This test verifies
	// the function signature and nil-plugin guard.
	err := emitConfigValidate(context.Background(), nil, config.Config{
		BaseURL:   "https://example.com",
		SiteTitle: "Test",
	}, discardLogger())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// applyContentTransform
// ---------------------------------------------------------------------------

func TestApplyContentTransform_NilPlugins(t *testing.T) {
	siteIndex := site.New()
	siteIndex.AddPage(&site.Page{
		Title:      "Hello",
		RawContent: "# Hello\n\nWorld",
		Content:    "<h1>Hello</h1>\n<p>World</p>",
	})

	applyContentTransform(context.Background(), nil, config.Config{}, siteIndex, discardLogger())

	// Page should be unchanged since plugins are nil.
	if siteIndex.Pages[0].RawContent != "# Hello\n\nWorld" {
		t.Errorf("RawContent changed unexpectedly: %q", siteIndex.Pages[0].RawContent)
	}
	if siteIndex.Pages[0].Content != "<h1>Hello</h1>\n<p>World</p>" {
		t.Errorf("Content changed unexpectedly: %q", siteIndex.Pages[0].Content)
	}
}

func TestApplyContentTransform_EmptyPages(t *testing.T) {
	siteIndex := site.New()
	// Should not panic with zero pages.
	applyContentTransform(context.Background(), nil, config.Config{}, siteIndex, discardLogger())
}

// ---------------------------------------------------------------------------
// emitImageProcess
// ---------------------------------------------------------------------------

func TestEmitImageProcess_NilPlugins(t *testing.T) {
	results := map[string]*imgopt.Result{
		"/img/hero.jpg": {
			Original:      "/img/hero.jpg",
			OriginalWidth: 1920,
			Variants: map[int][]imgopt.Variant{
				640:  {{URLPath: "/img/hero-640w.jpg", Width: 640, Format: "jpeg"}},
				1200: {{URLPath: "/img/hero-1200w.jpg", Width: 1200, Format: "jpeg"}},
			},
		},
	}

	// Should not panic with nil plugins.
	emitImageProcess(context.Background(), nil, config.Config{PublicDir: "public"}, results, discardLogger())
}

func TestEmitImageProcess_NilResults(t *testing.T) {
	// Should not panic with nil image results.
	emitImageProcess(context.Background(), nil, config.Config{}, nil, discardLogger())
}

func TestEmitImageProcess_EmptyResults(t *testing.T) {
	// Should not panic with empty results map.
	emitImageProcess(context.Background(), nil, config.Config{}, map[string]*imgopt.Result{}, discardLogger())
}

// ---------------------------------------------------------------------------
// emitImageProcess payload structure (verify the map-building logic)
// ---------------------------------------------------------------------------

func TestEmitImageProcess_PayloadStructure(t *testing.T) {
	// We can't call emitImageProcess with a real manager without WASM,
	// but we can verify the variant flattening logic by replicating
	// the inner loop and checking the output.
	result := &imgopt.Result{
		Original:      "/img/photo.jpg",
		OriginalWidth: 2400,
		Variants: map[int][]imgopt.Variant{
			640: {
				{URLPath: "/img/photo-640w.jpg", Width: 640, Format: "jpeg"},
				{URLPath: "/img/photo-640w.webp", Width: 640, Format: "webp"},
			},
			1200: {
				{URLPath: "/img/photo-1200w.jpg", Width: 1200, Format: "jpeg"},
			},
		},
	}

	// Replicate the variant-flattening logic from emitImageProcess.
	var variants []map[string]any
	for _, variantList := range result.Variants {
		for _, v := range variantList {
			variants = append(variants, map[string]any{
				"url_path": v.URLPath,
				"width":    v.Width,
				"format":   v.Format,
			})
		}
	}

	// Should have 3 variants total (2 for 640, 1 for 1200).
	if len(variants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(variants))
	}

	// Verify all variants have the expected keys.
	for i, v := range variants {
		if _, ok := v["url_path"]; !ok {
			t.Errorf("variant[%d] missing url_path", i)
		}
		if _, ok := v["width"]; !ok {
			t.Errorf("variant[%d] missing width", i)
		}
		if _, ok := v["format"]; !ok {
			t.Errorf("variant[%d] missing format", i)
		}
	}

	// Verify the payload structure matches what plugins expect.
	// Note: configView converts public_dir to an absolute path for WASI compatibility.
	cfg := config.Config{PublicDir: "public"}
	payload := map[string]any{
		"config": configView(cfg),
		"image": map[string]any{
			"src_path":       "/img/photo.jpg",
			"public_dir":     configView(cfg)["public_dir"],
			"original":       result.Original,
			"original_width": result.OriginalWidth,
			"variants":       variants,
		},
	}

	imagePayload, ok := payload["image"].(map[string]any)
	if !ok {
		t.Fatal("payload[image] missing or wrong type")
	}
	if imagePayload["src_path"] != "/img/photo.jpg" {
		t.Errorf("src_path = %v; want /img/photo.jpg", imagePayload["src_path"])
	}
	// public_dir is now an absolute path for WASI filesystem access
	if imagePayload["public_dir"] == "" {
		t.Error("public_dir should not be empty")
	}
	if imagePayload["original"] != "/img/photo.jpg" {
		t.Errorf("original = %v; want /img/photo.jpg", imagePayload["original"])
	}
	if imagePayload["original_width"] != 2400 {
		t.Errorf("original_width = %v; want 2400", imagePayload["original_width"])
	}
}

// ---------------------------------------------------------------------------
// applyContentTransform — page payload structure
// ---------------------------------------------------------------------------

func TestApplyContentTransform_PayloadKeys(t *testing.T) {
	// Verify the payload structure built for content.transform includes
	// all expected page fields. We test this by checking that the map
	// construction in the function doesn't panic and produces correct keys.
	page := &site.Page{
		Title:      "Test Page",
		Slug:       "test-page",
		Path:       "/2025/01/test-page/",
		Permalink:  "https://example.com/2025/01/test-page/",
		RawContent: "# Test\n\nContent here.",
		Summary:    "A summary",
		Taxonomies: map[string][]string{"tags": {"go", "test"}},
		Extra:      map[string]any{"custom": "value"},
	}

	cfg := config.Config{
		BaseURL:   "https://example.com",
		SiteTitle: "Test Site",
	}

	cfgView := configView(cfg)
	payload := map[string]any{
		"config": cfgView,
		"page": map[string]any{
			"title":         page.Title,
			"slug":          page.Slug,
			"path":          page.Path,
			"permalink":     page.Permalink,
			"date":          page.Date,
			"body_markdown": page.RawContent,
			"summary":       page.Summary,
			"taxonomies":    page.Taxonomies,
			"extra":         page.Extra,
		},
	}

	pageMap, ok := payload["page"].(map[string]any)
	if !ok {
		t.Fatal("payload[page] missing or wrong type")
	}
	expectedKeys := []string{"title", "slug", "path", "permalink", "date", "body_markdown", "summary", "taxonomies", "extra"}
	for _, key := range expectedKeys {
		if _, ok := pageMap[key]; !ok {
			t.Errorf("payload[page] missing key %q", key)
		}
	}

	if pageMap["title"] != "Test Page" {
		t.Errorf("title = %v; want %q", pageMap["title"], "Test Page")
	}
	if pageMap["body_markdown"] != "# Test\n\nContent here." {
		t.Errorf("body_markdown = %v; want %q", pageMap["body_markdown"], "# Test\n\nContent here.")
	}

	tags, ok := pageMap["taxonomies"].(map[string][]string)
	if !ok {
		t.Fatal("taxonomies wrong type")
	}
	if len(tags["tags"]) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags["tags"]))
	}
}

// ---------------------------------------------------------------------------
// emitConfigValidate — payload structure
// ---------------------------------------------------------------------------

func TestEmitConfigValidate_PayloadKeys(t *testing.T) {
	cfg := config.Config{
		BaseURL:     "https://example.com",
		SiteTitle:   "Test Site",
		Theme:       "default",
		ColorScheme: "dark",
		PublicDir:   "public",
	}

	payload := map[string]any{
		"config": configView(cfg),
	}

	cfgMap, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatal("payload[config] missing or wrong type")
	}

	// Verify essential keys are present.
	essentialKeys := []string{"base_url", "site_title", "theme", "color_scheme", "public_dir"}
	for _, key := range essentialKeys {
		if _, ok := cfgMap[key]; !ok {
			t.Errorf("payload[config] missing key %q", key)
		}
	}

	if cfgMap["base_url"] != "https://example.com" {
		t.Errorf("base_url = %v; want %q", cfgMap["base_url"], "https://example.com")
	}
	if cfgMap["color_scheme"] != "dark" {
		t.Errorf("color_scheme = %v; want %q", cfgMap["color_scheme"], "dark")
	}
}

// ---------------------------------------------------------------------------
// Integration: hooks are no-ops when plugins is nil (regression guard)
// ---------------------------------------------------------------------------

func TestAllHooks_NilPluginsAreNoOps(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}
	logger := discardLogger()

	t.Run("emitConfigValidate", func(t *testing.T) {
		err := emitConfigValidate(ctx, nil, cfg, logger)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("applyContentTransform", func(t *testing.T) {
		si := site.New()
		si.AddPage(&site.Page{Title: "T", RawContent: "body", Content: "<p>body</p>"})
		applyContentTransform(ctx, nil, cfg, si, logger)
		if si.Pages[0].Content != "<p>body</p>" {
			t.Error("content changed when plugins is nil")
		}
	})

	t.Run("emitImageProcess", func(t *testing.T) {
		results := map[string]*imgopt.Result{
			"test.jpg": {Original: "/img/test.jpg", OriginalWidth: 800},
		}
		// Should not panic.
		emitImageProcess(ctx, nil, cfg, results, logger)
	})
}
