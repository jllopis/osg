package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ParseHugo reads Hugo content files from dir and returns posts.
// Supports both YAML (---) and TOML (+++) frontmatter delimiters.
func ParseHugo(dir string) ([]Post, error) {
	var posts []Post
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		post, err := parseHugoFile(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if post != nil {
			posts = append(posts, *post)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk hugo dir: %w", err)
	}
	return posts, nil
}

func parseHugoFile(content string) (*Post, error) {
	fm, body, err := splitHugoFrontmatter(content)
	if err != nil {
		return nil, err
	}
	if fm == nil {
		return nil, nil
	}

	title, _ := fm["title"].(string)
	slug, _ := fm["slug"].(string)
	draft, _ := fm["draft"].(bool)

	date := parseHugoDate(fm)
	tags := extractStringList(fm, "tags")
	categories := extractStringList(fm, "categories")

	return &Post{
		Title:      title,
		Date:       date,
		Tags:       tags,
		Categories: categories,
		Content:    strings.TrimSpace(body),
		Slug:       slug,
		Draft:      draft,
	}, nil
}

func splitHugoFrontmatter(content string) (map[string]any, string, error) {
	content = strings.TrimSpace(content)

	// YAML frontmatter
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "---")
		if end < 0 {
			return nil, content, nil
		}
		fmRaw := content[3 : 3+end]
		body := content[3+end+3:]
		var fm map[string]any
		if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
			return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
		}
		return fm, body, nil
	}

	// TOML frontmatter
	if strings.HasPrefix(content, "+++") {
		end := strings.Index(content[3:], "+++")
		if end < 0 {
			return nil, content, nil
		}
		fmRaw := content[3 : 3+end]
		body := content[3+end+3:]
		var fm map[string]any
		if _, err := toml.Decode(fmRaw, &fm); err != nil {
			return nil, "", fmt.Errorf("parse TOML frontmatter: %w", err)
		}
		return fm, body, nil
	}

	return nil, content, nil
}

func parseHugoDate(fm map[string]any) time.Time {
	for _, key := range []string{"date", "publishDate", "publishdate"} {
		if v, ok := fm[key]; ok {
			switch d := v.(type) {
			case string:
				for _, layout := range []string{
					time.RFC3339,
					"2006-01-02T15:04:05",
					"2006-01-02",
				} {
					if t, err := time.Parse(layout, d); err == nil {
						return t
					}
				}
			case time.Time:
				return d
			}
		}
	}
	return time.Now()
}

func extractStringList(fm map[string]any, key string) []string {
	v, ok := fm[key]
	if !ok {
		return nil
	}
	switch list := v.(type) {
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return list
	}
	return nil
}
