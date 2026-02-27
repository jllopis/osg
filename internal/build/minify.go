package build

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

// mimeByExt maps file extensions to MIME types for minification.
var mimeByExt = map[string]string{
	".html": "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".svg":  "image/svg+xml",
	".xml":  "text/xml",
}

// newMinifier creates a minify.M configured for all supported types.
func newMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("application/json", json.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFunc("text/xml", xml.Minify)
	return m
}

// minifyDir walks dir and minifies all supported files in-place.
// Returns the number of files minified and any error.
func minifyDir(dir string, logger *slog.Logger) (int, error) {
	m := newMinifier()
	count := 0

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		mime, ok := mimeByExt[ext]
		if !ok {
			return nil
		}

		if err := minifyFile(m, path, mime); err != nil {
			logger.Warn("minify failed", "file", path, "error", err)
			// Non-fatal: skip and continue.
			return nil
		}

		count++
		return nil
	})

	return count, err
}

// minifyFile reads a file, minifies its content, and writes it back.
func minifyFile(m *minify.M, path string, mime string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	minified, err := m.Bytes(mime, data)
	if err != nil {
		return fmt.Errorf("minify %s: %w", path, err)
	}

	return os.WriteFile(path, minified, 0o644)
}
