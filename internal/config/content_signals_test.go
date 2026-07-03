package config

import "testing"

func boolPtr(v bool) *bool { return &v }

func TestContentSignalsLine_Defaults(t *testing.T) {
	var cs ContentSignalsConfig
	got := cs.Line()
	want := "ai-train=no, search=yes, ai-input=yes"
	if got != want {
		t.Errorf("Line() = %q, want %q", got, want)
	}
}

func TestContentSignalsLine_Disabled(t *testing.T) {
	cs := ContentSignalsConfig{Enabled: boolPtr(false)}
	if got := cs.Line(); got != "" {
		t.Errorf("Line() = %q, want empty", got)
	}
}

func TestContentSignalsLine_Overrides(t *testing.T) {
	cs := ContentSignalsConfig{
		Enabled: boolPtr(true),
		AITrain: boolPtr(true),
		Search:  boolPtr(false),
		AIInput: boolPtr(false),
	}
	got := cs.Line()
	want := "ai-train=yes, search=no, ai-input=no"
	if got != want {
		t.Errorf("Line() = %q, want %q", got, want)
	}
}

func TestContentSignalsLine_PartialOverride(t *testing.T) {
	// Only ai_train set; the rest keep their defaults.
	cs := ContentSignalsConfig{AITrain: boolPtr(true)}
	got := cs.Line()
	want := "ai-train=yes, search=yes, ai-input=yes"
	if got != want {
		t.Errorf("Line() = %q, want %q", got, want)
	}
}
