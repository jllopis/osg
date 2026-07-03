package build

import (
	"testing"
	"time"

	"osg/internal/site"
	"osg/internal/taxonomy"
)

// A changed page must dirty the pages that render it in their navigation:
// chronological neighbors, series mates, backlink targets — and, via the
// previous build's cached order/link graph, the neighbors of removed pages.
func TestStaleNeighborSources(t *testing.T) {
	day := 24 * time.Hour
	now := time.Now()
	newPage := func(name string, age time.Duration) *site.Page {
		return &site.Page{
			Title: name, Path: "/" + name + "/", SourcePath: name + ".md",
			Date: now.Add(-age),
		}
	}
	a := newPage("a", 0*day)
	b := newPage("b", 1*day)
	c := newPage("c", 2*day)
	target := newPage("target", 30*day)
	far := newPage("far", 31*day)
	b.Content = `<p>see <a href="/target/">target</a></p>`

	idx := site.New()
	for _, p := range []*site.Page{a, b, c, target, far} {
		idx.AddPage(p)
	}
	postPages := []*site.Page{a, b, c, target, far} // newest-first
	pagePos := map[*site.Page]int{}
	for i, p := range postPages {
		pagePos[p] = i
	}
	backlinks := buildBacklinkIndex(idx.Pages)
	series := buildSeriesIndex(idx.Pages)
	noTax := map[string]*taxonomy.Index{}

	// Editing b must dirty its neighbors a and c, and its link target.
	plan := buildPlan{changedFiles: map[string]bool{"b.md": true}}
	dirty := staleNeighborSources(plan, idx, postPages, pagePos, series, backlinks, noTax)
	for _, want := range []string{"a.md", "c.md", "target.md"} {
		if !dirty[want] {
			t.Errorf("editing b: %s not marked dirty (got %v)", want, dirty)
		}
	}
	if dirty["far.md"] {
		t.Errorf("editing b: far.md should not be dirty")
	}
	if dirty["b.md"] {
		t.Errorf("changed pages must not appear in the expansion set")
	}

	// Removing x must dirty its former neighbors and former link targets,
	// reconstructed from the previous build's cache.
	plan = buildPlan{
		changedFiles:  map[string]bool{},
		removedFiles:  []string{"x.md"},
		prevPageOrder: []string{"a.md", "x.md", "c.md"},
		prevPageLinks: map[string][]string{"x.md": {"/target/"}},
	}
	dirty = staleNeighborSources(plan, idx, postPages, pagePos, series, backlinks, noTax)
	for _, want := range []string{"a.md", "c.md", "target.md"} {
		if !dirty[want] {
			t.Errorf("removing x: %s not marked dirty (got %v)", want, dirty)
		}
	}
}
