package build

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

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

// minifyJob is a unit of work for the parallel minifier.
type minifyJob struct {
	path string
	mime string
}

// minifyDir walks dir and minifies all supported files in-place using a
// parallel worker pool.  Returns the number of files minified and any error.
func minifyDir(dir string, logger *slog.Logger) (int, error) {
	m := newMinifier()

	// Phase 1: discover all minifiable files.
	var jobs []minifyJob
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

		jobs = append(jobs, minifyJob{path: path, mime: mime})
		return nil
	})
	if err != nil {
		return 0, err
	}

	if len(jobs) == 0 {
		return 0, nil
	}

	// Phase 2: minify in parallel.
	workers := runtime.NumCPU()
	if workers > len(jobs) {
		workers = len(jobs)
	}

	var count atomic.Int64
	jobCh := make(chan minifyJob, len(jobs))
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				if err := minifyFile(m, j.path, j.mime); err != nil {
					logger.Warn("minify failed", "file", j.path, "error", err)
					continue
				}
				count.Add(1)
			}
		}()
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()

	return int(count.Load()), nil
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
