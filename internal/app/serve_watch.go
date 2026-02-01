package app

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"osg/internal/config"
)

type watchKind int

const (
	watchVault watchKind = iota
	watchBuild
)

type watchRoot struct {
	path string
	kind watchKind
}

type watchEvent struct {
	kind watchKind
	path string
}

func startWatch(ctx context.Context, cfg config.Config, configPath string, logger *slog.Logger) (<-chan watchEvent, <-chan error, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	roots, configFile, configDir := buildWatchRoots(cfg, configPath)
	if len(roots) == 0 {
		_ = watcher.Close()
		return nil, nil, nil
	}

	for _, root := range roots {
		if err := addRecursive(watcher, root.path); err != nil {
			if logger != nil {
				logger.Warn("watch root failed", "path", root.path, "error", err)
			}
		}
	}

	if configDir != "" && !rootExists(roots, configDir) {
		if err := addRecursive(watcher, configDir); err != nil {
			if logger != nil {
				logger.Warn("watch config dir failed", "path", configDir, "error", err)
			}
		}
	}

	events := make(chan watchEvent, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				errs <- err
			case evt, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !isRelevantOp(evt.Op) {
					continue
				}

				evtPath := normalizePath(evt.Name)
				if configFile != "" && samePath(evtPath, configFile) {
					events <- watchEvent{kind: watchBuild, path: evtPath}
					continue
				}

				kind, ok := matchRoot(evtPath, roots)
				if !ok {
					continue
				}

				if evt.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(evtPath); err == nil && info.IsDir() {
						_ = addRecursive(watcher, evtPath)
					}
				}

				events <- watchEvent{kind: kind, path: evtPath}
			}
		}
	}()

	return events, errs, nil
}

func buildWatchRoots(cfg config.Config, configPath string) ([]watchRoot, string, string) {
	var roots []watchRoot
	seen := map[string]watchKind{}

	if strings.TrimSpace(cfg.VaultPath) != "" {
		addRoot(&roots, seen, cfg.VaultPath, watchVault)
	} else {
		addRoot(&roots, seen, cfg.ContentDir, watchBuild)
	}

	addRoot(&roots, seen, cfg.TemplatesDir, watchBuild)
	addRoot(&roots, seen, cfg.StaticDir, watchBuild)
	addRoot(&roots, seen, cfg.SassDir, watchBuild)
	addRoot(&roots, seen, cfg.ThemesDir, watchBuild)
	addRoot(&roots, seen, cfg.PluginsDir, watchBuild)

	configFile := ""
	configDir := ""
	if strings.TrimSpace(configPath) != "" {
		configFile = normalizePath(configPath)
		configDir = filepath.Dir(configFile)
	}

	sortRoots(roots)
	return roots, configFile, configDir
}

func addRoot(roots *[]watchRoot, seen map[string]watchKind, root string, kind watchKind) {
	if strings.TrimSpace(root) == "" {
		return
	}
	normalized := normalizePath(root)
	if existing, ok := seen[normalized]; ok {
		if existing == watchVault {
			return
		}
		if kind == watchVault {
			for i := range *roots {
				if normalizePath((*roots)[i].path) == normalized {
					(*roots)[i].kind = watchVault
				}
			}
			seen[normalized] = watchVault
		}
		return
	}
	seen[normalized] = kind
	*roots = append(*roots, watchRoot{path: normalized, kind: kind})
}

func sortRoots(roots []watchRoot) {
	sort.SliceStable(roots, func(i, j int) bool {
		return len(roots[i].path) > len(roots[j].path)
	})
}

func addRecursive(w *fsnotify.Watcher, root string) error {
	root = normalizePath(root)
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		return nil
	})
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".")
}

func matchRoot(path string, roots []watchRoot) (watchKind, bool) {
	for _, root := range roots {
		if hasPathPrefix(path, root.path) {
			return root.kind, true
		}
	}
	return watchBuild, false
}

func rootExists(roots []watchRoot, dir string) bool {
	dir = normalizePath(dir)
	for _, root := range roots {
		if normalizePath(root.path) == dir {
			return true
		}
	}
	return false
}

func hasPathPrefix(path string, root string) bool {
	if path == root {
		return true
	}
	if strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return true
	}
	return false
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		return abs
	}
	return filepath.Clean(path)
}

func samePath(a string, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func isRelevantOp(op fsnotify.Op) bool {
	if op&fsnotify.Write != 0 {
		return true
	}
	if op&fsnotify.Create != 0 {
		return true
	}
	if op&fsnotify.Remove != 0 {
		return true
	}
	if op&fsnotify.Rename != 0 {
		return true
	}
	return false
}

func runWatchLoop(ctx context.Context, events <-chan watchEvent, errs <-chan error, opts CLIOptions, logger *slog.Logger, hub *reloadHub, debounce time.Duration) {
	if debounce <= 0 {
		debounce = 300 * time.Millisecond
	}

	var (
		pendingUpdate bool
		pendingBuild  bool
		running       bool
		timer         *time.Timer
		timerCh       <-chan time.Time
	)

	trigger := func() {
		if timer == nil {
			timer = time.NewTimer(debounce)
			timerCh = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounce)
		timerCh = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case err, ok := <-errs:
			if ok && logger != nil {
				logger.Warn("watch error", "error", err)
			}
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.kind == watchVault {
				pendingUpdate = true
				pendingBuild = true
			} else {
				pendingBuild = true
			}
			trigger()
		case <-timerCh:
			if running {
				trigger()
				continue
			}
			running = true
			update := pendingUpdate
			build := pendingBuild
			pendingUpdate = false
			pendingBuild = false

			if update {
				if logger != nil {
					logger.Info("watch update-content")
				}
				if err := RunUpdateContent(ctx, opts); err != nil && logger != nil {
					logger.Warn("watch update-content failed", "error", err)
				}
			}

			if build {
				if logger != nil {
					logger.Info("watch build")
				}
				if err := RunBuild(ctx, opts); err != nil && logger != nil {
					logger.Warn("watch build failed", "error", err)
				} else if hub != nil {
					hub.Broadcast()
				}
			}

			running = false
			if pendingUpdate || pendingBuild {
				trigger()
			}
		}
	}
}

func runInitialBuild(ctx context.Context, opts CLIOptions, logger *slog.Logger, withUpdate bool) error {
	if withUpdate {
		if err := RunUpdateContent(ctx, opts); err != nil {
			return err
		}
	}
	if err := RunBuild(ctx, opts); err != nil {
		return err
	}
	if logger != nil {
		logger.Info("initial build complete")
	}
	return nil
}
