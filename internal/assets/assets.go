package assets

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"osg/internal/config"
)

func Prepare(cfg config.Config, logger *slog.Logger) error {
	if err := copyStatic(cfg, logger); err != nil {
		return err
	}

	if err := copyContentAssets(cfg, logger); err != nil {
		return err
	}

	if err := compileSass(cfg, logger); err != nil {
		return err
	}

	return nil
}

// PrepareWithChain is like Prepare but uses a resolved theme inheritance chain
// for static file copying and sass compilation.  The chain is ordered child-first;
// ancestor static files are copied first so that child themes override them.
func PrepareWithChain(cfg config.Config, themeChain []string, logger *slog.Logger) error {
	if err := copyStaticChain(themeChain, cfg.StaticDir, cfg.PublicDir, logger); err != nil {
		return err
	}

	if err := copyContentAssets(cfg, logger); err != nil {
		return err
	}

	if err := compileSassChain(themeChain, cfg, logger); err != nil {
		return err
	}

	return nil
}

func copyStatic(cfg config.Config, logger *slog.Logger) error {
	if cfg.Theme != "" {
		themeStatic := filepath.Join(cfg.ThemesDir, cfg.Theme, "static")
		if err := copyDir(themeStatic, cfg.PublicDir, logger); err != nil {
			return err
		}
	}

	return copyDir(cfg.StaticDir, cfg.PublicDir, logger)
}

// copyStaticChain copies static assets from the theme chain (root ancestor
// first, child last) followed by user static dir.
func copyStaticChain(themeChain []string, userStaticDir string, publicDir string, logger *slog.Logger) error {
	// Copy from root ancestor first so child overrides parent.
	for i := len(themeChain) - 1; i >= 0; i-- {
		themeStatic := filepath.Join(themeChain[i], "static")
		if err := copyDir(themeStatic, publicDir, logger); err != nil {
			return err
		}
	}
	return copyDir(userStaticDir, publicDir, logger)
}

// compileSassChain compiles sass from the theme chain (root ancestor first,
// child last) followed by user sass.
func compileSassChain(themeChain []string, cfg config.Config, logger *slog.Logger) error {
	for i := len(themeChain) - 1; i >= 0; i-- {
		themeSass := filepath.Join(themeChain[i], "sass")
		if err := compileSassDir(themeSass, cfg.PublicDir, logger); err != nil {
			return err
		}
	}
	if !cfg.CompileSass {
		return nil
	}
	return compileSassDir(cfg.SassDir, cfg.PublicDir, logger)
}

func copyContentAssets(cfg config.Config, logger *slog.Logger) error {
	return copyDirFiltered(cfg.ContentDir, cfg.PublicDir, logger, func(path string, d fs.DirEntry) bool {
		if d.IsDir() {
			return false
		}
		if strings.HasPrefix(d.Name(), ".") {
			return true
		}
		return strings.EqualFold(filepath.Ext(d.Name()), ".md")
	})
}

func copyDir(src string, dest string, logger *slog.Logger) error {
	return copyDirFiltered(src, dest, logger, func(_ string, d fs.DirEntry) bool {
		return strings.HasPrefix(d.Name(), ".")
	})
}

func copyDirFiltered(src string, dest string, logger *slog.Logger, skip func(path string, d fs.DirEntry) bool) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat dir %s: %w", src, err)
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		if err := copyFile(path, destPath); err != nil {
			return err
		}

		logger.Debug("copied asset", "src", path, "dest", destPath)
		return nil
	})
}

func copyFile(src string, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func compileSass(cfg config.Config, logger *slog.Logger) error {
	if cfg.Theme != "" {
		themeSass := filepath.Join(cfg.ThemesDir, cfg.Theme, "sass")
		if err := compileSassDir(themeSass, cfg.PublicDir, logger); err != nil {
			return err
		}
	}

	if !cfg.CompileSass {
		return nil
	}

	return compileSassDir(cfg.SassDir, cfg.PublicDir, logger)
}

func compileSassDir(src string, dest string, logger *slog.Logger) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat sass dir %s: %w", src, err)
	}
	if !info.IsDir() {
		return nil
	}

	conflicts, err := sassConflicts(src)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("sass conflict: %s", strings.Join(conflicts, ", "))
	}

	if _, err := exec.LookPath("sass"); err != nil {
		return fmt.Errorf("sass binary not found in PATH")
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), "_") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".scss" && ext != ".sass" {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		outRel := strings.TrimSuffix(rel, ext) + ".css"
		outPath := filepath.Join(dest, outRel)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		if err := runSass(path, outPath, filepath.Dir(path), src); err != nil {
			return err
		}

		logger.Info("compiled sass", "src", path, "dest", outPath)
		return nil
	})
}

func sassConflicts(root string) ([]string, error) {
	conflicts := []string{}
	seen := map[string]map[string]map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".scss" && ext != ".sass" {
			return nil
		}

		dir := filepath.Dir(path)
		base := strings.TrimSuffix(d.Name(), ext)
		if seen[dir] == nil {
			seen[dir] = map[string]map[string]bool{}
		}
		if seen[dir][base] == nil {
			seen[dir][base] = map[string]bool{}
		}
		seen[dir][base][ext] = true
		if seen[dir][base][".scss"] && seen[dir][base][".sass"] {
			conflicts = append(conflicts, filepath.Join(dir, base))
		}

		return nil
	})

	return conflicts, err
}

func runSass(input string, output string, loadPaths ...string) error {
	args := []string{"--no-source-map", "--style", "compressed"}
	for _, loadPath := range loadPaths {
		if strings.TrimSpace(loadPath) == "" {
			continue
		}
		args = append(args, "--load-path", loadPath)
	}
	args = append(args, input, output)

	cmd := exec.Command("sass", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
