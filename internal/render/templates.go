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

	htmltpl "html/template"
	texttpl "text/template"
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
	ThemeDir string // Single theme dir (legacy, used when ThemeChain is empty).
	// ThemeChain is an ordered list of theme directories from the active
	// theme through its parent chain (child first, root ancestor last).
	// Templates are loaded root-ancestor-first so that child themes
	// override parent templates.
	ThemeChain []string
	Funcs      htmltpl.FuncMap
}

// Load returns two template trees: one html/template (for .html files with
// auto-escaping) and one text/template (for .xml and .txt files without
// escaping).  Both trees share the same FuncMap.
func (l TemplateLoader) Load() (*htmltpl.Template, *texttpl.Template, error) {
	htmlRoot := htmltpl.New("root")
	textRoot := texttpl.New("root")
	if len(l.Funcs) > 0 {
		htmlRoot = htmlRoot.Funcs(l.Funcs)
		textRoot = textRoot.Funcs(texttpl.FuncMap(l.Funcs))
	}

	if err := parseEmbedded(htmlRoot, textRoot, builtinFS); err != nil {
		return nil, nil, err
	}

	// Load theme templates.  When a chain is provided, iterate from
	// root ancestor to child so that child definitions win.
	themeDirs := l.themeTemplateDirs()
	for _, dir := range themeDirs {
		if dir != "" {
			if err := parseDir(htmlRoot, textRoot, dir); err != nil {
				return nil, nil, err
			}
		}
	}

	if l.UserDir != "" {
		if err := parseDir(htmlRoot, textRoot, l.UserDir); err != nil {
			return nil, nil, err
		}
	}

	return htmlRoot, textRoot, nil
}

// themeTemplateDirs returns theme template directories in load order
// (root ancestor first, active theme last).
func (l TemplateLoader) themeTemplateDirs() []string {
	if len(l.ThemeChain) > 0 {
		// Reverse: chain is [child, parent, grandparent] -> load grandparent first.
		dirs := make([]string, 0, len(l.ThemeChain))
		for i := len(l.ThemeChain) - 1; i >= 0; i-- {
			dirs = append(dirs, l.ThemeChain[i])
		}
		return dirs
	}
	// Legacy single-dir mode.
	if l.ThemeDir != "" {
		return []string{l.ThemeDir}
	}
	return nil
}

func parseDir(htmlRoot *htmltpl.Template, textRoot *texttpl.Template, dir string) error {
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

		if isTextTemplate(rel) {
			_, err = textRoot.New(rel).Parse(string(data))
		} else {
			_, err = htmlRoot.New(rel).Parse(string(data))
		}
		return err
	})
}

func parseEmbedded(htmlRoot *htmltpl.Template, textRoot *texttpl.Template, fsys embed.FS) error {
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
		if isTextTemplate(name) {
			_, err = textRoot.New(name).Parse(string(data))
		} else {
			_, err = htmlRoot.New(name).Parse(string(data))
		}
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
