package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	_ "image/gif"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Options controls image optimization behaviour.
type Options struct {
	// Quality for JPEG/WebP encoding (1-100).
	Quality int
	// Widths are the responsive breakpoints to generate (e.g. [640, 1200]).
	Widths []int
	// WebP enables WebP variant generation via cwebp (if available).
	WebP bool
	// AVIF enables AVIF variant generation via avifenc (if available).
	AVIF bool
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Quality: 80,
		Widths:  []int{640, 1200},
		WebP:    true,
		AVIF:    true,
	}
}

// Result describes all variants generated for a single source image.
type Result struct {
	// Original is the URL-path of the original image (e.g. "/img/hero.jpg").
	Original string
	// OriginalWidth is the pixel width of the source image (0 if unknown).
	OriginalWidth int
	// Variants maps width -> list of URL paths (JPEG first, then WebP).
	Variants map[int][]Variant
}

// Variant is a single generated file.
type Variant struct {
	URLPath string // e.g. "/img/hero-640w.webp"
	Width   int
	Format  string // "jpeg", "webp", "png"
}

// imageJob is a unit of work for the parallel image optimizer.
type imageJob struct {
	path    string
	urlPath string
	hash    string // SHA-256 of source content (set during cache check)
}

// imageResult collects the output of a single image optimization job.
type imageResult struct {
	urlPath string
	result  *Result
	count   int
	hash    string // carried from job for cache update
}

// Optimize walks publicDir looking for raster images that were copied from
// content, generates resized JPEG and optional WebP variants, and returns a
// map from original URL path ("/2025/01/hero.jpg") to its Result.
//
// Image processing runs in parallel using a worker pool sized to the number
// of available CPUs.
func Optimize(publicDir string, opts Options, logger *slog.Logger) (map[string]*Result, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// WebP and AVIF encoders are embedded (libwebp/libavif compiled to
	// WASM, executed via the same wazero runtime that powers the plugin
	// system). Available everywhere — Options.WebP / Options.AVIF stay
	// only as user-facing kill switches.
	hasWebP := opts.WebP
	hasAVIF := opts.AVIF

	// Phase 1: discover all optimizable images (fast filesystem walk).
	var jobs []imageJob
	err := filepath.WalkDir(publicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !isOptimizable(d.Name()) {
			return nil
		}

		rel, relErr := filepath.Rel(publicDir, path)
		if relErr != nil {
			return relErr
		}
		jobs = append(jobs, imageJob{
			path:    path,
			urlPath: "/" + filepath.ToSlash(rel),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk public dir for images: %w", err)
	}

	if len(jobs) == 0 {
		return map[string]*Result{}, nil
	}

	// Load image cache to skip unchanged images.
	cache := loadImageCache()

	// Phase 2: separate cached vs uncached jobs.
	var uncached []imageJob
	results := make(map[string]*Result, len(jobs))
	cached := 0

	for _, j := range jobs {
		hash, err := hashFile(j.path)
		if err != nil {
			uncached = append(uncached, j)
			continue
		}
		if entry, ok := cache.Entries[j.urlPath]; ok && entry.Hash == hash {
			// Cache hit — verify variant files still exist.
			if variantsExist(publicDir, entry.Result) {
				results[j.urlPath] = entry.Result
				cached++
				continue
			}
		}
		j.hash = hash
		uncached = append(uncached, j)
	}

	// Phase 3: process uncached images in parallel.
	generated := 0
	if len(uncached) > 0 {
		workers := runtime.NumCPU()
		if workers > len(uncached) {
			workers = len(uncached)
		}

		jobCh := make(chan imageJob, len(uncached))
		resCh := make(chan imageResult, len(uncached))
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobCh {
					res, n, err := optimizeFile(j.path, j.urlPath, opts, hasWebP, hasAVIF, logger)
					if err != nil {
						logger.Warn("image optimize failed", "path", j.path, "error", err)
						continue
					}
					if res != nil {
						resCh <- imageResult{urlPath: j.urlPath, result: res, count: n, hash: j.hash}
					}
				}
			}()
		}

		for _, j := range uncached {
			jobCh <- j
		}
		close(jobCh)

		go func() {
			wg.Wait()
			close(resCh)
		}()

		for r := range resCh {
			results[r.urlPath] = r.result
			generated += r.count
			if r.hash != "" {
				cache.Entries[r.urlPath] = &cacheEntry{
					Hash:   r.hash,
					Result: r.result,
					Count:  r.count,
				}
			}
		}
	}

	// Phase 4: downsize oversized originals.  This runs for ALL images
	// (including cached ones whose originals were restored by asset copy)
	// so that the fallback <img src> file stays compact.
	//
	// The original's quality is intentionally a notch lower than the
	// AVIF/WebP variants: it is only consumed by old browsers, social
	// crawlers, RSS readers and direct hotlinks — none of which warrant
	// the extra bytes. Modern browsers fetch the AVIF/WebP variants
	// at full configured quality.
	maxW := maxConfiguredWidth(opts.Widths)
	if maxW > 0 {
		originalQuality := opts.Quality - 5
		if originalQuality < 50 {
			originalQuality = 50
		}
		downsized := 0
		for _, j := range jobs {
			if downsizeIfNeeded(j.path, maxW, originalQuality, hasWebP, logger) {
				downsized++
			}
		}
		if downsized > 0 {
			logger.Info("downsized oversized originals", "count", downsized, "maxWidth", maxW, "quality", originalQuality)
		}
	}

	// Save updated cache.
	if err := saveImageCache(cache); err != nil {
		logger.Warn("failed to save image cache", "error", err)
	}

	total := len(jobs)
	if generated > 0 || cached > 0 {
		logger.Info("images optimized", "total", total, "optimized", total-cached, "cached", cached, "variants", generated)
	}

	return results, nil
}

// isOptimizable returns true for raster image extensions we can process.
func isOptimizable(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".avif":
		return true
	}
	return false
}

// isVariant returns true if the filename already looks like a generated
// variant (e.g. "hero-640w.jpg").
func isVariant(name string) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	// Check for "-<digits>w" suffix.
	idx := strings.LastIndex(base, "-")
	if idx < 0 {
		return false
	}
	suffix := base[idx+1:]
	if len(suffix) < 2 || suffix[len(suffix)-1] != 'w' {
		return false
	}
	for _, c := range suffix[:len(suffix)-1] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// optimizeFile generates responsive variants for a single image file.
// Returns nil result if the image is already a variant or too small.
func optimizeFile(path string, urlPath string, opts Options, hasWebP bool, hasAVIF bool, logger *slog.Logger) (*Result, int, error) {
	if isVariant(filepath.Base(path)) {
		return nil, 0, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = file.Close() }()

	src, format, err := image.Decode(file)
	if err != nil {
		return nil, 0, fmt.Errorf("decode %s: %w", path, err)
	}

	bounds := src.Bounds()
	srcWidth := bounds.Dx()

	maxWidth := maxConfiguredWidth(opts.Widths)
	effectiveWidth := srcWidth
	if maxWidth > 0 && srcWidth > maxWidth {
		effectiveWidth = maxWidth
	}

	res := &Result{
		Original:      urlPath,
		OriginalWidth: effectiveWidth,
		Variants:      map[int][]Variant{},
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	count := 0

	for _, w := range opts.Widths {
		if w >= srcWidth {
			// Don't upscale. If the original is smaller than the
			// breakpoint, skip this width entirely.
			continue
		}

		// Skip the per-width resize entirely when neither modern
		// encoder is available: the only output we'd produce would
		// be intermediate JPEGs which we no longer emit, so there's
		// nothing useful left for this width. The original at base
		// path still serves as <img> fallback.
		if !hasWebP && !hasAVIF {
			continue
		}

		// Resize once, then encode WebP and AVIF directly from the
		// resized image. JPEG variants are no longer emitted in
		// <picture>: every browser that understands <picture> also
		// understands WebP, so JPEG srcset entries were dead code.
		resized := resize(src, w)

		variants := []Variant{}

		if hasWebP {
			webpName := fmt.Sprintf("%s-%dw.webp", base, w)
			webpPath := filepath.Join(dir, webpName)
			webpURL := urlDir(urlPath) + webpName

			if err := writeWebP(webpPath, resized, opts.Quality); err != nil {
				logger.Warn("webp conversion failed", "path", webpPath, "error", err)
			} else {
				variants = append(variants, Variant{URLPath: webpURL, Width: w, Format: "webp"})
				count++
				logger.Debug("image variant", "src", urlPath, "variant", webpURL, "width", w, "format", "webp")
			}
		}

		if hasAVIF {
			avifName := fmt.Sprintf("%s-%dw.avif", base, w)
			avifPath := filepath.Join(dir, avifName)
			avifURL := urlDir(urlPath) + avifName

			if err := writeAVIF(avifPath, resized, opts.Quality); err != nil {
				logger.Warn("avif conversion failed", "path", avifPath, "error", err)
			} else {
				variants = append(variants, Variant{URLPath: avifURL, Width: w, Format: "avif"})
				count++
				logger.Debug("image variant", "src", urlPath, "variant", avifURL, "width", w, "format", "avif")
			}
		}

		if len(variants) > 0 {
			res.Variants[w] = variants
		}
	}

	// Generate WebP of the original size only when the original is not
	// oversized (i.e. at or below the largest configured width).  For
	// oversized originals the largest width variant already provides a WebP.
	if hasWebP && format != "webp" && (maxWidth == 0 || srcWidth <= maxWidth) {
		webpName := base + ".webp"
		webpPath := filepath.Join(dir, webpName)
		webpURL := urlDir(urlPath) + webpName

		if err := writeWebP(webpPath, src, opts.Quality); err != nil {
			logger.Warn("webp conversion of original failed", "path", path, "error", err)
		} else {
			if res.Variants[srcWidth] == nil {
				res.Variants[srcWidth] = []Variant{}
			}
			res.Variants[srcWidth] = append(res.Variants[srcWidth], Variant{
				URLPath: webpURL, Width: srcWidth, Format: "webp",
			})
			count++
		}
	}

	// Generate AVIF of the original size (same logic as WebP above).
	if hasAVIF && (maxWidth == 0 || srcWidth <= maxWidth) {
		avifName := base + ".avif"
		avifPath := filepath.Join(dir, avifName)
		avifURL := urlDir(urlPath) + avifName

		if err := writeAVIF(avifPath, src, opts.Quality); err != nil {
			logger.Warn("avif conversion of original failed", "path", path, "error", err)
		} else {
			if res.Variants[srcWidth] == nil {
				res.Variants[srcWidth] = []Variant{}
			}
			res.Variants[srcWidth] = append(res.Variants[srcWidth], Variant{
				URLPath: avifURL, Width: srcWidth, Format: "avif",
			})
			count++
		}
	}

	if count == 0 {
		return nil, 0, nil
	}

	return res, count, nil
}

// resize scales src to the target width, preserving aspect ratio.
func resize(src image.Image, targetWidth int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || targetWidth <= 0 {
		return src
	}
	targetHeight := srcH * targetWidth / srcW
	if targetHeight < 1 {
		targetHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// writeJPEG encodes an image as JPEG with the given quality.
func writeJPEG(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

// writeWebP encodes the given image to WebP using the embedded
// libwebp (gen2brain/webp -> wazero). Method 4 is the library's
// default quality/speed trade-off; we keep it.
func writeWebP(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return webp.Encode(f, img, webp.Options{Quality: quality, Method: 4})
}

// writeAVIF encodes the given image to AVIF using the embedded
// libavif (gen2brain/avif -> wazero). Speed 6 mirrors the speed flag
// we passed to the avifenc CLI before — the libavif default is 6.
func writeAVIF(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return avif.Encode(f, img, avif.Options{
		Quality:      quality,
		QualityAlpha: quality,
		Speed:        6,
	})
}

// urlDir returns the directory portion of a URL path with a trailing slash.
func urlDir(urlPath string) string {
	dir := filepath.ToSlash(filepath.Dir(urlPath))
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

// PictureHTML generates a <picture> element with <source> for WebP and JPEG
// srcset, falling back to the original <img>. If no variants exist, returns
// a plain <img> tag. When fetchpriority is non-empty (e.g. "high"), the
// attribute is added to the <img> element for LCP optimisation.
func PictureHTML(src string, alt string, loading string, fetchpriority string, results map[string]*Result) string {
	if loading == "" {
		loading = "lazy"
	}

	fpAttr := ""
	if fetchpriority != "" {
		fpAttr = fmt.Sprintf(` fetchpriority="%s"`, fetchpriority)
	}

	res, ok := results[src]
	if !ok || len(res.Variants) == 0 {
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="%s"%s />`, src, escAttr(alt), loading, fpAttr)
	}

	// Collect srcset entries grouped by format.
	avifSrcset := []string{}
	webpSrcset := []string{}
	jpgSrcset := []string{}

	// Sort widths for deterministic output.
	widths := sortedWidths(res.Variants)

	for _, w := range widths {
		for _, v := range res.Variants[w] {
			entry := fmt.Sprintf("%s %dw", v.URLPath, v.Width)
			switch v.Format {
			case "avif":
				avifSrcset = append(avifSrcset, entry)
			case "webp":
				webpSrcset = append(webpSrcset, entry)
			case "jpeg", "jpg", "png":
				jpgSrcset = append(jpgSrcset, entry)
			}
		}
	}

	// If no format-specific srcsets at all, fall back to plain <img>.
	if len(jpgSrcset) == 0 && len(webpSrcset) == 0 && len(avifSrcset) == 0 {
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="%s"%s />`, src, escAttr(alt), loading, fpAttr)
	}

	var b strings.Builder
	b.WriteString("<picture>\n")

	// AVIF first (smallest, best compression, newest format).
	if len(avifSrcset) > 0 {
		fmt.Fprintf(&b, `  <source type="image/avif" srcset="%s" />`, strings.Join(avifSrcset, ", "))
		b.WriteString("\n")
	}

	if len(webpSrcset) > 0 {
		fmt.Fprintf(&b, `  <source type="image/webp" srcset="%s" />`, strings.Join(webpSrcset, ", "))
		b.WriteString("\n")
	}

	if len(jpgSrcset) > 0 {
		fmt.Fprintf(&b, `  <source type="image/jpeg" srcset="%s" />`, strings.Join(jpgSrcset, ", "))
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, `  <img src="%s" alt="%s" loading="%s"%s />`, src, escAttr(alt), loading, fpAttr)
	b.WriteString("\n</picture>")

	return b.String()
}

// sortedWidths returns the map keys sorted ascending.
func sortedWidths(m map[int][]Variant) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort for tiny slices.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

// escAttr escapes double quotes in HTML attribute values.
func escAttr(s string) string {
	return strings.ReplaceAll(s, `"`, "&quot;")
}

// maxConfiguredWidth returns the largest width in the list, or 0 if empty.
func maxConfiguredWidth(widths []int) int {
	m := 0
	for _, w := range widths {
		if w > m {
			m = w
		}
	}
	return m
}

// downsizeIfNeeded replaces an oversized original with a version resized to
// maxWidth.  Returns true if the file was downsized.  It uses DecodeConfig
// for a fast header-only check before doing the full decode+resize.
func downsizeIfNeeded(path string, maxWidth int, quality int, hasWebP bool, logger *slog.Logger) bool {
	if isVariant(filepath.Base(path)) {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		return false
	}

	// Fast header-only check.
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	cfg, _, err := image.DecodeConfig(f)
	_ = f.Close()
	if err != nil || cfg.Width <= maxWidth {
		return false
	}

	// Full decode + resize.
	f, err = os.Open(path)
	if err != nil {
		return false
	}
	src, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return false
	}

	resized := resize(src, maxWidth)

	switch ext {
	case ".jpg", ".jpeg":
		if err := writeJPEG(path, resized, quality); err != nil {
			logger.Warn("downsize original failed", "path", path, "error", err)
			return false
		}
	case ".webp":
		if !hasWebP {
			return false
		}
		if err := writeWebP(path, resized, quality); err != nil {
			logger.Warn("downsize original failed", "path", path, "error", err)
			return false
		}
	}

	logger.Debug("downsized original", "path", path, "from", cfg.Width, "to", maxWidth)
	return true
}
