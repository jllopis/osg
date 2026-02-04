package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func UpdatePluginsEnabled(path string, enabled []string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "config.yaml"
	}

	enabled = normalizePluginNames(enabled)

	doc := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &doc)
	} else if os.IsNotExist(err) {
		_ = yaml.Unmarshal([]byte(DefaultConfigYAML()), &doc)
	} else {
		return err
	}

	if doc == nil {
		doc = map[string]any{}
	}
	doc["plugins_enabled"] = enabled

	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}

	return os.WriteFile(path, out, 0o644)
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
