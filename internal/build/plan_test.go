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
		if !plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("non-incremental plan should always render")
		}
	})

	t.Run("full rebuild always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true, full: true}
		if !plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("full rebuild should always render")
		}
	})

	t.Run("nil page always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true}
		if !plan.shouldRenderPage(nil, existingOutput, "page.html") {
			t.Error("nil page should always render")
		}
	})

	t.Run("empty source path always renders", func(t *testing.T) {
		plan := buildPlan{incremental: true}
		emptySource := &site.Page{SourcePath: ""}
		if !plan.shouldRenderPage(emptySource, existingOutput, "page.html") {
			t.Error("empty source path should always render")
		}
	})

	t.Run("changed file renders", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{"content/changed.md": true},
		}
		if !plan.shouldRenderPage(changedPage, existingOutput, "page.html") {
			t.Error("changed page should render")
		}
	})

	t.Run("unchanged file with existing output skips", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{},
		}
		if plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("unchanged page with existing output should not render")
		}
	})

	t.Run("unchanged file with missing output renders", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{},
		}
		if !plan.shouldRenderPage(page, missingOutput, "page.html") {
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

	t.Run("template change triggers collection render", func(t *testing.T) {
		plan := buildPlan{incremental: true, contentChanged: false, templatesChanged: true}
		if !plan.shouldRenderCollection(existingOutput) {
			t.Error("template change should trigger collection render")
		}
	})
}

// ---------------------------------------------------------------------------
// Template-aware shouldRenderPage
// ---------------------------------------------------------------------------

func TestShouldRenderPage_TemplateChange(t *testing.T) {
	tmp := t.TempDir()
	existingOutput := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(existingOutput, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	page := &site.Page{SourcePath: "content/hello.md"}

	t.Run("page template changed", func(t *testing.T) {
		plan := buildPlan{
			incremental:      true,
			changedFiles:     map[string]bool{},
			templatesChanged: true,
			changedTemplates: map[string]bool{"page.html": true},
		}
		if !plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("should render when page's template changed")
		}
	})

	t.Run("different template changed does not affect page", func(t *testing.T) {
		plan := buildPlan{
			incremental:      true,
			changedFiles:     map[string]bool{},
			templatesChanged: true,
			changedTemplates: map[string]bool{"section.html": true},
		}
		if plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("should not render when a different template changed")
		}
	})

	t.Run("shared partial change triggers all pages", func(t *testing.T) {
		plan := buildPlan{
			incremental:      true,
			changedFiles:     map[string]bool{},
			templatesChanged: true,
			changedTemplates: map[string]bool{"partials/head.html": true},
		}
		if !plan.shouldRenderPage(page, existingOutput, "page.html") {
			t.Error("shared partial change should trigger all pages")
		}
	})
}

// ---------------------------------------------------------------------------
// shouldRenderSection
// ---------------------------------------------------------------------------

func TestShouldRenderSection(t *testing.T) {
	tmp := t.TempDir()
	existingOutput := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(existingOutput, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("page in section changed", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{"content/blog/post.md": true},
			prevSectionPages: map[string][]string{
				"/blog/": {"content/blog/post.md", "content/blog/other.md"},
			},
		}
		if !plan.shouldRenderSection("/blog/", existingOutput, "section.html") {
			t.Error("should render when a page in the section changed")
		}
	})

	t.Run("no pages changed in section", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{"content/other/post.md": true},
			prevSectionPages: map[string][]string{
				"/blog/": {"content/blog/post.md"},
			},
		}
		if plan.shouldRenderSection("/blog/", existingOutput, "section.html") {
			t.Error("should not render when no pages in section changed")
		}
	})

	t.Run("section template changed", func(t *testing.T) {
		plan := buildPlan{
			incremental:      true,
			changedFiles:     map[string]bool{},
			templatesChanged: true,
			changedTemplates: map[string]bool{"section.html": true},
			prevSectionPages: map[string][]string{
				"/blog/": {"content/blog/post.md"},
			},
		}
		if !plan.shouldRenderSection("/blog/", existingOutput, "section.html") {
			t.Error("should render when section template changed")
		}
	})

	t.Run("removed pages trigger section render", func(t *testing.T) {
		plan := buildPlan{
			incremental:  true,
			changedFiles: map[string]bool{},
			removed:      1,
			prevSectionPages: map[string][]string{
				"/blog/": {"content/blog/post.md"},
			},
		}
		if !plan.shouldRenderSection("/blog/", existingOutput, "section.html") {
			t.Error("should render when pages were removed")
		}
	})

	t.Run("new section without prev data renders", func(t *testing.T) {
		plan := buildPlan{
			incremental:      true,
			changedFiles:     map[string]bool{},
			prevSectionPages: map[string][]string{},
		}
		if !plan.shouldRenderSection("/new-section/", existingOutput, "section.html") {
			t.Error("should render new section without previous data")
		}
	})
}

// ---------------------------------------------------------------------------
// diffTemplates
// ---------------------------------------------------------------------------

func TestDiffTemplates(t *testing.T) {
	t.Run("nil prev returns all current as changed", func(t *testing.T) {
		current := map[string]string{"page.html": "abc", "section.html": "def"}
		changed := diffTemplates(nil, current)
		if len(changed) != 2 {
			t.Errorf("expected 2 changed, got %d", len(changed))
		}
	})

	t.Run("detects modified template", func(t *testing.T) {
		prev := map[string]string{"page.html": "abc"}
		current := map[string]string{"page.html": "xyz"}
		changed := diffTemplates(prev, current)
		if !changed["page.html"] {
			t.Error("page.html should be changed")
		}
	})

	t.Run("detects new template", func(t *testing.T) {
		prev := map[string]string{"page.html": "abc"}
		current := map[string]string{"page.html": "abc", "custom.html": "new"}
		changed := diffTemplates(prev, current)
		if !changed["custom.html"] {
			t.Error("custom.html should be in changed")
		}
		if changed["page.html"] {
			t.Error("page.html should not be changed")
		}
	})

	t.Run("detects removed template", func(t *testing.T) {
		prev := map[string]string{"page.html": "abc", "old.html": "def"}
		current := map[string]string{"page.html": "abc"}
		changed := diffTemplates(prev, current)
		if !changed["old.html"] {
			t.Error("removed old.html should be in changed")
		}
	})
}

// ---------------------------------------------------------------------------
// hasSharedPartialChanged
// ---------------------------------------------------------------------------

func TestHasSharedPartialChanged(t *testing.T) {
	t.Run("partial in changed set", func(t *testing.T) {
		if !hasSharedPartialChanged(map[string]bool{"partials/head.html": true}) {
			t.Error("should detect partial change")
		}
	})

	t.Run("no partials changed", func(t *testing.T) {
		if hasSharedPartialChanged(map[string]bool{"page.html": true}) {
			t.Error("should not detect partial change")
		}
	})

	t.Run("empty set", func(t *testing.T) {
		if hasSharedPartialChanged(map[string]bool{}) {
			t.Error("should not detect partial change in empty set")
		}
	})
}
