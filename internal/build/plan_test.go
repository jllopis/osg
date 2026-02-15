package build

import (
	"os"
	"path/filepath"
	"testing"

	"osg/internal/site"
)

// ---------------------------------------------------------------------------
// outputMissing
// ---------------------------------------------------------------------------

func TestOutputMissing(t *testing.T) {
	t.Run("empty path returns true", func(t *testing.T) {
		if !outputMissing("") {
			t.Error("outputMissing(\"\") = false; want true")
		}
	})

	t.Run("nonexistent path returns true", func(t *testing.T) {
		if !outputMissing("/nonexistent/path/to/file.html") {
			t.Error("outputMissing(nonexistent) = false; want true")
		}
	})

	t.Run("existing file returns false", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "index.html")
		if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
		if outputMissing(f) {
			t.Errorf("outputMissing(%q) = true; want false", f)
		}
	})
}

// ---------------------------------------------------------------------------
// buildPlan.shouldRenderPage
// ---------------------------------------------------------------------------

func TestShouldRenderPage(t *testing.T) {
	tmp := t.TempDir()
	existingOutput := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(existingOutput, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingOutput := filepath.Join(tmp, "missing", "index.html")

	page := &site.Page{SourcePath: "content/hello.md"}
	changedPage := &site.Page{SourcePath: "content/changed.md"}

	t.Run("non-incremental always renders", func(t *testing.T) {
		plan := buildPlan{incremental: false}
		if !plan.shouldRenderPage(page, existingOutput) {
			t.Error("non-incremental plan should always render")
		}
	})

	t.Run("full rebuild always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true, full: true}
		if !plan.shouldRenderPage(page, existingOutput) {
			t.Error("full rebuild should always render")
		}
	})

	t.Run("nil page always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true}
		if !plan.shouldRenderPage(nil, existingOutput) {
			t.Error("nil page should always render")
		}
	})

	t.Run("empty source path always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true}
		emptySource := &site.Page{SourcePath: ""}
		if !plan.shouldRenderPage(emptySource, existingOutput) {
			t.Error("empty source path should always render")
		}
	})

	t.Run("changed file renders", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{"content/changed.md": true},
		}
		if !plan.shouldRenderPage(changedPage, existingOutput) {
			t.Error("changed page should render")
		}
	})

	t.Run("unchanged file with existing output skips", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{},
		}
		if plan.shouldRenderPage(page, existingOutput) {
			t.Error("unchanged page with existing output should not render")
		}
	})

	t.Run("unchanged file with missing output renders", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{},
		}
		if !plan.shouldRenderPage(page, missingOutput) {
			t.Error("unchanged page with missing output should render")
		}
	})
}

// ---------------------------------------------------------------------------
// buildPlan.shouldRenderCollection
// ---------------------------------------------------------------------------

func TestShouldRenderCollection(t *testing.T) {
	tmp := t.TempDir()
	existingOutput := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(existingOutput, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingOutput := filepath.Join(tmp, "missing", "index.html")

	t.Run("non-incremental always renders", func(t *testing.T) {
		plan := buildPlan{incremental: false}
		if !plan.shouldRenderCollection(existingOutput) {
			t.Error("non-incremental plan should always render")
		}
	})

	t.Run("full rebuild always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true, full: true}
		if !plan.shouldRenderCollection(existingOutput) {
			t.Error("full rebuild should always render")
		}
	})

	t.Run("content changed always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true, contentChanged: true}
		if !plan.shouldRenderCollection(existingOutput) {
			t.Error("content changed should always render collection")
		}
	})

	t.Run("no content change with existing output skips", func(t *testing.T) {
		plan := buildPlan{incremental: true, contentChanged: false}
		if plan.shouldRenderCollection(existingOutput) {
			t.Error("no content change with existing output should not render")
		}
	})

	t.Run("no content change with missing output renders", func(t *testing.T) {
		plan := buildPlan{incremental: true, contentChanged: false}
		if !plan.shouldRenderCollection(missingOutput) {
			t.Error("no content change with missing output should render")
		}
	})
}
