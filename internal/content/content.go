package content

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/publish"
	slugpkg "osg/internal/slug"

	"gopkg.in/yaml.v3"
)

func NormalizeFrontmatter(fm map[string]any, slug string, dateISO string, isDraft bool, sourceName string) map[string]any {
	out := map[string]any{}
	osg := publish.GetOSGBlock(fm)

	// Title precedence: osg.title > fm.title > fm.name > filename
	title := pickString(osg, "title")
	if title == "" {
		title = pickString(fm, "title", "name")
	}
	if title == "" {
		title = strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	}

	out["title"] = title
	out["date"] = dateISO
	out["slug"] = slug
	out["draft"] = isDraft

	// Publish-at: preserved as a top-level field so site.ParseFile's
	// pickTime picks it up after the vault → content sync. Without
	// this, scheduled drafts never reach the scheduler classification
	// (publish.PublishAt reads osg.* or top-level; NormalizeFrontmatter
	// builds the output map from scratch, so anything we don't copy
	// here is lost).
	if at := publish.PublishAt(fm); !at.IsZero() {
		out["publish_at"] = at
	}

	if tags := toStringSlice(fm["tags"]); len(tags) > 0 {
		out["tags"] = tags
	}

	if area, ok := fm["area"]; ok && area != nil {
		out["area"] = area
	}

	if t, ok := fm["type"]; ok && t != nil {
		out["type"] = t
	}

	if template := pickString(fm, "template"); template != "" {
		out["template"] = template
	}

	// Summary: osg.abstract takes precedence, then top-level summary/description/excerpt
	if abstract := pickString(osg, "abstract"); abstract != "" {
		out["summary"] = abstract
	} else if summary := pickString(fm, "summary", "description", "excerpt"); summary != "" {
		out["summary"] = summary
	}

	if lang := pickString(fm, "lang", "language"); lang != "" {
		out["lang"] = lang
	}

	// Author: osg.author takes precedence, then top-level author
	if author := pickString(osg, "author"); author != "" {
		out["author"] = author
	} else if author := pickString(fm, "author"); author != "" {
		out["author"] = author
	}

	// Image: osg.image takes precedence, then top-level image/cover/banner
	image := ""
	if osg != nil {
		image = pickString(osg, "image")
	}
	if image == "" {
		image = pickString(fm, "image", "cover", "banner")
	}
	if image != "" {
		out["image"] = image
	}

	// Image credit: osg.image_credit takes precedence, then top-level image_credit
	if credit := pickString(osg, "image_credit"); credit != "" {
		out["image_credit"] = credit
	} else if credit := pickString(fm, "image_credit"); credit != "" {
		out["image_credit"] = credit
	}

	// Featured: osg.featured takes precedence, then extra.featured
	featured := false
	if osg != nil {
		if v, ok := osg["featured"]; ok {
			switch b := v.(type) {
			case bool:
				featured = b
			case string:
				featured = strings.EqualFold(strings.TrimSpace(b), "true")
			}
		}
	}
	if !featured {
		if extra, ok := fm["extra"].(map[string]any); ok {
			if v, ok := extra["featured"]; ok {
				if b, ok := v.(bool); ok {
					featured = b
				}
			}
		}
	}
	if featured {
		out["featured"] = true
	}

	// Menu: osg.menu marks a page for navigation menu display.
	// Menu pages are also excluded from homepage listings.
	// menu_title: osg.path is used as the menu label when available,
	// so the user can have a long page title but a short nav entry.
	if osg != nil {
		if menu := pickBool(osg, "menu"); menu {
			out["menu"] = true
			if menuTitle := pickString(osg, "path"); menuTitle != "" {
				out["menu_title"] = menuTitle
			}
		}
	}

	out["obsidian"] = map[string]any{
		"source":      sourceName,
		"frontmatter": fm,
	}

	return out
}

func RenderMarkdown(frontmatter map[string]any, body []byte) ([]byte, error) {
	fmBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	if len(fmBytes) == 0 || fmBytes[len(fmBytes)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString("---\n")
	buf.Write(body)

	return buf.Bytes(), nil
}

// BuildOutputPath expands placeholders in layout and returns the content file path.
// Supported placeholders: {date}, {year}, {month}, {day}, {slug}, {title}.
func BuildOutputPath(contentDir, layout string, pageDate time.Time, pageSlug, title string) string {
	if strings.TrimSpace(layout) == "" {
		layout = "{date}/{slug}"
	}
	expanded := expandPlaceholders(layout, pageDate, pageSlug, title)
	return filepath.Join(contentDir, filepath.Clean(expanded), "index.md")
}

// ExpandPermalink expands placeholders in a permalink pattern and returns
// a clean relative path suitable for content directory placement.
// Supported placeholders: {date}, {year}, {month}, {day}, {slug}, {title}.
func ExpandPermalink(pattern string, pageDate time.Time, pageSlug, title string) string {
	return filepath.Clean(expandPlaceholders(pattern, pageDate, pageSlug, title))
}

// expandPlaceholders replaces permalink placeholders in a pattern string.
func expandPlaceholders(pattern string, pageDate time.Time, pageSlug, title string) string {
	year := fmt.Sprintf("%04d", pageDate.Year())
	month := fmt.Sprintf("%02d", int(pageDate.Month()))
	day := fmt.Sprintf("%02d", pageDate.Day())
	datePath := year + "/" + month + "/" + day

	path := strings.ReplaceAll(pattern, "{date}", datePath)
	path = strings.ReplaceAll(path, "{year}", year)
	path = strings.ReplaceAll(path, "{month}", month)
	path = strings.ReplaceAll(path, "{day}", day)
	path = strings.ReplaceAll(path, "{slug}", pageSlug)
	if strings.Contains(path, "{title}") {
		path = strings.ReplaceAll(path, "{title}", slugpkg.Slugify(title))
	}

	return strings.TrimLeft(path, "/\\")
}

func pickString(fm map[string]any, keys ...string) string {
	for _, key := range keys {
		if val := stringFrom(fm, key); val != "" {
			return val
		}
	}
	return ""
}

func stringFrom(fm map[string]any, key string) string {
	if fm == nil {
		return ""
	}
	val, ok := fm[key]
	if !ok {
		return ""
	}
	if str, ok := val.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func toStringSlice(value any) []string {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return compactStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	case string:
		s := strings.TrimSpace(v)
		if s != "" {
			return []string{s}
		}
	}

	return nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, s := range values {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func pickBool(fm map[string]any, key string) bool {
	if fm == nil {
		return false
	}
	val, ok := fm[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}
