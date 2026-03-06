package tui

import (
	"fmt"
	"strconv"
	"strings"

	"osg/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
)

// FieldEditor handles inline editing for different config field types.
type FieldEditor struct {
	active    bool
	field     config.ConfigField
	textInput textinput.Model
	original  string // value before editing started
}

// NewFieldEditor creates a new field editor.
func NewFieldEditor() FieldEditor {
	ti := textinput.New()
	ti.CharLimit = 512
	ti.Width = 40
	return FieldEditor{
		textInput: ti,
	}
}

// Start begins editing a field with the given current value.
func (fe *FieldEditor) Start(field config.ConfigField, currentValue string) {
	fe.active = true
	fe.field = field
	fe.original = currentValue

	switch field.Type {
	case config.FieldBool:
		// Bools use space toggle, not the text editor.
		// But if somehow Start is called, just store the value.
		fe.textInput.SetValue(currentValue)
	case config.FieldStringList, config.FieldIntList:
		// Show comma-separated for editing.
		fe.textInput.SetValue(currentValue)
		fe.textInput.Placeholder = "comma-separated values"
	case config.FieldStringMap:
		fe.textInput.SetValue(currentValue)
		fe.textInput.Placeholder = "key=value, key=value"
	case config.FieldInt:
		fe.textInput.SetValue(currentValue)
		fe.textInput.Placeholder = "integer value"
	default:
		fe.textInput.SetValue(currentValue)
		fe.textInput.Placeholder = ""
	}

	if field.Sensitive {
		fe.textInput.EchoMode = textinput.EchoPassword
	} else {
		fe.textInput.EchoMode = textinput.EchoNormal
	}

	fe.textInput.Focus()
	fe.textInput.CursorEnd()
}

// Reset clears the editor state.
func (fe *FieldEditor) Reset() {
	fe.active = false
	fe.field = config.ConfigField{}
	fe.original = ""
	fe.textInput.SetValue("")
	fe.textInput.Blur()
	fe.textInput.EchoMode = textinput.EchoNormal
}

// Value returns the current value from the editor.
func (fe *FieldEditor) Value() string {
	return fe.textInput.Value()
}

// Validate checks if the current value is valid for the field type.
func (fe *FieldEditor) Validate() error {
	val := fe.textInput.Value()
	switch fe.field.Type {
	case config.FieldInt:
		if val != "" {
			if _, err := strconv.Atoi(strings.TrimSpace(val)); err != nil {
				return fmt.Errorf("must be an integer")
			}
		}
	case config.FieldIntList:
		for _, item := range splitListValue(val) {
			if _, err := strconv.Atoi(strings.TrimSpace(item)); err != nil {
				return fmt.Errorf("%q is not an integer", item)
			}
		}
	case config.FieldBool:
		v := strings.TrimSpace(val)
		if v != "true" && v != "false" && v != "" {
			return fmt.Errorf("must be true or false")
		}
	}

	// Check against Options if defined.
	if len(fe.field.Options) > 0 && val != "" {
		found := false
		for _, opt := range fe.field.Options {
			if opt == val {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("must be one of: %s", strings.Join(fe.field.Options, ", "))
		}
	}

	return nil
}

// HandleKey processes a key event during editing.
// Returns: (value changed bool, done editing bool).
func (fe *FieldEditor) HandleKey(keyStr string) (changed bool, done bool) {
	switch keyStr {
	case "enter":
		if err := fe.Validate(); err != nil {
			// Invalid — stay in edit mode (the error will be visible in View).
			return false, false
		}
		return fe.Value() != fe.original, true
	case "esc":
		// Cancel edit.
		fe.textInput.SetValue(fe.original)
		return false, true
	default:
		// Forward to text input.
		fe.textInput, _ = fe.textInput.Update(keyToMsg(keyStr))
		return false, false
	}
}

// View renders the editor UI lines.
func (fe *FieldEditor) View(maxWidth int) []string {
	if !fe.active {
		return nil
	}

	var lines []string
	lines = append(lines, cfgFieldEditingStyle.Render(fmt.Sprintf("  Editing: %s", fe.field.Label)))

	// Show options hint if available.
	if len(fe.field.Options) > 0 {
		lines = append(lines, cfgFieldDescStyle.Render(fmt.Sprintf("    Options: %s", strings.Join(fe.field.Options, ", "))))
	}

	// Text input.
	inputView := "  " + fe.textInput.View()
	lines = append(lines, inputView)

	// Validation error.
	if err := fe.Validate(); err != nil {
		lines = append(lines, "  "+lipglossRenderError(err.Error()))
	}

	return lines
}

// UpdateTextInput forwards a tea.Msg to the text input.
func (fe *FieldEditor) UpdateTextInput(msg interface{}) {
	fe.textInput, _ = fe.textInput.Update(msg)
}

// lipglossRenderError renders an error string in red.
func lipglossRenderError(s string) string {
	return cfgDirtyStyle.Render(s)
}

// keyToMsg creates a minimal tea.KeyMsg for forwarding to the text input.
// This is used to forward typed characters during editing.
func keyToMsg(keyStr string) interface{} {
	// For the text input, we return the key string directly as a tea.KeyMsg.
	// The bubbles textinput accepts tea.KeyMsg or tea.Msg.
	// We rely on the parent Update to forward the actual msg.
	return nil
}
