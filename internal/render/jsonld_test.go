package render

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildArticleSchema(t *testing.T) {
	page := map[string]any{
		"title":      "My Post",
		"permalink":  "https://example.com/2024/my-post/",
		"summary":    "A great post",
		"image":      "/img/hero.jpg",
		"date":       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		"updated":    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		"author":     "Alice",
		"word_count": 500,
		"taxonomies": map[string][]string{"tags": {"go", "web"}},
	}

	schema := buildArticleSchema(page, "https://example.com", "My Site", "en", "Blog")
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	if schema["@type"] != "BlogPosting" {
		t.Errorf("expected BlogPosting, got %s", schema["@type"])
	}
	if schema["headline"] != "My Post" {
		t.Errorf("expected My Post, got %s", schema["headline"])
	}
	if schema["description"] != "A great post" {
		t.Errorf("expected summary, got %v", schema["description"])
	}
	// Image should be made absolute.
	if schema["image"] != "https://example.com/img/hero.jpg" {
		t.Errorf("expected absolute image URL, got %s", schema["image"])
	}
	if schema["datePublished"] != "2024-01-15T00:00:00Z" {
		t.Errorf("unexpected datePublished: %s", schema["datePublished"])
	}
	if schema["dateModified"] != "2024-02-01T00:00:00Z" {
		t.Errorf("unexpected dateModified: %s", schema["dateModified"])
	}
	if schema["wordCount"] != 500 {
		t.Errorf("expected wordCount 500, got %v", schema["wordCount"])
	}

	author, _ := schema["author"].(map[string]any)
	if author == nil || author["name"] != "Alice" {
		t.Errorf("expected author Alice, got %v", schema["author"])
	}

	publisher, _ := schema["publisher"].(map[string]any)
	if publisher == nil || publisher["name"] != "My Site" {
		t.Errorf("expected publisher My Site, got %v", schema["publisher"])
	}

	if schema["inLanguage"] != "en" {
		t.Errorf("expected inLanguage en, got %v", schema["inLanguage"])
	}
	if schema["articleSection"] != "Blog" {
		t.Errorf("expected articleSection Blog, got %v", schema["articleSection"])
	}
	if schema["keywords"] != "go, web" {
		t.Errorf("expected keywords 'go, web', got %v", schema["keywords"])
	}

	// Verify it's valid JSON.
	if _, err := json.Marshal(schema); err != nil {
		t.Errorf("schema is not valid JSON: %v", err)
	}
}

func TestBuildArticleSchema_EmptyTitle(t *testing.T) {
	page := map[string]any{"title": ""}
	schema := buildArticleSchema(page, "https://example.com", "Site", "", "")
	if schema != nil {
		t.Error("expected nil for empty title")
	}
}

func TestBuildArticleSchema_MinimalFields(t *testing.T) {
	page := map[string]any{"title": "Minimal"}
	schema := buildArticleSchema(page, "", "", "", "")
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
	if schema["headline"] != "Minimal" {
		t.Errorf("expected Minimal, got %s", schema["headline"])
	}
	// No publisher when siteTitle is empty.
	if _, ok := schema["publisher"]; ok {
		t.Error("expected no publisher for empty siteTitle")
	}
}

func TestBuildArticleSchema_ExternalImage(t *testing.T) {
	page := map[string]any{
		"title": "Post",
		"image": "https://cdn.example.com/photo.jpg",
	}
	schema := buildArticleSchema(page, "https://example.com", "", "", "")
	// External image should be left as-is.
	if schema["image"] != "https://cdn.example.com/photo.jpg" {
		t.Errorf("expected external URL preserved, got %s", schema["image"])
	}
}

func TestBuildWebSiteSchema(t *testing.T) {
	schema := buildWebSiteSchema("https://example.com", "My Site", "A description", "en")
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	if schema["@type"] != "WebSite" {
		t.Errorf("expected WebSite, got %s", schema["@type"])
	}
	if schema["url"] != "https://example.com" {
		t.Errorf("expected base URL, got %s", schema["url"])
	}
	if schema["name"] != "My Site" {
		t.Errorf("expected My Site, got %s", schema["name"])
	}
	if schema["description"] != "A description" {
		t.Errorf("expected description, got %v", schema["description"])
	}

	action, _ := schema["potentialAction"].(map[string]any)
	if action == nil {
		t.Fatal("expected SearchAction")
	}
	if action["@type"] != "SearchAction" {
		t.Errorf("expected SearchAction type, got %s", action["@type"])
	}
	target, _ := action["target"].(string)
	if !strings.Contains(target, "/search/") {
		t.Errorf("expected search target, got %s", target)
	}
	if schema["inLanguage"] != "en" {
		t.Errorf("expected inLanguage en, got %v", schema["inLanguage"])
	}
}

func TestBuildWebSiteSchema_EmptyBaseURL(t *testing.T) {
	schema := buildWebSiteSchema("", "Site", "Desc", "")
	if schema != nil {
		t.Error("expected nil for empty baseURL")
	}
}

func TestBuildBreadcrumbSchema(t *testing.T) {
	page := map[string]any{
		"title":     "Post Title",
		"path":      "/blog/2024/my-post/",
		"permalink": "https://example.com/blog/2024/my-post/",
	}

	schema := buildBreadcrumbSchema(page, "https://example.com", "Home")
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	if schema["@type"] != "BreadcrumbList" {
		t.Errorf("expected BreadcrumbList, got %s", schema["@type"])
	}

	items, _ := schema["itemListElement"].([]map[string]any)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 breadcrumb items, got %d", len(items))
	}

	// First item: Home.
	if items[0]["name"] != "Home" {
		t.Errorf("expected Home, got %s", items[0]["name"])
	}
	if items[0]["position"] != 1 {
		t.Errorf("expected position 1, got %v", items[0]["position"])
	}

	// Last item: page title.
	last := items[len(items)-1]
	if last["name"] != "Post Title" {
		t.Errorf("expected Post Title, got %s", last["name"])
	}
}

func TestBuildBreadcrumbSchema_RootPage(t *testing.T) {
	page := map[string]any{"path": "/", "title": "Home"}
	schema := buildBreadcrumbSchema(page, "https://example.com", "Home")
	if schema != nil {
		t.Error("expected nil for root page")
	}
}

func TestBuildBreadcrumbSchema_EmptyBaseURL(t *testing.T) {
	page := map[string]any{"path": "/blog/post/", "title": "Post"}
	schema := buildBreadcrumbSchema(page, "", "Home")
	if schema != nil {
		t.Error("expected nil for empty baseURL")
	}
}

func TestJsonldFunc_PageContext(t *testing.T) {
	ctx := Context{BaseURL: "https://example.com"}
	fn := jsonldFunc(ctx)

	data := map[string]any{
		"config": map[string]any{
			"base_url":   "https://example.com",
			"site_title": "Test Site",
		},
		"page": map[string]any{
			"title":     "Test Post",
			"permalink": "https://example.com/test/",
			"path":      "/test/",
			"date":      time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	result := fn(data)
	html := string(result)

	if !strings.Contains(html, `"@type":"BlogPosting"`) {
		t.Error("expected BlogPosting in output")
	}
	if !strings.Contains(html, `"@type":"BreadcrumbList"`) {
		t.Error("expected BreadcrumbList in output")
	}
	if !strings.Contains(html, `<script type="application/ld+json">`) {
		t.Error("expected script tag in output")
	}
}

func TestJsonldFunc_IndexContext(t *testing.T) {
	ctx := Context{BaseURL: "https://example.com"}
	fn := jsonldFunc(ctx)

	data := map[string]any{
		"config": map[string]any{
			"base_url":         "https://example.com",
			"site_title":       "Test Site",
			"site_description": "A test site",
		},
		"current_path": "/",
	}

	result := fn(data)
	html := string(result)

	if !strings.Contains(html, `"@type":"WebSite"`) {
		t.Error("expected WebSite in output")
	}
	if !strings.Contains(html, `SearchAction`) {
		t.Error("expected SearchAction in output")
	}
}

func TestJsonldFunc_NoConfig(t *testing.T) {
	ctx := Context{}
	fn := jsonldFunc(ctx)

	result := fn(map[string]any{})
	if result != "" {
		t.Errorf("expected empty output for no config, got %s", result)
	}
}

func TestJsonldFunc_SectionContext(t *testing.T) {
	ctx := Context{}
	fn := jsonldFunc(ctx)

	// Non-root section without .page — should produce no output.
	data := map[string]any{
		"config": map[string]any{
			"base_url":   "https://example.com",
			"site_title": "Test",
		},
		"current_path": "/blog/",
		"section":      map[string]any{"title": "Blog"},
	}

	result := fn(data)
	if result != "" {
		t.Errorf("expected no JSON-LD for non-root section, got %s", result)
	}
}
