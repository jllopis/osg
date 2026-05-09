package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStage_NoExclusionsReturnsPublicDirVerbatim(t *testing.T) {
	pub := t.TempDir()
	staging, err := Stage(pub, nil, nil)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if staging.Dir != pub {
		t.Errorf("Dir = %q, want public dir verbatim %q", staging.Dir, pub)
	}
	if staging.Cleanup == nil {
		t.Error("Cleanup must be a no-op, not nil")
	}
	staging.Cleanup() // must not panic
}

func TestStage_EmptyAndWhitespaceExclusionsAreIgnored(t *testing.T) {
	pub := t.TempDir()
	staging, err := Stage(pub, []string{"", "   ", "\t"}, nil)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if staging.Dir != pub {
		t.Errorf("Dir should fall through to publicDir when no real exclusions, got %q", staging.Dir)
	}
}

func TestStage_RootExclusionIgnored(t *testing.T) {
	// "/" would empty the deploy entirely; treat as nothing.
	pub := t.TempDir()
	staging, err := Stage(pub, []string{"/"}, nil)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if staging.Dir != pub {
		t.Errorf("Dir = %q, want publicDir (root exclusion must be ignored)", staging.Dir)
	}
}

func TestStage_ExcludesSubtree(t *testing.T) {
	pub := t.TempDir()

	// Layout:
	//   pub/keep/index.html
	//   pub/keep/img.jpg
	//   pub/draft/index.html
	//   pub/draft/img.jpg
	//   pub/2026/02/draft-post/index.html
	mustWrite(t, filepath.Join(pub, "keep/index.html"), "k")
	mustWrite(t, filepath.Join(pub, "keep/img.jpg"), "kimg")
	mustWrite(t, filepath.Join(pub, "draft/index.html"), "d")
	mustWrite(t, filepath.Join(pub, "draft/img.jpg"), "dimg")
	mustWrite(t, filepath.Join(pub, "2026/02/draft-post/index.html"), "dp")

	staging, err := Stage(pub, []string{"/draft/", "/2026/02/draft-post/"}, nil)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	defer staging.Cleanup()

	if staging.Dir == pub {
		t.Fatal("staging Dir should be a separate directory when exclusions apply")
	}
	if !exists(filepath.Join(staging.Dir, "keep/index.html")) {
		t.Error("keep/index.html missing from staging")
	}
	if !exists(filepath.Join(staging.Dir, "keep/img.jpg")) {
		t.Error("keep/img.jpg missing from staging")
	}
	if exists(filepath.Join(staging.Dir, "draft/index.html")) {
		t.Error("draft/index.html should have been excluded")
	}
	if exists(filepath.Join(staging.Dir, "draft")) {
		t.Error("draft/ directory should not exist in staging")
	}
	if exists(filepath.Join(staging.Dir, "2026/02/draft-post/index.html")) {
		t.Error("nested deferred path should have been excluded")
	}
	// Sibling under same date prefix must remain.
	mustWrite(t, filepath.Join(pub, "2026/02/keep-post/index.html"), "kp")
	staging2, _ := Stage(pub, []string{"/2026/02/draft-post/"}, nil)
	defer staging2.Cleanup()
	if !exists(filepath.Join(staging2.Dir, "2026/02/keep-post/index.html")) {
		t.Error("sibling under date prefix must survive partial exclusion")
	}
}

func TestStage_CleanupRemovesStagingDir(t *testing.T) {
	pub := t.TempDir()
	mustWrite(t, filepath.Join(pub, "keep/index.html"), "k")
	mustWrite(t, filepath.Join(pub, "drop/index.html"), "d")

	staging, err := Stage(pub, []string{"/drop/"}, nil)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if !exists(staging.Dir) {
		t.Fatal("staging dir should exist after Stage")
	}
	staging.Cleanup()
	if exists(staging.Dir) {
		t.Error("Cleanup must remove the staging directory")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
