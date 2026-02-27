package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		highlighting.NewHighlighting(
			highlighting.WithStyle("nord"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		),
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
		renderer.WithNodeRenderers(
			// Override image rendering: wrap standalone images in <figure>
			// with data-lightbox attribute and optional <figcaption>.
			util.Prioritized(newFigureImageRenderer(html.WithUnsafe()), 100),
			// Override paragraph rendering: suppress <p> for standalone images.
			util.Prioritized(newFigureParagraphRenderer(html.WithUnsafe()), 100),
		),
	),
)

// orgTableSepRe matches Org-mode table separator rows that use + instead of |
// e.g. |---+---+---| should become |---|---|---|
var orgTableSepRe = regexp.MustCompile(`(?m)^(\|[-:]+)(\+[-:]+)*\|$`)

// orgTableSepPlus matches the + characters in separator rows
var orgTableSepPlus = regexp.MustCompile(`\+`)

func Render(input []byte) (string, error) {
	// Pre-process: expand shortcodes before Markdown rendering.
	processed := []byte(ExpandShortcodes(string(input)))

	// Pre-process: convert Org-mode table separators to GFM format
	// Org-mode uses |---+---| while GFM needs |---|---|
	processed = orgTableSepRe.ReplaceAllFunc(processed, func(match []byte) []byte {
		return orgTableSepPlus.ReplaceAll(match, []byte("|"))
	})

	var buf bytes.Buffer
	if err := md.Convert(processed, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderString(input string) (string, error) {
	return Render([]byte(input))
}
