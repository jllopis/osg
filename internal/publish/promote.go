package publish

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/date"
	"osg/internal/frontmatter"
)

// PublishAt extracts the publish_at frontmatter value (looking inside
// the osg block first, then the top level) and parses it with the
// shared date.Parse layouts. Returns the zero time when absent or
// unparseable.
func PublishAt(fm map[string]any) time.Time {
	if fm == nil {
		return time.Time{}
	}
	if osg := GetOSGBlock(fm); osg != nil {
		if t := parseTime(osg["publish_at"]); !t.IsZero() {
			return t
		}
	}
	return parseTime(fm["publish_at"])
}

func parseTime(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := date.Parse(v); ok {
		return t
	}
	return time.Time{}
}

// PromoteDueDrafts walks vaultPath, locates markdown files marked as
// drafts with a publish_at that has already arrived, and rewrites
// each one's frontmatter to flip osg.publish: draft → osg.publish:
// true. Returns the list of vault-relative paths that were promoted
// so callers can log / surface them in the operations audit.
//
// The rewrite is atomic per file (write-temp + rename) so a crash
// mid-promotion never leaves a half-written .md in the vault. Errors
// on individual files are logged but do not abort the sweep — we'd
// rather promote what we can than fail the whole publish window.
func PromoteDueDrafts(vaultPath string, now time.Time, logger *slog.Logger) ([]string, error) {
	vaultPath = strings.TrimSpace(vaultPath)
	if vaultPath == "" {
		return nil, nil
	}
	if _, err := os.Stat(vaultPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat vault: %w", err)
	}

	var promoted []string
	err := filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != vaultPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if logger != nil {
				logger.Warn("promote: read failed", "path", path, "error", err)
			}
			return nil
		}
		fm, _, _, err := frontmatter.SplitFrontmatter(data)
		if err != nil {
			if logger != nil {
				logger.Warn("promote: parse frontmatter failed", "path", path, "error", err)
			}
			return nil
		}
		if !shouldPromote(fm, now) {
			return nil
		}

		updated, err := frontmatter.UpdateField(data, "osg.publish", "true")
		if err != nil {
			if logger != nil {
				logger.Warn("promote: rewrite failed", "path", path, "error", err)
			}
			return nil
		}
		if err := writeAtomic(path, updated); err != nil {
			if logger != nil {
				logger.Warn("promote: write failed", "path", path, "error", err)
			}
			return nil
		}
		rel, relErr := filepath.Rel(vaultPath, path)
		if relErr != nil {
			rel = path
		}
		promoted = append(promoted, filepath.ToSlash(rel))
		if logger != nil {
			logger.Info("promote: draft published",
				"path", filepath.ToSlash(rel),
				"publish_at", PublishAt(fm).Format(time.RFC3339),
			)
		}
		return nil
	})
	if err != nil {
		return promoted, err
	}
	return promoted, nil
}

// shouldPromote returns true when fm represents a draft whose
// publish_at is set and lies at or before `now`. Drafts without a
// publish_at, scheduled-but-not-due drafts, and non-drafts are
// untouched — the function captures the exact "auto-publish window
// has arrived" condition.
func shouldPromote(fm map[string]any, now time.Time) bool {
	_, isDraft := ShouldPublish(fm)
	if !isDraft {
		return false
	}
	at := PublishAt(fm)
	if at.IsZero() {
		return false
	}
	return !at.After(now)
}

// writeAtomic writes data to path via a temp file in the same dir
// then renames into place. Mirrors the helper used by the UI editor
// so behaviour stays consistent across all vault writes.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".osg-promote-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
