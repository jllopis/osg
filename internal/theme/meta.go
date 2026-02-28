package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ThemeMeta holds metadata from a theme's theme.yaml file.
type ThemeMeta struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description,omitempty"`
	Author        string `yaml:"author,omitempty"`
	Version       string `yaml:"version,omitempty"`
	MinOSGVersion string `yaml:"min_osg_version,omitempty"`
	Parent        string `yaml:"parent,omitempty"`
}

// LoadMeta reads and parses a theme.yaml from the given theme directory.
// Returns a zero ThemeMeta (with Name set to the directory basename) if the
// file does not exist.
func LoadMeta(themeDir string) (ThemeMeta, error) {
	themeDir = strings.TrimSpace(themeDir)
	if themeDir == "" {
		return ThemeMeta{}, fmt.Errorf("theme directory is empty")
	}

	metaPath := filepath.Join(themeDir, "theme.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No theme.yaml — return minimal meta with directory name.
			return ThemeMeta{Name: filepath.Base(themeDir)}, nil
		}
		return ThemeMeta{}, fmt.Errorf("read theme.yaml: %w", err)
	}

	var meta ThemeMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return ThemeMeta{}, fmt.Errorf("parse theme.yaml in %s: %w", themeDir, err)
	}

	if strings.TrimSpace(meta.Name) == "" {
		meta.Name = filepath.Base(themeDir)
	}

	return meta, nil
}

// ResolveChain returns the ordered list of theme directories from the active
// theme up through its parent chain.  The first element is the active theme,
// the last is the root ancestor.  A cycle or missing parent returns an error.
func ResolveChain(themesDir string, themeName string) ([]string, error) {
	themesDir = strings.TrimSpace(themesDir)
	themeName = strings.TrimSpace(themeName)
	if themesDir == "" || themeName == "" {
		return nil, nil
	}

	var chain []string
	visited := map[string]bool{}
	current := themeName

	for current != "" {
		if visited[current] {
			return nil, fmt.Errorf("theme inheritance cycle detected: %s", current)
		}
		visited[current] = true

		dir := filepath.Join(themesDir, current)
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("theme not found: %s (required by parent chain)", current)
			}
			return nil, err
		}

		chain = append(chain, dir)

		meta, err := LoadMeta(dir)
		if err != nil {
			return nil, fmt.Errorf("load theme metadata for %s: %w", current, err)
		}
		current = strings.TrimSpace(meta.Parent)
	}

	return chain, nil
}

// ListThemes scans the themes directory and returns metadata for each theme found.
func ListThemes(themesDir string) ([]ThemeMeta, error) {
	themesDir = strings.TrimSpace(themesDir)
	if themesDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read themes dir: %w", err)
	}

	var themes []ThemeMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(themesDir, e.Name())
		meta, err := LoadMeta(dir)
		if err != nil {
			// Skip themes with unparseable metadata.
			continue
		}
		themes = append(themes, meta)
	}
	return themes, nil
}

// WriteMeta writes a ThemeMeta to theme.yaml in the given directory.
func WriteMeta(themeDir string, meta ThemeMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal theme.yaml: %w", err)
	}

	metaPath := filepath.Join(themeDir, "theme.yaml")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o644)
}
