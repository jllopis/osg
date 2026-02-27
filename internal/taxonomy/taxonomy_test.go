package taxonomy

import (
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/site"
)

func makePage(title string, date time.Time, taxonomies map[string][]string) *site.Page {
	return &site.Page{
		Title:      title,
		Slug:       title,
		Path:       "/" + title + "/",
		Date:       date,
		Taxonomies: taxonomies,
	}
}

func TestBuild_EmptyConfigs(t *testing.T) {
	t.Parallel()
	result := Build(nil, nil, "")
	if len(result) != 0 {
		t.Errorf("expected empty indices for nil configs, got %d", len(result))
	}
}

func TestBuild_EmptyPages(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{{Name: "tags"}}
	result := Build(cfgs, nil, "")
	if len(result) != 1 {
		t.Fatalf("expected 1 index, got %d", len(result))
	}
	if len(result["tags"].Terms) != 0 {
		t.Errorf("expected 0 terms, got %d", len(result["tags"].Terms))
	}
}

func TestBuild_SingleTaxonomy(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{{Name: "tags"}}
	pages := []*site.Page{
		makePage("post-1", time.Now(), map[string][]string{"tags": {"go", "testing"}}),
		makePage("post-2", time.Now(), map[string][]string{"tags": {"go"}}),
	}

	result := Build(cfgs, pages, "https://example.com")
	idx := result["tags"]
	if idx == nil {
		t.Fatal("expected 'tags' index")
	}
	if len(idx.Terms) != 2 {
		t.Fatalf("expected 2 terms (go, testing), got %d", len(idx.Terms))
	}

	goTerm := idx.Terms["go"]
	if goTerm == nil {
		t.Fatal("expected 'go' term")
	}
	if len(goTerm.Pages) != 2 {
		t.Errorf("expected 2 pages for 'go' term, got %d", len(goTerm.Pages))
	}
	if goTerm.Path != "/tags/go/" {
		t.Errorf("expected path /tags/go/, got %s", goTerm.Path)
	}
	if goTerm.Permalink != "https://example.com/tags/go/" {
		t.Errorf("expected permalink https://example.com/tags/go/, got %s", goTerm.Permalink)
	}

	testingTerm := idx.Terms["testing"]
	if testingTerm == nil {
		t.Fatal("expected 'testing' term")
	}
	if len(testingTerm.Pages) != 1 {
		t.Errorf("expected 1 page for 'testing' term, got %d", len(testingTerm.Pages))
	}
}

func TestBuild_MultipleTaxonomies(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{
		{Name: "tags"},
		{Name: "area"},
	}
	pages := []*site.Page{
		makePage("post-1", time.Now(), map[string][]string{
			"tags": {"go"},
			"area": {"backend"},
		}),
	}

	result := Build(cfgs, pages, "")
	if len(result) != 2 {
		t.Fatalf("expected 2 indices, got %d", len(result))
	}
	if result["tags"].Terms["go"] == nil {
		t.Error("expected 'go' term in tags")
	}
	if result["area"].Terms["backend"] == nil {
		t.Error("expected 'backend' term in area")
	}
}

func TestBuild_EmptyTermNameSkipped(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{{Name: "tags"}}
	pages := []*site.Page{
		makePage("post-1", time.Now(), map[string][]string{"tags": {"go", "", "  "}}),
	}

	result := Build(cfgs, pages, "")
	if len(result["tags"].Terms) != 1 {
		t.Errorf("expected 1 term (empty terms skipped), got %d", len(result["tags"].Terms))
	}
}

func TestBuild_IgnoresUnknownTaxonomy(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{{Name: "tags"}}
	pages := []*site.Page{
		makePage("post-1", time.Now(), map[string][]string{"category": {"misc"}}),
	}

	result := Build(cfgs, pages, "")
	if len(result["tags"].Terms) != 0 {
		t.Errorf("expected 0 terms for tags when pages only have category, got %d", len(result["tags"].Terms))
	}
}

func TestBuild_PagesSortedByDateDesc(t *testing.T) {
	t.Parallel()
	cfgs := []config.TaxonomyConfig{{Name: "tags"}}
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	pages := []*site.Page{
		makePage("old-post", older, map[string][]string{"tags": {"go"}}),
		makePage("new-post", newer, map[string][]string{"tags": {"go"}}),
	}

	result := Build(cfgs, pages, "")
	goTerm := result["tags"].Terms["go"]
	if goTerm.Pages[0].Title != "new-post" {
		t.Errorf("expected newest post first, got %s", goTerm.Pages[0].Title)
	}
}

func TestTermsSorted(t *testing.T) {
	t.Parallel()
	idx := &Index{Terms: map[string]*Term{
		"zebra": {Name: "Zebra"},
		"alpha": {Name: "Alpha"},
		"mid":   {Name: "Mid"},
	}}

	sorted := idx.TermsSorted()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(sorted))
	}
	if sorted[0].Name != "Alpha" || sorted[1].Name != "Mid" || sorted[2].Name != "Zebra" {
		t.Errorf("expected alphabetical order, got %s, %s, %s",
			sorted[0].Name, sorted[1].Name, sorted[2].Name)
	}
}

func TestConfigView(t *testing.T) {
	t.Parallel()
	cfg := config.TaxonomyConfig{
		Name:         "tags",
		PaginateBy:   10,
		PaginatePath: "page",
		Feed:         true,
		Render:       true,
	}
	v := ConfigView(cfg)
	if v["name"] != "tags" {
		t.Errorf("expected name 'tags', got %v", v["name"])
	}
	if v["paginate_by"] != 10 {
		t.Errorf("expected paginate_by 10, got %v", v["paginate_by"])
	}
}

func TestTermView(t *testing.T) {
	t.Parallel()
	term := &Term{
		Name:      "Go",
		Slug:      "go",
		Path:      "/tags/go/",
		Permalink: "https://example.com/tags/go/",
		Pages:     []*site.Page{makePage("post-1", time.Now(), nil)},
	}
	v := TermView(term)
	if v["name"] != "Go" {
		t.Errorf("expected name 'Go', got %v", v["name"])
	}
	if v["path"] != "/tags/go/" {
		t.Errorf("expected path /tags/go/, got %v", v["path"])
	}
	if v["page_count"] != 1 {
		t.Errorf("expected page_count 1, got %v", v["page_count"])
	}
}

func TestTermViews(t *testing.T) {
	t.Parallel()
	terms := []*Term{
		{Name: "A"},
		{Name: "B"},
	}
	views := TermViews(terms)
	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}
}

// ---- Pagination Tests ----

func TestBuildPaginator_NoPagination(t *testing.T) {
	t.Parallel()
	pages := make([]*site.Page, 5)
	for i := range pages {
		pages[i] = &site.Page{Title: "p"}
	}

	// paginateBy 0 -> no pagination
	result := BuildPaginator(pages, 0, "/tags/go/", "page")
	if result != nil {
		t.Errorf("expected nil for paginateBy=0, got %d paginators", len(result))
	}

	// All pages fit on one page
	result = BuildPaginator(pages, 10, "/tags/go/", "page")
	if result != nil {
		t.Errorf("expected nil when all pages fit, got %d paginators", len(result))
	}
}

func TestBuildPaginator_BasicPagination(t *testing.T) {
	t.Parallel()
	pages := make([]*site.Page, 7)
	for i := range pages {
		pages[i] = &site.Page{Title: "p"}
	}

	result := BuildPaginator(pages, 3, "/tags/go/", "page")
	if len(result) != 3 {
		t.Fatalf("expected 3 paginators for 7 items / 3 per page, got %d", len(result))
	}

	// Page 1
	p1 := result[0]
	if len(p1.Pages) != 3 {
		t.Errorf("page 1 should have 3 items, got %d", len(p1.Pages))
	}
	if p1.CurrentIndex != 1 {
		t.Errorf("page 1 CurrentIndex should be 1, got %d", p1.CurrentIndex)
	}
	if p1.Previous != "" {
		t.Errorf("page 1 should have no Previous, got %q", p1.Previous)
	}
	if p1.Next == "" {
		t.Error("page 1 should have Next")
	}
	if p1.First != "/tags/go/" {
		t.Errorf("First should be base URL, got %q", p1.First)
	}

	// Page 2
	p2 := result[1]
	if len(p2.Pages) != 3 {
		t.Errorf("page 2 should have 3 items, got %d", len(p2.Pages))
	}
	if p2.Previous != "/tags/go/" {
		t.Errorf("page 2 Previous should be base URL, got %q", p2.Previous)
	}

	// Page 3 (last)
	p3 := result[2]
	if len(p3.Pages) != 1 {
		t.Errorf("page 3 should have 1 item, got %d", len(p3.Pages))
	}
	if p3.Next != "" {
		t.Errorf("last page should have no Next, got %q", p3.Next)
	}
	if p3.Last != p3.BaseURL+"page/3/" {
		t.Errorf("Last should be page/3/, got %q", p3.Last)
	}
}

func TestBuildPaginator_DefaultPaginatePath(t *testing.T) {
	t.Parallel()
	pages := make([]*site.Page, 5)
	for i := range pages {
		pages[i] = &site.Page{Title: "p"}
	}

	result := BuildPaginator(pages, 2, "/sec/", "")
	if result == nil {
		t.Fatal("expected paginators")
	}
	if result[1].Previous != "/sec/" {
		t.Errorf("page 2 Previous should be base URL, got %q", result[1].Previous)
	}
	// Default path is "page"
	if result[1].Last == "" {
		t.Error("Last should not be empty")
	}
}

func TestPaginatorView(t *testing.T) {
	t.Parallel()
	p := Paginator{
		PaginateBy:   10,
		NumberPagers: 3,
		CurrentIndex: 2,
		TotalPages:   3,
		First:        "/a/",
		Last:         "/a/page/3/",
		Previous:     "/a/",
		Next:         "/a/page/3/",
		Pages:        []*site.Page{{Title: "p"}},
	}
	v := PaginatorView(p)
	if v["current_index"] != 2 {
		t.Errorf("expected current_index=2, got %v", v["current_index"])
	}
	if v["total_pages"] != 3 {
		t.Errorf("expected total_pages=3, got %v", v["total_pages"])
	}
	if v["first"] != "/a/" {
		t.Errorf("expected first=/a/, got %v", v["first"])
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/tags/go", "/tags/go/"},
		{"/tags/go/", "/tags/go/"},
		{"/", "/"},
		{"", "/"},
	}
	for _, tc := range tests {
		got := ensureTrailingSlash(tc.input)
		if got != tc.want {
			t.Errorf("ensureTrailingSlash(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
