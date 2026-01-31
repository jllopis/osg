package content

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func NormalizeFrontmatter(fm map[string]any, slug string, dateISO string, isDraft bool, sourceName string) map[string]any {
	out := map[string]any{}

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

	if categories := toStringSlice(pickValue(fm, "categories", "category")); len(categories) > 0 {
		out["categories"] = categories
	}

	if template := pickString(fm, "template"); template != "" {
		out["template"] = template
	}

	if summary := pickString(fm, "summary", "description", "excerpt"); summary != "" {
		out["summary"] = summary
	}

	if lang := pickString(fm, "lang", "language"); lang != "" {
		out["lang"] = lang
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
