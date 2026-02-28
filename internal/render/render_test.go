package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
