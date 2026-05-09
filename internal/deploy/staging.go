package deploy

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// StagingDirName is the staging dir created next to publicDir when a
// deploy needs to exclude paths. Hard-linked from public/ so the disk
// cost is one inode per file plus directory metadata.
const StagingDirName = ".osg-deploy-staging"

// Staging carries the directory the deploy provider should upload
// from plus a Cleanup hook the caller must call (typically via
// defer). When no exclusions apply, Dir is publicDir verbatim and
// Cleanup is a no-op — zero overhead for the common case.
type Staging struct {
	Dir      string
	Cleanup  func()
	Excluded []string
}

// Stage prepares the deploy source. When excludePaths is empty it
// returns publicDir unchanged. Otherwise it walks publicDir and
// hardlinks every entry into a sibling staging directory, skipping
// any subtree whose URL path matches an entry in excludePaths.
//
// excludePaths uses URL form ("/2026/02/foo/" or "/foo/bar/") and
// is treated as a prefix: the entire subtree under each path is
// removed from the staging dir. Files outside the listed prefixes
// are unchanged.
//
// Hardlinks are preferred (zero copy, instant). On EXDEV (staging
// dir on a different filesystem than publicDir) a regular file copy
// is used instead. Errors abort and clean up the partial staging.
func Stage(publicDir string, excludePaths []string, logger *slog.Logger) (*Staging, error) {
	cleaned := normaliseExcludes(excludePaths)
	if len(cleaned) == 0 {
		return &Staging{Dir: publicDir, Cleanup: func() {}}, nil
	}

	excludedFS := make(map[string]struct{}, len(cleaned))
	for _, p := range cleaned {
		// "/2026/02/foo/" -> "publicDir/2026/02/foo".
		rel := strings.Trim(p, "/")
		if rel == "" {
			// Excluding "/" would empty the deploy entirely; treat as
			// nothing to exclude so a typo can't take the whole site
			// offline by accident.
			continue
		}
		excludedFS[filepath.Join(publicDir, filepath.FromSlash(rel))] = struct{}{}
	}
	if len(excludedFS) == 0 {
		return &Staging{Dir: publicDir, Cleanup: func() {}}, nil
	}

	stagingDir := filepath.Join(filepath.Dir(publicDir), StagingDirName)
	if err := os.RemoveAll(stagingDir); err != nil {
		return nil, fmt.Errorf("clean staging dir: %w", err)
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(stagingDir) }

	err := filepath.WalkDir(publicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == publicDir {
			return nil
		}
		if _, skip := excludedFS[path]; skip {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(publicDir, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(stagingDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// Try hardlink first; on cross-device fall back to copy.
		if linkErr := os.Link(path, target); linkErr == nil {
			return nil
		} else if !errors.Is(linkErr, syscall.EXDEV) {
			// Some platforms also report cross-device differently
			// (e.g. errno 18 wrapped in *os.LinkError). Recover by
			// trying the copy path before giving up.
			if !isCrossDevice(linkErr) {
				return fmt.Errorf("hardlink %s -> %s: %w", path, target, linkErr)
			}
		}
		return copyFile(path, target)
	})
	if err != nil {
		cleanup()
		return nil, err
	}

	excluded := make([]string, 0, len(cleaned))
	for p := range excludedFS {
		rel, _ := filepath.Rel(publicDir, p)
		excluded = append(excluded, "/"+filepath.ToSlash(rel)+"/")
	}
	if logger != nil {
		logger.Info("deploy staging prepared",
			"dir", stagingDir,
			"excluded", len(excluded))
	}
	return &Staging{Dir: stagingDir, Cleanup: cleanup, Excluded: excluded}, nil
}

// normaliseExcludes trims and dedupes the input list, dropping empty
// entries. Keeps the order stable so logging is deterministic.
func normaliseExcludes(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// copyFile is the fallback used when os.Link cannot be applied
// (cross-device staging). Preserves mode bits but not ownership —
// adequate for a read-only artefact passed to a deploy provider.
func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer func() { _ = srcF.Close() }()
	info, err := srcF.Stat()
	if err != nil {
		return err
	}
	dstF, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer func() { _ = dstF.Close() }()
	if _, err := io.Copy(dstF, srcF); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	return nil
}

// isCrossDevice unwraps a LinkError to spot platform-specific
// cross-device markers that aren't reachable via errors.Is on
// some systems.
func isCrossDevice(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if errors.Is(linkErr.Err, syscall.EXDEV) {
			return true
		}
	}
	return false
}
