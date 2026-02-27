package markdown

import (
	"strings"
	"testing"
)

func TestExpandShortcodes_Note(t *testing.T) {
	input := `{{< note >}}This is important.{{< /note >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `class="admonition admonition-note"`) {
		t.Errorf("expected note admonition, got:\n%s", result)
	}
	if !strings.Contains(result, "This is important.") {
		t.Errorf("expected content preserved, got:\n%s", result)
	}
	if !strings.Contains(result, `class="admonition-title">info</p>`) {
		t.Errorf("expected default title 'info', got:\n%s", result)
	}
}

func TestExpandShortcodes_NoteWithTitle(t *testing.T) {
	input := `{{< note "Remember" >}}Don't forget.{{< /note >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `class="admonition-title">Remember</p>`) {
		t.Errorf("expected custom title 'Remember', got:\n%s", result)
	}
}

func TestExpandShortcodes_Warning(t *testing.T) {
	input := `{{< warning >}}Careful!{{< /warning >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `admonition-warning`) {
		t.Errorf("expected warning class, got:\n%s", result)
	}
}

func TestExpandShortcodes_Tip(t *testing.T) {
	input := `{{< tip >}}Pro tip here.{{< /tip >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `admonition-tip`) {
		t.Errorf("expected tip class, got:\n%s", result)
	}
}

func TestExpandShortcodes_Details(t *testing.T) {
	input := `{{< details "Click to expand" >}}Hidden content.{{< /details >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "<details>") {
		t.Errorf("expected <details>, got:\n%s", result)
	}
	if !strings.Contains(result, "<summary>Click to expand</summary>") {
		t.Errorf("expected summary, got:\n%s", result)
	}
	if !strings.Contains(result, "Hidden content.") {
		t.Errorf("expected content, got:\n%s", result)
	}
}

func TestExpandShortcodes_Unknown(t *testing.T) {
	input := `{{< unknown >}}content{{< /unknown >}}`
	result := ExpandShortcodes(input)
	if result != input {
		t.Errorf("unknown shortcode should be left as-is, got:\n%s", result)
	}
}

func TestExpandShortcodes_NoShortcodes(t *testing.T) {
	input := "Just regular **markdown** text."
	result := ExpandShortcodes(input)
	if result != input {
		t.Errorf("no shortcodes should return unchanged, got:\n%s", result)
	}
}

func TestExpandShortcodes_MultipleInline(t *testing.T) {
	input := `Before

{{< note >}}First note.{{< /note >}}

Middle text.

{{< warning >}}Second warning.{{< /warning >}}

After`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "admonition-note") {
		t.Error("missing note admonition")
	}
	if !strings.Contains(result, "admonition-warning") {
		t.Error("missing warning admonition")
	}
	if !strings.Contains(result, "Before") || !strings.Contains(result, "After") {
		t.Error("surrounding text should be preserved")
	}
}
