// Package summary provides pluggable summary generation for pages.
//
// Three strategies are supported via the Provider interface:
//
//   - "manual"  — only frontmatter summaries are used (NoopProvider).
//   - "auto"    — first sentences are extracted from the markdown body
//     when no frontmatter summary exists (ExtractProvider).
//   - "ai"      — reserved for future LLM-based generation via Kairos
//     (see https://github.com/jllopis/kairos).
//
// The build pipeline calls [ForPages] after parsing all content and
// building the site hierarchy to fill in any empty Page.Summary fields.
package summary

import (
	"context"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DefaultMaxLen is the default character limit for auto-extracted summaries.
// CSS clamps to 2 lines (~160 chars at typical widths).
const DefaultMaxLen = 160

// ---------------------------------------------------------------------------
// Provider interface — the extension point for Kairos / LLM integration.
// ---------------------------------------------------------------------------

// Provider generates a summary for a page.
//
// Implementations must be safe for concurrent use.  The context carries
// cancellation / timeout (relevant for the future AI provider).
type Provider interface {
	// Summarize returns a plain-text summary for the given page content.
	// title is the page title (useful for LLM prompts).
	// rawMarkdown is the markdown body (frontmatter already stripped).
	// An empty return means "no summary available".
	Summarize(ctx context.Context, title string, rawMarkdown string) (string, error)
}

// ---------------------------------------------------------------------------
// NoopProvider — "manual" strategy, does nothing.
// ---------------------------------------------------------------------------

// NoopProvider always returns an empty summary.  Use it when summary_strategy
// is "manual" so only explicit frontmatter summaries are used.
type NoopProvider struct{}

func (NoopProvider) Summarize(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

// ---------------------------------------------------------------------------
// ExtractProvider — "auto" strategy, extracts first sentences.
// ---------------------------------------------------------------------------

// ExtractProvider extracts the first sentences from the markdown body,
// stripping formatting to produce a plain-text excerpt.
type ExtractProvider struct {
	// MaxLen is the maximum character length of the generated summary.
	// If zero, DefaultMaxLen is used.
	MaxLen int
}

func (p ExtractProvider) Summarize(_ context.Context, _ string, rawMarkdown string) (string, error) {
	maxLen := p.MaxLen
	if maxLen <= 0 {
		maxLen = DefaultMaxLen
	}
	plain := PlainText(rawMarkdown)
	if plain == "" {
		return "", nil
	}
	return truncateSentence(plain, maxLen), nil
}

// ---------------------------------------------------------------------------
// Placeholder for future Kairos AI provider.
// ---------------------------------------------------------------------------
//
// When Kairos is ready, implement Provider with something like:
//
//   type KairosProvider struct {
//       Client *kairos.Client   // or agent handle
//       Model  string           // e.g. "gpt-4o-mini"
//       Prompt string           // system prompt template
//   }
//
//   func (k *KairosProvider) Summarize(ctx context.Context, title, raw string) (string, error) {
//       resp, err := k.Client.Complete(ctx, kairos.Request{
//           Model:  k.Model,
//           System: k.Prompt,
//           Messages: []kairos.Message{
//               {Role: "user", Content: fmt.Sprintf("Title: %s\n\n%s", title, raw)},
//           },
//       })
//       if err != nil { return "", err }
//       return strings.TrimSpace(resp.Text), nil
//   }
//
// Then register it in NewProvider() when strategy == "ai".

// ---------------------------------------------------------------------------
// NewProvider returns the appropriate Provider for the given strategy.
// ---------------------------------------------------------------------------

// NewProvider creates a Provider for the named strategy.
// Recognised values: "auto" (default), "manual", "ai".
// "ai" currently falls back to "auto" until Kairos integration is wired.
func NewProvider(strategy string) Provider {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "manual":
		return NoopProvider{}
	case "ai":
		// TODO: when Kairos is integrated, return KairosProvider here.
		// For now fall through to auto so pages still get summaries.
		return ExtractProvider{}
	default: // "auto" or empty
		return ExtractProvider{}
	}
}

// ---------------------------------------------------------------------------
// PlainText strips markdown formatting to produce readable plain text.
// ---------------------------------------------------------------------------

// Compiled regexps (package-level for performance).
var (
	reCodeBlock   = regexp.MustCompile("(?s)```[^`]*```")
	reCodeInline  = regexp.MustCompile("`[^`]+`")
	reMathBlock   = regexp.MustCompile(`(?s)\$\$[^$]+\$\$`)
	reMathInline  = regexp.MustCompile(`\$[^$\n]+\$`)
	reImage       = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reLink        = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reWikiDisplay = regexp.MustCompile(`\[\[([^|\]]+)\|([^\]]+)\]\]`)
	reWikiSimple  = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	reHeading     = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reHR          = regexp.MustCompile(`(?m)^[\s]*([-*_]){3,}\s*$`)
	reCallout     = regexp.MustCompile(`(?m)^>\s*\[![^\]]*\]\s*`)
	reBlockquote  = regexp.MustCompile(`(?m)^>\s?`)
	reListMarker  = regexp.MustCompile(`(?m)^[\s]*[-*+]\s`)
	reOrdList     = regexp.MustCompile(`(?m)^[\s]*\d+\.\s`)
	reBoldItalic3 = regexp.MustCompile(`\*{3}(.+?)\*{3}`)
	reBoldItalic2 = regexp.MustCompile(`\*{2}(.+?)\*{2}`)
	reBoldItalic1 = regexp.MustCompile(`\*(.+?)\*`)
	reUndBI3      = regexp.MustCompile(`_{3}(.+?)_{3}`)
	reUndBI2      = regexp.MustCompile(`_{2}(.+?)_{2}`)
	reUndBI1      = regexp.MustCompile(`_(.+?)_`)
	reStrike      = regexp.MustCompile(`~~(.+?)~~`)
	reHighlight   = regexp.MustCompile(`==(.+?)==`)
	reMultiSpace  = regexp.MustCompile(`[ \t]+`)
	reMultiNL     = regexp.MustCompile(`\n{2,}`)
)

// PlainText converts a raw markdown string into plain text suitable for
// use as a summary or excerpt.  It strips code blocks, math, images,
// links, wiki-links, HTML tags, headings, block quotes, list markers,
// bold/italic, strikethrough, highlights, and collapses whitespace.
func PlainText(md string) string {
	s := md

	// Remove fenced code blocks first (they may contain other patterns).
	s = reCodeBlock.ReplaceAllString(s, " ")
	// Remove inline code.
	s = reCodeInline.ReplaceAllString(s, " ")
	// Remove math (block then inline).
	s = reMathBlock.ReplaceAllString(s, " ")
	s = reMathInline.ReplaceAllString(s, " ")
	// Remove images entirely.
	s = reImage.ReplaceAllString(s, "")
	// Convert links to their display text.
	s = reLink.ReplaceAllString(s, "$1")
	// Convert wiki-links: [[target|display]] -> display, [[target]] -> target.
	s = reWikiDisplay.ReplaceAllString(s, "$2")
	s = reWikiSimple.ReplaceAllString(s, "$1")
	// Strip HTML tags.
	s = reHTMLTag.ReplaceAllString(s, "")
	// Strip headings markers.
	s = reHeading.ReplaceAllString(s, "")
	// Strip horizontal rules.
	s = reHR.ReplaceAllString(s, " ")
	// Strip Obsidian callout markers (> [!INFO] etc.).
	s = reCallout.ReplaceAllString(s, "")
	// Strip blockquote markers.
	s = reBlockquote.ReplaceAllString(s, "")
	// Strip list markers.
	s = reListMarker.ReplaceAllString(s, "")
	s = reOrdList.ReplaceAllString(s, "")
	// Strip bold/italic/strikethrough/highlight wrappers, keep content.
	// Process longest patterns first (***bold italic*** before **bold**).
	s = reBoldItalic3.ReplaceAllString(s, "$1")
	s = reBoldItalic2.ReplaceAllString(s, "$1")
	s = reBoldItalic1.ReplaceAllString(s, "$1")
	s = reUndBI3.ReplaceAllString(s, "$1")
	s = reUndBI2.ReplaceAllString(s, "$1")
	s = reUndBI1.ReplaceAllString(s, "$1")
	s = reStrike.ReplaceAllString(s, "$1")
	s = reHighlight.ReplaceAllString(s, "$1")
	// Collapse whitespace.
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reMultiNL.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// truncateSentence truncates text to at most maxLen characters, preferring
// to break at a sentence boundary (. ! ?) or, failing that, at a word
// boundary.  If truncated, an ellipsis is appended.
func truncateSentence(text string, maxLen int) string {
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}

	// Work with runes to handle multi-byte characters correctly.
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}

	candidate := runes[:maxLen]

	// Try to find the last sentence-ending punctuation.
	bestSentence := -1
	for i := len(candidate) - 1; i >= 0; i-- {
		if candidate[i] == '.' || candidate[i] == '!' || candidate[i] == '?' {
			bestSentence = i + 1
			break
		}
	}
	if bestSentence > maxLen/3 { // only use if we keep a reasonable chunk
		return strings.TrimSpace(string(runes[:bestSentence]))
	}

	// Fall back to last word boundary.
	bestWord := -1
	for i := len(candidate) - 1; i >= 0; i-- {
		if unicode.IsSpace(candidate[i]) {
			bestWord = i
			break
		}
	}
	if bestWord > maxLen/3 {
		return strings.TrimSpace(string(runes[:bestWord])) + "..."
	}

	// Hard truncate (very long word).
	return strings.TrimSpace(string(candidate)) + "..."
}
