package render

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"html/template"
)

//go:embed builtins/*
var builtinFS embed.FS

var (
	builtinsOnce      sync.Once
	builtinsSignature string
	builtinsErr       error
)

func BuiltinsSignature() (string, error) {
	builtinsOnce.Do(func() {
		entries := []string{}
		err := fs.WalkDir(builtinFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !isTemplateFile(path) {
				return nil
			}
			entries = append(entries, path)
			return nil
		})
		if err != nil {
			builtinsErr = err
			return
		}
		sort.Strings(entries)
		hasher := sha256.New()
		for _, path := range entries {
			data, err := fs.ReadFile(builtinFS, path)
			if err != nil {
				builtinsErr = err
				return
			}
			_, _ = hasher.Write([]byte(path))
			_, _ = hasher.Write(data)
		}
		builtinsSignature = hex.EncodeToString(hasher.Sum(nil))
	})
	return builtinsSignature, builtinsErr
}

type TemplateLoader struct {
	UserDir  string
	ThemeDir string
	Funcs    template.FuncMap
}

func (l TemplateLoader) Load() (*template.Template, error) {
	root := template.New("root")
	if len(l.Funcs) > 0 {
		root = root.Funcs(l.Funcs)
	}

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
