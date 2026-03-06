package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all global key bindings.
type KeyMap struct {
	Quit          key.Binding
	ToggleSidebar key.Binding
	ClearOutput   key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	Submit        key.Binding
	ToggleServe   key.Binding
	ToggleAPI     key.Binding
	ToggleLogs    key.Binding
	ToggleConfig  key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "sidebar"),
		),
		ClearOutput: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run command"),
		),
		ToggleServe: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("F5", "toggle serve"),
		),
		ToggleAPI: key.NewBinding(
			key.WithKeys("f6"),
			key.WithHelp("F6", "toggle API"),
		),
		ToggleLogs: key.NewBinding(
			key.WithKeys("f7"),
			key.WithHelp("F7", "toggle logs"),
		),
		ToggleConfig: key.NewBinding(
			key.WithKeys("f8"),
			key.WithHelp("F8", "config editor"),
		),
	}
}

var keys = DefaultKeyMap()
