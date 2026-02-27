package site

import (
	"strings"
	"testing"
	"time"
)

func TestMenuPages_ReturnsOnlyMenuPages(t *testing.T) {
	s := New()
	s.AddPage(&Page{Title: "Post 1", Path: "/2025/01/01/post-1/", Date: time.Now()})
	s.AddPage(&Page{Title: "About", Path: "/about/", Menu: true, Date: time.Now()})
	s.AddPage(&Page{Title: "Post 2", Path: "/2025/01/02/post-2/", Date: time.Now()})

	menu := s.MenuPages()
	if len(menu) != 1 {
		t.Fatalf("expected 1 menu page, got %d", len(menu))
	}
	if menu[0].Title != "About" {
		t.Fatalf("expected menu page title 'About', got %q", menu[0].Title)
	}
}

func TestMenuPages_EmptyWhenNoMenuPages(t *testing.T) {
	s := New()
	s.AddPage(&Page{Title: "Post 1", Path: "/2025/01/01/post-1/", Date: time.Now()})

	menu := s.MenuPages()
	if len(menu) != 0 {
		t.Fatalf("expected 0 menu pages, got %d", len(menu))
	}
}

func TestBuildHierarchy_MenuPagesExcludedFromSections(t *testing.T) {
	s := New()
	now := time.Now()
	s.AddPage(&Page{Title: "Post 1", Path: "/2025/01/01/post-1/", Date: now})
	s.AddPage(&Page{Title: "About", Path: "/about/", Menu: true, Date: now})
	s.BuildHierarchy()

	// The root section should only contain the regular post, not the menu page.
	root := s.Sections["/"]
	if root == nil {
		t.Fatal("expected root section to exist")
	}

	for _, p := range root.Pages {
		if p.Menu {
			t.Fatalf("menu page %q should not appear in root section pages", p.Title)
		}
	}
}

func TestBuildHierarchy_RootPagesExcludeMenuPages(t *testing.T) {
	s := New()
	now := time.Now()
	s.AddPage(&Page{Title: "Post 1", Path: "/2025/01/01/post-1/", Date: now})
	s.AddPage(&Page{Title: "About", Path: "/about/", Menu: true, Date: now})
	s.BuildHierarchy()

	root := s.Sections["/"]
	if root == nil {
		t.Fatal("expected root section to exist")
	}

	// When root has no direct pages (date-based layout), it gets populated
	// with all non-menu site pages.
	for _, p := range root.Pages {
		if p.Menu {
			t.Fatalf("menu page %q should not be in root.Pages population", p.Title)
		}
	}

	// But the menu page should still be in the site-wide Pages list.
	found := false
	for _, p := range s.Pages {
		if p.Title == "About" && p.Menu {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("menu page should still be in s.Pages")
	}
}

func TestPageView_IncludesMenuField(t *testing.T) {
	p := &Page{
		Title: "About",
		Path:  "/about/",
		Menu:  true,
	}
	view := p.View()
	if menu, ok := view["menu"]; !ok {
		t.Fatal("expected 'menu' key in page view")
	} else if menu != true {
		t.Fatalf("expected menu=true in view, got %v", menu)
	}
}

func TestPageView_MenuFalseByDefault(t *testing.T) {
	p := &Page{
		Title: "Regular Post",
		Path:  "/2025/01/01/post/",
	}
	view := p.View()
	if menu, ok := view["menu"]; !ok {
		t.Fatal("expected 'menu' key in page view")
	} else if menu != false {
		t.Fatalf("expected menu=false in view, got %v", menu)
	}
}

// ---- New ----

func TestNew_RootSection(t *testing.T) {
	s := New()
	if s.Root == nil {
		t.Fatal("expected root section")
	}
	if s.Root.Path != "/" {
		t.Errorf("root path = %q, want /", s.Root.Path)
	}
	if len(s.Sections) != 1 {
		t.Errorf("expected 1 section (root), got %d", len(s.Sections))
	}
}

// ---- AddPage ----

func TestAddPage(t *testing.T) {
	s := New()
	s.AddPage(&Page{Title: "A"})
	s.AddPage(&Page{Title: "B"})
	if len(s.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(s.Pages))
	}
}

// ---- AddSection ----

func TestAddSection_New(t *testing.T) {
	s := New()
	s.AddSection(&Section{Title: "Blog", Path: "/blog/"})
	if len(s.Sections) != 2 { // root + blog
		t.Errorf("expected 2 sections, got %d", len(s.Sections))
	}
}

func TestAddSection_Merge(t *testing.T) {
	s := New()
	s.AddSection(&Section{Title: "Blog", Path: "/blog/"})
	s.AddSection(&Section{Title: "Blog Updated", Path: "/blog/", Content: "new content"})

	blog := s.Sections["/blog/"]
	if blog == nil {
		t.Fatal("expected /blog/ section")
	}
	// Merge should overwrite title and content.
	if blog.Title != "Blog Updated" {
		t.Errorf("title = %q, want 'Blog Updated'", blog.Title)
	}
	if blog.Content != "new content" {
		t.Errorf("content = %q, want 'new content'", blog.Content)
	}
}

// ---- BuildHierarchy ----

func TestBuildHierarchy_PagesAssignedToSections(t *testing.T) {
	s := New()
	s.AddSection(&Section{Title: "Blog", Path: "/blog/"})
	now := time.Now()
	s.AddPage(&Page{Title: "Post 1", Path: "/blog/post-1/", Date: now})
	s.AddPage(&Page{Title: "Post 2", Path: "/blog/post-2/", Date: now.Add(-time.Hour)})
	s.BuildHierarchy()

	blog := s.Sections["/blog/"]
	if blog == nil {
		t.Fatal("expected /blog/ section")
	}
	if len(blog.Pages) != 2 {
		t.Fatalf("expected 2 pages in /blog/, got %d", len(blog.Pages))
	}
}

func TestBuildHierarchy_PagesSortedByDateDesc(t *testing.T) {
	s := New()
	s.AddSection(&Section{Title: "Blog", Path: "/blog/"})
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	s.AddPage(&Page{Title: "Old", Path: "/blog/old/", Date: older})
	s.AddPage(&Page{Title: "New", Path: "/blog/new/", Date: newer})
	s.BuildHierarchy()

	blog := s.Sections["/blog/"]
	if blog.Pages[0].Title != "New" {
		t.Errorf("expected newest page first, got %q", blog.Pages[0].Title)
	}
}

func TestBuildHierarchy_IntermediateSectionsCreated(t *testing.T) {
	s := New()
	// Add a deep page without creating the parent section explicitly.
	s.AddPage(&Page{Title: "Deep Post", Path: "/2025/01/post/", Date: time.Now()})
	s.BuildHierarchy()

	// The hierarchy should create intermediate sections.
	if s.Sections["/2025/"] == nil && s.Sections["/2025/01/"] == nil {
		// At minimum, a parent section should exist for the page.
		found := false
		for path := range s.Sections {
			if strings.HasPrefix(path, "/2025") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected intermediate sections to be created")
		}
	}
}

func TestBuildHierarchy_SubsectionsLinked(t *testing.T) {
	s := New()
	s.AddSection(&Section{Title: "Blog", Path: "/blog/"})
	s.AddSection(&Section{Title: "Tech", Path: "/blog/tech/"})
	s.AddPage(&Page{Title: "Post", Path: "/blog/tech/post/", Date: time.Now()})
	s.BuildHierarchy()

	blog := s.Sections["/blog/"]
	if blog == nil {
		t.Fatal("expected /blog/ section")
	}
	if len(blog.Subsections) != 1 {
		t.Errorf("expected 1 subsection in /blog/, got %d", len(blog.Subsections))
	}
}

func TestBuildHierarchy_RootPopulatedWhenEmpty(t *testing.T) {
	s := New()
	// No explicit root content, just date-based pages.
	now := time.Now()
	s.AddPage(&Page{Title: "Post A", Path: "/2025/01/01/a/", Date: now})
	s.AddPage(&Page{Title: "Post B", Path: "/2025/01/02/b/", Date: now})
	s.BuildHierarchy()

	root := s.Root
	if len(root.Pages) != 2 {
		t.Errorf("root should have 2 pages (populated), got %d", len(root.Pages))
	}
}

// ---- Site.View ----

func TestSiteView(t *testing.T) {
	s := New()
	s.AddPage(&Page{Title: "P1", Path: "/p1/", Date: time.Now()})
	s.BuildHierarchy()

	v := s.View()
	pages, ok := v["pages"]
	if !ok {
		t.Fatal("expected 'pages' key in site view")
	}
	pagesSlice, ok := pages.([]map[string]any)
	if !ok {
		t.Fatalf("pages should be []map[string]any, got %T", pages)
	}
	if len(pagesSlice) != 1 {
		t.Errorf("expected 1 page in view, got %d", len(pagesSlice))
	}
}

// ---- Page.View ----

func TestPageView_AllFields(t *testing.T) {
	p := &Page{
		Title:       "Test Post",
		Slug:        "test-post",
		Path:        "/2025/01/01/test-post/",
		Permalink:   "https://example.com/2025/01/01/test-post/",
		Date:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Summary:     "A test post",
		Content:     "<p>Hello</p>",
		WordCount:   100,
		ReadingTime: 1,
		Image:       "/img/hero.jpg",
		Lang:        "en",
		Taxonomies:  map[string][]string{"tags": {"go", "test"}},
		Extra:       map[string]any{"featured": true},
	}
	v := p.View()

	if v["title"] != "Test Post" {
		t.Errorf("title = %v", v["title"])
	}
	if v["slug"] != "test-post" {
		t.Errorf("slug = %v", v["slug"])
	}
	if v["summary"] != "A test post" {
		t.Errorf("summary = %v", v["summary"])
	}
	if v["word_count"] != 100 {
		t.Errorf("word_count = %v", v["word_count"])
	}
	if v["reading_time"] != 1 {
		t.Errorf("reading_time = %v", v["reading_time"])
	}
	if v["image"] != "/img/hero.jpg" {
		t.Errorf("image = %v", v["image"])
	}
	if v["lang"] != "en" {
		t.Errorf("lang = %v", v["lang"])
	}

	taxonomies, ok := v["taxonomies"].(map[string][]string)
	if !ok {
		t.Fatal("taxonomies should be map[string][]string")
	}
	if len(taxonomies["tags"]) != 2 {
		t.Errorf("expected 2 tags, got %d", len(taxonomies["tags"]))
	}

	extra, ok := v["extra"].(map[string]any)
	if !ok {
		t.Fatal("extra should be map[string]any")
	}
	if extra["featured"] != true {
		t.Errorf("extra.featured = %v", extra["featured"])
	}
}

// ---- Section.View ----

func TestSectionView_FeaturedPage(t *testing.T) {
	now := time.Now()
	s := &Section{
		Title: "Blog",
		Path:  "/blog/",
		Pages: []*Page{
			{Title: "Regular", Path: "/blog/regular/", Date: now.Add(-time.Hour)},
			{Title: "Featured", Path: "/blog/featured/", Date: now, Extra: map[string]any{"featured": true}},
		},
	}

	v := s.View()
	featured, ok := v["featured_page"]
	if !ok {
		t.Fatal("expected featured_page in section view")
	}
	fp, ok := featured.(map[string]any)
	if !ok {
		t.Fatal("featured_page should be map[string]any")
	}
	if fp["title"] != "Featured" {
		t.Errorf("featured page title = %v, want 'Featured'", fp["title"])
	}
}

func TestSectionView_DefaultFeaturedIsMostRecent(t *testing.T) {
	now := time.Now()
	s := &Section{
		Title: "Blog",
		Path:  "/blog/",
		Pages: []*Page{
			{Title: "Newest", Path: "/blog/newest/", Date: now},
			{Title: "Oldest", Path: "/blog/oldest/", Date: now.Add(-24 * time.Hour)},
		},
	}

	v := s.View()
	featured := v["featured_page"].(map[string]any)
	if featured["title"] != "Newest" {
		t.Errorf("default featured should be most recent, got %v", featured["title"])
	}
}
