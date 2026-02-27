package markdown

import (
	"testing"
)

func TestExtractTOC_NoHeadings(t *testing.T) {
	result := ExtractTOC("<p>Hello world</p>")
	if result != nil {
		t.Errorf("expected nil for no headings, got %v", result)
	}
}

func TestExtractTOC_H1Ignored(t *testing.T) {
	// Only h2-h6 are extracted, not h1.
	result := ExtractTOC(`<h1 id="title">Title</h1>`)
	if result != nil {
		t.Errorf("expected nil for h1, got %v", result)
	}
}

func TestExtractTOC_BasicHeadings(t *testing.T) {
	html := `<h2 id="intro">Introduction</h2><p>text</p><h3 id="setup">Setup</h3><p>text</p><h2 id="conclusion">Conclusion</h2>`
	entries := ExtractTOC(html)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Level != 2 || entries[0].ID != "intro" || entries[0].Title != "Introduction" {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if entries[1].Level != 3 || entries[1].ID != "setup" || entries[1].Title != "Setup" {
		t.Errorf("entry 1: %+v", entries[1])
	}
	if entries[2].Level != 2 || entries[2].ID != "conclusion" || entries[2].Title != "Conclusion" {
		t.Errorf("entry 2: %+v", entries[2])
	}
}

func TestExtractTOC_HTMLInHeading(t *testing.T) {
	html := `<h2 id="bold">Some <strong>bold</strong> text</h2>`
	entries := ExtractTOC(html)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "Some bold text" {
		t.Errorf("expected stripped HTML, got %q", entries[0].Title)
	}
}

func TestExtractTOC_EntityDecoding(t *testing.T) {
	html := `<h2 id="amps">A &amp; B</h2>`
	entries := ExtractTOC(html)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Title != "A & B" {
		t.Errorf("expected decoded entity, got %q", entries[0].Title)
	}
}

func TestTOCView_Nil(t *testing.T) {
	result := TOCView(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestTOCView_Basic(t *testing.T) {
	entries := []TOCEntry{
		{Level: 2, ID: "a", Title: "A"},
		{Level: 3, ID: "b", Title: "B"},
	}
	views := TOCView(entries)
	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}
	if views[0]["level"] != 2 || views[0]["id"] != "a" || views[0]["title"] != "A" {
		t.Errorf("view 0: %v", views[0])
	}
}
