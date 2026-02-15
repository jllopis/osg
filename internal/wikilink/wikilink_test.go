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
