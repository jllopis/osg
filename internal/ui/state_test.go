package ui

import (
	"testing"

	"osg/internal/build"
)

func TestSectionOf(t *testing.T) {
	cases := map[string]string{
		"/":                "(root)",
		"/post":            "(root)",
		"/blog/2024/post/": "blog",
		"blog/post":        "blog",
		"":                 "(root)",
	}
	for input, want := range cases {
		got := sectionOf(input)
		if got != want {
			t.Errorf("sectionOf(%q)=%q want %q", input, got, want)
		}
	}
}

func TestBars(t *testing.T) {
	got := bars(nil)
	if got != nil {
		t.Errorf("bars(nil)=%v want nil", got)
	}

	in := []build.MonthlyStats{
		{Month: "2024-01", Count: 5},
		{Month: "2024-02", Count: 1},
		{Month: "2024-03", Count: 10},
	}
	out := bars(in)
	if len(out) != 3 {
		t.Fatalf("len=%d want 3", len(out))
	}
	if out[2].Width != 100 {
		t.Errorf("max month width=%d want 100", out[2].Width)
	}
	if out[1].Width != 10 { // 1 / 10 * 100
		t.Errorf("min month width=%d want 10", out[1].Width)
	}
}

func TestBarsAllZero(t *testing.T) {
	in := []build.MonthlyStats{{Month: "2024-01", Count: 0}}
	out := bars(in)
	if len(out) != 1 || out[0].Width != 0 {
		t.Errorf("zero counts should produce zero width, got %+v", out)
	}
}
