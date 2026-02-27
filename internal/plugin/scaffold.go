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
//go:embed templates/go/*
//go:embed templates/go/*.tmpl
var pluginTemplates embed.FS

// Scaffold creates a new plugin project in baseDir/name using the given
// language template. Supported languages: "rust" (default), "go".
func Scaffold(baseDir string, name string, lang string) error {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		lang = "rust"
	}

	switch lang {
	case "rust":
		return scaffoldFromTemplate(baseDir, name, "templates/rust", rustReplacements(name))
	case "go", "tinygo":
		return scaffoldFromTemplate(baseDir, name, "templates/go", goReplacements(name))
	default:
		return fmt.Errorf("unsupported language %q: must be rust or go", lang)
	}
}

// ScaffoldRust creates a Rust plugin scaffold. Retained for backward
// compatibility; new code should use Scaffold(baseDir, name, "rust").
func ScaffoldRust(baseDir string, name string) error {
	return Scaffold(baseDir, name, "rust")
}

func scaffoldFromTemplate(baseDir string, name string, templateDir string, replacements map[string]string) error {
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

	return fs.WalkDir(pluginTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, templateDir+"/")
		if rel == path {
			return nil
		}
		// Strip .tmpl extension so e.g. "go.mod.tmpl" becomes "go.mod".
		rel = strings.TrimSuffix(rel, ".tmpl")
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

func rustReplacements(name string) map[string]string {
	return map[string]string{
		"{{name}}":       name,
		"{{crate_name}}": toCrateName(name),
	}
}

func goReplacements(name string) map[string]string {
	return map[string]string{
		"{{name}}":              name,
		"{{module_name}}":       toModuleName(name),
		"//go:build ignore\n\n": "", // strip build tag used to hide template from Go toolchain
	}
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

func toModuleName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	if strings.HasPrefix(name, "osg-") {
		return name
	}
	return "osg-" + name
}
