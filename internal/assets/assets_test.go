package assets

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

// ---------------------------------------------------------------------------
// copyFile
// ---------------------------------------------------------------------------

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(src, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "dest.txt")
	if err := copyFile(src, dest); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content = %q, want %q", string(got), "hello world")
	}
}

func TestCopyFile_MissingSrc(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dest.txt")
	err := copyFile("/nonexistent/file.txt", dest)
	if err == nil {
		t.Error("expected error for missing source")
	}
}

// ---------------------------------------------------------------------------
// copyDirFiltered
// ---------------------------------------------------------------------------

func TestCopyDirFiltered_EmptySrc(t *testing.T) {
	logger := slog.Default()
	err := copyDirFiltered("", t.TempDir(), logger, func(_ string, _ os.DirEntry) bool { return false })
	if err != nil {
		t.Errorf("empty src should return nil, got: %v", err)
	}
}

func TestCopyDirFiltered_WhitespaceSrc(t *testing.T) {
	logger := slog.Default()
	err := copyDirFiltered("  ", t.TempDir(), logger, func(_ string, _ os.DirEntry) bool { return false })
	if err != nil {
		t.Errorf("whitespace src should return nil, got: %v", err)
	}
}

func TestCopyDirFiltered_NonexistentSrc(t *testing.T) {
	logger := slog.Default()
	err := copyDirFiltered("/nonexistent/path", t.TempDir(), logger, func(_ string, _ os.DirEntry) bool { return false })
	if err != nil {
		t.Errorf("nonexistent src should return nil, got: %v", err)
	}
}

func TestCopyDirFiltered_SrcIsFile(t *testing.T) {
	// Create a file, not a directory
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := slog.Default()
	err := copyDirFiltered(f, t.TempDir(), logger, func(_ string, _ os.DirEntry) bool { return false })
	if err != nil {
		t.Errorf("src that is a file should return nil, got: %v", err)
	}
}

func TestCopyDirFiltered_CopiesFiles(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	logger := slog.Default()

	// Create files in src
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := copyDirFiltered(src, dest, logger, func(_ string, _ os.DirEntry) bool { return false })
	if err != nil {
		t.Fatalf("copyDirFiltered error: %v", err)
	}

	// Verify files exist in dest
	gotA, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	if err != nil {
		t.Fatalf("missing a.txt: %v", err)
	}
	if string(gotA) != "aaa" {
		t.Errorf("a.txt = %q, want %q", string(gotA), "aaa")
	}

	gotB, err := os.ReadFile(filepath.Join(dest, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("missing sub/b.txt: %v", err)
	}
	if string(gotB) != "bbb" {
		t.Errorf("sub/b.txt = %q, want %q", string(gotB), "bbb")
	}
}

func TestCopyDirFiltered_SkipCallback(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	logger := slog.Default()

	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skip.log"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Skip .log files
	err := copyDirFiltered(src, dest, logger, func(_ string, d os.DirEntry) bool {
		return filepath.Ext(d.Name()) == ".log"
	})
	if err != nil {
		t.Fatalf("copyDirFiltered error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "keep.txt")); err != nil {
		t.Error("keep.txt should exist")
	}
	if _, err := os.Stat(filepath.Join(dest, "skip.log")); err == nil {
		t.Error("skip.log should NOT exist (was skipped)")
	}
}

// ---------------------------------------------------------------------------
// copyDir (skips dotfiles)
// ---------------------------------------------------------------------------

func TestCopyDir_SkipsDotfiles(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	logger := slog.Default()

	if err := os.WriteFile(filepath.Join(src, "visible.txt"), []byte("vis"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".hidden"), []byte("hid"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dest, logger); err != nil {
		t.Fatalf("copyDir error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "visible.txt")); err != nil {
		t.Error("visible.txt should be copied")
	}
	if _, err := os.Stat(filepath.Join(dest, ".hidden")); err == nil {
		t.Error(".hidden should NOT be copied")
	}
}

// ---------------------------------------------------------------------------
// copyContentAssets
// ---------------------------------------------------------------------------

func TestCopyContentAssets(t *testing.T) {
	contentDir := t.TempDir()
	publicDir := t.TempDir()
	logger := slog.Default()

	// Create content files
	if err := os.WriteFile(filepath.Join(contentDir, "post.md"), []byte("# Title"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "image.png"), []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, ".DS_Store"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{ContentDir: contentDir, PublicDir: publicDir}
	if err := copyContentAssets(cfg, logger); err != nil {
		t.Fatalf("copyContentAssets error: %v", err)
	}

	// .md files should be skipped
	if _, err := os.Stat(filepath.Join(publicDir, "post.md")); err == nil {
		t.Error("post.md should NOT be copied")
	}
	// dotfiles should be skipped
	if _, err := os.Stat(filepath.Join(publicDir, ".DS_Store")); err == nil {
		t.Error(".DS_Store should NOT be copied")
	}
	// images should be copied
	if _, err := os.Stat(filepath.Join(publicDir, "image.png")); err != nil {
		t.Error("image.png should be copied")
	}
}

// ---------------------------------------------------------------------------
// sassConflicts
// ---------------------------------------------------------------------------

func TestSassConflicts_NoConflicts(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.scss"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "utils.scss"), []byte("a{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := sassConflicts(root)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}

func TestSassConflicts_WithConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.scss"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.sass"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := sassConflicts(root)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %v", len(conflicts), conflicts)
	}
	if filepath.Base(conflicts[0]) != "main" {
		t.Errorf("conflict base = %q, want %q", filepath.Base(conflicts[0]), "main")
	}
}

func TestSassConflicts_DifferentDirsNoConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Same base name but different directories — should NOT conflict
	if err := os.WriteFile(filepath.Join(root, "a", "main.scss"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b", "main.sass"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := sassConflicts(root)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts across dirs, got %v", conflicts)
	}
}

func TestSassConflicts_EmptyDir(t *testing.T) {
	root := t.TempDir()
	conflicts, err := sassConflicts(root)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts in empty dir, got %v", conflicts)
	}
}

func TestSassConflicts_SkipsDotDirs(t *testing.T) {
	root := t.TempDir()
	dotDir := filepath.Join(root, ".hidden")
	if err := os.MkdirAll(dotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Files inside a dotdir should be ignored
	if err := os.WriteFile(filepath.Join(dotDir, "main.scss"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dotDir, "main.sass"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := sassConflicts(root)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts in dotdir, got %v", conflicts)
	}
}

func TestSassConflicts_NonexistentRoot(t *testing.T) {
	_, err := sassConflicts("/nonexistent/path")
	// WalkDir on nonexistent path returns an error
	if err == nil {
		t.Error("expected error for nonexistent root")
	}
}

// ---------------------------------------------------------------------------
// copyStatic
// ---------------------------------------------------------------------------

func TestCopyStatic_WithTheme(t *testing.T) {
	themesDir := t.TempDir()
	publicDir := t.TempDir()
	logger := slog.Default()

	themeStatic := filepath.Join(themesDir, "default", "static")
	if err := os.MkdirAll(themeStatic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeStatic, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Theme:     "default",
		ThemesDir: themesDir,
		PublicDir: publicDir,
		StaticDir: "/nonexistent/static", // no site-level static
	}
	if err := copyStatic(cfg, logger); err != nil {
		t.Fatalf("copyStatic error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(publicDir, "style.css")); err != nil {
		t.Error("style.css should be copied from theme static")
	}
}

func TestCopyStatic_NoTheme(t *testing.T) {
	staticDir := t.TempDir()
	publicDir := t.TempDir()
	logger := slog.Default()

	if err := os.WriteFile(filepath.Join(staticDir, "robots.txt"), []byte("User-agent: *"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Theme:     "",
		StaticDir: staticDir,
		PublicDir: publicDir,
	}
	if err := copyStatic(cfg, logger); err != nil {
		t.Fatalf("copyStatic error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(publicDir, "robots.txt")); err != nil {
		t.Error("robots.txt should be copied from static dir")
	}
}

// ---------------------------------------------------------------------------
// compileSass (guard paths only — no sass binary needed)
// ---------------------------------------------------------------------------

func TestCompileSass_NoThemeNoCompile(t *testing.T) {
	// With no theme and CompileSass false, should be a no-op
	cfg := config.Config{
		Theme:       "",
		CompileSass: false,
		SassDir:     "/nonexistent",
	}
	logger := slog.Default()
	err := compileSass(cfg, logger)
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestCompileSassDir_EmptySrc(t *testing.T) {
	err := compileSassDir("", t.TempDir(), slog.Default())
	if err != nil {
		t.Errorf("empty src should return nil, got: %v", err)
	}
}

func TestCompileSassDir_NonexistentSrc(t *testing.T) {
	err := compileSassDir("/nonexistent/sass", t.TempDir(), slog.Default())
	if err != nil {
		t.Errorf("nonexistent src should return nil, got: %v", err)
	}
}

func TestCompileSassDir_SrcIsFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := compileSassDir(f, t.TempDir(), slog.Default())
	if err != nil {
		t.Errorf("src file should return nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Prepare (integration, guard paths)
// ---------------------------------------------------------------------------

func TestPrepare_EmptyDirs(t *testing.T) {
	// All dirs nonexistent → should succeed (all guard paths return nil)
	cfg := config.Config{
		Theme:       "",
		ContentDir:  "/nonexistent/content",
		PublicDir:   t.TempDir(),
		StaticDir:   "/nonexistent/static",
		CompileSass: false,
	}
	logger := slog.Default()
	err := Prepare(cfg, logger)
	if err != nil {
		t.Errorf("Prepare with empty dirs should succeed, got: %v", err)
	}
}
