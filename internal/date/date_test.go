package date

import (
	"os"
	"testing"
	"time"
)

func TestDeriveFromFrontmatter(t *testing.T) {
	fm := map[string]any{"date": "2025-11-02 20:57"}
	info := fakeFileInfo{modTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)}

	got := Derive(fm, info)
	if FormatISO(got) != "2025-11-02" {
		t.Fatalf("expected 2025-11-02, got %s", FormatISO(got))
	}
}

type fakeFileInfo struct {
	modTime time.Time
}

func (f fakeFileInfo) Name() string       { return "file.md" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }
