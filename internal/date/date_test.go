package date

import (
	"os"
	"testing"
	"time"
)

type fakeFileInfo struct {
	modTime time.Time
}

func (f fakeFileInfo) Name() string       { return "file.md" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestDeriveFromFrontmatter(t *testing.T) {
	fm := map[string]any{"date": "2025-11-02 20:57"}
	info := fakeFileInfo{modTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)}

	got := Derive(fm, info)
	if FormatISO(got) != "2025-11-02" {
		t.Fatalf("expected 2025-11-02, got %s", FormatISO(got))
	}
}

func TestDeriveFallsBackToModTime(t *testing.T) {
	info := fakeFileInfo{modTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)}
	got := Derive(nil, info)
	if FormatISO(got) != "2024-06-15" {
		t.Fatalf("expected 2024-06-15, got %s", FormatISO(got))
	}
}

func TestDeriveFallsBackWhenKeyMissing(t *testing.T) {
	fm := map[string]any{"title": "No date here"}
	info := fakeFileInfo{modTime: time.Date(2023, 3, 10, 0, 0, 0, 0, time.Local)}
	got := Derive(fm, info)
	if FormatISO(got) != "2023-03-10" {
		t.Fatalf("expected 2023-03-10 (modtime), got %s", FormatISO(got))
	}
}

func TestDeriveTriesCandidateKeys(t *testing.T) {
	for _, key := range []string{"date", "created", "created_at", "createdAt", "updated", "modified", "lastmod"} {
		t.Run(key, func(t *testing.T) {
			fm := map[string]any{key: "2025-07-04"}
			info := fakeFileInfo{modTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)}
			got := Derive(fm, info)
			if FormatISO(got) != "2025-07-04" {
				t.Fatalf("key %q: expected 2025-07-04, got %s", key, FormatISO(got))
			}
		})
	}
}

func TestDeriveUnparseableValueFallsBack(t *testing.T) {
	fm := map[string]any{"date": "not-a-date"}
	info := fakeFileInfo{modTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)}
	got := Derive(fm, info)
	if FormatISO(got) != "2024-01-01" {
		t.Fatalf("expected fallback to modtime, got %s", FormatISO(got))
	}
}

func TestFormatPath(t *testing.T) {
	tests := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), "2025/01/02"},
		{time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), "2024/12/31"},
		{time.Date(2000, 7, 4, 0, 0, 0, 0, time.UTC), "2000/07/04"},
	}
	for _, tc := range tests {
		got := FormatPath(tc.t)
		if got != tc.want {
			t.Errorf("FormatPath(%v) = %q, want %q", tc.t, got, tc.want)
		}
	}
}

func TestFormatISO(t *testing.T) {
	got := FormatISO(time.Date(2025, 11, 2, 0, 0, 0, 0, time.UTC))
	if got != "2025-11-02" {
		t.Fatalf("expected 2025-11-02, got %s", got)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		val  any
		ok   bool
		date string // FormatISO result when ok
	}{
		{"time.Time", time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC), true, "2025-03-15"},
		{"int64 unix", int64(1735689600), true, ""}, // just check ok
		{"int unix", 1735689600, true, ""},
		{"float64 unix", float64(1735689600), true, ""},
		{"RFC3339", "2025-03-15T10:30:00Z", true, "2025-03-15"},
		{"datetime space", "2025-03-15 10:30:00", true, "2025-03-15"},
		{"datetime minute", "2025-03-15 10:30", true, "2025-03-15"},
		{"datetimeT minute", "2025-03-15T10:30", true, "2025-03-15"},
		{"date only", "2025-03-15", true, "2025-03-15"},
		{"empty string", "", false, ""},
		{"whitespace only", "   ", false, ""},
		{"garbage string", "not-a-date", false, ""},
		{"nil", nil, false, ""},
		{"bool", true, false, ""},
		{"slice", []string{"a"}, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Parse(tc.val)
			if ok != tc.ok {
				t.Fatalf("Parse(%v): ok=%v, want %v", tc.val, ok, tc.ok)
			}
			if ok && tc.date != "" && FormatISO(got) != tc.date {
				t.Fatalf("Parse(%v) = %s, want %s", tc.val, FormatISO(got), tc.date)
			}
		})
	}
}
