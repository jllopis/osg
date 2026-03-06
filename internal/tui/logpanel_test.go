package tui

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// LogPanel construction and toggle
// ---------------------------------------------------------------------------

func TestNewLogPanel(t *testing.T) {
	lp := NewLogPanel()
	if lp.Visible() {
		t.Error("new panel should be hidden")
	}
	if lp.Tab() != LogTabAll {
		t.Errorf("default tab = %d; want LogTabAll (%d)", lp.Tab(), LogTabAll)
	}
}

func TestLogPanelToggle(t *testing.T) {
	lp := NewLogPanel()
	lp.Toggle()
	if !lp.Visible() {
		t.Error("panel should be visible after toggle")
	}
	lp.Toggle()
	if lp.Visible() {
		t.Error("panel should be hidden after second toggle")
	}
}

// ---------------------------------------------------------------------------
// Tab navigation
// ---------------------------------------------------------------------------

func TestLogPanelTabs(t *testing.T) {
	lp := NewLogPanel()

	// Default is All.
	if lp.Tab() != LogTabAll {
		t.Fatalf("default tab = %d; want %d", lp.Tab(), LogTabAll)
	}

	// SetTab.
	lp.SetTab(LogTabServe)
	if lp.Tab() != LogTabServe {
		t.Errorf("tab = %d; want LogTabServe", lp.Tab())
	}
	lp.SetTab(LogTabAPI)
	if lp.Tab() != LogTabAPI {
		t.Errorf("tab = %d; want LogTabAPI", lp.Tab())
	}

	// NextTab cycles Serve -> API -> All -> Serve.
	lp.SetTab(LogTabServe)
	lp.NextTab()
	if lp.Tab() != LogTabAPI {
		t.Errorf("after NextTab: tab = %d; want LogTabAPI", lp.Tab())
	}
	lp.NextTab()
	if lp.Tab() != LogTabAll {
		t.Errorf("after NextTab: tab = %d; want LogTabAll", lp.Tab())
	}
	lp.NextTab()
	if lp.Tab() != LogTabServe {
		t.Errorf("after NextTab: tab = %d; want LogTabServe", lp.Tab())
	}

	// PrevTab cycles All -> API -> Serve -> All.
	lp.SetTab(LogTabAll)
	lp.PrevTab()
	if lp.Tab() != LogTabAPI {
		t.Errorf("after PrevTab: tab = %d; want LogTabAPI", lp.Tab())
	}
	lp.PrevTab()
	if lp.Tab() != LogTabServe {
		t.Errorf("after PrevTab: tab = %d; want LogTabServe", lp.Tab())
	}
	lp.PrevTab()
	if lp.Tab() != LogTabAll {
		t.Errorf("after PrevTab: tab = %d; want LogTabAll", lp.Tab())
	}
}

// ---------------------------------------------------------------------------
// Resize and viewport
// ---------------------------------------------------------------------------

func TestLogPanelResize(t *testing.T) {
	lp := NewLogPanel()
	lp.Toggle()

	// Before resize, viewport is not ready.
	if lp.ready {
		t.Error("viewport should not be ready before Resize")
	}

	lp.Resize(120, 15)
	if !lp.ready {
		t.Error("viewport should be ready after Resize")
	}
	if lp.width != 120 {
		t.Errorf("width = %d; want 120", lp.width)
	}
	if lp.height != 15 {
		t.Errorf("height = %d; want 15", lp.height)
	}

	// Second resize updates dimensions.
	lp.Resize(80, 10)
	if lp.width != 80 {
		t.Errorf("width after re-resize = %d; want 80", lp.width)
	}
}

// ---------------------------------------------------------------------------
// SetContent and View
// ---------------------------------------------------------------------------

func TestLogPanelSetContent(t *testing.T) {
	lp := NewLogPanel()
	lp.Toggle()
	lp.Resize(80, 10)

	msgs := []Message{
		{Label: "INFO", Text: "first message", Time: time.Now()},
		{Label: "ERROR", Text: "second message", Time: time.Now()},
	}
	lp.SetContent(msgs)

	view := lp.View()
	if view == "" {
		t.Error("View() should not be empty when visible with content")
	}
}

func TestLogPanelViewHiddenEmpty(t *testing.T) {
	lp := NewLogPanel()
	// Hidden panel should return empty string.
	if lp.View() != "" {
		t.Error("View() should be empty when hidden")
	}
}

// ---------------------------------------------------------------------------
// PanelHeight
// ---------------------------------------------------------------------------

func TestPanelHeight(t *testing.T) {
	tests := []struct {
		termHeight int
		wantMin    int
		wantMax    int
	}{
		{10, 4, 4},  // 10/3 = 3, clamped to min 4
		{30, 4, 20}, // 30/3 = 10
		{60, 4, 20}, // 60/3 = 20, clamped to max 20
		{90, 4, 20}, // 90/3 = 30, clamped to max 20
	}
	for _, tt := range tests {
		h := PanelHeight(tt.termHeight)
		if h < tt.wantMin || h > tt.wantMax {
			t.Errorf("PanelHeight(%d) = %d; want %d..%d", tt.termHeight, h, tt.wantMin, tt.wantMax)
		}
	}
}

// ---------------------------------------------------------------------------
// MessagesForTab
// ---------------------------------------------------------------------------

func TestMessagesForTab(t *testing.T) {
	serve := []Message{{Label: "INFO", Text: "serve msg"}}
	api := []Message{{Label: "INFO", Text: "api msg"}}
	all := []Message{{Label: "INFO", Text: "all msg"}}

	got := MessagesForTab(LogTabServe, serve, api, all)
	if len(got) != 1 || got[0].Text != "serve msg" {
		t.Errorf("LogTabServe: got %v", got)
	}

	got = MessagesForTab(LogTabAPI, serve, api, all)
	if len(got) != 1 || got[0].Text != "api msg" {
		t.Errorf("LogTabAPI: got %v", got)
	}

	got = MessagesForTab(LogTabAll, serve, api, all)
	if len(got) != 1 || got[0].Text != "all msg" {
		t.Errorf("LogTabAll: got %v", got)
	}
}

// ---------------------------------------------------------------------------
// renderLogMessages
// ---------------------------------------------------------------------------

func TestRenderLogMessages(t *testing.T) {
	t.Run("empty messages", func(t *testing.T) {
		result := renderLogMessages(nil, 80)
		if result == "" {
			t.Error("should show empty message text")
		}
	})

	t.Run("formats messages with timestamp and label", func(t *testing.T) {
		msgs := []Message{
			{Label: "INFO", Text: "hello", Time: time.Date(2025, 1, 15, 14, 32, 1, 0, time.UTC)},
		}
		result := renderLogMessages(msgs, 80)
		if result == "" {
			t.Error("should not be empty")
		}
	})
}
