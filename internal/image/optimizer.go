package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "image/gif"

	"golang.org/x/image/draw"
)

// Options controls image optimization behaviour.
type Options struct {
	// Quality for JPEG/WebP encoding (1-100).
	Quality int
	// Widths are the responsive breakpoints to generate (e.g. [640, 1200]).
	Widths []int
	// WebP enables WebP variant generation via cwebp (if available).
	WebP bool
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Quality: 80,
		Widths:  []int{640, 1200},
		WebP:    true,
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

// Optimize walks publicDir looking for raster images that were copied from
// content, generates resized JPEG and optional WebP variants, and returns a
// map from original URL path ("/2025/01/hero.jpg") to its Result.
func Optimize(publicDir string, opts Options, logger *slog.Logger) (map[string]*Result, error) {
	if logger == nil {
		logger = slog.Default()
	}

	hasWebP := false
	if opts.WebP {
		if _, err := exec.LookPath("cwebp"); err == nil {
			hasWebP = true
		} else {
			logger.Info("cwebp not found in PATH, WebP generation disabled")
		}
	}

	results := map[string]*Result{}
	generated := 0

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

		rel, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}
		urlPath := "/" + filepath.ToSlash(rel)

		res, n, err := optimizeFile(path, urlPath, opts, hasWebP, logger)
		if err != nil {
			logger.Warn("image optimize failed", "path", path, "error", err)
			return nil // non-fatal
		}
		if res != nil {
			results[urlPath] = res
			generated += n
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk public dir for images: %w", err)
	}

	if generated > 0 {
		logger.Info("images optimized", "sources", len(results), "variants", generated)
	}

	return results, nil
}

// isOptimizable returns true for raster image extensions we can process.
func isOptimizable(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png":
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
func optimizeFile(path string, urlPath string, opts Options, hasWebP bool, logger *slog.Logger) (*Result, int, error) {
	if isVariant(filepath.Base(path)) {
		return nil, 0, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	src, format, err := image.Decode(file)
	if err != nil {
		return nil, 0, fmt.Errorf("decode %s: %w", path, err)
	}

	bounds := src.Bounds()
	srcWidth := bounds.Dx()

	res := &Result{
		Original:      urlPath,
		OriginalWidth: srcWidth,
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

		// Resize.
		resized := resize(src, w)

		// JPEG variant.
		jpgName := fmt.Sprintf("%s-%dw.jpg", base, w)
		jpgPath := filepath.Join(dir, jpgName)
		jpgURL := urlDir(urlPath) + jpgName

		if err := writeJPEG(jpgPath, resized, opts.Quality); err != nil {
			return nil, 0, fmt.Errorf("write jpeg %s: %w", jpgPath, err)
		}

		variants := []Variant{{URLPath: jpgURL, Width: w, Format: "jpeg"}}
		count++
		logger.Debug("image variant", "src", urlPath, "variant", jpgURL, "width", w, "format", "jpeg")

		// WebP variant via cwebp.
		if hasWebP {
			webpName := fmt.Sprintf("%s-%dw.webp", base, w)
			webpPath := filepath.Join(dir, webpName)
			webpURL := urlDir(urlPath) + webpName

			if err := writeWebP(jpgPath, webpPath, opts.Quality); err != nil {
				logger.Warn("webp conversion failed", "path", jpgPath, "error", err)
			} else {
				variants = append(variants, Variant{URLPath: webpURL, Width: w, Format: "webp"})
				count++
				logger.Debug("image variant", "src", urlPath, "variant", webpURL, "width", w, "format", "webp")
			}
		}

		res.Variants[w] = variants
	}

	// Also generate WebP of the original size if we have cwebp.
	if hasWebP && format != "webp" {
		webpName := base + ".webp"
		webpPath := filepath.Join(dir, webpName)
		webpURL := urlDir(urlPath) + webpName

		if err := writeWebP(path, webpPath, opts.Quality); err != nil {
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
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

// writePNG encodes an image as PNG (unused for variants but kept for completeness).
func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// writeWebP converts an image to WebP using the cwebp binary.
func writeWebP(srcPath string, destPath string, quality int) error {
	cmd := exec.Command("cwebp", "-q", fmt.Sprintf("%d", quality), srcPath, "-o", destPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
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
// a plain <img> tag.
func PictureHTML(src string, alt string, loading string, results map[string]*Result) string {
	if loading == "" {
		loading = "lazy"
	}

	res, ok := results[src]
	if !ok || len(res.Variants) == 0 {
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="%s" />`, src, escAttr(alt), loading)
	}

	// Collect srcset entries grouped by format.
	webpSrcset := []string{}
	jpgSrcset := []string{}

	// Sort widths for deterministic output.
	widths := sortedWidths(res.Variants)

	for _, w := range widths {
		for _, v := range res.Variants[w] {
			entry := fmt.Sprintf("%s %dw", v.URLPath, v.Width)
			switch v.Format {
			case "webp":
				webpSrcset = append(webpSrcset, entry)
			case "jpeg", "jpg", "png":
				jpgSrcset = append(jpgSrcset, entry)
			}
		}
	}

	// If we only have WebP of the original (no smaller variants), still
	// include the original in the JPEG srcset for the fallback.
	if len(jpgSrcset) == 0 && len(webpSrcset) == 0 {
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="%s" />`, src, escAttr(alt), loading)
	}

	var b strings.Builder
	b.WriteString("<picture>\n")

	if len(webpSrcset) > 0 {
		fmt.Fprintf(&b, `  <source type="image/webp" srcset="%s" />`, strings.Join(webpSrcset, ", "))
		b.WriteString("\n")
	}

	if len(jpgSrcset) > 0 {
		fmt.Fprintf(&b, `  <source type="image/jpeg" srcset="%s" />`, strings.Join(jpgSrcset, ", "))
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, `  <img src="%s" alt="%s" loading="%s" />`, src, escAttr(alt), loading)
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
