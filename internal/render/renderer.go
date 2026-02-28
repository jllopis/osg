package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	htmltpl "html/template"
	texttpl "text/template"
)

// Renderer holds both an html/template tree (for .html files, with
// automatic HTML escaping) and a text/template tree (for .xml and .txt
// files, where escaping is unwanted — XML feeds, sitemap, robots.txt).
type Renderer struct {
	htmlTemplates *htmltpl.Template
	textTemplates *texttpl.Template
}

func New(userDir string, themeDir string, ctx Context) (*Renderer, error) {
	loader := TemplateLoader{UserDir: userDir, ThemeDir: themeDir, Funcs: FuncMap(ctx)}
	html, text, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &Renderer{htmlTemplates: html, textTemplates: text}, nil
}

// NewWithChain creates a Renderer using a theme inheritance chain instead of
// a single theme directory.  The chain should be ordered child-first (as
// returned by theme.ResolveChain); the loader reverses it internally so that
// ancestor templates are loaded first and child templates override them.
func NewWithChain(userDir string, themeChain []string, ctx Context) (*Renderer, error) {
	loader := TemplateLoader{UserDir: userDir, ThemeChain: themeChain, Funcs: FuncMap(ctx)}
	html, text, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &Renderer{htmlTemplates: html, textTemplates: text}, nil
}

func (r *Renderer) HasTemplate(name string) bool {
	if isTextTemplate(name) {
		return r.textTemplates.Lookup(name) != nil
	}
	return r.htmlTemplates.Lookup(name) != nil
}

func (r *Renderer) RenderToFile(name string, ctx map[string]any, outputPath string) error {
	if isTextTemplate(name) {
		if r.textTemplates.Lookup(name) == nil {
			return fmt.Errorf("template not found: %s", name)
		}
	} else {
		if r.htmlTemplates.Lookup(name) == nil {
			return fmt.Errorf("template not found: %s", name)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if isTextTemplate(name) {
		return r.textTemplates.ExecuteTemplate(file, name, ctx)
	}
	return r.htmlTemplates.ExecuteTemplate(file, name, ctx)
}

// isTextTemplate returns true for template names that should use
// text/template (no HTML escaping) instead of html/template.
func isTextTemplate(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".xml" || ext == ".txt"
}
