package theme

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed default
var defaultThemeFS embed.FS

// EnsureDefaultTheme writes the embedded default theme into themesDir/default,
// always overwriting existing files to keep the on-disk copy in sync with the
// version compiled into the binary.
func EnsureDefaultTheme(themesDir string) error {
	return installTheme(themesDir, "default", true)
}

// ScaffoldTheme creates a new theme based on the embedded default theme.
// This creates a full copy — use ScaffoldChildTheme for an inheritance-based child.
func ScaffoldTheme(themesDir string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("theme name is required")
	}
	if name == "default" {
		return EnsureDefaultTheme(themesDir)
	}

	themesDir = strings.TrimSpace(themesDir)
	if themesDir == "" {
		return fmt.Errorf("themes dir is empty")
	}

	target := filepath.Join(themesDir, name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("theme already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return err
	}

	return installTheme(themesDir, name, false)
}

// ScaffoldChildTheme creates a minimal child theme that inherits from a parent.
// It writes theme.yaml with the parent field and creates empty template/static/i18n
// directories ready for the user to add overrides.
func ScaffoldChildTheme(themesDir string, name string, parent string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("theme name is required")
	}
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return fmt.Errorf("parent theme is required")
	}
	if name == parent {
		return fmt.Errorf("child theme cannot be its own parent")
	}

	themesDir = strings.TrimSpace(themesDir)
	if themesDir == "" {
		return fmt.Errorf("themes dir is empty")
	}

	// Verify parent exists.
	parentDir := filepath.Join(themesDir, parent)
	if _, err := os.Stat(parentDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent theme not found: %s", parent)
		}
		return err
	}

	target := filepath.Join(themesDir, name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("theme already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return err
	}

	// Create directory structure.
	for _, sub := range []string{"templates/partials", "static", "i18n"} {
		if err := os.MkdirAll(filepath.Join(target, sub), 0o755); err != nil {
			return err
		}
	}

	// Write theme.yaml with parent.
	meta := ThemeMeta{
		Name:   name,
		Parent: parent,
	}
	return WriteMeta(target, meta)
}

func installTheme(themesDir string, name string, overwrite bool) error {
	themesDir = strings.TrimSpace(themesDir)
	if themesDir == "" {
		return nil
	}

	target := filepath.Join(themesDir, name)
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
		if !overwrite {
			if _, err := os.Stat(destPath); err == nil {
				return nil
			} else if !os.IsNotExist(err) {
				return err
			}
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
