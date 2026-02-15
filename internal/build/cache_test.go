package build

import (
	"io/fs"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

// ---------------------------------------------------------------------------
// diffContent
// ---------------------------------------------------------------------------

func TestDiffContent(t *testing.T) {
	t.Run("nil prev treats all current as changed", func(t *testing.T) {
		current := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
			"b.md": {ModTime: 200, Size: 20},
		}
		changed, removed := diffContent(nil, current)
		if len(changed) != 2 {
			t.Errorf("changed count = %d; want 2", len(changed))
		}
		if !changed["a.md"] || !changed["b.md"] {
			t.Errorf("expected both files to be marked changed")
		}
		if len(removed) != 0 {
			t.Errorf("removed = %v; want empty", removed)
		}
	})

	t.Run("detects changed files", func(t *testing.T) {
		prev := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
			"b.md": {ModTime: 200, Size: 20},
		}
		current := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10}, // unchanged
			"b.md": {ModTime: 300, Size: 20}, // changed (different mod time)
		}
		changed, removed := diffContent(prev, current)
		if len(changed) != 1 {
			t.Fatalf("changed count = %d; want 1", len(changed))
		}
		if !changed["b.md"] {
			t.Error("expected b.md to be marked changed")
		}
		if len(removed) != 0 {
			t.Errorf("removed = %v; want empty", removed)
		}
	})

	t.Run("detects removed files", func(t *testing.T) {
		prev := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
			"b.md": {ModTime: 200, Size: 20},
		}
		current := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
		}
		changed, removed := diffContent(prev, current)
		if len(changed) != 0 {
			t.Errorf("changed count = %d; want 0", len(changed))
		}
		if len(removed) != 1 || removed[0] != "b.md" {
			t.Errorf("removed = %v; want [b.md]", removed)
		}
	})

	t.Run("detects new files", func(t *testing.T) {
		prev := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
		}
		current := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
			"c.md": {ModTime: 500, Size: 50},
		}
		changed, removed := diffContent(prev, current)
		if len(changed) != 1 || !changed["c.md"] {
			t.Errorf("changed = %v; want {c.md: true}", changed)
		}
		if len(removed) != 0 {
			t.Errorf("removed = %v; want empty", removed)
		}
	})

	t.Run("no changes", func(t *testing.T) {
		stamps := map[string]fileStamp{
			"a.md": {ModTime: 100, Size: 10},
		}
		changed, removed := diffContent(stamps, stamps)
		if len(changed) != 0 {
			t.Errorf("changed count = %d; want 0", len(changed))
		}
		if len(removed) != 0 {
			t.Errorf("removed count = %d; want 0", len(removed))
		}
	})
}

// ---------------------------------------------------------------------------
// hashBytes
// ---------------------------------------------------------------------------

func TestHashBytes(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		a := hashBytes([]byte("hello"))
		b := hashBytes([]byte("hello"))
		if a != b {
			t.Errorf("hashBytes not deterministic: %q != %q", a, b)
		}
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		a := hashBytes([]byte("hello"))
		b := hashBytes([]byte("world"))
		if a == b {
			t.Error("different inputs produced the same hash")
		}
	})

	t.Run("returns 64-char hex string", func(t *testing.T) {
		h := hashBytes([]byte("test"))
		if len(h) != 64 {
			t.Errorf("hash length = %d; want 64", len(h))
		}
	})
}

// ---------------------------------------------------------------------------
// hashStrings
// ---------------------------------------------------------------------------

func TestHashStrings(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		a := hashStrings([]string{"a", "b", "c"})
		b := hashStrings([]string{"a", "b", "c"})
		if a != b {
			t.Errorf("hashStrings not deterministic: %q != %q", a, b)
		}
	})

	t.Run("order matters", func(t *testing.T) {
		a := hashStrings([]string{"a", "b"})
		b := hashStrings([]string{"b", "a"})
		if a == b {
			t.Error("different order produced the same hash")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		h := hashStrings(nil)
		if len(h) != 64 {
			t.Errorf("hash length = %d; want 64", len(h))
		}
	})
}

// ---------------------------------------------------------------------------
// isTemplateFile
// ---------------------------------------------------------------------------

// fakeDirEntry implements fs.DirEntry for testing.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestIsTemplateFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"page.html", true},
		{"sitemap.xml", true},
		{"robots.txt", true},
		{"page.HTML", true},
		{"style.css", false},
		{"main.go", false},
		{"data.json", false},
		{"image.png", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := isTemplateFile(tc.path, fakeDirEntry{name: filepath.Base(tc.path)})
			if got != tc.want {
				t.Errorf("isTemplateFile(%q) = %v; want %v", tc.path, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// shouldSkipDir
// ---------------------------------------------------------------------------

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".git", true},
		{".hidden", true},
		{".", true},
		{"templates", false},
		{"public", false},
		{"node_modules", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSkipDir(tc.name)
			if got != tc.want {
				t.Errorf("shouldSkipDir(%q) = %v; want %v", tc.name, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildCachePath
// ---------------------------------------------------------------------------

func TestBuildCachePath(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{
			name: "with cache dir",
			cfg:  config.Config{BuildCacheDir: ".cache"},
			want: filepath.Join(".cache", "build.json"),
		},
		{
			name: "empty cache dir",
			cfg:  config.Config{BuildCacheDir: ""},
			want: "",
		},
		{
			name: "whitespace cache dir",
			cfg:  config.Config{BuildCacheDir: "  "},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildCachePath(tc.cfg)
			if got != tc.want {
				t.Errorf("buildCachePath = %q; want %q", got, tc.want)
			}
		})
	}
}
