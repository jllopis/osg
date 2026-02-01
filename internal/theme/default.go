package theme

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed default/templates/* default/static/*
var defaultThemeFS embed.FS

// EnsureDefaultTheme writes the embedded default theme into themesDir/default
// without overwriting existing files.
func EnsureDefaultTheme(themesDir string) error {
	themesDir = strings.TrimSpace(themesDir)
	if themesDir == "" {
		return nil
	}

	target := filepath.Join(themesDir, "default")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}

	return fs.WalkDir(defaultThemeFS, "default", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel("default", path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(target, rel)
		if _, err := os.Stat(destPath); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		data, err := defaultThemeFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0o644)
	})
}
