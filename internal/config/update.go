package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// UpdatePluginsEnabled updates the plugins_enabled list in the given config
// file, preserving all YAML comments and formatting.
func UpdatePluginsEnabled(path string, enabled []string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "config.yaml"
	}

	enabled = normalizePluginNames(enabled)

	// If the file doesn't exist, create it from the default template first.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
			return err
		}
		if err := os.WriteFile(path, []byte(DefaultConfigYAML()), 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	doc, err := LoadNode(path)
	if err != nil {
		return err
	}

	SetNodeSequence(doc, "plugins_enabled", enabled)
	return SaveNode(path, doc)
}

func normalizePluginNames(input []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(input))
	for _, name := range input {
		name = normalizePluginName(name)
		if name == "" {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func normalizePluginName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".wasm") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return strings.TrimSpace(name)
}
