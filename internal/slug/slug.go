package slug

import (
	"path/filepath"
	"strings"
	"unicode"
)

func Derive(fm map[string]any, filename string) string {
	if fm != nil {
		if value := stringFrom(fm, "slug"); value != "" {
			return Slugify(value)
		}
		if value := stringFrom(fm, "title"); value != "" {
			return Slugify(value)
		}
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	return Slugify(base)
}

func Slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return "untitled"
	}

	var b strings.Builder
	b.Grow(len(input))
	lastDash := false

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '-' || r == '_' || unicode.IsSpace(r) {
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
			continue
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "untitled"
	}
	return slug
}

func stringFrom(fm map[string]any, key string) string {
	val, ok := fm[key]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}
