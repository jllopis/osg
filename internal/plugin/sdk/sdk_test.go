package sdk

import (
	"encoding/json"
	"testing"
)

func TestNewPlugin(t *testing.T) {
	p := NewPlugin(PluginMeta{
		Name:    "test-plugin",
		Version: "1.0.0",
		Hooks:   []string{"build.finished"},
	})
	if p == nil {
		t.Fatal("NewPlugin returned nil")
	}
	if globalPlugin != p {
		t.Fatal("global plugin not set")
	}
}

func TestHandleEvent_NoPlugin(t *testing.T) {
	globalPlugin = nil
	out := HandleEvent([]byte(`{"type":"build.finished","payload":{}}`))
	if out != nil {
		t.Fatal("expected nil when no plugin registered")
	}
}

func TestHandleEvent_InvalidJSON(t *testing.T) {
	NewPlugin(PluginMeta{Name: "test"})
	out := HandleEvent([]byte(`not json`))
	if out != nil {
		t.Fatal("expected nil for invalid JSON")
	}
}

func TestHandleEvent_UnhandledHook(t *testing.T) {
	p := NewPlugin(PluginMeta{Name: "test"})
	p.On("build.finished", func(ev Event) *Response {
		return &Response{Payload: map[string]any{"handled": true}}
	})

	out := HandleEvent([]byte(`{"type":"build.started","payload":{}}`))
	if out != nil {
		t.Fatal("expected nil for unhandled hook")
	}
}

func TestHandleEvent_HandledHook(t *testing.T) {
	p := NewPlugin(PluginMeta{Name: "test"})
	var receivedType string
	p.On("build.finished", func(ev Event) *Response {
		receivedType = ev.Type
		return &Response{Payload: map[string]any{
			"custom_key": "custom_value",
		}}
	})

	input := `{"type":"build.finished","payload":{"config":{"site_title":"Test"}}}`
	out := HandleEvent([]byte(input))
	if out == nil {
		t.Fatal("expected non-nil response")
	}
	if receivedType != "build.finished" {
		t.Fatalf("expected type build.finished, got %s", receivedType)
	}

	var resp Response
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Payload["custom_key"] != "custom_value" {
		t.Fatalf("expected custom_value, got %v", resp.Payload["custom_key"])
	}
}

func TestHandleEvent_NilResponse(t *testing.T) {
	p := NewPlugin(PluginMeta{Name: "test"})
	p.On("build.finished", func(ev Event) *Response {
		return nil
	})

	out := HandleEvent([]byte(`{"type":"build.finished","payload":{}}`))
	if out != nil {
		t.Fatal("expected nil for nil response")
	}
}

func TestGetPluginInfo(t *testing.T) {
	NewPlugin(PluginMeta{
		Name:        "my-plugin",
		Version:     "2.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		Hooks:       []string{"build.finished", "after.build"},
	})

	data := GetPluginInfo()
	if data == nil {
		t.Fatal("expected non-nil plugin info")
	}

	var meta PluginMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if meta.Name != "my-plugin" {
		t.Fatalf("expected my-plugin, got %s", meta.Name)
	}
	if meta.Version != "2.0.0" {
		t.Fatalf("expected 2.0.0, got %s", meta.Version)
	}
	if len(meta.Hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(meta.Hooks))
	}
}

func TestGetPluginInfo_NoPlugin(t *testing.T) {
	globalPlugin = nil
	data := GetPluginInfo()
	if data != nil {
		t.Fatal("expected nil when no plugin")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"config": map[string]any{
			"site_title": "My Site",
			"public_dir": "/tmp/public",
		},
		"simple": "value",
	}

	if s := GetString(m, "config", "site_title"); s != "My Site" {
		t.Fatalf("expected 'My Site', got %q", s)
	}
	if s := GetString(m, "config", "public_dir"); s != "/tmp/public" {
		t.Fatalf("expected '/tmp/public', got %q", s)
	}
	if s := GetString(m, "simple"); s != "value" {
		t.Fatalf("expected 'value', got %q", s)
	}
	if s := GetString(m, "missing"); s != "" {
		t.Fatalf("expected empty, got %q", s)
	}
	if s := GetString(m, "config", "missing"); s != "" {
		t.Fatalf("expected empty, got %q", s)
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{
		"config": map[string]any{
			"lightbox": true,
			"drafts":   false,
		},
	}

	if b := GetBool(m, "config", "lightbox"); !b {
		t.Fatal("expected true")
	}
	if b := GetBool(m, "config", "drafts"); b {
		t.Fatal("expected false")
	}
	if b := GetBool(m, "config", "missing"); b {
		t.Fatal("expected false for missing key")
	}
}

func TestGetFloat(t *testing.T) {
	m := map[string]any{
		"stats": map[string]any{
			"total": float64(42),
		},
	}

	if f := GetFloat(m, "stats", "total"); f != 42 {
		t.Fatalf("expected 42, got %f", f)
	}
	if f := GetFloat(m, "stats", "missing"); f != 0 {
		t.Fatalf("expected 0, got %f", f)
	}
}

func TestGetSlice(t *testing.T) {
	m := map[string]any{
		"tags": []any{"go", "wasm", "osg"},
	}

	s := GetSlice(m, "tags")
	if len(s) != 3 {
		t.Fatalf("expected 3, got %d", len(s))
	}
	if s[0] != "go" {
		t.Fatalf("expected 'go', got %v", s[0])
	}
}

func TestGetMap(t *testing.T) {
	m := map[string]any{
		"config": map[string]any{
			"nested": map[string]any{
				"key": "val",
			},
		},
	}

	inner := GetMap(m, "config", "nested")
	if inner == nil {
		t.Fatal("expected non-nil")
	}
	if inner["key"] != "val" {
		t.Fatalf("expected 'val', got %v", inner["key"])
	}
}

func TestPackPtrLen(t *testing.T) {
	packed := PackPtrLen(0x1000, 256)
	ptr := uint32(packed >> 32)
	length := uint32(packed)
	if ptr != 0x1000 {
		t.Fatalf("expected ptr 0x1000, got 0x%x", ptr)
	}
	if length != 256 {
		t.Fatalf("expected len 256, got %d", length)
	}
}

func TestPackPtrLen_Zero(t *testing.T) {
	packed := PackPtrLen(0, 0)
	if packed != 0 {
		t.Fatalf("expected 0, got %d", packed)
	}
}

func TestMultipleHandlers(t *testing.T) {
	p := NewPlugin(PluginMeta{Name: "multi"})

	var calls []string
	p.On("build.started", func(ev Event) *Response {
		calls = append(calls, "started")
		return nil
	})
	p.On("build.finished", func(ev Event) *Response {
		calls = append(calls, "finished")
		return &Response{Payload: map[string]any{"done": true}}
	})

	HandleEvent([]byte(`{"type":"build.started","payload":{}}`))
	out := HandleEvent([]byte(`{"type":"build.finished","payload":{}}`))

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if out == nil {
		t.Fatal("expected response from build.finished")
	}
}

func TestEventPayloadAccess(t *testing.T) {
	p := NewPlugin(PluginMeta{Name: "test"})

	var publicDir string
	p.On("build.finished", func(ev Event) *Response {
		publicDir = GetString(ev.Payload, "config", "public_dir")
		return nil
	})

	input := `{"type":"build.finished","payload":{"config":{"public_dir":"/abs/public"}}}`
	HandleEvent([]byte(input))

	if publicDir != "/abs/public" {
		t.Fatalf("expected '/abs/public', got %q", publicDir)
	}
}
