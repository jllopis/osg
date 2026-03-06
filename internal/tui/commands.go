package tui

import (
	"strings"
)

// slashCommand defines a single slash command in the registry.
type slashCommand struct {
	Name    string   // canonical name e.g. "/build"
	Aliases []string // bare aliases e.g. "b"
	Hint    string   // short description
	Args    string   // argument placeholder, e.g. "<name>"
}

// commandRegistry is the ordered list of available commands.
var commandRegistry = []slashCommand{
	{Name: "/init", Aliases: []string{"i"}, Hint: "Initialize project"},
	{Name: "/update", Aliases: []string{"update-content", "a"}, Hint: "Sync vault content"},
	{Name: "/build", Aliases: []string{"b"}, Hint: "Build static site"},
	{Name: "/serve", Aliases: []string{"s"}, Hint: "Start preview server", Args: "[--api]"},
	{Name: "/api", Hint: "Toggle standalone API server"},
	{Name: "/new", Hint: "Create a new post", Args: "<title>"},
	{Name: "/stop", Hint: "Stop a service", Args: "serve|api"},
	{Name: "/logs", Hint: "Toggle log panel"},
	{Name: "/config", Hint: "Open config editor"},
	{Name: "/doctor", Hint: "Run diagnostics"},
	{Name: "/next", Aliases: []string{"n"}, Hint: "Run next workflow step"},
	{Name: "/theme", Hint: "Theme commands", Args: "init|list"},
	{Name: "/plugin", Hint: "Plugin management", Args: "enable|disable|toggle|list|install|init|search|update"},
	{Name: "/version", Aliases: []string{"v"}, Hint: "Show version info"},
	{Name: "/clear", Hint: "Clear output"},
	{Name: "/help", Aliases: []string{"h"}, Hint: "Show help"},
	{Name: "/quit", Aliases: []string{"exit", "q"}, Hint: "Exit OSG"},
}

// matchCommands returns commands whose name or aliases start with the given prefix.
// If prefix is empty or just "/", all commands are returned.
func matchCommands(prefix string) []slashCommand {
	prefix = strings.TrimSpace(prefix)
	if prefix == "/" || prefix == "" {
		return commandRegistry
	}

	// Normalize: ensure prefix starts with /
	search := prefix
	if !strings.HasPrefix(search, "/") {
		search = "/" + search
	}
	search = strings.ToLower(search)

	var matches []slashCommand
	for _, cmd := range commandRegistry {
		if strings.HasPrefix(strings.ToLower(cmd.Name), search) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// resolveCommand normalizes user input into a canonical command name and arguments.
// It accepts both "/build" and "build" forms.
func resolveCommand(raw string) (cmd string, args []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	normalized := strings.TrimPrefix(raw, "/")
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return "", nil
	}
	cmd = strings.ToLower(fields[0])
	args = fields[1:]

	// Check canonical names (strip /)
	for _, entry := range commandRegistry {
		canonical := strings.TrimPrefix(entry.Name, "/")
		if cmd == canonical {
			return canonical, args
		}
		for _, alias := range entry.Aliases {
			if cmd == strings.ToLower(alias) {
				return canonical, args
			}
		}
	}
	return cmd, args
}

// helpText returns a formatted help string listing all commands.
func helpText() string {
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, cmd := range commandRegistry {
		line := "  " + cmd.Name
		if cmd.Args != "" {
			line += " " + cmd.Args
		}
		if len(cmd.Aliases) > 0 {
			line += "  (aliases: " + strings.Join(cmd.Aliases, ", ") + ")"
		}
		line += "  - " + cmd.Hint
		b.WriteString(line + "\n")
	}
	b.WriteString("\nBare commands work too: build = /build")
	return b.String()
}
