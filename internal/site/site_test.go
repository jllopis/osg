package site

import (
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
