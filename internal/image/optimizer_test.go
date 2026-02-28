package image

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestJPEG writes a solid-colour JPEG to path.
func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

// createTestPNG writes a solid-colour PNG to path.
func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestIsOptimizable(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.JPG", true},
		{"photo.png", true},
		{"photo.PNG", true},
		{"photo.svg", false},
		{"photo.webp", false},
		{"photo.gif", false},
		{"file.md", false},
		{"file.html", false},
	}
	for _, tt := range tests {
		if got := isOptimizable(tt.name); got != tt.want {
			t.Errorf("isOptimizable(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsVariant(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"hero-640w.jpg", true},
		{"hero-1200w.webp", true},
		{"hero-1w.png", true},
		{"hero.jpg", false},
		{"hero-big.jpg", false},
		{"hero-w.jpg", false},    // no digits
		{"hero-640x.jpg", false}, // not 'w'
		{"hero-640w", true},      // no extension but still variant pattern
		{"-640w.jpg", true},      // edge case: empty base
		{"photo-abc-640w.jpg", true},
	}
	for _, tt := range tests {
		if got := isVariant(tt.name); got != tt.want {
			t.Errorf("isVariant(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestResize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := resize(src, 640)
	bounds := resized.Bounds()
	if bounds.Dx() != 640 {
		t.Errorf("resize width = %d, want 640", bounds.Dx())
	}
	// Aspect ratio: 1600/900 = 16/9, so 640*900/1600 = 360
	if bounds.Dy() != 360 {
		t.Errorf("resize height = %d, want 360", bounds.Dy())
	}
}

func TestResizeNoUpscale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 300))
	// resize to larger width — should return original
	resized := resize(src, 800)
	bounds := resized.Bounds()
	// resize doesn't enforce no-upscale itself, the caller does.
	// But if called, it still works (just upscales).
	if bounds.Dx() != 800 {
		t.Errorf("resize width = %d, want 800", bounds.Dx())
	}
}

func TestSortedWidths(t *testing.T) {
	m := map[int][]Variant{
		1200: {{URLPath: "/a", Width: 1200}},
		640:  {{URLPath: "/b", Width: 640}},
		1920: {{URLPath: "/c", Width: 1920}},
	}
	got := sortedWidths(m)
	want := []int{640, 1200, 1920}
	if len(got) != len(want) {
		t.Fatalf("sortedWidths len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("sortedWidths[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestEscAttr(t *testing.T) {
	if got := escAttr(`foo "bar" baz`); got != `foo &quot;bar&quot; baz` {
		t.Errorf("escAttr = %q, want escaped quotes", got)
	}
	if got := escAttr("plain"); got != "plain" {
		t.Errorf("escAttr plain = %q", got)
	}
}

func TestURLDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/img/hero.jpg", "/img/"},
		{"/2025/01/photo.jpg", "/2025/01/"},
		{"/hero.jpg", "/"},
	}
	for _, tt := range tests {
		if got := urlDir(tt.input); got != tt.want {
			t.Errorf("urlDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOptimizeFile_JPEG(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "hero.jpg")
	createTestJPEG(t, imgPath, 1600, 900)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 75, Widths: []int{640, 1200}, WebP: false}

	res, count, err := optimizeFile(imgPath, "/img/hero.jpg", opts, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (640w + 1200w JPEG)", count)
	}
	if res.Original != "/img/hero.jpg" {
		t.Errorf("Original = %q, want /img/hero.jpg", res.Original)
	}
	if res.OriginalWidth != 1600 {
		t.Errorf("OriginalWidth = %d, want 1600", res.OriginalWidth)
	}

	// Check 640w variant exists on disk.
	jpgPath := filepath.Join(dir, "hero-640w.jpg")
	if _, err := os.Stat(jpgPath); err != nil {
		t.Errorf("640w JPEG not found: %v", err)
	}

	// Check 1200w variant exists on disk.
	jpgPath1200 := filepath.Join(dir, "hero-1200w.jpg")
	if _, err := os.Stat(jpgPath1200); err != nil {
		t.Errorf("1200w JPEG not found: %v", err)
	}

	// Variants should have width keys.
	if _, ok := res.Variants[640]; !ok {
		t.Error("missing 640 width in Variants")
	}
	if _, ok := res.Variants[1200]; !ok {
		t.Error("missing 1200 width in Variants")
	}
}

func TestOptimizeFile_SkipsSmallImages(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "small.jpg")
	createTestJPEG(t, imgPath, 400, 300) // smaller than all breakpoints

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 75, Widths: []int{640, 1200}, WebP: false}

	res, count, err := optimizeFile(imgPath, "/img/small.jpg", opts, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	// No variants should be generated since 400 < 640 < 1200.
	if res != nil {
		t.Errorf("expected nil result for small image, got %+v (count=%d)", res, count)
	}
}

func TestOptimizeFile_SkipsVariants(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "hero-640w.jpg")
	createTestJPEG(t, imgPath, 640, 360)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 75, Widths: []int{320}, WebP: false}

	res, _, err := optimizeFile(imgPath, "/img/hero-640w.jpg", opts, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Error("expected nil result for existing variant file")
	}
}

func TestOptimizeFile_PNG(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "chart.png")
	createTestPNG(t, imgPath, 2000, 1000)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 80, Widths: []int{640, 1200}, WebP: false}

	res, count, err := optimizeFile(imgPath, "/chart.png", opts, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result for PNG")
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestOptimize_FullWalk(t *testing.T) {
	dir := t.TempDir()

	// Create subdirectory structure like public/
	createTestJPEG(t, filepath.Join(dir, "img", "hero.jpg"), 1600, 900)
	createTestPNG(t, filepath.Join(dir, "2025", "01", "screenshot.png"), 2400, 1200)

	// Create an SVG (should be ignored).
	svgPath := filepath.Join(dir, "img", "placeholder.svg")
	_ = os.WriteFile(svgPath, []byte(`<svg></svg>`), 0o644)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 80, Widths: []int{640, 1200}, WebP: false}

	results, err := Optimize(dir, opts, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Should have results for hero.jpg and screenshot.png.
	if len(results) != 2 {
		t.Errorf("results count = %d, want 2", len(results))
		for k := range results {
			t.Logf("  result key: %s", k)
		}
	}

	if _, ok := results["/img/hero.jpg"]; !ok {
		t.Error("missing result for /img/hero.jpg")
	}
	if _, ok := results["/2025/01/screenshot.png"]; !ok {
		t.Error("missing result for /2025/01/screenshot.png")
	}
}

func TestPictureHTML_NoVariants(t *testing.T) {
	html := PictureHTML("/img/hero.jpg", "Hero image", "eager", nil)
	want := `<img src="/img/hero.jpg" alt="Hero image" loading="eager" />`
	if html != want {
		t.Errorf("PictureHTML no variants =\n%s\nwant:\n%s", html, want)
	}
}

func TestPictureHTML_NotInResults(t *testing.T) {
	results := map[string]*Result{}
	html := PictureHTML("/img/hero.jpg", "Hero", "lazy", results)
	if !strings.HasPrefix(html, "<img") {
		t.Errorf("expected plain <img>, got: %s", html)
	}
}

func TestPictureHTML_WithVariants(t *testing.T) {
	results := map[string]*Result{
		"/img/hero.jpg": {
			Original:      "/img/hero.jpg",
			OriginalWidth: 1600,
			Variants: map[int][]Variant{
				640: {
					{URLPath: "/img/hero-640w.jpg", Width: 640, Format: "jpeg"},
					{URLPath: "/img/hero-640w.webp", Width: 640, Format: "webp"},
				},
				1200: {
					{URLPath: "/img/hero-1200w.jpg", Width: 1200, Format: "jpeg"},
					{URLPath: "/img/hero-1200w.webp", Width: 1200, Format: "webp"},
				},
			},
		},
	}

	html := PictureHTML("/img/hero.jpg", "Hero", "eager", results)

	if !strings.Contains(html, "<picture>") {
		t.Error("expected <picture> element")
	}
	if !strings.Contains(html, `type="image/webp"`) {
		t.Error("expected webp source")
	}
	if !strings.Contains(html, `type="image/jpeg"`) {
		t.Error("expected jpeg source")
	}
	if !strings.Contains(html, "hero-640w.webp 640w") {
		t.Error("expected 640w webp srcset entry")
	}
	if !strings.Contains(html, "hero-1200w.jpg 1200w") {
		t.Error("expected 1200w jpeg srcset entry")
	}
	if !strings.Contains(html, `loading="eager"`) {
		t.Error("expected loading=eager on fallback img")
	}
	if !strings.Contains(html, `alt="Hero"`) {
		t.Error("expected alt attribute")
	}
}

func TestPictureHTML_QuotesInAlt(t *testing.T) {
	html := PictureHTML("/x.jpg", `He said "hello"`, "lazy", nil)
	if !strings.Contains(html, "&quot;") {
		t.Errorf("expected escaped quotes in alt, got: %s", html)
	}
}

func TestPictureHTML_DefaultLoading(t *testing.T) {
	html := PictureHTML("/x.jpg", "alt", "", nil)
	if !strings.Contains(html, `loading="lazy"`) {
		t.Errorf("expected default loading=lazy, got: %s", html)
	}
}

func TestOptimizeFile_OnlyOneWidth(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "medium.jpg")
	createTestJPEG(t, imgPath, 800, 600) // 800 > 640 but 800 < 1200

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 80, Widths: []int{640, 1200}, WebP: false}

	res, count, err := optimizeFile(imgPath, "/medium.jpg", opts, false, logger)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	// Only 640w variant, not 1200w (would upscale).
	if count != 1 {
		t.Errorf("count = %d, want 1 (only 640w)", count)
	}
	if _, ok := res.Variants[640]; !ok {
		t.Error("missing 640 width in Variants")
	}
	if _, ok := res.Variants[1200]; ok {
		t.Error("unexpected 1200 width in Variants (should not upscale)")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Quality != 80 {
		t.Errorf("Quality = %d, want 80", opts.Quality)
	}
	if len(opts.Widths) != 2 || opts.Widths[0] != 640 || opts.Widths[1] != 1200 {
		t.Errorf("Widths = %v, want [640 1200]", opts.Widths)
	}
	if !opts.WebP {
		t.Error("WebP = false, want true")
	}
}

func TestWriteJPEG(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	path := filepath.Join(dir, "test.jpg")

	if err := writeJPEG(path, img, 80); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("JPEG file is empty")
	}

	// Verify it's a valid JPEG.
	f, _ := os.Open(path)
	defer func() { _ = f.Close() }()
	_, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("cannot decode written JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("format = %q, want jpeg", format)
	}
}

func TestOptimize_SkipsDotDirs(t *testing.T) {
	dir := t.TempDir()
	// Image inside a dot-directory should be skipped.
	createTestJPEG(t, filepath.Join(dir, ".cache", "old.jpg"), 1600, 900)
	// Image outside dot-directory should be processed.
	createTestJPEG(t, filepath.Join(dir, "photo.jpg"), 1600, 900)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := Options{Quality: 80, Widths: []int{640}, WebP: false}

	results, err := Optimize(dir, opts, logger)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
	if _, ok := results["/photo.jpg"]; !ok {
		keys := []string{}
		for k := range results {
			keys = append(keys, k)
		}
		t.Errorf("expected /photo.jpg in results, got: %v", keys)
	}
}

func TestOptimize_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	opts := DefaultOptions()

	results, err := Optimize(dir, opts, logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestPictureHTML_WebPOnly(t *testing.T) {
	// Only WebP variants, no JPEG srcset.
	results := map[string]*Result{
		"/img/hero.jpg": {
			Original:      "/img/hero.jpg",
			OriginalWidth: 1600,
			Variants: map[int][]Variant{
				1600: {
					{URLPath: "/img/hero.webp", Width: 1600, Format: "webp"},
				},
			},
		},
	}

	html := PictureHTML("/img/hero.jpg", "Hero", "lazy", results)
	if !strings.Contains(html, "<picture>") {
		t.Error("expected <picture> element")
	}
	if !strings.Contains(html, `type="image/webp"`) {
		t.Error("expected webp source")
	}
	// Should NOT have jpeg source since there are no jpeg variants.
	if strings.Contains(html, `type="image/jpeg"`) {
		t.Error("unexpected jpeg source when only webp exists")
	}
	// Fallback img should still be the original.
	if !strings.Contains(html, `src="/img/hero.jpg"`) {
		t.Error("expected original src in fallback img")
	}
}
