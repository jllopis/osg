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

func New(userDir string, themeDir string) (*Renderer, error) {
	loader := TemplateLoader{UserDir: userDir, ThemeDir: themeDir}
	templates, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
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
