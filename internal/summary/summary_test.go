package summary

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// NewProvider
// ---------------------------------------------------------------------------

func TestNewProvider_Auto(t *testing.T) {
	p := NewProvider("auto")
	if _, ok := p.(ExtractProvider); !ok {
		t.Fatalf("expected ExtractProvider, got %T", p)
	}
}

func TestNewProvider_Manual(t *testing.T) {
	p := NewProvider("manual")
	if _, ok := p.(NoopProvider); !ok {
		t.Fatalf("expected NoopProvider, got %T", p)
	}
}

func TestNewProvider_AI_FallsBackToExtract(t *testing.T) {
	p := NewProvider("ai")
	if _, ok := p.(ExtractProvider); !ok {
		t.Fatalf("expected ExtractProvider (ai fallback), got %T", p)
	}
}

func TestNewProvider_Empty_DefaultsToAuto(t *testing.T) {
	p := NewProvider("")
	if _, ok := p.(ExtractProvider); !ok {
		t.Fatalf("expected ExtractProvider for empty strategy, got %T", p)
	}
}

func TestNewProvider_CaseInsensitive(t *testing.T) {
	for _, s := range []string{"MANUAL", "Manual", " manual "} {
		p := NewProvider(s)
		if _, ok := p.(NoopProvider); !ok {
			t.Errorf("NewProvider(%q): expected NoopProvider, got %T", s, p)
		}
	}
}

// ---------------------------------------------------------------------------
// NoopProvider
// ---------------------------------------------------------------------------

func TestNoopProvider_ReturnsEmpty(t *testing.T) {
	p := NoopProvider{}
	text, err := p.Summarize(context.Background(), "Title", "Some markdown content here.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty string, got %q", text)
	}
}

// ---------------------------------------------------------------------------
// ExtractProvider
// ---------------------------------------------------------------------------

func TestExtractProvider_BasicSentence(t *testing.T) {
	p := ExtractProvider{}
	md := "This is the first sentence. This is the second sentence. And a third one."
	text, err := p.Summarize(context.Background(), "Title", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty summary")
	}
	// Should be plain text, no markdown
	if strings.Contains(text, "#") || strings.Contains(text, "**") {
		t.Errorf("summary still contains markdown formatting: %q", text)
	}
}

func TestExtractProvider_EmptyContent(t *testing.T) {
	p := ExtractProvider{}
	text, err := p.Summarize(context.Background(), "Title", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty string for empty content, got %q", text)
	}
}

func TestExtractProvider_OnlyHeadings(t *testing.T) {
	p := ExtractProvider{}
	md := "# Heading One\n## Heading Two\n### Heading Three\n"
	text, err := p.Summarize(context.Background(), "Title", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Headings become plain text — they should appear as content
	if text == "" {
		t.Log("headings-only content produced empty summary (acceptable)")
	}
}

func TestExtractProvider_CustomMaxLen(t *testing.T) {
	p := ExtractProvider{MaxLen: 30}
	md := "This is a sentence that is definitely longer than thirty characters by quite a bit."
	text, err := p.Summarize(context.Background(), "Title", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Allow a little extra for the "..." suffix
	if len([]rune(text)) > 35 {
		t.Errorf("expected truncated to ~30 chars, got %d: %q", len([]rune(text)), text)
	}
}

func TestExtractProvider_RespectsMaxLenZeroDefault(t *testing.T) {
	p := ExtractProvider{MaxLen: 0}
	md := "Short."
	text, err := p.Summarize(context.Background(), "Title", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Short." {
		t.Errorf("expected %q, got %q", "Short.", text)
	}
}

// ---------------------------------------------------------------------------
// PlainText
// ---------------------------------------------------------------------------

func TestPlainText_CodeBlock(t *testing.T) {
	md := "Before.\n```go\nfmt.Println(\"hello\")\n```\nAfter."
	result := PlainText(md)
	if strings.Contains(result, "fmt.Println") {
		t.Errorf("code block not stripped: %q", result)
	}
	if !strings.Contains(result, "Before") || !strings.Contains(result, "After") {
		t.Errorf("surrounding text lost: %q", result)
	}
}

func TestPlainText_InlineCode(t *testing.T) {
	md := "Use the `fmt.Println` function."
	result := PlainText(md)
	if strings.Contains(result, "`") {
		t.Errorf("inline code backticks not stripped: %q", result)
	}
	if strings.Contains(result, "fmt.Println") {
		t.Errorf("inline code content not stripped: %q", result)
	}
}

func TestPlainText_MathBlock(t *testing.T) {
	md := "Formula: $$E = mc^2$$ is famous."
	result := PlainText(md)
	if strings.Contains(result, "mc^2") {
		t.Errorf("math block not stripped: %q", result)
	}
	if !strings.Contains(result, "Formula") {
		t.Errorf("surrounding text lost: %q", result)
	}
}

func TestPlainText_MathInline(t *testing.T) {
	md := "The value $x + y$ is computed."
	result := PlainText(md)
	if strings.Contains(result, "x + y") {
		t.Errorf("inline math not stripped: %q", result)
	}
}

func TestPlainText_Image(t *testing.T) {
	md := "See ![alt text](image.png) for details."
	result := PlainText(md)
	if strings.Contains(result, "alt text") || strings.Contains(result, "image.png") {
		t.Errorf("image not stripped: %q", result)
	}
}

func TestPlainText_Link(t *testing.T) {
	md := "Visit [Google](https://google.com) now."
	result := PlainText(md)
	if strings.Contains(result, "https://google.com") {
		t.Errorf("link URL not stripped: %q", result)
	}
	if !strings.Contains(result, "Google") {
		t.Errorf("link display text lost: %q", result)
	}
}

func TestPlainText_WikiLinkSimple(t *testing.T) {
	md := "See [[My Page]] for more."
	result := PlainText(md)
	if strings.Contains(result, "[[") || strings.Contains(result, "]]") {
		t.Errorf("wiki-link brackets not stripped: %q", result)
	}
	if !strings.Contains(result, "My Page") {
		t.Errorf("wiki-link target text lost: %q", result)
	}
}

func TestPlainText_WikiLinkDisplay(t *testing.T) {
	md := "See [[target|Display Text]] for more."
	result := PlainText(md)
	if !strings.Contains(result, "Display Text") {
		t.Errorf("wiki-link display text lost: %q", result)
	}
	if strings.Contains(result, "target") {
		t.Errorf("wiki-link target should be stripped: %q", result)
	}
}

func TestPlainText_HTMLTags(t *testing.T) {
	md := "This has <strong>bold</strong> and <em>italic</em> HTML."
	result := PlainText(md)
	if strings.Contains(result, "<") || strings.Contains(result, ">") {
		t.Errorf("HTML tags not stripped: %q", result)
	}
	if !strings.Contains(result, "bold") || !strings.Contains(result, "italic") {
		t.Errorf("HTML content lost: %q", result)
	}
}

func TestPlainText_PreBlock(t *testing.T) {
	md := "Before.\n\n<pre class=\"mermaid\">\ngraph TD\n  A --> B\n</pre>\n\nAfter."
	result := PlainText(md)
	if strings.Contains(result, "graph TD") || strings.Contains(result, "A -->") {
		t.Errorf("pre block content not stripped: %q", result)
	}
	if !strings.Contains(result, "Before") || !strings.Contains(result, "After") {
		t.Errorf("surrounding text lost: %q", result)
	}
}

func TestPlainText_ScriptBlock(t *testing.T) {
	md := "Before.\n<script>(function(){mermaid.init()})()</script>\nAfter."
	result := PlainText(md)
	if strings.Contains(result, "mermaid") || strings.Contains(result, "function") {
		t.Errorf("script block content not stripped: %q", result)
	}
	if !strings.Contains(result, "Before") || !strings.Contains(result, "After") {
		t.Errorf("surrounding text lost: %q", result)
	}
}

func TestPlainText_MermaidTransformed(t *testing.T) {
	// Simulates what content.transform + page.before_render produces:
	// mermaid blocks become <pre class="mermaid"> + inline <script>
	md := `Intro paragraph about the Lindy effect.

<pre class="mermaid">
graph LR
  A[Idea] --&gt; B{Survived?}
  B --&gt; |Yes| C[More likely to survive]
  B --&gt; |No| D[Forgotten]
</pre>

<script>(function(){var d=document.querySelectorAll('pre.mermaid');if(!d.length)return})()</script>

Conclusion paragraph.`
	result := PlainText(md)
	if strings.Contains(result, "graph LR") || strings.Contains(result, "Survived") {
		t.Errorf("mermaid diagram content not stripped: %q", result)
	}
	if strings.Contains(result, "script") || strings.Contains(result, "querySelectorAll") {
		t.Errorf("script content not stripped: %q", result)
	}
	if !strings.Contains(result, "Intro paragraph") || !strings.Contains(result, "Conclusion paragraph") {
		t.Errorf("text content lost: %q", result)
	}
}

func TestPlainText_Headings(t *testing.T) {
	md := "# Heading\nBody text here."
	result := PlainText(md)
	if strings.Contains(result, "#") {
		t.Errorf("heading marker not stripped: %q", result)
	}
	if !strings.Contains(result, "Heading") {
		t.Errorf("heading text lost: %q", result)
	}
}

func TestPlainText_Blockquote(t *testing.T) {
	md := "> This is quoted.\n> Another line."
	result := PlainText(md)
	if strings.HasPrefix(result, ">") || strings.Contains(result, "\n>") {
		t.Errorf("blockquote markers not stripped: %q", result)
	}
	if !strings.Contains(result, "This is quoted") {
		t.Errorf("blockquote text lost: %q", result)
	}
}

func TestPlainText_ObsidianCallout(t *testing.T) {
	md := "> [!INFO] Important note\n> Details here."
	result := PlainText(md)
	if strings.Contains(result, "[!INFO]") {
		t.Errorf("callout marker not stripped: %q", result)
	}
}

func TestPlainText_ListMarkers(t *testing.T) {
	md := "- Item one\n- Item two\n* Item three\n1. First\n2. Second"
	result := PlainText(md)
	if strings.Contains(result, "- ") || strings.Contains(result, "* ") {
		t.Errorf("list markers not stripped: %q", result)
	}
}

func TestPlainText_BoldItalic(t *testing.T) {
	md := "This is **bold** and *italic* and ***both***."
	result := PlainText(md)
	if strings.Contains(result, "*") {
		t.Errorf("bold/italic markers not stripped: %q", result)
	}
	if !strings.Contains(result, "bold") || !strings.Contains(result, "italic") {
		t.Errorf("content lost: %q", result)
	}
}

func TestPlainText_Strikethrough(t *testing.T) {
	md := "This is ~~struck~~ text."
	result := PlainText(md)
	if strings.Contains(result, "~~") {
		t.Errorf("strikethrough markers not stripped: %q", result)
	}
	if !strings.Contains(result, "struck") {
		t.Errorf("content lost: %q", result)
	}
}

func TestPlainText_Highlight(t *testing.T) {
	md := "This is ==highlighted== text."
	result := PlainText(md)
	if strings.Contains(result, "==") {
		t.Errorf("highlight markers not stripped: %q", result)
	}
	if !strings.Contains(result, "highlighted") {
		t.Errorf("content lost: %q", result)
	}
}

func TestPlainText_HorizontalRule(t *testing.T) {
	md := "Before.\n\n---\n\nAfter."
	result := PlainText(md)
	if strings.Contains(result, "---") {
		t.Errorf("horizontal rule not stripped: %q", result)
	}
}

func TestPlainText_Empty(t *testing.T) {
	if got := PlainText(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestPlainText_WhitespaceOnly(t *testing.T) {
	if got := PlainText("   \n\n   \t  "); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestPlainText_CollapsesWhitespace(t *testing.T) {
	md := "Word    one.\n\n\nWord   two."
	result := PlainText(md)
	if strings.Contains(result, "  ") {
		t.Errorf("multiple spaces not collapsed: %q", result)
	}
}

// ---------------------------------------------------------------------------
// truncateSentence
// ---------------------------------------------------------------------------

func TestTruncateSentence_ShortText(t *testing.T) {
	got := truncateSentence("Short text.", 100)
	if got != "Short text." {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestTruncateSentence_ExactLength(t *testing.T) {
	text := strings.Repeat("a", 160)
	got := truncateSentence(text, 160)
	if got != text {
		t.Errorf("expected unchanged text at exact length")
	}
}

func TestTruncateSentence_BreaksAtSentence(t *testing.T) {
	text := "First sentence. Second sentence that makes this much longer than the limit we will set."
	got := truncateSentence(text, 20)
	if !strings.HasSuffix(got, ".") {
		t.Errorf("expected sentence break (ending with '.'), got %q", got)
	}
	if strings.Contains(got, "...") {
		t.Errorf("sentence break should not have ellipsis, got %q", got)
	}
}

func TestTruncateSentence_BreaksAtWord(t *testing.T) {
	text := "This has no sentence boundary just words that go on and on and on and on and on forever"
	got := truncateSentence(text, 40)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected word-break with ellipsis, got %q", got)
	}
	// Should not break mid-word (soft check; main thing is it ends at a word boundary)
	trimmed := strings.TrimSuffix(got, "...")
	if strings.HasSuffix(trimmed, "a") {
		t.Log("word-break ended with 'a', possibly mid-word")
	}
}

func TestTruncateSentence_VeryLongWord(t *testing.T) {
	text := strings.Repeat("x", 200)
	got := truncateSentence(text, 50)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected hard truncate with ellipsis, got %q", got)
	}
	if len([]rune(got)) > 55 {
		t.Errorf("hard truncate exceeded limit: %d chars", len([]rune(got)))
	}
}

func TestTruncateSentence_Unicode(t *testing.T) {
	// Each emoji is one rune but multiple bytes.
	text := "Hello " + strings.Repeat("🎉", 200)
	got := truncateSentence(text, 20)
	if len([]rune(got)) > 25 { // allow a little slack for "..."
		t.Errorf("unicode truncation exceeded limit: %d runes in %q", len([]rune(got)), got)
	}
}

// ---------------------------------------------------------------------------
// Integration: ExtractProvider with realistic markdown
// ---------------------------------------------------------------------------

func TestExtractProvider_RealisticMarkdown(t *testing.T) {
	md := `# My Great Post

This is the **opening paragraph** with a [[wiki link]] and some $math$.

> [!NOTE] Remember
> This is a callout that should be stripped.

Second paragraph with ` + "`inline code`" + ` and [a link](https://example.com).

$$
E = mc^2
$$

Third paragraph continues here.`

	p := ExtractProvider{MaxLen: 100}
	text, err := p.Summarize(context.Background(), "My Great Post", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatal("expected non-empty summary from realistic markdown")
	}
	// Should not contain any markdown artifacts
	for _, artifact := range []string{"#", "**", "[[", "]]", "`", "[!", "$$", "<"} {
		if strings.Contains(text, artifact) {
			t.Errorf("summary contains markdown artifact %q: %q", artifact, text)
		}
	}
	t.Logf("Generated summary: %q", text)
}
