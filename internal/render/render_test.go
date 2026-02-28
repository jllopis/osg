package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/i18n"
	"osg/internal/site"
	"osg/internal/taxonomy"
)

// ---- Template function tests ----

func TestBase64Encode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input any
		want  string
	}{
		{"hello", "aGVsbG8="},
		{[]byte("hello"), "aGVsbG8="},
		{nil, ""},
	}
	for _, tc := range tests {
		got, err := base64Encode(tc.input)
		if err != nil {
			t.Fatalf("base64Encode(%v) error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("base64Encode(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBase64EncodeUnsupportedType(t *testing.T) {
	t.Parallel()
	_, err := base64Encode(42)
	if err == nil {
		t.Error("expected error for int input")
	}
}

func TestBase64Decode(t *testing.T) {
	t.Parallel()
	got, err := base64Decode("aGVsbG8=")
	if err != nil {
		t.Fatalf("base64Decode error: %v", err)
	}
	if got != "hello" {
		t.Errorf("base64Decode = %q, want %q", got, "hello")
	}
}

func TestBase64DecodeNil(t *testing.T) {
	t.Parallel()
	got, err := base64Decode(nil)
	if err != nil {
		t.Fatalf("base64Decode(nil) error: %v", err)
	}
	if got != "" {
		t.Errorf("base64Decode(nil) = %q, want empty", got)
	}
}

func TestBase64DecodeInvalidInput(t *testing.T) {
	t.Parallel()
	_, err := base64Decode("not valid base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestRegexReplace(t *testing.T) {
	t.Parallel()
	got, err := regexReplace("hello world", `w\w+`, "Go")
	if err != nil {
		t.Fatalf("regexReplace error: %v", err)
	}
	if got != "hello Go" {
		t.Errorf("regexReplace = %q, want %q", got, "hello Go")
	}
}

func TestRegexReplaceNil(t *testing.T) {
	t.Parallel()
	got, err := regexReplace(nil, "pattern", "repl")
	if err != nil {
		t.Fatalf("regexReplace(nil) error: %v", err)
	}
	if got != "" {
		t.Errorf("regexReplace(nil) = %q, want empty", got)
	}
}

func TestRegexReplaceInvalidPattern(t *testing.T) {
	t.Parallel()
	_, err := regexReplace("input", "[invalid", "repl")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestNumFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  any
		locale string
		want   string
	}{
		{nil, "en", ""},
		{1234, "en", "1,234"},
		{int64(1234567), "en", "1,234,567"},
		{"", "en", ""},
		{"42", "en", "42"},
	}
	for _, tc := range tests {
		got, err := numFormat(tc.input, tc.locale)
		if err != nil {
			t.Fatalf("numFormat(%v, %q) error: %v", tc.input, tc.locale, err)
		}
		if got != tc.want {
			t.Errorf("numFormat(%v, %q) = %q, want %q", tc.input, tc.locale, got, tc.want)
		}
	}
}

func TestMarkdownFilter(t *testing.T) {
	t.Parallel()
	got, err := markdownFilter("**bold**")
	if err != nil {
		t.Fatalf("markdownFilter error: %v", err)
	}
	if !strings.Contains(string(got), "<strong>bold</strong>") {
		t.Errorf("markdownFilter = %q, want to contain <strong>bold</strong>", got)
	}
}

func TestMarkdownFilterNil(t *testing.T) {
	t.Parallel()
	got, err := markdownFilter(nil)
	if err != nil {
		t.Fatalf("markdownFilter(nil) error: %v", err)
	}
	if got != "" {
		t.Errorf("markdownFilter(nil) = %q, want empty", got)
	}
}

func TestMarkdownFilterBytes(t *testing.T) {
	t.Parallel()
	got, err := markdownFilter([]byte("*italic*"))
	if err != nil {
		t.Fatalf("markdownFilter(bytes) error: %v", err)
	}
	if !strings.Contains(string(got), "<em>italic</em>") {
		t.Errorf("markdownFilter(bytes) = %q, want <em>", got)
	}
}

func TestMarkdownFilterUnsupported(t *testing.T) {
	t.Parallel()
	_, err := markdownFilter(42)
	if err == nil {
		t.Error("expected error for int input")
	}
}

// ---- NormalizePath tests ----

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/tags/go/", "/tags/go/"},
		{"/tags/go", "/tags/go/"},
		{"tags/go", "/tags/go/"},
		{"/", "/"},
		{"", "/"},
	}
	for _, tc := range tests {
		got := normalizePath(tc.input)
		if got != tc.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---- TemplateLoader tests ----

func TestTemplateLoaderBuiltins(t *testing.T) {
	t.Parallel()
	loader := TemplateLoader{Funcs: FuncMap(Context{})}
	htmlTmpl, textTmpl, err := loader.Load()
	if err != nil {
		t.Fatalf("Load builtins failed: %v", err)
	}

	// Check that expected builtin HTML templates exist
	for _, name := range []string{"page.html", "index.html", "section.html", "404.html"} {
		if htmlTmpl.Lookup(name) == nil {
			t.Errorf("expected builtin html template %q to be loaded", name)
		}
	}
	// Check that expected builtin text templates exist (XML, TXT)
	for _, name := range []string{"sitemap.xml", "rss.xml", "atom.xml", "robots.txt"} {
		if textTmpl.Lookup(name) == nil {
			t.Errorf("expected builtin text template %q to be loaded", name)
		}
	}
}

func TestTemplateLoaderUserOverride(t *testing.T) {
	t.Parallel()
	userDir := t.TempDir()
	err := os.WriteFile(
		filepath.Join(userDir, "custom.html"),
		[]byte(`<p>custom template</p>`),
		0o644,
	)
	if err != nil {
		t.Fatal(err)
	}

	loader := TemplateLoader{UserDir: userDir, Funcs: FuncMap(Context{})}
	htmlTmpl, _, err := loader.Load()
	if err != nil {
		t.Fatalf("Load with user dir failed: %v", err)
	}

	if htmlTmpl.Lookup("custom.html") == nil {
		t.Error("expected custom.html template to be loaded from user dir")
	}
}

func TestTemplateLoaderThemeOverride(t *testing.T) {
	t.Parallel()
	themeDir := t.TempDir()
	// Create a subdirectory for partials
	partialsDir := filepath.Join(themeDir, "partials")
	_ = os.MkdirAll(partialsDir, 0o755)
	err := os.WriteFile(
		filepath.Join(partialsDir, "footer.html"),
		[]byte(`<footer>Theme footer</footer>`),
		0o644,
	)
	if err != nil {
		t.Fatal(err)
	}

	loader := TemplateLoader{ThemeDir: themeDir, Funcs: FuncMap(Context{})}
	htmlTmpl, _, err := loader.Load()
	if err != nil {
		t.Fatalf("Load with theme dir failed: %v", err)
	}

	if htmlTmpl.Lookup("partials/footer.html") == nil {
		t.Error("expected partials/footer.html from theme dir")
	}
}

// ---- Renderer tests ----

func TestRendererHasTemplate(t *testing.T) {
	t.Parallel()
	r, err := New("", "", Context{})
	if err != nil {
		t.Fatalf("New renderer failed: %v", err)
	}

	if !r.HasTemplate("page.html") {
		t.Error("expected page.html to exist")
	}
	if r.HasTemplate("nonexistent.html") {
		t.Error("expected nonexistent.html to not exist")
	}
}

func TestRendererRenderToFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a simple template in user dir
	userDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(userDir, 0o755)
	_ = os.WriteFile(
		filepath.Join(userDir, "test.html"),
		[]byte(`Title: {{ .title }}`),
		0o644,
	)

	r, err := New(userDir, "", Context{})
	if err != nil {
		t.Fatalf("New renderer failed: %v", err)
	}

	outPath := filepath.Join(tmpDir, "output", "test.html")
	err = r.RenderToFile("test.html", map[string]any{"title": "Hello"}, outPath)
	if err != nil {
		t.Fatalf("RenderToFile failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "Title: Hello" {
		t.Errorf("output = %q, want %q", string(data), "Title: Hello")
	}
}

func TestRendererRenderToFileMissingTemplate(t *testing.T) {
	t.Parallel()
	r, err := New("", "", Context{})
	if err != nil {
		t.Fatalf("New renderer failed: %v", err)
	}

	err = r.RenderToFile("nonexistent.html", nil, "/tmp/out.html")
	if err == nil {
		t.Error("expected error for missing template")
	}
}

// ---- BuiltinsSignature tests ----

func TestBuiltinsSignature(t *testing.T) {
	t.Parallel()
	sig, err := BuiltinsSignature()
	if err != nil {
		t.Fatalf("BuiltinsSignature error: %v", err)
	}
	if len(sig) != 64 {
		t.Errorf("expected SHA256 hex string (64 chars), got %d chars", len(sig))
	}

	// Should be deterministic
	sig2, _ := BuiltinsSignature()
	if sig != sig2 {
		t.Error("BuiltinsSignature should be deterministic")
	}
}

// ---- FuncMap tests ----

func TestFuncMap_ContainsExpectedFunctions(t *testing.T) {
	t.Parallel()
	fm := FuncMap(Context{})
	expected := []string{
		"markdown", "base64_encode", "base64_decode", "regex_replace",
		"num_format", "get_page", "get_section", "get_taxonomy_url",
		"get_taxonomy", "get_url", "get_hash", "get_image_metadata",
		"load_data", "trans", "date_format", "picture",
	}
	for _, name := range expected {
		if fm[name] == nil {
			t.Errorf("FuncMap missing %q", name)
		}
	}
}

// ---------------------------------------------------------------------------
// detectFormat
// ---------------------------------------------------------------------------

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		override string
		want     string
	}{
		{"override wins", "data.csv", "json", "json"},
		{"override trimmed", "data.csv", "  JSON  ", "json"},
		{"from extension", "data.csv", "", "csv"},
		{"json extension", "file.json", "", "json"},
		{"xml extension", "feed.xml", "", "xml"},
		{"no extension", "noext", "", ""},
		{"empty input", "", "", ""},
		{"uppercase extension", "FILE.CSV", "", "csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(tt.input, tt.override)
			if got != tt.want {
				t.Errorf("detectFormat(%q, %q) = %q, want %q", tt.input, tt.override, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseCSV
// ---------------------------------------------------------------------------

func TestParseCSV(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		data := []byte("name,age\nAlice,30\nBob,25\n")
		got, err := parseCSV(data)
		if err != nil {
			t.Fatal(err)
		}
		rows, ok := got.([]map[string]string)
		if !ok {
			t.Fatalf("expected []map[string]string, got %T", got)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		if rows[0]["name"] != "Alice" || rows[0]["age"] != "30" {
			t.Errorf("row[0] = %v", rows[0])
		}
	})

	t.Run("header only", func(t *testing.T) {
		data := []byte("name,age\n")
		got, err := parseCSV(data)
		if err != nil {
			t.Fatal(err)
		}
		rows := got.([]map[string]string)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows for header-only, got %d", len(rows))
		}
	})

	t.Run("empty", func(t *testing.T) {
		got, err := parseCSV([]byte(""))
		if err != nil {
			t.Fatal(err)
		}
		rows := got.([]map[string]string)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows for empty, got %d", len(rows))
		}
	})
}

// ---------------------------------------------------------------------------
// hashBytes
// ---------------------------------------------------------------------------

func TestHashBytes(t *testing.T) {
	data := []byte("hello")

	t.Run("sha256 default", func(t *testing.T) {
		h1, err := hashBytes(data, "")
		if err != nil {
			t.Fatal(err)
		}
		h2, err := hashBytes(data, "sha256")
		if err != nil {
			t.Fatal(err)
		}
		if string(h1) != string(h2) {
			t.Error("empty algo should default to sha256")
		}
		if len(h1) != 32 {
			t.Errorf("sha256 should produce 32 bytes, got %d", len(h1))
		}
	})

	t.Run("sha1", func(t *testing.T) {
		h, err := hashBytes(data, "sha1")
		if err != nil {
			t.Fatal(err)
		}
		if len(h) != 20 {
			t.Errorf("sha1 should produce 20 bytes, got %d", len(h))
		}
	})

	t.Run("md5", func(t *testing.T) {
		h, err := hashBytes(data, "md5")
		if err != nil {
			t.Fatal(err)
		}
		if len(h) != 16 {
			t.Errorf("md5 should produce 16 bytes, got %d", len(h))
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		h1, _ := hashBytes(data, "sha256")
		h2, _ := hashBytes(data, "sha256")
		if string(h1) != string(h2) {
			t.Error("same input should produce same hash")
		}
	})
}

// ---------------------------------------------------------------------------
// boolArg / stringArg
// ---------------------------------------------------------------------------

func TestBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		index    int
		fallback bool
		want     bool
	}{
		{"bool true", []any{true}, 0, false, true},
		{"bool false", []any{false}, 0, true, false},
		{"string true", []any{"true"}, 0, false, true},
		{"string TRUE", []any{"TRUE"}, 0, false, true},
		{"string false", []any{"false"}, 0, true, false},
		{"out of range", []any{}, 0, true, true},
		{"non-bool fallback", []any{42}, 0, true, true},
		{"index 1", []any{"x", true}, 1, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolArg(tt.args, tt.index, tt.fallback)
			if got != tt.want {
				t.Errorf("boolArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		index    int
		fallback string
		want     string
	}{
		{"found", []any{"hello"}, 0, "default", "hello"},
		{"trimmed", []any{"  hello  "}, 0, "", "hello"},
		{"out of range", []any{}, 0, "default", "default"},
		{"non-string fallback", []any{42}, 0, "default", "default"},
		{"index 1", []any{"a", "b"}, 1, "", "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringArg(tt.args, tt.index, tt.fallback)
			if got != tt.want {
				t.Errorf("stringArg() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildURL (render-local version)
// ---------------------------------------------------------------------------

func TestBuildURL_Render(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{"normal", "https://example.com", "/blog/", "https://example.com/blog/"},
		{"trailing slash", "https://example.com/", "/blog/", "https://example.com/blog/"},
		{"empty base", "", "/blog/", "/blog/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildURL(tt.baseURL, tt.path)
			if got != tt.want {
				t.Errorf("buildURL(%q, %q) = %q, want %q", tt.baseURL, tt.path, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ensureTrailingSlash (render-local version)
// ---------------------------------------------------------------------------

func TestEnsureTrailingSlash_Render(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/blog", "/blog/"},
		{"/blog/", "/blog/"},
		{"", "/"},
		{"/", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := ensureTrailingSlash(tt.in)
			if got != tt.want {
				t.Errorf("ensureTrailingSlash(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// transFunc
// ---------------------------------------------------------------------------

func TestTransFunc_NilBundle(t *testing.T) {
	t.Parallel()
	fn := transFunc(Context{})
	// With nil bundle, should return the key itself.
	got := fn("hello")
	if got != "hello" {
		t.Errorf("transFunc(nil bundle) = %q, want %q", got, "hello")
	}
}

func TestTransFunc_WithBundle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Write a minimal i18n YAML file.
	i18nDir := filepath.Join(dir, "i18n")
	_ = os.MkdirAll(i18nDir, 0o755)
	_ = os.WriteFile(filepath.Join(i18nDir, "es.yaml"), []byte("greeting: Hola\n"), 0o644)
	_ = os.WriteFile(filepath.Join(i18nDir, "en.yaml"), []byte("greeting: Hello\n"), 0o644)

	bundle := loadTestBundle(t, i18nDir, "es")
	fn := transFunc(Context{I18n: bundle, DefaultLanguage: "es"})

	// Default language.
	got := fn("greeting")
	if got != "Hola" {
		t.Errorf("trans('greeting') = %q, want 'Hola'", got)
	}

	// Explicit language override.
	got = fn("greeting", "en")
	if got != "Hello" {
		t.Errorf("trans('greeting', 'en') = %q, want 'Hello'", got)
	}

	// Unknown key returns the key.
	got = fn("unknown_key")
	if got != "unknown_key" {
		t.Errorf("trans('unknown_key') = %q, want 'unknown_key'", got)
	}
}

// ---------------------------------------------------------------------------
// dateFormatFunc
// ---------------------------------------------------------------------------

func TestDateFormatFunc_DefaultLang(t *testing.T) {
	t.Parallel()
	fn := dateFormatFunc(Context{DefaultLanguage: "en"})

	date := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
	got := fn(date, "2006-01-02")
	if got != "2025-01-15" {
		t.Errorf("dateFormatFunc = %q, want '2025-01-15'", got)
	}
}

func TestDateFormatFunc_ExplicitLang(t *testing.T) {
	t.Parallel()
	fn := dateFormatFunc(Context{DefaultLanguage: "en"})

	date := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
	// Explicit language override — should still format the date.
	got := fn(date, "2006-01-02", "es")
	if got != "2025-01-15" {
		t.Errorf("dateFormatFunc(es) = %q, want '2025-01-15'", got)
	}
}

// ---------------------------------------------------------------------------
// getPageFunc
// ---------------------------------------------------------------------------

func TestGetPageFunc_Found(t *testing.T) {
	t.Parallel()
	page := &site.Page{
		Title: "Test Page",
		Path:  "/blog/test/",
	}
	index := map[string]*site.Page{"/blog/test/": page}
	fn := getPageFunc(index)

	got := fn("/blog/test/")
	if got == nil {
		t.Fatal("expected page view, got nil")
	}
	if got["title"] != "Test Page" {
		t.Errorf("title = %v, want 'Test Page'", got["title"])
	}
}

func TestGetPageFunc_NotFound(t *testing.T) {
	t.Parallel()
	fn := getPageFunc(map[string]*site.Page{})

	got := fn("/nonexistent/")
	if got != nil {
		t.Errorf("expected nil for missing page, got %v", got)
	}
}

func TestGetPageFunc_NormalizesPath(t *testing.T) {
	t.Parallel()
	page := &site.Page{
		Title: "Normalized",
		Path:  "/test/",
	}
	index := map[string]*site.Page{"/test/": page}
	fn := getPageFunc(index)

	// Without trailing slash — should be normalized.
	got := fn("test")
	if got == nil {
		t.Fatal("expected page view after normalization, got nil")
	}
}

// ---------------------------------------------------------------------------
// getSectionFunc
// ---------------------------------------------------------------------------

func TestGetSectionFunc_Found(t *testing.T) {
	t.Parallel()
	sec := &site.Section{
		Title: "Blog",
		Path:  "/blog/",
	}
	index := map[string]*site.Section{"/blog/": sec}
	fn := getSectionFunc(index)

	got := fn("/blog/")
	if got == nil {
		t.Fatal("expected section view, got nil")
	}
}

func TestGetSectionFunc_NotFound(t *testing.T) {
	t.Parallel()
	fn := getSectionFunc(map[string]*site.Section{})
	got := fn("/missing/")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetSectionFunc_MetadataOnly(t *testing.T) {
	t.Parallel()
	sec := &site.Section{
		Title: "Blog",
		Path:  "/blog/",
	}
	index := map[string]*site.Section{"/blog/": sec}
	fn := getSectionFunc(index)

	got := fn("/blog/", true)
	if got == nil {
		t.Fatal("expected section view, got nil")
	}
	// With metadataOnly=true, pages and subsections should be empty.
	pages, ok := got["pages"].([]map[string]any)
	if !ok {
		t.Fatalf("pages type = %T, want []map[string]any", got["pages"])
	}
	if len(pages) != 0 {
		t.Errorf("expected empty pages with metadataOnly, got %d", len(pages))
	}
}

// ---------------------------------------------------------------------------
// getTaxonomyURLFunc
// ---------------------------------------------------------------------------

func TestGetTaxonomyURLFunc_KnownTerm(t *testing.T) {
	t.Parallel()
	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags"},
		Terms: map[string]*taxonomy.Term{
			"go": {Slug: "go", Permalink: "/tags/go/"},
		},
	}
	ctx := Context{
		BaseURL:    "https://example.com",
		Taxonomies: map[string]*taxonomy.Index{"tags": idx},
	}
	fn := getTaxonomyURLFunc(ctx)

	got := fn("tags", "go")
	if got != "/tags/go/" {
		t.Errorf("getTaxonomyURL('tags','go') = %q, want '/tags/go/'", got)
	}
}

func TestGetTaxonomyURLFunc_UnknownTerm(t *testing.T) {
	t.Parallel()
	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags"},
		Terms:  map[string]*taxonomy.Term{},
	}
	ctx := Context{
		BaseURL:    "https://example.com",
		Taxonomies: map[string]*taxonomy.Index{"tags": idx},
	}
	fn := getTaxonomyURLFunc(ctx)

	got := fn("tags", "unknown")
	if got != "https://example.com/tags/unknown/" {
		t.Errorf("getTaxonomyURL('tags','unknown') = %q, want fallback URL", got)
	}
}

func TestGetTaxonomyURLFunc_UnknownKind(t *testing.T) {
	t.Parallel()
	ctx := Context{Taxonomies: map[string]*taxonomy.Index{}}
	fn := getTaxonomyURLFunc(ctx)

	got := fn("categories", "test")
	if got != "" {
		t.Errorf("expected empty string for unknown kind, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// getTaxonomyFunc
// ---------------------------------------------------------------------------

func TestGetTaxonomyFunc_Found(t *testing.T) {
	t.Parallel()
	idx := &taxonomy.Index{
		Config: config.TaxonomyConfig{Name: "tags"},
		Terms:  map[string]*taxonomy.Term{},
	}
	ctx := Context{Taxonomies: map[string]*taxonomy.Index{"tags": idx}}
	fn := getTaxonomyFunc(ctx)

	got := fn("tags")
	if got == nil {
		t.Fatal("expected taxonomy view, got nil")
	}
	if got["taxonomy"] == nil {
		t.Error("expected 'taxonomy' key in view")
	}
	if got["terms"] == nil {
		t.Error("expected 'terms' key in view")
	}
}

func TestGetTaxonomyFunc_NotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{Taxonomies: map[string]*taxonomy.Index{}}
	fn := getTaxonomyFunc(ctx)

	got := fn("nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown taxonomy, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// getURLFunc
// ---------------------------------------------------------------------------

func TestGetURLFunc_Plain(t *testing.T) {
	t.Parallel()
	ctx := Context{BaseURL: "https://example.com"}
	fn := getURLFunc(ctx)

	got := fn("/blog/")
	if got != "https://example.com/blog/" {
		t.Errorf("getURL('/blog/') = %q", got)
	}
}

func TestGetURLFunc_EmptyBase(t *testing.T) {
	t.Parallel()
	fn := getURLFunc(Context{})

	got := fn("/blog/")
	if got != "/blog/" {
		t.Errorf("getURL('/blog/') with empty base = %q", got)
	}
}

func TestGetURLFunc_TrailingSlash(t *testing.T) {
	t.Parallel()
	fn := getURLFunc(Context{BaseURL: "https://example.com"})

	got := fn("/blog", true)
	if got != "https://example.com/blog/" {
		t.Errorf("getURL with trailing = %q", got)
	}
}

// ---------------------------------------------------------------------------
// pictureFunc
// ---------------------------------------------------------------------------

func TestPictureFunc_BasicSrc(t *testing.T) {
	t.Parallel()
	fn := pictureFunc(Context{})

	got := fn("/img/photo.jpg")
	html := string(got)
	if !strings.Contains(html, "photo.jpg") {
		t.Errorf("pictureFunc output = %q, expected to contain 'photo.jpg'", html)
	}
}

func TestPictureFunc_WithAlt(t *testing.T) {
	t.Parallel()
	fn := pictureFunc(Context{})

	got := fn("/img/photo.jpg", "A nice photo")
	html := string(got)
	if !strings.Contains(html, "A nice photo") {
		t.Errorf("pictureFunc output = %q, expected alt text", html)
	}
}

func TestPictureFunc_WithLoading(t *testing.T) {
	t.Parallel()
	fn := pictureFunc(Context{})

	got := fn("/img/photo.jpg", "alt", "eager")
	html := string(got)
	if !strings.Contains(html, "eager") {
		t.Errorf("pictureFunc output = %q, expected loading='eager'", html)
	}
}

// ---------------------------------------------------------------------------
// parseXML
// ---------------------------------------------------------------------------

func TestParseXML_Valid(t *testing.T) {
	t.Parallel()
	data := []byte(`<root><child>text</child></root>`)
	got, err := parseXML(data)
	if err != nil {
		t.Fatalf("parseXML error: %v", err)
	}
	node, ok := got.(xmlNode)
	if !ok {
		t.Fatalf("expected xmlNode, got %T", got)
	}
	if node.XMLName.Local != "root" {
		t.Errorf("root name = %q, want 'root'", node.XMLName.Local)
	}
}

func TestParseXML_Invalid(t *testing.T) {
	t.Parallel()
	_, err := parseXML([]byte(`<not closed`))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

// ---------------------------------------------------------------------------
// resolveFilePath
// ---------------------------------------------------------------------------

func TestResolveFilePath_InPublicDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pubDir := filepath.Join(dir, "public")
	_ = os.MkdirAll(pubDir, 0o755)
	_ = os.WriteFile(filepath.Join(pubDir, "style.css"), []byte("body{}"), 0o644)

	ctx := Context{PublicDir: pubDir}
	got, err := resolveFilePath(ctx, "/style.css")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != filepath.Join(pubDir, "style.css") {
		t.Errorf("resolved = %q, want %q", got, filepath.Join(pubDir, "style.css"))
	}
}

func TestResolveFilePath_InStaticDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	staticDir := filepath.Join(dir, "static")
	_ = os.MkdirAll(staticDir, 0o755)
	_ = os.WriteFile(filepath.Join(staticDir, "logo.png"), []byte("PNG"), 0o644)

	ctx := Context{StaticDir: staticDir}
	got, err := resolveFilePath(ctx, "/logo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != filepath.Join(staticDir, "logo.png") {
		t.Errorf("resolved = %q, want %q", got, filepath.Join(staticDir, "logo.png"))
	}
}

func TestResolveFilePath_Missing(t *testing.T) {
	t.Parallel()
	ctx := Context{PublicDir: "/nonexistent"}
	_, err := resolveFilePath(ctx, "/missing.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestResolveFilePath_Empty(t *testing.T) {
	t.Parallel()
	_, err := resolveFilePath(Context{}, "")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

// ---------------------------------------------------------------------------
// loadDataFunc
// ---------------------------------------------------------------------------

func TestLoadDataFunc_LocalJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"key":"value"}`), 0o644)

	ctx := Context{PublicDir: dir}
	fn := loadDataFunc(ctx)

	got, err := fn("/data.json")
	if err != nil {
		t.Fatalf("load_data error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want 'value'", m["key"])
	}
}

func TestLoadDataFunc_LocalYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("key: value\n"), 0o644)

	ctx := Context{PublicDir: dir}
	fn := loadDataFunc(ctx)

	got, err := fn("/data.yaml")
	if err != nil {
		t.Fatalf("load_data error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want 'value'", m["key"])
	}
}

func TestLoadDataFunc_MissingOptional(t *testing.T) {
	t.Parallel()
	ctx := Context{PublicDir: "/nonexistent"}
	fn := loadDataFunc(ctx)

	// required=false (default) — should return nil, no error.
	got, err := fn("/missing.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing optional, got %v", got)
	}
}

func TestLoadDataFunc_MissingRequired(t *testing.T) {
	t.Parallel()
	ctx := Context{PublicDir: "/nonexistent"}
	fn := loadDataFunc(ctx)

	// required=true — should return error.
	_, err := fn("/missing.json", "", true)
	if err == nil {
		t.Error("expected error for required missing file")
	}
}

func TestLoadDataFunc_CSV(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "data.csv"), []byte("name,age\nAlice,30\n"), 0o644)

	ctx := Context{PublicDir: dir}
	fn := loadDataFunc(ctx)

	got, err := fn("/data.csv")
	if err != nil {
		t.Fatalf("load_data csv error: %v", err)
	}
	rows, ok := got.([]map[string]string)
	if !ok {
		t.Fatalf("expected []map[string]string, got %T", got)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// hashForPath
// ---------------------------------------------------------------------------

func TestHashForPath_FileExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644)

	ctx := Context{PublicDir: dir}
	hash, err := hashForPath(ctx, "/file.txt", "sha256", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	// Hash should be hex-encoded.
	if len(hash) != 64 {
		t.Errorf("expected 64-char sha256 hex, got %d chars", len(hash))
	}
}

func TestHashForPath_FileExists_Base64(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644)

	ctx := Context{PublicDir: dir}
	hash, err := hashForPath(ctx, "/file.txt", "sha256", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	// Base64 SHA256 should end with = padding.
	if !strings.HasSuffix(hash, "=") {
		t.Errorf("expected base64 hash ending with '=', got %q", hash)
	}
}

func TestHashForPath_FallbackToStringHash(t *testing.T) {
	t.Parallel()
	ctx := Context{PublicDir: "/nonexistent"}
	// When file not found, hashes the input string itself.
	hash, err := hashForPath(ctx, "some-string", "md5", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for string fallback")
	}
	if len(hash) != 32 {
		t.Errorf("expected 32-char md5 hex, got %d chars", len(hash))
	}
}

// ---------------------------------------------------------------------------
// helper
// ---------------------------------------------------------------------------

func loadTestBundle(t *testing.T, dir string, defaultLang string) *i18n.Bundle {
	t.Helper()
	bundle := i18n.New(defaultLang)
	if err := bundle.LoadDir(dir); err != nil {
		t.Fatalf("loadDir: %v", err)
	}
	return bundle
}
