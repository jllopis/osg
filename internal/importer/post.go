package importer

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Post represents an imported post ready to be written as a Markdown file.
type Post struct {
	Title      string
	Date       time.Time
	Tags       []string
	Categories []string
	Content    string // Markdown body
	Slug       string
	Draft      bool
}

// OutputPath returns the relative path for this post within the content dir.
func (p Post) OutputPath() string {
	y := p.Date.Format("2006")
	m := p.Date.Format("01")
	return filepath.Join(y, m, safeSlug(p.Slug, p.Title)+".md")
}

// safeSlug returns a single, traversal-free path segment for the output
// filename. An importer source (WXR <wp:post_name>, Hugo frontmatter slug) is
// attacker-controllable, so a value like "../../../etc/evil" must not let the
// write escape the content dir. We collapse it to its base segment and reject
// "."/".." , falling back to a slug derived from the title.
func safeSlug(slug, title string) string {
	slug = strings.ReplaceAll(strings.TrimSpace(slug), "\\", "/")
	if slug != "" {
		slug = strings.TrimSpace(filepath.Base(filepath.FromSlash(slug)))
		if slug == "." || slug == ".." {
			slug = ""
		}
	}
	if slug == "" {
		slug = slugify(title)
	}
	return slug
}

// ToMarkdown returns the full Markdown file content with YAML frontmatter.
func (p Post) ToMarkdown() string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", p.Title)
	fmt.Fprintf(&b, "date: %s\n", p.Date.Format("2006-01-02"))
	if p.Draft {
		b.WriteString("publish: draft\n")
	}
	if len(p.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range p.Tags {
			fmt.Fprintf(&b, "  - %q\n", t)
		}
	}
	if len(p.Categories) > 0 {
		b.WriteString("categories:\n")
		for _, c := range p.Categories {
			fmt.Fprintf(&b, "  - %q\n", c)
		}
	}
	b.WriteString("---\n\n")
	b.WriteString(p.Content)
	b.WriteString("\n")
	return b.String()
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []byte
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, byte(c))
		} else if c == ' ' || c == '-' || c == '_' {
			if len(out) > 0 && out[len(out)-1] != '-' {
				out = append(out, '-')
			}
		}
	}
	return strings.Trim(string(out), "-")
}
