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

// --- YouTube ---

func TestExpandShortcodes_YouTube_BareID(t *testing.T) {
	input := `{{< youtube "dQw4w9WgXcQ" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "embed-youtube") {
		t.Errorf("expected youtube embed class, got:\n%s", result)
	}
	if !strings.Contains(result, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected video ID in iframe src, got:\n%s", result)
	}
	if !strings.Contains(result, "<iframe") {
		t.Errorf("expected iframe, got:\n%s", result)
	}
}

func TestExpandShortcodes_YouTube_FullURL(t *testing.T) {
	input := `{{< youtube "https://www.youtube.com/watch?v=dQw4w9WgXcQ" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected extracted video ID, got:\n%s", result)
	}
}

func TestExpandShortcodes_YouTube_ShortURL(t *testing.T) {
	input := `{{< youtube "https://youtu.be/dQw4w9WgXcQ" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Errorf("expected extracted video ID from short URL, got:\n%s", result)
	}
}

func TestExpandShortcodes_YouTube_SelfClosing(t *testing.T) {
	input := `{{< youtube "dQw4w9WgXcQ" />}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "embed-youtube") {
		t.Errorf("expected self-closing youtube to work, got:\n%s", result)
	}
}

func TestExpandShortcodes_YouTube_Empty(t *testing.T) {
	input := `{{< youtube >}}`
	result := ExpandShortcodes(input)
	if strings.Contains(result, "iframe") {
		t.Errorf("empty youtube should produce no iframe, got:\n%s", result)
	}
}

// --- Twitter ---

func TestExpandShortcodes_Twitter(t *testing.T) {
	input := `{{< twitter "https://twitter.com/user/status/123456" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "embed-twitter") {
		t.Errorf("expected twitter embed class, got:\n%s", result)
	}
	if !strings.Contains(result, "twitter-tweet") {
		t.Errorf("expected twitter-tweet blockquote, got:\n%s", result)
	}
	if !strings.Contains(result, "platform.twitter.com/widgets.js") {
		t.Errorf("expected twitter widgets script, got:\n%s", result)
	}
}

func TestExpandShortcodes_Twitter_XCom(t *testing.T) {
	input := `{{< twitter "https://x.com/user/status/123456" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "twitter.com/user/status/123456") {
		t.Errorf("expected x.com normalized to twitter.com, got:\n%s", result)
	}
}

func TestExpandShortcodes_Twitter_Empty(t *testing.T) {
	input := `{{< twitter >}}`
	result := ExpandShortcodes(input)
	if strings.Contains(result, "twitter-tweet") {
		t.Errorf("empty twitter should produce nothing, got:\n%s", result)
	}
}

// --- CodePen ---

func TestExpandShortcodes_CodePen(t *testing.T) {
	input := `{{< codepen "https://codepen.io/jdoe/pen/abcDEF" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "embed-codepen") {
		t.Errorf("expected codepen embed class, got:\n%s", result)
	}
	if !strings.Contains(result, "codepen.io/jdoe/embed/abcDEF") {
		t.Errorf("expected codepen embed URL, got:\n%s", result)
	}
	if !strings.Contains(result, `default-tab=result`) {
		t.Errorf("expected default tab, got:\n%s", result)
	}
}

func TestExpandShortcodes_CodePen_WithOptions(t *testing.T) {
	input := `{{< codepen url="https://codepen.io/jdoe/pen/abcDEF" height="600" tab="css,result" theme="light" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `height="600"`) {
		t.Errorf("expected custom height, got:\n%s", result)
	}
	if !strings.Contains(result, `default-tab=css,result`) {
		t.Errorf("expected custom tab, got:\n%s", result)
	}
	if !strings.Contains(result, `theme-id=light`) {
		t.Errorf("expected custom theme, got:\n%s", result)
	}
}

func TestExpandShortcodes_CodePen_InvalidURL(t *testing.T) {
	input := `{{< codepen "https://example.com/not-codepen" >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "View on CodePen") {
		t.Errorf("expected fallback link for invalid URL, got:\n%s", result)
	}
}

// --- Figure ---

func TestExpandShortcodes_Figure(t *testing.T) {
	input := `{{< figure src="/img/photo.jpg" caption="A nice photo" >}}{{< /figure >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `class="figure"`) {
		t.Errorf("expected figure class, got:\n%s", result)
	}
	if !strings.Contains(result, `src="/img/photo.jpg"`) {
		t.Errorf("expected src attribute, got:\n%s", result)
	}
	if !strings.Contains(result, "<figcaption>A nice photo</figcaption>") {
		t.Errorf("expected figcaption from caption arg, got:\n%s", result)
	}
}

func TestExpandShortcodes_Figure_WithContent(t *testing.T) {
	input := `{{< figure src="/img/photo.jpg" >}}Custom **caption** text.{{< /figure >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, "<figcaption>Custom **caption** text.</figcaption>") {
		t.Errorf("expected figcaption from inner content, got:\n%s", result)
	}
}

func TestExpandShortcodes_Figure_WithClassAndWidth(t *testing.T) {
	input := `{{< figure src="/img/wide.jpg" class="full-width" width="800" >}}{{< /figure >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `class="figure full-width"`) {
		t.Errorf("expected custom class, got:\n%s", result)
	}
	if !strings.Contains(result, `width="800"`) {
		t.Errorf("expected width attribute, got:\n%s", result)
	}
}

func TestExpandShortcodes_Figure_WithLink(t *testing.T) {
	input := `{{< figure src="/img/photo.jpg" link="https://example.com" >}}{{< /figure >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `<a href="https://example.com">`) {
		t.Errorf("expected link wrapper, got:\n%s", result)
	}
	if !strings.Contains(result, "</a>") {
		t.Errorf("expected closing link tag, got:\n%s", result)
	}
}

// --- Tabs ---

func TestExpandShortcodes_Tabs(t *testing.T) {
	input := `{{< tabs >}}
{{< tab "Go" >}}` + "`" + `go fmt` + "`" + `{{< /tab >}}
{{< tab "Rust" >}}` + "`" + `cargo fmt` + "`" + `{{< /tab >}}
{{< /tabs >}}`
	result := ExpandShortcodes(input)
	if !strings.Contains(result, `class="tabs"`) {
		t.Errorf("expected tabs wrapper, got:\n%s", result)
	}
	if !strings.Contains(result, `data-tab-title="Go"`) {
		t.Errorf("expected Go tab, got:\n%s", result)
	}
	if !strings.Contains(result, `data-tab-title="Rust"`) {
		t.Errorf("expected Rust tab, got:\n%s", result)
	}
	if !strings.Contains(result, `class="tab-content"`) {
		t.Errorf("expected tab-content, got:\n%s", result)
	}
}

// --- parseArgs ---

func TestParseArgs_KeyValue(t *testing.T) {
	args := parseArgs(`src="/img/photo.jpg" caption="A nice photo" width=800`)
	if args["src"] != "/img/photo.jpg" {
		t.Errorf("expected src, got %q", args["src"])
	}
	if args["caption"] != "A nice photo" {
		t.Errorf("expected caption, got %q", args["caption"])
	}
	if args["width"] != "800" {
		t.Errorf("expected width, got %q", args["width"])
	}
}

func TestParseArgs_BarePositional(t *testing.T) {
	args := parseArgs(`"dQw4w9WgXcQ"`)
	if args["_pos"] != "dQw4w9WgXcQ" {
		t.Errorf("expected bare positional arg, got %q", args["_pos"])
	}
}

func TestParseArgs_Empty(t *testing.T) {
	args := parseArgs("")
	if len(args) != 0 {
		t.Errorf("expected empty map, got %v", args)
	}
}

// --- extractVideoID ---

func TestExtractVideoID_BareID(t *testing.T) {
	if id := extractVideoID("dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Errorf("expected bare ID, got %q", id)
	}
}

func TestExtractVideoID_WatchURL(t *testing.T) {
	if id := extractVideoID("https://www.youtube.com/watch?v=dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Errorf("expected extracted ID from watch URL, got %q", id)
	}
}

func TestExtractVideoID_ShortURL(t *testing.T) {
	if id := extractVideoID("https://youtu.be/dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Errorf("expected extracted ID from short URL, got %q", id)
	}
}

func TestExtractVideoID_EmbedURL(t *testing.T) {
	if id := extractVideoID("https://www.youtube.com/embed/dQw4w9WgXcQ"); id != "dQw4w9WgXcQ" {
		t.Errorf("expected extracted ID from embed URL, got %q", id)
	}
}

func TestExtractVideoID_Empty(t *testing.T) {
	if id := extractVideoID(""); id != "" {
		t.Errorf("expected empty for empty input, got %q", id)
	}
}
