package markdown

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "basic text",
			input:    "Hello **world**",
			contains: "<strong>world</strong>",
		},
		{
			name:     "GFM table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: "<table>",
		},
		{
			name:     "Org-mode table separator converts to GFM",
			input:    "| A | B |\n|---+---|\n| 1 | 2 |",
			contains: "<table>",
		},
		{
			name:     "Org-mode table with colons for alignment",
			input:    "| Left | Center |\n|-------+:-------:|\n| L | C |",
			contains: "<table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got:\n%s", tt.contains, output)
			}
		})
	}
}

func TestOrgModeTableConversion(t *testing.T) {
	// This specifically tests the Org-mode table separator conversion
	orgTable := `| Aspecto | Tales | Anaximandro |
|---------+-------+-------------|
| Arché   | Agua  | Ápeiron     |`

	output, err := Render([]byte(orgTable))
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Should produce a table
	if !strings.Contains(output, "<table>") {
		t.Error("expected <table> in output")
	}
	if !strings.Contains(output, "<thead>") {
		t.Error("expected <thead> in output")
	}
	if !strings.Contains(output, "<th>Aspecto</th>") {
		t.Error("expected <th>Aspecto</th> in output")
	}
	if !strings.Contains(output, "<td>Agua</td>") {
		t.Error("expected <td>Agua</td> in output")
	}

	// Should NOT contain raw pipe chars from the separator
	if strings.Contains(output, "|---") {
		t.Error("output should not contain raw markdown table syntax")
	}
}
