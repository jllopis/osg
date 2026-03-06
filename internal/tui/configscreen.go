package tui

import (
	"fmt"
	"strings"

	"osg/internal/config"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// ConfigScreen is the full-screen modal config editor.
type ConfigScreen struct {
	sections   []config.ConfigSection
	sectionIdx int    // selected section in left panel
	fieldIdx   int    // selected field in right panel
	focusPanel string // "sections" | "fields"
	editing    bool   // true when a field is in edit mode

	// Values loaded from the YAML file.
	configNode *yaml.Node
	configPath string

	// Dirty tracking.
	dirtyFields map[string]bool

	// Field editors.
	fieldEditor FieldEditor

	// Confirmation dialog for unsaved changes.
	confirmQuit bool

	// Dimensions.
	width  int
	height int
}

// NewConfigScreen creates a config editor screen.
// It loads the YAML node tree from the given config path.
func NewConfigScreen(configPath string) (*ConfigScreen, error) {
	node, err := config.LoadNode(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	cs := &ConfigScreen{
		sections:    config.ConfigSchema(),
		focusPanel:  "sections",
		configNode:  node,
		configPath:  configPath,
		dirtyFields: make(map[string]bool),
		fieldEditor: NewFieldEditor(),
	}
	return cs, nil
}

// IsDirty returns true if any field has been modified.
func (cs *ConfigScreen) IsDirty() bool {
	return len(cs.dirtyFields) > 0
}

// ConfirmQuitVisible returns true when the unsaved-changes dialog is shown.
func (cs *ConfigScreen) ConfirmQuitVisible() bool {
	return cs.confirmQuit
}

// Editing returns true when a field is being edited.
func (cs *ConfigScreen) Editing() bool {
	return cs.editing
}

// Save writes the current yaml.Node tree to disk.
func (cs *ConfigScreen) Save() error {
	if err := config.SaveNode(cs.configPath, cs.configNode); err != nil {
		return err
	}
	cs.dirtyFields = make(map[string]bool)
	return nil
}

// Resize updates the screen dimensions.
func (cs *ConfigScreen) Resize(w, h int) {
	cs.width = w
	cs.height = h
}

// ---- Navigation ----

// SectionCount returns how many sections there are.
func (cs *ConfigScreen) SectionCount() int { return len(cs.sections) }

// SectionIdx returns the current section index.
func (cs *ConfigScreen) SectionIdx() int { return cs.sectionIdx }

// FieldIdx returns the current field index.
func (cs *ConfigScreen) FieldIdx() int { return cs.fieldIdx }

// FocusPanel returns "sections" or "fields".
func (cs *ConfigScreen) FocusPanel() string { return cs.focusPanel }

// currentSection returns the currently selected section.
func (cs *ConfigScreen) currentSection() config.ConfigSection {
	if cs.sectionIdx >= 0 && cs.sectionIdx < len(cs.sections) {
		return cs.sections[cs.sectionIdx]
	}
	return config.ConfigSection{}
}

// currentField returns the currently selected field.
func (cs *ConfigScreen) currentField() (config.ConfigField, bool) {
	sec := cs.currentSection()
	if cs.fieldIdx >= 0 && cs.fieldIdx < len(sec.Fields) {
		return sec.Fields[cs.fieldIdx], true
	}
	return config.ConfigField{}, false
}

// fieldValue returns the current value for a field key from the YAML tree.
func (cs *ConfigScreen) fieldValue(key string) string {
	val, ok := config.GetNodeValue(cs.configNode, key)
	if !ok {
		return ""
	}
	return val
}

// GetValue returns the current value for a dotted config key, e.g. "site_title".
// Used by the TUI to refresh sidebar options after a config save.
func (cs *ConfigScreen) GetValue(key string) string {
	return cs.fieldValue(key)
}

// GetSequence returns the raw sequence items for a dotted config key.
func (cs *ConfigScreen) GetSequence(key string) ([]string, bool) {
	return config.GetNodeSequence(cs.configNode, key)
}

// MoveSection moves the section selection by delta.
func (cs *ConfigScreen) MoveSection(delta int) {
	n := len(cs.sections)
	if n == 0 {
		return
	}
	cs.sectionIdx += delta
	if cs.sectionIdx < 0 {
		cs.sectionIdx = 0
	}
	if cs.sectionIdx >= n {
		cs.sectionIdx = n - 1
	}
	// Reset field index when changing sections.
	cs.fieldIdx = 0
}

// MoveField moves the field selection by delta.
func (cs *ConfigScreen) MoveField(delta int) {
	sec := cs.currentSection()
	n := len(sec.Fields)
	if n == 0 {
		return
	}
	cs.fieldIdx += delta
	if cs.fieldIdx < 0 {
		cs.fieldIdx = 0
	}
	if cs.fieldIdx >= n {
		cs.fieldIdx = n - 1
	}
}

// SwitchPanel toggles focus between sections and fields.
func (cs *ConfigScreen) SwitchPanel() {
	if cs.focusPanel == "sections" {
		cs.focusPanel = "fields"
	} else {
		cs.focusPanel = "sections"
	}
}

// ---- Editing ----

// StartEdit begins editing the currently selected field.
func (cs *ConfigScreen) StartEdit() {
	field, ok := cs.currentField()
	if !ok {
		return
	}
	cs.editing = true
	val := cs.fieldValue(field.Key)
	cs.fieldEditor.Start(field, val)
}

// CancelEdit stops editing without saving changes.
func (cs *ConfigScreen) CancelEdit() {
	cs.editing = false
	cs.fieldEditor.Reset()
}

// ConfirmEdit applies the edit and marks the field dirty.
func (cs *ConfigScreen) ConfirmEdit() {
	field, ok := cs.currentField()
	if !ok {
		cs.editing = false
		return
	}

	newVal := cs.fieldEditor.Value()
	oldVal := cs.fieldValue(field.Key)

	if newVal != oldVal {
		// Apply to the YAML tree.
		switch field.Type {
		case config.FieldStringList, config.FieldIntList:
			items := splitListValue(newVal)
			config.SetNodeSequence(cs.configNode, field.Key, items)
		default:
			config.SetNodeValue(cs.configNode, field.Key, newVal)
		}
		cs.dirtyFields[field.Key] = true
	}

	cs.editing = false
	cs.fieldEditor.Reset()
}

// ToggleBool toggles a boolean field without entering edit mode.
func (cs *ConfigScreen) ToggleBool() {
	field, ok := cs.currentField()
	if !ok || field.Type != config.FieldBool {
		return
	}
	val := cs.fieldValue(field.Key)
	newVal := "true"
	if val == "true" {
		newVal = "false"
	}
	config.SetNodeValue(cs.configNode, field.Key, newVal)
	cs.dirtyFields[field.Key] = true
}

// ---- List editing helpers ----

// AddListItem adds an empty item to a StringList/IntList field.
func (cs *ConfigScreen) AddListItem() {
	field, ok := cs.currentField()
	if !ok {
		return
	}
	if field.Type != config.FieldStringList && field.Type != config.FieldIntList {
		return
	}
	items, _ := config.GetNodeSequence(cs.configNode, field.Key)
	items = append(items, "")
	config.SetNodeSequence(cs.configNode, field.Key, items)
	cs.dirtyFields[field.Key] = true
}

// DeleteListItem deletes the last item from a StringList/IntList field.
func (cs *ConfigScreen) DeleteListItem() {
	field, ok := cs.currentField()
	if !ok {
		return
	}
	if field.Type != config.FieldStringList && field.Type != config.FieldIntList {
		return
	}
	items, ok := config.GetNodeSequence(cs.configNode, field.Key)
	if !ok || len(items) == 0 {
		return
	}
	items = items[:len(items)-1]
	config.SetNodeSequence(cs.configNode, field.Key, items)
	cs.dirtyFields[field.Key] = true
}

// ---- Confirm dialog ----

// ShowConfirmQuit shows the unsaved changes dialog.
func (cs *ConfigScreen) ShowConfirmQuit() {
	cs.confirmQuit = true
}

// HideConfirmQuit hides the dialog.
func (cs *ConfigScreen) HideConfirmQuit() {
	cs.confirmQuit = false
}

// ---- Rendering ----

// View renders the full config screen.
func (cs *ConfigScreen) View() string {
	if cs.width == 0 || cs.height == 0 {
		return ""
	}

	header := cs.renderHeader()
	hintBar := cs.renderHintBar()

	bodyHeight := cs.height - lipgloss.Height(header) - lipgloss.Height(hintBar)
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	// Left panel: section list.
	sectionListWidth := 20
	if sectionListWidth > cs.width/3 {
		sectionListWidth = cs.width / 3
	}
	sectionList := cs.renderSectionList(sectionListWidth, bodyHeight)

	// Right panel: fields.
	fieldPanelWidth := cs.width - sectionListWidth - 3 // border/padding
	if fieldPanelWidth < 20 {
		fieldPanelWidth = 20
	}
	fieldPanel := cs.renderFieldPanel(fieldPanelWidth, bodyHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sectionList, fieldPanel)

	screen := lipgloss.JoinVertical(lipgloss.Left, header, body, hintBar)

	// Overlay the confirm dialog if active.
	if cs.confirmQuit {
		dialog := cs.renderConfirmDialog()
		screen = overlayCenter(screen, dialog, cs.width, cs.height)
	}

	return screen
}

func (cs *ConfigScreen) renderHeader() string {
	title := "Configuration"
	right := ""
	if cs.IsDirty() {
		right = cfgDirtyStyle.Render("[* modified]")
	}
	gap := cs.width - lipgloss.Width(title) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	content := title + strings.Repeat(" ", gap) + right
	return cfgHeaderStyle.Width(cs.width).Render(content)
}

func (cs *ConfigScreen) renderHintBar() string {
	var hints []string
	switch {
	case cs.editing:
		hints = append(hints,
			hintKeyStyle.Render("Enter")+hintDescStyle.Render(" confirm"),
			hintKeyStyle.Render("Esc")+hintDescStyle.Render(" cancel"),
		)
	case cs.confirmQuit:
		hints = append(hints,
			hintKeyStyle.Render("y")+hintDescStyle.Render(" save+quit"),
			hintKeyStyle.Render("n")+hintDescStyle.Render(" quit"),
			hintKeyStyle.Render("Esc")+hintDescStyle.Render(" cancel"),
		)
	default:
		hints = append(hints,
			hintKeyStyle.Render("Ctrl+S")+hintDescStyle.Render(" save"),
			hintKeyStyle.Render("Esc")+hintDescStyle.Render(" back"),
			hintKeyStyle.Render("Enter")+hintDescStyle.Render(" edit"),
			hintKeyStyle.Render("Tab")+hintDescStyle.Render(" panel"),
			hintKeyStyle.Render("^/v")+hintDescStyle.Render(" nav"),
		)
		if cs.focusPanel == "fields" {
			field, ok := cs.currentField()
			if ok && field.Type == config.FieldBool {
				hints = append(hints, hintKeyStyle.Render("Space")+hintDescStyle.Render(" toggle"))
			}
			if ok && (field.Type == config.FieldStringList || field.Type == config.FieldIntList) {
				hints = append(hints,
					hintKeyStyle.Render("a")+hintDescStyle.Render(" add"),
					hintKeyStyle.Render("d")+hintDescStyle.Render(" del"),
				)
			}
		}
	}
	return cfgHintBarStyle.Width(cs.width).Render(strings.Join(hints, "  "))
}

func (cs *ConfigScreen) renderSectionList(width, height int) string {
	var lines []string
	for i, sec := range cs.sections {
		name := truncateStr(sec.Name, width-4)
		if i == cs.sectionIdx {
			prefix := "  "
			if cs.focusPanel == "sections" {
				prefix = "> "
			}
			lines = append(lines, cfgSectionActiveStyle.Width(width-2).Render(prefix+name))
		} else {
			lines = append(lines, cfgSectionNormalStyle.Render("  "+name))
		}
	}

	// Pad to fill height.
	for len(lines) < height-2 {
		lines = append(lines, "")
	}
	// If too many lines, scroll to keep selected visible.
	if len(lines) > height-2 {
		lines = scrollWindow(lines, cs.sectionIdx, height-2)
	}

	content := strings.Join(lines, "\n")
	return cfgSectionListStyle.Width(width).Height(height - 2).Render(content)
}

func (cs *ConfigScreen) renderFieldPanel(width, height int) string {
	sec := cs.currentSection()
	if len(sec.Fields) == 0 {
		return cfgFieldPanelStyle.Width(width).Height(height).Render(
			cfgFieldDescStyle.Render("No fields in this section"))
	}

	var lines []string
	// Section title + description.
	lines = append(lines, cfgStructHeaderStyle.Render(sec.Name))
	if sec.Description != "" {
		lines = append(lines, cfgFieldDescStyle.Render(sec.Description))
	}
	lines = append(lines, "")

	for i, field := range sec.Fields {
		fieldLines := cs.renderField(field, i, width-4)
		lines = append(lines, fieldLines...)
		lines = append(lines, "") // blank separator
	}

	// If editing, append the editor view.
	if cs.editing {
		lines = append(lines, cs.fieldEditor.View(width-4)...)
	}

	// Pad or scroll.
	maxLines := height - 2
	if len(lines) > maxLines && maxLines > 0 {
		// Find the line offset for the selected field.
		targetLine := cs.fieldLineOffset(sec.Fields, cs.fieldIdx)
		lines = scrollWindow(lines, targetLine, maxLines)
	}
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return cfgFieldPanelStyle.Width(width).Height(height).Render(content)
}

func (cs *ConfigScreen) renderField(field config.ConfigField, idx int, maxWidth int) []string {
	var lines []string

	// Label and value on the same line.
	label := field.Label
	val := cs.fieldValue(field.Key)
	if val == "" {
		val = cfgFieldDescStyle.Render("(empty)")
	} else if field.Sensitive {
		val = cfgSensitiveStyle.Render(strings.Repeat("*", min(len(val), 12)))
	}

	// Dirty indicator.
	dirty := ""
	if cs.dirtyFields[field.Key] {
		dirty = cfgDirtyStyle.Render(" *")
	}

	// Bool fields show a toggle marker.
	if field.Type == config.FieldBool {
		boolVal := cs.fieldValue(field.Key)
		if boolVal == "true" {
			val = cfgFieldValueStyle.Foreground(nordGreen).Render("[x] true")
		} else {
			val = cfgFieldValueStyle.Render("[ ] false")
		}
	}

	isSelected := idx == cs.fieldIdx && cs.focusPanel == "fields"
	labelStr := cfgFieldLabelStyle.Render(label)
	valStr := cfgFieldValueStyle.Render(val) + dirty

	line := fmt.Sprintf("  %s  %s", labelStr, valStr)
	if isSelected && !cs.editing {
		line = cfgFieldSelectedStyle.Width(maxWidth).Render(line)
	}
	lines = append(lines, line)

	// Description below.
	if field.Description != "" {
		desc := "    " + field.Description
		if len(desc) > maxWidth {
			desc = desc[:maxWidth-1] + "~"
		}
		lines = append(lines, cfgFieldDescStyle.Render(desc))
	}

	return lines
}

// fieldLineOffset computes the approximate line index where field[idx] starts.
func (cs *ConfigScreen) fieldLineOffset(fields []config.ConfigField, idx int) int {
	// Header is 3 lines (title + desc + blank), each field is ~3 lines (label+desc+blank).
	offset := 3
	for i := 0; i < idx && i < len(fields); i++ {
		offset += 2 // label + desc
		if fields[i].Description != "" {
			offset++
		}
	}
	return offset
}

func (cs *ConfigScreen) renderConfirmDialog() string {
	msg := "Unsaved changes. Save before closing?\n\n" +
		hintKeyStyle.Render("y") + " save and close   " +
		hintKeyStyle.Render("n") + " discard and close   " +
		hintKeyStyle.Render("Esc") + " cancel"
	return cfgConfirmStyle.Render(msg)
}

// ---- Utility ----

// overlayCenter places an overlay string centered on a background.
func overlayCenter(bg, overlay string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	ovLines := strings.Split(overlay, "\n")

	ovWidth := lipgloss.Width(overlay)
	ovHeight := len(ovLines)

	startRow := (height - ovHeight) / 2
	startCol := (width - ovWidth) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	// Pad bg lines to height if needed.
	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	for i, ovLine := range ovLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		// Ensure background line is wide enough.
		bgLine := bgLines[row]
		for lipgloss.Width(bgLine) < width {
			bgLine += " "
		}
		// Simple character overlay — replace substring.
		runes := []rune(bgLine)
		ovRunes := []rune(ovLine)
		for j, r := range ovRunes {
			pos := startCol + j
			if pos < len(runes) {
				runes[pos] = r
			}
		}
		bgLines[row] = string(runes)
	}

	return strings.Join(bgLines, "\n")
}

// scrollWindow returns a slice of lines around the target index.
func scrollWindow(lines []string, target, windowSize int) []string {
	if len(lines) <= windowSize {
		return lines
	}
	start := target - windowSize/2
	if start < 0 {
		start = 0
	}
	end := start + windowSize
	if end > len(lines) {
		end = len(lines)
		start = end - windowSize
		if start < 0 {
			start = 0
		}
	}
	return lines[start:end]
}

// truncateStr truncates a string to maxLen, adding "~" if truncated.
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "~"
	}
	return s[:maxLen-1] + "~"
}

// splitListValue splits a comma-separated value string into items.
func splitListValue(val string) []string {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
