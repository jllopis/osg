package assets

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"osg/internal/config"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newLogger() *slog.Logger { return slog.Default() }

// writeFile is a test helper that creates parent dirs and writes content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readFile is a test helper that reads a file's content as a string.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

// fileExists reports whether a file (or dir) exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ---------------------------------------------------------------------------
// copyStaticChain
// ---------------------------------------------------------------------------

func TestCopyStaticChain_MultipleThemes(t *testing.T) {
	// Set up three theme dirs (grandparent, parent, child) each with a static/
	// directory.  The chain is ordered child-first so we pass
	// [child, parent, grandparent].  The function iterates root-ancestor first
	// so grandparent files are laid down first and child files override.
	grandparent := t.TempDir()
	parent := t.TempDir()
	child := t.TempDir()
	userStatic := t.TempDir()
	publicDir := t.TempDir()

	// Grandparent provides base.css and shared.css
	writeFile(t, filepath.Join(grandparent, "static", "base.css"), "grandparent-base")
	writeFile(t, filepath.Join(grandparent, "static", "shared.css"), "grandparent-shared")

	// Parent overrides shared.css
	writeFile(t, filepath.Join(parent, "static", "shared.css"), "parent-shared")
	writeFile(t, filepath.Join(parent, "static", "parent-only.js"), "parent-js")

	// Child overrides shared.css again and adds child-only file
	writeFile(t, filepath.Join(child, "static", "shared.css"), "child-shared")
	writeFile(t, filepath.Join(child, "static", "child-only.txt"), "child-txt")

	// User static provides a robots.txt
	writeFile(t, filepath.Join(userStatic, "robots.txt"), "User-agent: *")

	chain := []string{child, parent, grandparent}
	if err := copyStaticChain(chain, userStatic, publicDir, newLogger()); err != nil {
		t.Fatalf("copyStaticChain error: %v", err)
	}

	// base.css comes from grandparent (nobody overrides it)
	if got := readFile(t, filepath.Join(publicDir, "base.css")); got != "grandparent-base" {
		t.Errorf("base.css = %q, want grandparent-base", got)
	}

	// shared.css should be the child's version (last to copy)
	if got := readFile(t, filepath.Join(publicDir, "shared.css")); got != "child-shared" {
		t.Errorf("shared.css = %q, want child-shared", got)
	}

	// parent-only.js should exist
	if got := readFile(t, filepath.Join(publicDir, "parent-only.js")); got != "parent-js" {
		t.Errorf("parent-only.js = %q, want parent-js", got)
	}

	// child-only.txt should exist
	if got := readFile(t, filepath.Join(publicDir, "child-only.txt")); got != "child-txt" {
		t.Errorf("child-only.txt = %q, want child-txt", got)
	}

	// user robots.txt should exist (copied last)
	if got := readFile(t, filepath.Join(publicDir, "robots.txt")); got != "User-agent: *" {
		t.Errorf("robots.txt = %q, want User-agent: *", got)
	}
}

func TestCopyStaticChain_EmptyChain(t *testing.T) {
	// An empty chain should still copy user static dir.
	userStatic := t.TempDir()
	publicDir := t.TempDir()

	writeFile(t, filepath.Join(userStatic, "site.js"), "js-content")

	if err := copyStaticChain(nil, userStatic, publicDir, newLogger()); err != nil {
		t.Fatalf("copyStaticChain error: %v", err)
	}

	if got := readFile(t, filepath.Join(publicDir, "site.js")); got != "js-content" {
		t.Errorf("site.js = %q, want js-content", got)
	}
}

func TestCopyStaticChain_NonexistentStaticSubdirs(t *testing.T) {
	// Theme dirs exist but have no static/ subdirectory — should be fine.
	theme1 := t.TempDir()
	theme2 := t.TempDir()
	publicDir := t.TempDir()

	chain := []string{theme1, theme2}
	if err := copyStaticChain(chain, "/nonexistent/user-static", publicDir, newLogger()); err != nil {
		t.Fatalf("copyStaticChain error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// compileSassChain
// ---------------------------------------------------------------------------

func TestCompileSassChain_CompileSassFalse(t *testing.T) {
	// When CompileSass is false and themes have no sass dir, should be a no-op.
	theme := t.TempDir()
	publicDir := t.TempDir()
	cfg := config.Config{
		CompileSass: false,
		SassDir:     "/nonexistent/sass",
		PublicDir:   publicDir,
	}
	chain := []string{theme}

	if err := compileSassChain(chain, cfg, newLogger()); err != nil {
		t.Fatalf("compileSassChain error: %v", err)
	}
}

func TestCompileSassChain_WithConflict(t *testing.T) {
	// Create a theme sass dir that has conflicting .scss and .sass for
	// the same base name — compileSassDir should return an error.
	theme := t.TempDir()
	sassDir := filepath.Join(theme, "sass")
	writeFile(t, filepath.Join(sassDir, "main.scss"), "body{}")
	writeFile(t, filepath.Join(sassDir, "main.sass"), "body")

	publicDir := t.TempDir()
	cfg := config.Config{
		CompileSass: false,
		SassDir:     "/nonexistent",
		PublicDir:   publicDir,
	}
	chain := []string{theme}

	err := compileSassChain(chain, cfg, newLogger())
	if err == nil {
		t.Fatal("expected sass conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "sass conflict") {
		t.Errorf("error = %q, want it to contain 'sass conflict'", err.Error())
	}
}

func TestCompileSassChain_NoSassDirs(t *testing.T) {
	// Themes have no sass/ subdirectory, CompileSass is true but user SassDir
	// doesn't exist either — everything should succeed silently.
	theme1 := t.TempDir()
	theme2 := t.TempDir()
	publicDir := t.TempDir()
	cfg := config.Config{
		CompileSass: true,
		SassDir:     "/nonexistent/user-sass",
		PublicDir:   publicDir,
	}
	chain := []string{theme1, theme2}

	if err := compileSassChain(chain, cfg, newLogger()); err != nil {
		t.Fatalf("compileSassChain error: %v", err)
	}
}

func TestCompileSassChain_UserSassSkippedWhenCompileFalse(t *testing.T) {
	// Even if the user SassDir has a conflict, it should not be reached
	// if CompileSass is false.
	publicDir := t.TempDir()
	userSass := t.TempDir()
	writeFile(t, filepath.Join(userSass, "main.scss"), "body{}")
	writeFile(t, filepath.Join(userSass, "main.sass"), "body")

	cfg := config.Config{
		CompileSass: false,
		SassDir:     userSass,
		PublicDir:   publicDir,
	}
	// No themes in chain.
	if err := compileSassChain(nil, cfg, newLogger()); err != nil {
		t.Fatalf("compileSassChain should skip user sass when CompileSass=false, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// compileSassDir (additional coverage)
// ---------------------------------------------------------------------------

func TestCompileSassDir_WhitespaceSrc(t *testing.T) {
	err := compileSassDir("   ", t.TempDir(), newLogger())
	if err != nil {
		t.Errorf("whitespace src should return nil, got: %v", err)
	}
}

func TestCompileSassDir_ConflictDetection(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "theme.scss"), "a{}")
	writeFile(t, filepath.Join(src, "theme.sass"), "a")

	err := compileSassDir(src, t.TempDir(), newLogger())
	if err == nil {
		t.Fatal("expected sass conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "sass conflict") {
		t.Errorf("error = %q, want it to contain 'sass conflict'", err.Error())
	}
}

func TestCompileSassDir_NoConflictNoSassBinary(t *testing.T) {
	// When there are .scss files but no conflicts, compileSassDir proceeds to
	// check for the sass binary.  In a test environment the binary is likely
	// absent, so we expect either a "sass binary not found" error or (if sass
	// is installed) success.  Either outcome is acceptable — we are exercising
	// the code path up to and including the LookPath check.
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "style.scss"), "body { color: red; }")

	err := compileSassDir(src, t.TempDir(), newLogger())
	if err != nil && !strings.Contains(err.Error(), "sass binary not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompileSassDir_EmptyDirNoScssFiles(t *testing.T) {
	// Directory exists but has no .scss/.sass files — should pass conflict
	// detection, then hit LookPath (or succeed if sass is installed).
	// We can't guarantee sass is installed, but the dir-existence and
	// conflict-check paths are exercised.
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "readme.txt"), "not a sass file")

	err := compileSassDir(src, t.TempDir(), newLogger())
	// With no .scss files the conflict check passes.  Then LookPath runs.
	// Accept either nil (sass installed) or "sass binary not found".
	if err != nil && !strings.Contains(err.Error(), "sass binary not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompileSassDir_SkipsUnderscoreAndNonSass(t *testing.T) {
	// If sass binary IS available this exercises the walk skipping logic;
	// if sass is NOT available we still exercise everything up to LookPath.
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "_partial.scss"), "$x: 1;")
	writeFile(t, filepath.Join(src, "notes.txt"), "text")

	err := compileSassDir(src, t.TempDir(), newLogger())
	if err != nil && !strings.Contains(err.Error(), "sass binary not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// compileSass
// ---------------------------------------------------------------------------

func TestCompileSass_WithThemeNoSassDir(t *testing.T) {
	// Theme exists but has no sass/ subdirectory — should succeed.
	themesDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(themesDir, "mytheme"), 0o755); err != nil {
		t.Fatal(err)
	}
	publicDir := t.TempDir()

	cfg := config.Config{
		Theme:       "mytheme",
		ThemesDir:   themesDir,
		CompileSass: false,
		SassDir:     "/nonexistent",
		PublicDir:   publicDir,
	}
	if err := compileSass(cfg, newLogger()); err != nil {
		t.Errorf("compileSass error: %v", err)
	}
}

func TestCompileSass_WithThemeConflict(t *testing.T) {
	// Theme sass dir has a conflict — should return error.
	themesDir := t.TempDir()
	sassDir := filepath.Join(themesDir, "badtheme", "sass")
	writeFile(t, filepath.Join(sassDir, "a.scss"), "a{}")
	writeFile(t, filepath.Join(sassDir, "a.sass"), "a")

	cfg := config.Config{
		Theme:       "badtheme",
		ThemesDir:   themesDir,
		CompileSass: false,
		SassDir:     "/nonexistent",
		PublicDir:   t.TempDir(),
	}
	err := compileSass(cfg, newLogger())
	if err == nil {
		t.Fatal("expected sass conflict error")
	}
	if !strings.Contains(err.Error(), "sass conflict") {
		t.Errorf("error = %q, want sass conflict", err.Error())
	}
}

func TestCompileSass_NoThemeCompileTrue_NonexistentSassDir(t *testing.T) {
	// No theme, CompileSass true, but SassDir doesn't exist — should succeed.
	cfg := config.Config{
		Theme:       "",
		CompileSass: true,
		SassDir:     "/nonexistent/sass",
		PublicDir:   t.TempDir(),
	}
	if err := compileSass(cfg, newLogger()); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestCompileSass_EmptySassDir(t *testing.T) {
	cfg := config.Config{
		Theme:       "",
		CompileSass: true,
		SassDir:     "",
		PublicDir:   t.TempDir(),
	}
	if err := compileSass(cfg, newLogger()); err != nil {
		t.Errorf("expected nil for empty sass dir, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// copyStatic (additional paths)
// ---------------------------------------------------------------------------

func TestCopyStatic_NonexistentStaticDir(t *testing.T) {
	cfg := config.Config{
		Theme:     "",
		StaticDir: "/nonexistent/static",
		PublicDir: t.TempDir(),
	}
	if err := copyStatic(cfg, newLogger()); err != nil {
		t.Errorf("expected nil for nonexistent static dir, got: %v", err)
	}
}

func TestCopyStatic_EmptyStaticDir(t *testing.T) {
	cfg := config.Config{
		Theme:     "",
		StaticDir: "",
		PublicDir: t.TempDir(),
	}
	if err := copyStatic(cfg, newLogger()); err != nil {
		t.Errorf("expected nil for empty static dir, got: %v", err)
	}
}

func TestCopyStatic_ThemeOverriddenBySiteStatic(t *testing.T) {
	// Site-level static dir is processed after theme static, so site files
	// override theme files with the same name.
	themesDir := t.TempDir()
	themeStatic := filepath.Join(themesDir, "default", "static")
	writeFile(t, filepath.Join(themeStatic, "common.css"), "theme-version")

	siteStatic := t.TempDir()
	writeFile(t, filepath.Join(siteStatic, "common.css"), "site-version")

	publicDir := t.TempDir()
	cfg := config.Config{
		Theme:     "default",
		ThemesDir: themesDir,
		StaticDir: siteStatic,
		PublicDir: publicDir,
	}
	if err := copyStatic(cfg, newLogger()); err != nil {
		t.Fatalf("copyStatic error: %v", err)
	}

	if got := readFile(t, filepath.Join(publicDir, "common.css")); got != "site-version" {
		t.Errorf("common.css = %q, want site-version (site should override theme)", got)
	}
}

// ---------------------------------------------------------------------------
// PrepareWithChain
// ---------------------------------------------------------------------------

func TestPrepareWithChain_StaticCopyOnly(t *testing.T) {
	// Set up a single theme in the chain with a static file.
	// CompileSass is false so only static copy + content assets run.
	theme := t.TempDir()
	writeFile(t, filepath.Join(theme, "static", "app.js"), "console.log('hi')")

	contentDir := t.TempDir()
	writeFile(t, filepath.Join(contentDir, "pic.png"), "PNG-DATA")
	writeFile(t, filepath.Join(contentDir, "post.md"), "# Title")

	publicDir := t.TempDir()
	userStatic := t.TempDir()
	writeFile(t, filepath.Join(userStatic, "favicon.ico"), "ICO")

	cfg := config.Config{
		StaticDir:   userStatic,
		PublicDir:   publicDir,
		ContentDir:  contentDir,
		CompileSass: false,
		SassDir:     "",
	}
	chain := []string{theme}

	if err := PrepareWithChain(cfg, chain, newLogger()); err != nil {
		t.Fatalf("PrepareWithChain error: %v", err)
	}

	// Theme static file
	if !fileExists(filepath.Join(publicDir, "app.js")) {
		t.Error("app.js should be copied from theme static")
	}
	// User static file
	if !fileExists(filepath.Join(publicDir, "favicon.ico")) {
		t.Error("favicon.ico should be copied from user static")
	}
	// Content asset (non-md, non-dotfile)
	if !fileExists(filepath.Join(publicDir, "pic.png")) {
		t.Error("pic.png should be copied from content")
	}
	// Markdown should NOT be copied
	if fileExists(filepath.Join(publicDir, "post.md")) {
		t.Error("post.md should NOT be copied")
	}
}

func TestPrepareWithChain_EmptyChain(t *testing.T) {
	publicDir := t.TempDir()
	cfg := config.Config{
		StaticDir:   "/nonexistent",
		PublicDir:   publicDir,
		ContentDir:  "/nonexistent",
		CompileSass: false,
		SassDir:     "",
	}
	if err := PrepareWithChain(cfg, nil, newLogger()); err != nil {
		t.Fatalf("PrepareWithChain with empty chain should succeed, got: %v", err)
	}
}

func TestPrepareWithChain_SassConflictReturnsError(t *testing.T) {
	// Theme has conflicting sass files — should propagate error.
	theme := t.TempDir()
	sassDir := filepath.Join(theme, "sass")
	writeFile(t, filepath.Join(sassDir, "x.scss"), "x{}")
	writeFile(t, filepath.Join(sassDir, "x.sass"), "x")

	publicDir := t.TempDir()
	cfg := config.Config{
		StaticDir:   "/nonexistent",
		PublicDir:   publicDir,
		ContentDir:  "/nonexistent",
		CompileSass: false,
		SassDir:     "",
	}
	chain := []string{theme}

	err := PrepareWithChain(cfg, chain, newLogger())
	if err == nil {
		t.Fatal("expected error from sass conflict")
	}
	if !strings.Contains(err.Error(), "sass conflict") {
		t.Errorf("error = %q, want it to contain 'sass conflict'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Prepare (additional coverage)
// ---------------------------------------------------------------------------

func TestPrepare_BasicFlow_CompileSassFalse(t *testing.T) {
	// A realistic scenario: theme with static, content with images, no sass.
	themesDir := t.TempDir()
	themeStatic := filepath.Join(themesDir, "minimal", "static")
	writeFile(t, filepath.Join(themeStatic, "reset.css"), "* { margin: 0; }")

	contentDir := t.TempDir()
	writeFile(t, filepath.Join(contentDir, "photo.jpg"), "JPEG-DATA")
	writeFile(t, filepath.Join(contentDir, "index.md"), "# Home")
	writeFile(t, filepath.Join(contentDir, ".obsidian"), "vault-meta")

	publicDir := t.TempDir()

	cfg := config.Config{
		Theme:       "minimal",
		ThemesDir:   themesDir,
		StaticDir:   "/nonexistent/static",
		PublicDir:   publicDir,
		ContentDir:  contentDir,
		CompileSass: false,
		SassDir:     "",
	}
	if err := Prepare(cfg, newLogger()); err != nil {
		t.Fatalf("Prepare error: %v", err)
	}

	if !fileExists(filepath.Join(publicDir, "reset.css")) {
		t.Error("reset.css should exist from theme static")
	}
	if !fileExists(filepath.Join(publicDir, "photo.jpg")) {
		t.Error("photo.jpg should be copied from content")
	}
	if fileExists(filepath.Join(publicDir, "index.md")) {
		t.Error("index.md should NOT be copied")
	}
	if fileExists(filepath.Join(publicDir, ".obsidian")) {
		t.Error(".obsidian should NOT be copied")
	}
}

func TestPrepare_NoThemeNoContent(t *testing.T) {
	cfg := config.Config{
		Theme:       "",
		ThemesDir:   "",
		StaticDir:   "",
		PublicDir:   t.TempDir(),
		ContentDir:  "",
		CompileSass: false,
		SassDir:     "",
	}
	if err := Prepare(cfg, newLogger()); err != nil {
		t.Errorf("Prepare with all empty dirs should succeed, got: %v", err)
	}
}

func TestPrepare_WithThemeSassConflict(t *testing.T) {
	themesDir := t.TempDir()
	sassDir := filepath.Join(themesDir, "broken", "sass")
	writeFile(t, filepath.Join(sassDir, "f.scss"), "f{}")
	writeFile(t, filepath.Join(sassDir, "f.sass"), "f")

	cfg := config.Config{
		Theme:       "broken",
		ThemesDir:   themesDir,
		StaticDir:   "",
		PublicDir:   t.TempDir(),
		ContentDir:  "",
		CompileSass: false,
		SassDir:     "",
	}
	err := Prepare(cfg, newLogger())
	if err == nil {
		t.Fatal("expected sass conflict error from Prepare")
	}
	if !strings.Contains(err.Error(), "sass conflict") {
		t.Errorf("error = %q, want sass conflict", err.Error())
	}
}

// ---------------------------------------------------------------------------
// copyStaticChain — child overrides ancestor
// ---------------------------------------------------------------------------

func TestCopyStaticChain_ChildOverridesAncestor(t *testing.T) {
	// Explicitly verify the override semantics with just 2 themes.
	ancestor := t.TempDir()
	child := t.TempDir()
	publicDir := t.TempDir()

	writeFile(t, filepath.Join(ancestor, "static", "logo.svg"), "ancestor-logo")
	writeFile(t, filepath.Join(child, "static", "logo.svg"), "child-logo")

	chain := []string{child, ancestor}
	if err := copyStaticChain(chain, "", publicDir, newLogger()); err != nil {
		t.Fatalf("copyStaticChain error: %v", err)
	}

	if got := readFile(t, filepath.Join(publicDir, "logo.svg")); got != "child-logo" {
		t.Errorf("logo.svg = %q, want child-logo (child should override ancestor)", got)
	}
}

// ---------------------------------------------------------------------------
// compileSassChain — multiple themes with no conflicts
// ---------------------------------------------------------------------------

func TestCompileSassChain_MultipleSassDirsNoConflict(t *testing.T) {
	// Two themes each with a sass/ dir containing non-conflicting files.
	// No sass binary likely available, so accept "sass binary not found" or nil.
	theme1 := t.TempDir()
	theme2 := t.TempDir()
	publicDir := t.TempDir()

	writeFile(t, filepath.Join(theme1, "sass", "a.scss"), "a { color: red; }")
	writeFile(t, filepath.Join(theme2, "sass", "b.scss"), "b { color: blue; }")

	cfg := config.Config{
		CompileSass: false,
		SassDir:     "/nonexistent",
		PublicDir:   publicDir,
	}
	chain := []string{theme1, theme2}

	err := compileSassChain(chain, cfg, newLogger())
	if err != nil && !strings.Contains(err.Error(), "sass binary not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
