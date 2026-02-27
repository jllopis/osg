package wikilink

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// wikiImageRe matches Obsidian image wikilinks: ![[image.png]] or ![[image.png|alt text]]
// It also matches links with folder paths: ![[folder/image.png]]
var wikiImageRe = regexp.MustCompile(`!\[\[([^\]|]+?)(?:\|([^\]]*))?\]\]`)

// wikiTextRe matches Obsidian text wikilinks: [[Note Title]] or [[Note Title|display text]]
var wikiTextRe = regexp.MustCompile(`\[\[([^\]|]+?)(?:\|([^\]]*))?\]\]`)

// Match represents a single wikilink image reference found in the body.
type Match struct {
	Full    string // the full match, e.g. "![[photo.png|My photo]]"
	Ref     string // the image reference, e.g. "photo.png" or "folder/photo.png"
	AltText string // optional alt text after |, e.g. "My photo"
}

// TextMatch represents a single wikilink text reference found in the body.
type TextMatch struct {
	Full    string // the full match, e.g. "[[Note Title|display]]"
	Title   string // the note title, e.g. "Note Title"
	Display string // optional display text after |, empty if not present
}

// FindImageLinks finds all image wikilinks in the body text.
// It only returns matches where the reference looks like an image file
// (has an image extension). Non-image wikilinks like ![[Note Title]]
// are ignored.
func FindImageLinks(body []byte) []Match {
	var matches []Match

	for _, m := range wikiImageRe.FindAllSubmatch(body, -1) {
		ref := strings.TrimSpace(string(m[1]))
		if ref == "" {
			continue
		}

		// Only match image extensions
		ext := strings.ToLower(filepath.Ext(ref))
		if !isImageExt(ext) {
			continue
		}

		match := Match{
			Full: string(m[0]),
			Ref:  ref,
		}
		if len(m) > 2 {
			match.AltText = strings.TrimSpace(string(m[2]))
		}

		matches = append(matches, match)
	}

	return matches
}

// RewriteImageLinks replaces all image wikilinks in the body with standard
// markdown image syntax. The resolver function maps image references to
// local filenames (the filename to use in the markdown output).
// If resolver returns ("", false), the wikilink is left unchanged.
func RewriteImageLinks(body []byte, resolver func(ref string) (localName string, ok bool)) []byte {
	result := wikiImageRe.ReplaceAllFunc(body, func(match []byte) []byte {
		sub := wikiImageRe.FindSubmatch(match)
		if sub == nil {
			return match
		}

		ref := strings.TrimSpace(string(sub[1]))
		ext := strings.ToLower(filepath.Ext(ref))
		if !isImageExt(ext) {
			return match // not an image, leave unchanged
		}

		localName, ok := resolver(ref)
		if !ok {
			return match // couldn't resolve, leave unchanged
		}

		alt := ""
		if len(sub) > 2 {
			alt = strings.TrimSpace(string(sub[2]))
		}
		if alt == "" {
			alt = strings.TrimSuffix(filepath.Base(ref), filepath.Ext(ref))
		}

		return []byte(fmt.Sprintf("![%s](%s)", alt, urlEncodePath(localName)))
	})

	return result
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".avif":
		return true
	}
	return false
}

// urlEncodePath encodes a filename/path for use in a markdown image URL.
// Spaces and other special characters are percent-encoded.
func urlEncodePath(name string) string {
	return url.PathEscape(name)
}

// FindTextLinks finds all text wikilinks in the body text.
// These are [[Note Title]] or [[Note Title|display text]] patterns.
// Image wikilinks (![[...]]) are NOT included.
func FindTextLinks(body []byte) []TextMatch {
	var matches []TextMatch

	for _, m := range wikiTextRe.FindAllSubmatch(body, -1) {
		title := strings.TrimSpace(string(m[1]))
		if title == "" {
			continue
		}

		// Skip if this is inside an image wikilink (preceded by !)
		// The regex will match the inner part of ![[...]] so we need to filter
		fullMatch := string(m[0])
		idx := strings.Index(string(body), fullMatch)
		if idx > 0 && string(body[idx-1]) == "!" {
			continue
		}

		match := TextMatch{
			Full:  fullMatch,
			Title: title,
		}
		if len(m) > 2 {
			match.Display = strings.TrimSpace(string(m[2]))
		}

		matches = append(matches, match)
	}

	return matches
}

// RewriteTextLinks replaces all text wikilinks in the body with markdown links.
// The resolver function maps note titles to URL paths.
// If resolver returns ("", false), the wikilink is converted to plain text (removing [[ ]]).
// If display text is provided, it's used; otherwise the title is used.
func RewriteTextLinks(body []byte, resolver func(title string) (href string, ok bool)) []byte {
	result := wikiTextRe.ReplaceAllFunc(body, func(match []byte) []byte {
		// Check if preceded by ! (image wikilink) - skip those
		fullMatch := string(match)
		idx := strings.Index(string(body), fullMatch)
		if idx > 0 && string(body[idx-1]) == "!" {
			return match
		}

		sub := wikiTextRe.FindSubmatch(match)
		if sub == nil {
			return match
		}

		title := strings.TrimSpace(string(sub[1]))
		if title == "" {
			return match
		}

		display := title
		if len(sub) > 2 {
			d := strings.TrimSpace(string(sub[2]))
			if d != "" {
				display = d
			}
		}

		href, ok := resolver(title)
		if !ok {
			// Not found: convert to plain text (remove [[ ]])
			return []byte(display)
		}

		// Found: convert to markdown link
		return []byte(fmt.Sprintf("[%s](%s)", display, href))
	})

	return result
}

// NormalizeTitle normalizes a note title for matching.
// It lowercases and trims whitespace.
func NormalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}
