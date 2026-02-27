package plugin

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed bundled
var bundledFS embed.FS

// BundledPlugins lists the plugin names embedded in the binary.
// These are extracted to the site's plugins directory on each build
// unless the user has placed their own version there.
var BundledPlugins = []string{"search"}

// EnsureBundledPlugins extracts embedded .wasm files into pluginsDir.
// It does NOT overwrite files that already exist, allowing users to
// provide their own version of a bundled plugin.
// If pluginsDir is empty, the function is a no-op.
func EnsureBundledPlugins(pluginsDir string) error {
	if pluginsDir == "" {
		return nil
	}

	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("create plugins dir: %w", err)
	}

	return fs.WalkDir(bundledFS, "bundled", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel("bundled", path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(pluginsDir, rel)

		// Do not overwrite user-provided plugins.
		if _, err := os.Stat(destPath); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		data, err := bundledFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0o644)
	})
}
