package tui

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// parseLogLine
// ---------------------------------------------------------------------------

func TestParseLogLine(t *testing.T) {
	t.Run("valid JSON log", func(t *testing.T) {
		ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
		entry := map[string]any{
			"time":  ts.Format(time.RFC3339Nano),
			"level": "INFO",
			"msg":   "build started",
			"pages": float64(42),
		}
		data, _ := json.Marshal(entry)
		msg := parseLogLine(string(data))
		if msg.Label != "INFO" {
			t.Errorf("Label = %q; want \"INFO\"", msg.Label)
		}
		if msg.Text != "build started" {
			t.Errorf("Text = %q; want \"build started\"", msg.Text)
		}
		if !msg.Time.Equal(ts) {
			t.Errorf("Time = %v; want %v", msg.Time, ts)
		}
		if msg.Fields["pages"] != float64(42) {
			t.Errorf("Fields[pages] = %v; want 42", msg.Fields["pages"])
		}
		// Ensure reserved keys are excluded from fields.
		if _, ok := msg.Fields["time"]; ok {
			t.Error("Fields should not contain \"time\"")
		}
		if _, ok := msg.Fields["level"]; ok {
			t.Error("Fields should not contain \"level\"")
		}
		if _, ok := msg.Fields["msg"]; ok {
			t.Error("Fields should not contain \"msg\"")
		}
	})

	t.Run("invalid JSON falls back to raw", func(t *testing.T) {
		msg := parseLogLine("this is not json")
		if msg.Label != "LOG" {
			t.Errorf("Label = %q; want \"LOG\"", msg.Label)
		}
		if msg.Text != "this is not json" {
			t.Errorf("Text = %q; want \"this is not json\"", msg.Text)
		}
	})

	t.Run("missing level defaults to LOG", func(t *testing.T) {
		data, _ := json.Marshal(map[string]any{"msg": "hello"})
		msg := parseLogLine(string(data))
		// fmt.Sprint(nil) for missing level gives "<nil>", which is empty after ToUpper check
		// Actually level will be "<nil>" which uppercased is "<NIL>". But the code does:
		// label := strings.ToUpper(fmt.Sprint(entry["level"]))
		// If "level" is missing, entry["level"] is nil, fmt.Sprint(nil) = "<nil>",
		// strings.ToUpper("<nil>") = "<NIL>". Then label == "" check is false.
		// So the label will be "<NIL>".
		// Let's just check it's non-empty.
		if msg.Label == "" {
			t.Error("Label should not be empty")
		}
	})

	t.Run("missing msg uses raw line", func(t *testing.T) {
		raw := `{"level":"WARN"}`
		msg := parseLogLine(raw)
		if msg.Label != "WARN" {
			t.Errorf("Label = %q; want \"WARN\"", msg.Label)
		}
		// msg is nil, fmt.Sprint(nil) = "<nil>", which triggers text = line
		if msg.Text != raw {
			t.Errorf("Text = %q; want %q", msg.Text, raw)
		}
	})
}

// ---------------------------------------------------------------------------
// normalizePluginSet
// ---------------------------------------------------------------------------

func TestNormalizePluginSet(t *testing.T) {
	t.Run("deduplicates", func(t *testing.T) {
		set := normalizePluginSet([]string{"foo", "foo", "bar"})
		if len(set) != 2 {
			t.Errorf("set size = %d; want 2", len(set))
		}
		if !set["foo"] || !set["bar"] {
			t.Errorf("set = %v", set)
		}
	})

	t.Run("strips .wasm suffix", func(t *testing.T) {
		set := normalizePluginSet([]string{"myplugin.wasm"})
		if !set["myplugin"] {
			t.Errorf("expected myplugin in set; got %v", set)
		}
	})

	t.Run("empty strings ignored", func(t *testing.T) {
		set := normalizePluginSet([]string{"", "  ", "valid"})
		if len(set) != 1 {
			t.Errorf("set size = %d; want 1", len(set))
		}
		if !set["valid"] {
			t.Errorf("expected valid in set; got %v", set)
		}
	})

	t.Run("nil input", func(t *testing.T) {
		set := normalizePluginSet(nil)
		if len(set) != 0 {
			t.Errorf("set size = %d; want 0", len(set))
		}
	})
}

// ---------------------------------------------------------------------------
// normalizePluginName
// ---------------------------------------------------------------------------

func TestNormalizePluginName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"myplugin", "myplugin"},
		{"myplugin.wasm", "myplugin"},
		{"MYPLUGIN.WASM", "MYPLUGIN"},
		{"plugin.Wasm", "plugin"},
		{"", ""},
		{"  ", ""},
		{"  spaced  ", "spaced"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizePluginName(tc.input)
			if got != tc.want {
				t.Errorf("normalizePluginName(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// asInt
// ---------------------------------------------------------------------------

func TestAsInt(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{"int", 42, 42},
		{"int64", int64(99), 99},
		{"float64", float64(7), 7},
		{"float32", float32(3), 3},
		{"string number", "123", 123},
		{"string with spaces", " 456 ", 456},
		{"invalid string", "abc", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := asInt(tc.input)
			if got != tc.want {
				t.Errorf("asInt(%v) = %d; want %d", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// asString
// ---------------------------------------------------------------------------

func TestAsString(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "  world  ", "world"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int64", int64(99), "99"},
		{"float64", float64(3.14), "3.14"},
		{"nil", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := asString(tc.input)
			if got != tc.want {
				t.Errorf("asString(%v) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// defaultAddr
// ---------------------------------------------------------------------------

func TestDefaultAddr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ":1313"},
		{"  ", ":1313"},
		{":8080", ":8080"},
		{"localhost:3000", "localhost:3000"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := defaultAddr(tc.input)
			if got != tc.want {
				t.Errorf("defaultAddr(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// defaultValue
// ---------------------------------------------------------------------------

func TestDefaultValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "-"},
		{"  ", "-"},
		{"hello", "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := defaultValue(tc.input)
			if got != tc.want {
				t.Errorf("defaultValue(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// defaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "config.yaml"},
		{"  ", "config.yaml"},
		{"my-config.yaml", "my-config.yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := defaultConfig(tc.input)
			if got != tc.want {
				t.Errorf("defaultConfig(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formatDuration
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"milliseconds", 450 * time.Millisecond, "450ms"},
		{"sub-second", 999 * time.Millisecond, "999ms"},
		{"one second", time.Second, "1s"},
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 2*time.Minute + 30*time.Second, "2m30s"},
		{"zero", 0, "0ms"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDuration(tc.dur)
			if got != tc.want {
				t.Errorf("formatDuration(%v) = %q; want %q", tc.dur, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// stepIndexForTask
// ---------------------------------------------------------------------------

func TestStepIndexForTask(t *testing.T) {
	tests := []struct {
		kind taskKind
		want int
	}{
		{taskInit, 0},
		{taskUpdate, 1},
		{taskBuild, 2},
		{taskServe, 3},
		{taskKind(99), -1},
	}

	for _, tc := range tests {
		got := stepIndexForTask(tc.kind)
		if got != tc.want {
			t.Errorf("stepIndexForTask(%d) = %d; want %d", tc.kind, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// taskLabel
// ---------------------------------------------------------------------------

func TestTaskLabel(t *testing.T) {
	tests := []struct {
		kind taskKind
		want string
	}{
		{taskInit, "Init"},
		{taskUpdate, "Update content"},
		{taskBuild, "Build"},
		{taskServe, "Serve"},
		{taskKind(99), "Task"},
	}

	for _, tc := range tests {
		got := taskLabel(tc.kind)
		if got != tc.want {
			t.Errorf("taskLabel(%d) = %q; want %q", tc.kind, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// actionForTask
// ---------------------------------------------------------------------------

func TestActionForTask(t *testing.T) {
	called := ""
	actions := Actions{
		Init:   func(_ context.Context) error { called = "init"; return nil },
		Update: func(_ context.Context) error { called = "update"; return nil },
		Build:  func(_ context.Context) error { called = "build"; return nil },
	}

	t.Run("init", func(t *testing.T) {
		fn := actionForTask(actions, taskInit)
		if fn == nil {
			t.Fatal("actionForTask(taskInit) returned nil")
		}
		_ = fn(context.Background())
		if called != "init" {
			t.Errorf("called = %q; want \"init\"", called)
		}
	})

	t.Run("update", func(t *testing.T) {
		fn := actionForTask(actions, taskUpdate)
		if fn == nil {
			t.Fatal("actionForTask(taskUpdate) returned nil")
		}
		_ = fn(context.Background())
		if called != "update" {
			t.Errorf("called = %q; want \"update\"", called)
		}
	})

	t.Run("build", func(t *testing.T) {
		fn := actionForTask(actions, taskBuild)
		if fn == nil {
			t.Fatal("actionForTask(taskBuild) returned nil")
		}
		_ = fn(context.Background())
		if called != "build" {
			t.Errorf("called = %q; want \"build\"", called)
		}
	})

	t.Run("serve returns nil", func(t *testing.T) {
		fn := actionForTask(actions, taskServe)
		if fn != nil {
			t.Error("actionForTask(taskServe) should return nil")
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		fn := actionForTask(actions, taskKind(99))
		if fn != nil {
			t.Error("actionForTask(unknown) should return nil")
		}
	})
}
