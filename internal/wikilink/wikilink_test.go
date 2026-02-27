package wikilink

import (
	"testing"
)

func TestFindImageLinks(t *testing.T) {
	body := []byte(`Some text before.

![[photo.png]]
![[Attachments/hero.jpg|Hero Image]]
![[diagram.svg]]
![[Not an image note]]
![[video.mp4]]
![[  spaces.png  |  alt with spaces  ]]

Some text after.`)

	matches := FindImageLinks(body)

	expected := []struct {
		ref string
		alt string
	}{
		{"photo.png", ""},
		{"Attachments/hero.jpg", "Hero Image"},
		{"diagram.svg", ""},
		{"spaces.png", "alt with spaces"},
	}

	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d: %+v", len(expected), len(matches), matches)
	}

	for i, exp := range expected {
		if matches[i].Ref != exp.ref {
			t.Errorf("match[%d].Ref = %q, want %q", i, matches[i].Ref, exp.ref)
		}
		if matches[i].AltText != exp.alt {
			t.Errorf("match[%d].AltText = %q, want %q", i, matches[i].AltText, exp.alt)
		}
	}
}

func TestFindImageLinks_Empty(t *testing.T) {
	matches := FindImageLinks([]byte("No wikilinks here."))
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestRewriteImageLinks(t *testing.T) {
	body := []byte(`Text before.

![[photo.png]]
![[Attachments/hero.jpg|Hero Image]]
![[unknown.png]]
![[Not a link]]

Text after.`)

	resolver := func(ref string) (string, bool) {
		switch ref {
		case "photo.png":
			return "photo.png", true
		case "Attachments/hero.jpg":
			return "hero.jpg", true
		default:
			return "", false
		}
	}

	result := RewriteImageLinks(body, resolver)
	resultStr := string(result)

	// photo.png -> resolved, alt derived from filename (no encoding needed)
	if expected := "![photo](photo.png)"; !contains(resultStr, expected) {
		t.Errorf("expected %q in result, got:\n%s", expected, resultStr)
	}

	// hero.jpg -> resolved, alt from wikilink (no encoding needed)
	if expected := "![Hero Image](hero.jpg)"; !contains(resultStr, expected) {
		t.Errorf("expected %q in result, got:\n%s", expected, resultStr)
	}

	// unknown.png -> not resolved, left unchanged
	if expected := "![[unknown.png]]"; !contains(resultStr, expected) {
		t.Errorf("expected %q unchanged in result, got:\n%s", expected, resultStr)
	}

	// Non-image wikilink -> left unchanged
	if expected := "![[Not a link]]"; !contains(resultStr, expected) {
		t.Errorf("expected %q unchanged in result, got:\n%s", expected, resultStr)
	}
}

func TestRewriteImageLinks_SpacesInFilename(t *testing.T) {
	body := []byte(`![[Pasted image 20240822.png|My Photo]]`)

	resolver := func(ref string) (string, bool) {
		if ref == "Pasted image 20240822.png" {
			return "Pasted image 20240822.png", true
		}
		return "", false
	}

	result := string(RewriteImageLinks(body, resolver))
	expected := "![My Photo](Pasted%20image%2020240822.png)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// NormalizeTitle
// ---------------------------------------------------------------------------

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hello World", "hello world"},
		{"  Hello  ", "hello"},
		{"", ""},
		{"UPPER", "upper"},
		{"already lower", "already lower"},
		{"  ", ""},
		{"Mixed CASE Title", "mixed case title"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := NormalizeTitle(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isImageExt
// ---------------------------------------------------------------------------

func TestIsImageExt(t *testing.T) {
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".avif"}
	for _, ext := range imageExts {
		if !isImageExt(ext) {
			t.Errorf("isImageExt(%q) = false, want true", ext)
		}
	}

	nonImageExts := []string{".mp4", ".pdf", ".txt", ".md", ".html", "", ".PNG", ".JPG"}
	for _, ext := range nonImageExts {
		if isImageExt(ext) {
			t.Errorf("isImageExt(%q) = true, want false", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// urlEncodePath
// ---------------------------------------------------------------------------

func TestUrlEncodePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no encoding needed", "photo.png", "photo.png"},
		{"spaces", "my photo.png", "my%20photo.png"},
		{"special chars", "file (1).png", "file%20%281%29.png"},
		{"empty", "", ""},
		{"path separators", "dir/file.png", "dir%2Ffile.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlEncodePath(tt.in)
			if got != tt.want {
				t.Errorf("urlEncodePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FindTextLinks
// ---------------------------------------------------------------------------

func TestFindTextLinks(t *testing.T) {
	body := []byte(`Some text before.

[[Note Title]]
[[Another Note|Display Text]]
![[image.png]]
[[ ]]
[[Trimmed  |  Custom Display  ]]

Some text after.`)

	matches := FindTextLinks(body)

	// Expect 3 matches: Note Title, Another Note, Trimmed
	// ![[image.png]] should be excluded
	// [[ ]] should be excluded (empty title)
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d: %+v", len(matches), matches)
	}

	// Check first match
	if matches[0].Title != "Note Title" {
		t.Errorf("match[0].Title = %q, want %q", matches[0].Title, "Note Title")
	}
	if matches[0].Display != "" {
		t.Errorf("match[0].Display = %q, want empty", matches[0].Display)
	}

	// Check second match (with display text)
	if matches[1].Title != "Another Note" {
		t.Errorf("match[1].Title = %q, want %q", matches[1].Title, "Another Note")
	}
	if matches[1].Display != "Display Text" {
		t.Errorf("match[1].Display = %q, want %q", matches[1].Display, "Display Text")
	}

	// Check third match (with trimming)
	if matches[2].Title != "Trimmed" {
		t.Errorf("match[2].Title = %q, want %q", matches[2].Title, "Trimmed")
	}
	if matches[2].Display != "Custom Display" {
		t.Errorf("match[2].Display = %q, want %q", matches[2].Display, "Custom Display")
	}
}

func TestFindTextLinks_Empty(t *testing.T) {
	matches := FindTextLinks([]byte("No wikilinks here."))
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestFindTextLinks_OnlyImageLinks(t *testing.T) {
	body := []byte(`![[photo.png]]
![[other.jpg|Alt]]`)
	matches := FindTextLinks(body)
	if len(matches) != 0 {
		t.Fatalf("expected 0 text matches for image-only body, got %d: %+v", len(matches), matches)
	}
}

// ---------------------------------------------------------------------------
// RewriteTextLinks
// ---------------------------------------------------------------------------

func TestRewriteTextLinks_Resolved(t *testing.T) {
	body := []byte("Check out [[My Note]] for details.")
	resolver := func(title string) (string, bool) {
		if title == "My Note" {
			return "/blog/my-note/", true
		}
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	expected := "Check out [My Note](/blog/my-note/) for details."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewriteTextLinks_WithDisplayText(t *testing.T) {
	body := []byte("See [[My Note|click here]] for more.")
	resolver := func(title string) (string, bool) {
		if title == "My Note" {
			return "/blog/my-note/", true
		}
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	expected := "See [click here](/blog/my-note/) for more."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewriteTextLinks_Unresolved(t *testing.T) {
	body := []byte("Reference to [[Unknown Page]] here.")
	resolver := func(title string) (string, bool) {
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	// Unresolved links become plain text (display text, which defaults to title)
	expected := "Reference to Unknown Page here."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewriteTextLinks_UnresolvedWithDisplay(t *testing.T) {
	body := []byte("See [[Unknown|display text]] here.")
	resolver := func(title string) (string, bool) {
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	expected := "See display text here."
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewriteTextLinks_DoesNotTouchImageLinks(t *testing.T) {
	body := []byte("Image: ![[photo.png]] and text [[Note]].")
	resolver := func(title string) (string, bool) {
		if title == "Note" {
			return "/note/", true
		}
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	// Image wikilink should remain untouched
	if !contains(result, "![[photo.png]]") {
		t.Errorf("image wikilink should be preserved, got: %s", result)
	}
	if !contains(result, "[Note](/note/)") {
		t.Errorf("text wikilink should be resolved, got: %s", result)
	}
}

func TestRewriteTextLinks_EmptyBody(t *testing.T) {
	result := RewriteTextLinks([]byte(""), func(string) (string, bool) { return "", false })
	if len(result) != 0 {
		t.Errorf("expected empty result, got %q", string(result))
	}
}

func TestRewriteTextLinks_MultipleLinks(t *testing.T) {
	body := []byte("See [[A]] and [[B]] and [[C|display]].")
	resolver := func(title string) (string, bool) {
		switch title {
		case "A":
			return "/a/", true
		case "B":
			return "/b/", true
		case "C":
			return "/c/", true
		}
		return "", false
	}

	result := string(RewriteTextLinks(body, resolver))
	if !contains(result, "[A](/a/)") {
		t.Errorf("missing link A in: %s", result)
	}
	if !contains(result, "[B](/b/)") {
		t.Errorf("missing link B in: %s", result)
	}
	if !contains(result, "[display](/c/)") {
		t.Errorf("missing link C with display in: %s", result)
	}
}
