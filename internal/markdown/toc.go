package markdown

import (
	"html"
	"regexp"
)

// TOCEntry represents a single heading in the table of contents.
type TOCEntry struct {
	Level int    // heading level (2-6)
	ID    string // heading id attribute
	Title string // heading text content (HTML stripped)
}

// headingRe matches <h2-6 id="...">...</h2-6> in rendered HTML.
// Captures: (1) level digit, (2) id attribute, (3) inner HTML.
var headingRe = regexp.MustCompile(`<h([2-6])\s+id="([^"]+)"[^>]*>(.*?)</h[2-6]>`)

// stripTagsRe removes HTML tags from inner heading content.
var stripTagsRe = regexp.MustCompile(`<[^>]+>`)

// ExtractTOC parses rendered HTML and returns a slice of TOCEntry for headings
// with id attributes (levels 2 through 6). Returns nil if no headings found.
func ExtractTOC(htmlContent string) []TOCEntry {
	matches := headingRe.FindAllStringSubmatch(htmlContent, -1)
	if len(matches) == 0 {
		return nil
	}

	entries := make([]TOCEntry, 0, len(matches))
	for _, m := range matches {
		level := int(m[1][0] - '0')
		id := m[2]
		title := stripTagsRe.ReplaceAllString(m[3], "")
		title = html.UnescapeString(title)
		entries = append(entries, TOCEntry{
			Level: level,
			ID:    id,
			Title: title,
		})
	}
	return entries
}

// TOCView converts TOC entries to template-friendly maps.
func TOCView(entries []TOCEntry) []map[string]any {
	if len(entries) == 0 {
		return nil
	}
	views := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		views = append(views, map[string]any{
			"level": e.Level,
			"id":    e.ID,
			"title": e.Title,
		})
	}
	return views
}
