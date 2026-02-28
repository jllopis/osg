package render

import (
	"fmt"
	"os"
	"path/filepath"

	"html/template"
)

type Renderer struct {
	templates *template.Template
}

func New(userDir string, themeDir string, ctx Context) (*Renderer, error) {
	loader := TemplateLoader{UserDir: userDir, ThemeDir: themeDir, Funcs: FuncMap(ctx)}
	templates, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
}

// NewWithChain creates a Renderer using a theme inheritance chain instead of
// a single theme directory.  The chain should be ordered child-first (as
// returned by theme.ResolveChain); the loader reverses it internally so that
// ancestor templates are loaded first and child templates override them.
func NewWithChain(userDir string, themeChain []string, ctx Context) (*Renderer, error) {
	loader := TemplateLoader{UserDir: userDir, ThemeChain: themeChain, Funcs: FuncMap(ctx)}
	templates, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
}

func (r *Renderer) HasTemplate(name string) bool {
	return r.templates.Lookup(name) != nil
}

func (r *Renderer) RenderToFile(name string, ctx map[string]any, outputPath string) error {
	if r.templates.Lookup(name) == nil {
		return fmt.Errorf("template not found: %s", name)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return r.templates.ExecuteTemplate(file, name, ctx)
}
