package tui

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// matchCommands
// ---------------------------------------------------------------------------

func TestMatchCommands(t *testing.T) {
	allCount := len(commandRegistry)

	t.Run("empty prefix returns all commands", func(t *testing.T) {
		matches := matchCommands("")
		if len(matches) != allCount {
			t.Errorf("matchCommands(\"\") returned %d; want %d", len(matches), allCount)
		}
	})

	t.Run("slash only returns all commands", func(t *testing.T) {
		matches := matchCommands("/")
		if len(matches) != allCount {
			t.Errorf("matchCommands(\"/\") returned %d; want %d", len(matches), allCount)
		}
	})

	t.Run("exact match", func(t *testing.T) {
		matches := matchCommands("/build")
		if len(matches) != 1 {
			t.Fatalf("matchCommands(\"/build\") returned %d; want 1", len(matches))
		}
		if matches[0].Name != "/build" {
			t.Errorf("match name = %q; want /build", matches[0].Name)
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		matches := matchCommands("/b")
		if len(matches) != 1 {
			t.Fatalf("matchCommands(\"/b\") returned %d; want 1", len(matches))
		}
		if matches[0].Name != "/build" {
			t.Errorf("match name = %q; want /build", matches[0].Name)
		}
	})

	t.Run("prefix without slash", func(t *testing.T) {
		matches := matchCommands("bu")
		if len(matches) != 1 {
			t.Fatalf("matchCommands(\"bu\") returned %d; want 1", len(matches))
		}
		if matches[0].Name != "/build" {
			t.Errorf("match name = %q; want /build", matches[0].Name)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		matches := matchCommands("/BUILD")
		if len(matches) != 1 {
			t.Fatalf("matchCommands(\"/BUILD\") returned %d; want 1", len(matches))
		}
	})

	t.Run("no match", func(t *testing.T) {
		matches := matchCommands("/xyz")
		if len(matches) != 0 {
			t.Errorf("matchCommands(\"/xyz\") returned %d; want 0", len(matches))
		}
	})

	t.Run("multiple matches", func(t *testing.T) {
		// Both /stop and /serve start with /s
		matches := matchCommands("/s")
		if len(matches) < 2 {
			t.Errorf("matchCommands(\"/s\") returned %d; want >= 2", len(matches))
		}
	})
}

// ---------------------------------------------------------------------------
// resolveCommand
// ---------------------------------------------------------------------------

func TestResolveCommand(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		cmd, args := resolveCommand("")
		if cmd != "" || args != nil {
			t.Errorf("resolveCommand(\"\") = (%q, %v); want (\"\", nil)", cmd, args)
		}
	})

	t.Run("canonical name with slash", func(t *testing.T) {
		cmd, args := resolveCommand("/build")
		if cmd != "build" {
			t.Errorf("cmd = %q; want \"build\"", cmd)
		}
		if len(args) != 0 {
			t.Errorf("args = %v; want empty", args)
		}
	})

	t.Run("canonical name without slash", func(t *testing.T) {
		cmd, _ := resolveCommand("build")
		if cmd != "build" {
			t.Errorf("cmd = %q; want \"build\"", cmd)
		}
	})

	t.Run("alias resolves to canonical", func(t *testing.T) {
		cmd, _ := resolveCommand("b")
		if cmd != "build" {
			t.Errorf("alias 'b' resolved to %q; want \"build\"", cmd)
		}
	})

	t.Run("alias with slash", func(t *testing.T) {
		cmd, _ := resolveCommand("/b")
		if cmd != "build" {
			t.Errorf("alias '/b' resolved to %q; want \"build\"", cmd)
		}
	})

	t.Run("alias for quit", func(t *testing.T) {
		cmd, _ := resolveCommand("q")
		if cmd != "quit" {
			t.Errorf("alias 'q' resolved to %q; want \"quit\"", cmd)
		}
	})

	t.Run("alias for quit (exit)", func(t *testing.T) {
		cmd, _ := resolveCommand("exit")
		if cmd != "quit" {
			t.Errorf("alias 'exit' resolved to %q; want \"quit\"", cmd)
		}
	})

	t.Run("command with arguments", func(t *testing.T) {
		cmd, args := resolveCommand("/theme init mytheme")
		if cmd != "theme" {
			t.Errorf("cmd = %q; want \"theme\"", cmd)
		}
		if len(args) != 2 || args[0] != "init" || args[1] != "mytheme" {
			t.Errorf("args = %v; want [init mytheme]", args)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		cmd, _ := resolveCommand("/BUILD")
		if cmd != "build" {
			t.Errorf("cmd = %q; want \"build\"", cmd)
		}
	})

	t.Run("unknown command returns as-is", func(t *testing.T) {
		cmd, _ := resolveCommand("unknown")
		if cmd != "unknown" {
			t.Errorf("cmd = %q; want \"unknown\"", cmd)
		}
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		cmd, _ := resolveCommand("  /build  ")
		if cmd != "build" {
			t.Errorf("cmd = %q; want \"build\"", cmd)
		}
	})

	t.Run("update-content alias", func(t *testing.T) {
		cmd, _ := resolveCommand("update-content")
		if cmd != "update" {
			t.Errorf("alias 'update-content' resolved to %q; want \"update\"", cmd)
		}
	})

	t.Run("alias a for update", func(t *testing.T) {
		cmd, _ := resolveCommand("a")
		if cmd != "update" {
			t.Errorf("alias 'a' resolved to %q; want \"update\"", cmd)
		}
	})

	t.Run("new command with title", func(t *testing.T) {
		cmd, args := resolveCommand("/new My Great Post")
		if cmd != "new" {
			t.Errorf("cmd = %q; want \"new\"", cmd)
		}
		if len(args) != 3 || args[0] != "My" || args[1] != "Great" || args[2] != "Post" {
			t.Errorf("args = %v; want [My Great Post]", args)
		}
	})

	t.Run("new command without slash", func(t *testing.T) {
		cmd, args := resolveCommand("new A Title")
		if cmd != "new" {
			t.Errorf("cmd = %q; want \"new\"", cmd)
		}
		if len(args) != 2 {
			t.Errorf("args = %v; want [A Title]", args)
		}
	})
}

// ---------------------------------------------------------------------------
// helpText
// ---------------------------------------------------------------------------

func TestHelpText(t *testing.T) {
	text := helpText()

	if !strings.Contains(text, "Available commands:") {
		t.Error("helpText missing header")
	}

	// Check that all commands appear.
	for _, cmd := range commandRegistry {
		if !strings.Contains(text, cmd.Name) {
			t.Errorf("helpText missing command %q", cmd.Name)
		}
		if !strings.Contains(text, cmd.Hint) {
			t.Errorf("helpText missing hint %q for %s", cmd.Hint, cmd.Name)
		}
	}

	if !strings.Contains(text, "Bare commands work too") {
		t.Error("helpText missing footer note")
	}
}
