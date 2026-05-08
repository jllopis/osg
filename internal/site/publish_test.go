package site

import (
	"testing"
	"time"
)

func TestPageIsScheduled(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		publishAt time.Time
		want      bool
	}{
		{"zero -> not scheduled", time.Time{}, false},
		{"past -> not scheduled", now.Add(-time.Hour), false},
		{"future -> scheduled", now.Add(time.Hour), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &Page{PublishAt: c.publishAt}
			if got := p.IsScheduled(); got != c.want {
				t.Errorf("IsScheduled()=%v want %v", got, c.want)
			}
		})
	}
}

func TestPickTimeRFC3339(t *testing.T) {
	fm := map[string]any{"publish_at": "2030-01-15T10:00:00Z"}
	got := pickTime(fm, "publish_at")
	if got.IsZero() {
		t.Fatalf("pickTime returned zero for valid RFC3339")
	}
	if got.Year() != 2030 || got.Month() != time.January || got.Day() != 15 {
		t.Errorf("pickTime parsed wrong date: %v", got)
	}
}

func TestPickTimeMissing(t *testing.T) {
	if got := pickTime(nil, "x"); !got.IsZero() {
		t.Errorf("pickTime(nil)=%v want zero", got)
	}
	if got := pickTime(map[string]any{"y": 1}, "x"); !got.IsZero() {
		t.Errorf("pickTime missing key=%v want zero", got)
	}
	if got := pickTime(map[string]any{"x": "not-a-date"}, "x"); !got.IsZero() {
		t.Errorf("pickTime invalid string=%v want zero", got)
	}
}
