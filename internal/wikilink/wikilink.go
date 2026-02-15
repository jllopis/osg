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

// Match represents a single wikilink image reference found in the body.
type Match struct {
	Full    string // the full match, e.g. "![[photo.png|My photo]]"
	Ref     string // the image reference, e.g. "photo.png" or "folder/photo.png"
	AltText string // optional alt text after |, e.g. "My photo"
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
