package tui

import (
	"os"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

// testConfigYAML is minimal YAML for testing.
const testConfigYAML = `# Site config
site_title: Test Site
site_description: A test site
base_url: https://example.com
default_language: es
color_scheme: auto
include_drafts: false
minify: true
plugins_enabled:
  - search
image_widths:
  - 640
  - 1200
ai:
  provider: gemini
  model: gemini-3-flash-preview
  timeout: 30
logging:
  level: info
  format: json
social:
  x: testhandle
  github: testuser
interactions:
  enabled: false
  listen: ":8090"
  cors_origins:
    - http://localhost:1313
`

func writeTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(testConfigYAML), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}

// ---- Construction ----

func TestNewConfigScreen(t *testing.T) {
	path := writeTestConfig(t)
	cs, err := NewConfigScreen(path)
	if err != nil {
		t.Fatalf("NewConfigScreen: %v", err)
	}
	if cs.SectionCount() != len(config.ConfigSchema()) {
		t.Errorf("SectionCount = %d, want %d", cs.SectionCount(), len(config.ConfigSchema()))
	}
	if cs.SectionIdx() != 0 {
		t.Errorf("SectionIdx = %d, want 0", cs.SectionIdx())
	}
	if cs.FocusPanel() != "sections" {
		t.Errorf("FocusPanel = %q, want 'sections'", cs.FocusPanel())
	}
	if cs.IsDirty() {
		t.Error("should not be dirty initially")
	}
}

func TestNewConfigScreen_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")
	cs, err := NewConfigScreen(path)
	if err != nil {
		t.Fatalf("NewConfigScreen should not fail for missing file: %v", err)
	}
	// Should get an empty doc with empty values.
	val := cs.fieldValue("site_title")
	if val != "" {
		t.Errorf("expected empty for missing file, got %q", val)
	}
}

// ---- Navigation ----

func TestMoveSection(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.MoveSection(1)
	if cs.SectionIdx() != 1 {
		t.Errorf("SectionIdx after +1 = %d, want 1", cs.SectionIdx())
	}

	cs.MoveSection(3)
	if cs.SectionIdx() != 4 {
		t.Errorf("SectionIdx after +3 = %d, want 4", cs.SectionIdx())
	}

	cs.MoveSection(-10)
	if cs.SectionIdx() != 0 {
		t.Errorf("SectionIdx after -10 = %d, want 0 (clamped)", cs.SectionIdx())
	}

	// Move past end.
	cs.MoveSection(1000)
	if cs.SectionIdx() != cs.SectionCount()-1 {
		t.Errorf("SectionIdx after +1000 = %d, want %d", cs.SectionIdx(), cs.SectionCount()-1)
	}
}

func TestMoveSection_ResetsFieldIdx(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.SwitchPanel()
	cs.MoveField(2) // move to field 2 in first section
	if cs.FieldIdx() != 2 {
		t.Fatalf("expected fieldIdx 2, got %d", cs.FieldIdx())
	}

	cs.SwitchPanel()
	cs.MoveSection(1)
	if cs.FieldIdx() != 0 {
		t.Errorf("fieldIdx should reset to 0 after section change, got %d", cs.FieldIdx())
	}
}

func TestMoveField(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.SwitchPanel()
	cs.MoveField(1)
	if cs.FieldIdx() != 1 {
		t.Errorf("FieldIdx = %d, want 1", cs.FieldIdx())
	}

	cs.MoveField(-5)
	if cs.FieldIdx() != 0 {
		t.Errorf("FieldIdx clamped = %d, want 0", cs.FieldIdx())
	}
}

func TestSwitchPanel(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	if cs.FocusPanel() != "sections" {
		t.Errorf("initial panel = %q", cs.FocusPanel())
	}
	cs.SwitchPanel()
	if cs.FocusPanel() != "fields" {
		t.Errorf("after switch = %q, want 'fields'", cs.FocusPanel())
	}
	cs.SwitchPanel()
	if cs.FocusPanel() != "sections" {
		t.Errorf("after second switch = %q, want 'sections'", cs.FocusPanel())
	}
}

// ---- Reading values ----

func TestFieldValue(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	tests := []struct {
		key  string
		want string
	}{
		{"site_title", "Test Site"},
		{"base_url", "https://example.com"},
		{"default_language", "es"},
		{"include_drafts", "false"},
		{"ai.provider", "gemini"},
		{"ai.timeout", "30"},
		{"logging.level", "info"},
		{"interactions.enabled", "false"},
		{"nonexistent_key", ""},
	}
	for _, tt := range tests {
		got := cs.fieldValue(tt.key)
		if got != tt.want {
			t.Errorf("fieldValue(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestFieldValue_List(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	val := cs.fieldValue("plugins_enabled")
	if val != "search" {
		t.Errorf("plugins_enabled = %q, want 'search'", val)
	}

	val = cs.fieldValue("image_widths")
	if val != "640, 1200" {
		t.Errorf("image_widths = %q, want '640, 1200'", val)
	}
}

// ---- Editing ----

func TestStartAndConfirmEdit(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	// Navigate to site_title field (section 0, field 0).
	cs.SwitchPanel()
	cs.StartEdit()
	if !cs.Editing() {
		t.Fatal("should be editing")
	}

	// Change the value.
	cs.fieldEditor.textInput.SetValue("New Title")
	cs.ConfirmEdit()

	if cs.Editing() {
		t.Error("should not be editing after confirm")
	}

	// Check value is updated.
	val := cs.fieldValue("site_title")
	if val != "New Title" {
		t.Errorf("site_title = %q, want 'New Title'", val)
	}

	if !cs.IsDirty() {
		t.Error("should be dirty after edit")
	}
	if !cs.dirtyFields["site_title"] {
		t.Error("site_title should be in dirty fields")
	}
}

func TestCancelEdit(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.SwitchPanel()
	cs.StartEdit()
	cs.fieldEditor.textInput.SetValue("Changed")
	cs.CancelEdit()

	if cs.Editing() {
		t.Error("should not be editing after cancel")
	}

	val := cs.fieldValue("site_title")
	if val != "Test Site" {
		t.Errorf("value should be unchanged, got %q", val)
	}
	if cs.IsDirty() {
		t.Error("should not be dirty after cancel")
	}
}

func TestEditSameValue_NotDirty(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.SwitchPanel()
	cs.StartEdit()
	// Don't change the value — confirm with same text.
	cs.ConfirmEdit()

	if cs.IsDirty() {
		t.Error("should not be dirty when value unchanged")
	}
}

// ---- Bool toggle ----

func TestToggleBool(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	// Navigate to Content section (idx 2) -> include_drafts (idx 3)
	cs.MoveSection(2)
	cs.SwitchPanel()
	cs.MoveField(3) // include_drafts

	field, ok := cs.currentField()
	if !ok || field.Key != "include_drafts" {
		t.Fatalf("expected include_drafts field, got %v", field.Key)
	}

	val := cs.fieldValue("include_drafts")
	if val != "false" {
		t.Fatalf("initial value = %q, want 'false'", val)
	}

	cs.ToggleBool()
	val = cs.fieldValue("include_drafts")
	if val != "true" {
		t.Errorf("after toggle = %q, want 'true'", val)
	}
	if !cs.dirtyFields["include_drafts"] {
		t.Error("should be dirty after toggle")
	}

	cs.ToggleBool()
	val = cs.fieldValue("include_drafts")
	if val != "false" {
		t.Errorf("after second toggle = %q, want 'false'", val)
	}
}

// ---- List operations ----

func TestAddAndDeleteListItem(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	// Navigate to Plugins section (idx 10), plugins_enabled (idx 1).
	cs.MoveSection(10)
	cs.SwitchPanel()
	cs.MoveField(1) // plugins_enabled

	field, ok := cs.currentField()
	if !ok || field.Key != "plugins_enabled" {
		t.Fatalf("expected plugins_enabled, got %v", field.Key)
	}

	cs.AddListItem()
	val := cs.fieldValue("plugins_enabled")
	// Should now be "search, " (the empty item gets ignored in display).
	if !cs.dirtyFields["plugins_enabled"] {
		t.Error("should be dirty after add")
	}

	cs.DeleteListItem()
	val = cs.fieldValue("plugins_enabled")
	if val != "search" {
		t.Errorf("after delete = %q, want 'search'", val)
	}
}

// ---- Save ----

func TestSave(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	// Edit a field.
	cs.SwitchPanel()
	cs.StartEdit()
	cs.fieldEditor.textInput.SetValue("Saved Title")
	cs.ConfirmEdit()

	if !cs.IsDirty() {
		t.Fatal("should be dirty before save")
	}

	if err := cs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if cs.IsDirty() {
		t.Error("should not be dirty after save")
	}

	// Verify the file was actually written with the new value.
	cs2, _ := NewConfigScreen(path)
	val := cs2.fieldValue("site_title")
	if val != "Saved Title" {
		t.Errorf("re-read site_title = %q, want 'Saved Title'", val)
	}
}

func TestSave_PreservesComments(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	cs.SwitchPanel()
	cs.StartEdit()
	cs.fieldEditor.textInput.SetValue("Commented Title")
	cs.ConfirmEdit()
	cs.Save()

	data, _ := os.ReadFile(path)
	content := string(data)
	if !contains(content, "# Site config") {
		t.Error("comment should be preserved after save")
	}
	if !contains(content, "Commented Title") {
		t.Error("new value should be present after save")
	}
}

// ---- Confirm dialog ----

func TestConfirmQuitDialog(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)

	if cs.ConfirmQuitVisible() {
		t.Error("dialog should not be visible initially")
	}

	cs.ShowConfirmQuit()
	if !cs.ConfirmQuitVisible() {
		t.Error("dialog should be visible after show")
	}

	cs.HideConfirmQuit()
	if cs.ConfirmQuitVisible() {
		t.Error("dialog should be hidden after hide")
	}
}

// ---- Rendering ----

func TestView_NonEmpty(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)
	cs.Resize(80, 24)

	view := cs.View()
	if view == "" {
		t.Error("View should not be empty")
	}
	if !contains(view, "Configuration") {
		t.Error("View should contain 'Configuration' header")
	}
	if !contains(view, "Site Identity") {
		t.Error("View should contain first section name")
	}
}

func TestView_ZeroSize(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)
	// Don't resize — size is 0x0.
	if cs.View() != "" {
		t.Error("View with zero size should be empty")
	}
}

func TestView_DirtyIndicator(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)
	cs.Resize(80, 24)

	// Not dirty — no indicator.
	view := cs.View()
	if contains(view, "modified") {
		t.Error("should not show modified indicator when clean")
	}

	// Make dirty.
	cs.SwitchPanel()
	cs.StartEdit()
	cs.fieldEditor.textInput.SetValue("Changed")
	cs.ConfirmEdit()

	view = cs.View()
	if !contains(view, "modified") {
		t.Error("should show modified indicator when dirty")
	}
}

func TestView_ConfirmDialog(t *testing.T) {
	path := writeTestConfig(t)
	cs, _ := NewConfigScreen(path)
	cs.Resize(80, 24)

	cs.ShowConfirmQuit()
	view := cs.View()
	if !contains(view, "Unsaved changes") {
		t.Error("View should show confirm dialog text")
	}
}

// ---- FieldEditor ----

func TestFieldEditor_ValidateInt(t *testing.T) {
	fe := NewFieldEditor()
	fe.Start(config.ConfigField{Key: "test", Type: config.FieldInt}, "30")
	fe.textInput.SetValue("notanumber")
	if err := fe.Validate(); err == nil {
		t.Error("should reject non-integer")
	}
	fe.textInput.SetValue("42")
	if err := fe.Validate(); err != nil {
		t.Errorf("should accept integer: %v", err)
	}
}

func TestFieldEditor_ValidateBool(t *testing.T) {
	fe := NewFieldEditor()
	fe.Start(config.ConfigField{Key: "test", Type: config.FieldBool}, "true")
	fe.textInput.SetValue("maybe")
	if err := fe.Validate(); err == nil {
		t.Error("should reject non-bool")
	}
	fe.textInput.SetValue("false")
	if err := fe.Validate(); err != nil {
		t.Errorf("should accept bool: %v", err)
	}
}

func TestFieldEditor_ValidateOptions(t *testing.T) {
	fe := NewFieldEditor()
	fe.Start(config.ConfigField{
		Key:     "test",
		Type:    config.FieldString,
		Options: []string{"a", "b", "c"},
	}, "a")
	fe.textInput.SetValue("d")
	if err := fe.Validate(); err == nil {
		t.Error("should reject value not in options")
	}
	fe.textInput.SetValue("b")
	if err := fe.Validate(); err != nil {
		t.Errorf("should accept valid option: %v", err)
	}
}

func TestFieldEditor_IntList(t *testing.T) {
	fe := NewFieldEditor()
	fe.Start(config.ConfigField{Key: "test", Type: config.FieldIntList}, "640, 1200")
	if err := fe.Validate(); err != nil {
		t.Errorf("should accept valid int list: %v", err)
	}
	fe.textInput.SetValue("640, abc")
	if err := fe.Validate(); err == nil {
		t.Error("should reject non-integer in list")
	}
}

func TestFieldEditor_Reset(t *testing.T) {
	fe := NewFieldEditor()
	fe.Start(config.ConfigField{Key: "test", Type: config.FieldString}, "hello")
	if !fe.active {
		t.Error("should be active after Start")
	}
	fe.Reset()
	if fe.active {
		t.Error("should not be active after Reset")
	}
	if fe.Value() != "" {
		t.Error("value should be empty after Reset")
	}
}

func TestFieldEditor_View(t *testing.T) {
	fe := NewFieldEditor()
	// Not active — no view.
	lines := fe.View(60)
	if len(lines) != 0 {
		t.Error("inactive editor should produce no view lines")
	}

	fe.Start(config.ConfigField{Key: "test", Label: "Test Field", Type: config.FieldString}, "hello")
	lines = fe.View(60)
	if len(lines) < 2 {
		t.Errorf("active editor should produce at least 2 lines, got %d", len(lines))
	}
}

// ---- Utility ----

func TestScrollWindow(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	// Window fits all.
	result := scrollWindow(lines, 0, 10)
	if len(result) != 8 {
		t.Errorf("should return all when window fits, got %d", len(result))
	}

	// Window smaller than content.
	result = scrollWindow(lines, 4, 3)
	if len(result) != 3 {
		t.Errorf("should return 3 lines, got %d", len(result))
	}

	// Target at beginning.
	result = scrollWindow(lines, 0, 3)
	if result[0] != "a" {
		t.Errorf("first line should be 'a', got %q", result[0])
	}

	// Target at end.
	result = scrollWindow(lines, 7, 3)
	if result[len(result)-1] != "h" {
		t.Errorf("last line should be 'h', got %q", result[len(result)-1])
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		s      string
		max    int
		expect string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel~"},
		{"hello", 1, "~"},
		{"hello", 0, ""},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncateStr(tt.s, tt.max)
		if got != tt.expect {
			t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.expect)
		}
	}
}

func TestSplitListValue(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"  ", 0},
		{"a, b, c", 3},
		{"single", 1},
		{"a,,b", 2},
	}
	for _, tt := range tests {
		got := splitListValue(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitListValue(%q) len = %d, want %d", tt.input, len(got), tt.want)
		}
	}
}

// ---- Integration with Model ----

func TestModelOpenConfigEditor(t *testing.T) {
	path := writeTestConfig(t)
	m := New(Actions{}, Options{ConfigPath: path}, nil)
	m.width = 80
	m.height = 24

	m2, _ := m.openConfigEditor()
	if !m2.configActive {
		t.Error("config should be active after open")
	}
	if m2.configScreen == nil {
		t.Error("configScreen should not be nil")
	}
}

func TestModelCloseConfigEditor(t *testing.T) {
	path := writeTestConfig(t)
	m := New(Actions{}, Options{ConfigPath: path}, nil)
	m.width = 80
	m.height = 24
	m, _ = m.openConfigEditor()

	m.closeConfigEditor()
	if m.configActive {
		t.Error("config should not be active after close")
	}
	if m.configScreen != nil {
		t.Error("configScreen should be nil after close")
	}
}

// contains checks if a string contains a substring.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsStr(s, sub)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
