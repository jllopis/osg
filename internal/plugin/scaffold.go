package plugin

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/rust/*
//go:embed templates/rust/src/*
var pluginTemplates embed.FS

func ScaffoldRust(baseDir string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}

	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "plugins_src"
	}

	target := filepath.Join(baseDir, name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("plugin already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return err
	}

	crateName := toCrateName(name)
	replacements := map[string]string{
		"{{name}}":       name,
		"{{crate_name}}": crateName,
	}

	return fs.WalkDir(pluginTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, "templates/rust/")
		if rel == path {
			return nil
		}
		dest := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		data, err := fs.ReadFile(pluginTemplates, path)
		if err != nil {
			return err
		}
		for key, value := range replacements {
			data = bytes.ReplaceAll(data, []byte(key), []byte(value))
		}

		return os.WriteFile(dest, data, 0o644)
	})
}

func toCrateName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	if strings.HasPrefix(name, "osg-") {
		return name
	}
	return "osg-" + name
}
