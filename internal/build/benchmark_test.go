package build

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleHTML is a realistic rendered page for benchmarking minification.
var sampleHTML = `<!doctype html>
<html lang="es" data-color-scheme="auto">
  <head>
    <title>Test Article &mdash; My Site</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="/style.css?v=abc123">
  </head>
  <body>
    <div class="page">
      <header class="site-header">
        <nav class="container">
          <a href="/" class="brand">My Site</a>
          <div class="nav-links">
            <a href="/tags/">Tags</a>
            <a href="/categories/">Categories</a>
          </div>
        </nav>
      </header>
      <main id="main-content" role="main">
        <article class="article">
          <header class="article-header container-narrow">
            <h1>Test Article Title</h1>
            <div class="meta">
              <time datetime="2025-01-15">15 de enero de 2025</time>
              <span class="reading-badge">5 min</span>
            </div>
          </header>
          <div class="article-content container-narrow">
            <div class="prose">` + strings.Repeat(`
              <p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do
              eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>
              <h2 id="section-one">Section One</h2>
              <p>Ut enim ad minim veniam, quis nostrud exercitation ullamco
              laboris nisi ut aliquip ex ea commodo consequat.</p>
              <pre><code class="language-go">func main() {
    fmt.Println("hello world")
}</code></pre>
              <blockquote><p>A notable quote from someone important.</p></blockquote>
              <ul>
                <li>First item</li>
                <li>Second item with <strong>bold</strong></li>
                <li>Third item with <a href="https://example.com">link</a></li>
              </ul>`, 5) + `
            </div>
          </div>
        </article>
      </main>
      <footer class="site-footer">
        <div class="container">
          <p>&copy; 2025 My Site</p>
        </div>
      </footer>
    </div>
  </body>
</html>`

func BenchmarkMinifyDir(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for b.Loop() {
		// Each iteration needs a fresh temp dir with files.
		dir := b.TempDir()
		for j := range 20 {
			name := filepath.Join(dir, "page"+string(rune('0'+j/10))+string(rune('0'+j%10))+".html")
			_ = os.WriteFile(name, []byte(sampleHTML), 0o644)
		}
		// Also add a CSS file.
		_ = os.WriteFile(filepath.Join(dir, "style.css"), []byte(`
			body { margin: 0; padding: 0; font-family: sans-serif; }
			.container { width: min(1080px, 100%); margin-inline: auto; }
			.prose { max-width: 80ch; color: #333; font-size: 20px; line-height: 1.8; }
		`), 0o644)

		_, _ = minifyDir(dir, logger)
	}
}

func BenchmarkMinifyFile_HTML(b *testing.B) {
	m := newMinifier()
	dir := b.TempDir()
	path := filepath.Join(dir, "test.html")
	_ = os.WriteFile(path, []byte(sampleHTML), 0o644)

	for b.Loop() {
		// Re-write the file each iteration since minifyFile overwrites it.
		_ = os.WriteFile(path, []byte(sampleHTML), 0o644)
		_ = minifyFile(m, path, "text/html")
	}
}

func BenchmarkTimingStage(b *testing.B) {
	for b.Loop() {
		bt := &BuildTimings{}
		done := bt.stage("test")
		// Simulate minimal work.
		_ = 1 + 1
		done()
	}
}
