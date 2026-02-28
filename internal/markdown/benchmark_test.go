package markdown

import (
	"strings"
	"testing"
)

// sampleMarkdown is a realistic article body used for benchmarking.
var sampleMarkdown = strings.Repeat(`# Heading One

This is a paragraph with **bold**, *italic*, and `+"`inline code`"+`. It also
has a [link](https://example.com) and an image reference.

## Second Heading

Here is a list:

- Item one with some text
- Item two with **bold** text
- Item three with a [link](https://example.com/page)

### Code Block

`+"```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"+`

> A blockquote with multiple lines of text that wraps around to test
> how the renderer handles longer content blocks.

| Column A | Column B | Column C |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

Some text with a footnote[^1].

[^1]: This is the footnote content.

`, 3) // ~3x to simulate a medium-length article

func BenchmarkRender(b *testing.B) {
	input := []byte(sampleMarkdown)
	for b.Loop() {
		_, _ = Render(input)
	}
}

func BenchmarkRenderString(b *testing.B) {
	for b.Loop() {
		_, _ = RenderString(sampleMarkdown)
	}
}

// sampleWithShortcodes exercises shortcode expansion before Goldmark.
var sampleWithShortcodes = `Some intro text.

{{< note >}}
This is a note admonition with **bold** content.
{{< /note >}}

{{< warning >}}
Be careful with this operation.
{{< /warning >}}

{{< tip >}}
A helpful tip for the reader.
{{< /tip >}}

{{< details summary="Click to expand" >}}
Hidden content with a [link](https://example.com).
{{< /details >}}

{{< youtube dQw4w9WgXcQ >}}

{{< twitter https://twitter.com/jack/status/20 >}}

Some more text here.

{{< tabs >}}
{{< tab title="Go" >}}
` + "```go\nfmt.Println(\"hello\")\n```" + `
{{< /tab >}}
{{< tab title="Rust" >}}
` + "```rust\nprintln!(\"hello\");\n```" + `
{{< /tab >}}
{{< /tabs >}}
`

func BenchmarkExpandShortcodes(b *testing.B) {
	for b.Loop() {
		ExpandShortcodes(sampleWithShortcodes)
	}
}

func BenchmarkRenderWithShortcodes(b *testing.B) {
	input := []byte(sampleWithShortcodes)
	for b.Loop() {
		_, _ = Render(input)
	}
}

func BenchmarkExtractTOC(b *testing.B) {
	html, _ := Render([]byte(sampleMarkdown))
	b.ResetTimer()
	for b.Loop() {
		ExtractTOC(html)
	}
}
