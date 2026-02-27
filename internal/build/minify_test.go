package build

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMinifyDir_HTMLFile(t *testing.T) {
	dir := t.TempDir()
	html := `<!doctype html>
<html>
  <head>
    <title>Test</title>
  </head>
  <body>
    <p>Hello   world</p>
  </body>
</html>`
	path := filepath.Join(dir, "index.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	count, err := minifyDir(dir, slog.Default())
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 minified, got %d", count)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	// Minified HTML should be shorter and have less whitespace.
	if len(result) >= len(html) {
		t.Errorf("expected minified output to be shorter than input")
	}
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected content preserved, got:\n%s", result)
	}
}

func TestMinifyDir_CSSFile(t *testing.T) {
	dir := t.TempDir()
	css := `body {
  margin: 0;
  padding: 0;
  color: #333;
}`
	path := filepath.Join(dir, "style.css")
	if err := os.WriteFile(path, []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}

	count, err := minifyDir(dir, slog.Default())
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if strings.Contains(result, "\n") {
		t.Errorf("expected minified CSS without newlines, got:\n%s", result)
	}
}

func TestMinifyDir_SkipsImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	count, err := minifyDir(dir, slog.Default())
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 minified (images skipped), got %d", count)
	}
}

func TestMinifyDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	count, err := minifyDir(dir, slog.Default())
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestMinifyDir_MultipleTypes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html> <body> </body> </html>"), 0o644)
	os.WriteFile(filepath.Join(dir, "app.js"), []byte("var x = 1 ;"), 0o644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{ "a" : 1 }`), 0o644)
	os.WriteFile(filepath.Join(dir, "feed.xml"), []byte(`<?xml version="1.0" ?>\n<root> </root>`), 0o644)

	count, err := minifyDir(dir, slog.Default())
	if err != nil {
		t.Fatalf("minifyDir error: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 minified, got %d", count)
	}
}
