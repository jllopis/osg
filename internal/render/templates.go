package render

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"html/template"
)

//go:embed builtins/*
var builtinFS embed.FS

type TemplateLoader struct {
	UserDir  string
	ThemeDir string
}

func (l TemplateLoader) Load() (*template.Template, error) {
	root := template.New("root").Funcs(FuncMap())

	if err := parseEmbedded(root, builtinFS); err != nil {
		return nil, err
	}

	if l.ThemeDir != "" {
		if err := parseDir(root, l.ThemeDir); err != nil {
			return nil, err
		}
	}

	if l.UserDir != "" {
		if err := parseDir(root, l.UserDir); err != nil {
			return nil, err
		}
	}

	return root, nil
}

func parseDir(root *template.Template, dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat templates dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("templates path is not a directory: %s", dir)
	}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isTemplateFile(path) {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = root.New(rel).Parse(string(data))
		return err
	})
}

func parseEmbedded(root *template.Template, fsys embed.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isTemplateFile(path) {
			return nil
		}

		name := strings.TrimPrefix(path, "builtins/")
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		_, err = root.New(name).Parse(string(data))
		return err
	})
}

func isTemplateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".xml", ".txt":
		return true
	default:
		return false
	}
}
