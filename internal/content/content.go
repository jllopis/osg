package content

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"osg/internal/publish"

	"gopkg.in/yaml.v3"
)

func NormalizeFrontmatter(fm map[string]any, slug string, dateISO string, isDraft bool, sourceName string) map[string]any {
	out := map[string]any{}
	osg := publish.GetOSGBlock(fm)

	title := pickString(fm, "title", "name")
	if title == "" {
		title = strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	}

	out["title"] = title
	out["date"] = dateISO
	out["slug"] = slug
	out["draft"] = isDraft

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
	if osg != nil {
		if menu := pickBool(osg, "menu"); menu {
			out["menu"] = true
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

func BuildOutputPath(contentDir string, layout string, datePath string, slug string) string {
	if strings.TrimSpace(layout) == "" {
		layout = "{date}/{slug}"
	}

	path := strings.ReplaceAll(layout, "{date}", datePath)
	path = strings.ReplaceAll(path, "{slug}", slug)
	path = strings.TrimLeft(path, "/\\")

	return filepath.Join(contentDir, filepath.Clean(path), "index.md")
}

func pickString(fm map[string]any, keys ...string) string {
	for _, key := range keys {
		if val := stringFrom(fm, key); val != "" {
			return val
		}
	}
	return ""
}

func pickValue(fm map[string]any, keys ...string) any {
	for _, key := range keys {
		if fm != nil {
			if val, ok := fm[key]; ok {
				return val
			}
		}
	}
	return nil
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
