package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// figureImageRenderer wraps standalone images in <figure> elements with
// an optional <figcaption> derived from the alt text.  It also adds a
// data-lightbox attribute so the lightbox JS can target these images.
//
// An image is considered "standalone" when it is the only child of its
// parent paragraph.  Inline images (mixed with text) are left untouched.
type figureImageRenderer struct {
	html.Config
}

func newFigureImageRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &figureImageRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *figureImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *figureImageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)
	standalone := isStandaloneImage(n)

	if standalone {
		// Replace the wrapping <p> with <figure> by closing any open <p>
		// tag and opening <figure> instead.  The paragraph renderer has
		// already written "<p>" by the time we get here, so we cannot
		// retroactively remove it.  Instead, we handle the paragraph
		// suppression in the paragraph renderer override below.
		_, _ = w.WriteString("<figure data-lightbox>")
	}

	_, _ = w.WriteString("<img src=\"")
	if r.Unsafe || !html.IsDangerousURL(n.Destination) {
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
	}
	_, _ = w.WriteString("\"")

	alt := nodeText(n, source)
	_, _ = w.WriteString(" alt=\"")
	_, _ = w.Write(util.EscapeHTML([]byte(alt)))
	_, _ = w.WriteString("\"")

	if n.Title != nil {
		_, _ = w.WriteString(" title=\"")
		_, _ = w.Write(util.EscapeHTML(n.Title))
		_, _ = w.WriteString("\"")
	}

	_, _ = w.WriteString(" loading=\"lazy\"")

	if r.XHTML {
		_, _ = w.WriteString(" />")
	} else {
		_, _ = w.WriteString(">")
	}

	if standalone && alt != "" {
		_, _ = w.WriteString("<figcaption>")
		_, _ = w.Write(util.EscapeHTML([]byte(alt)))
		_, _ = w.WriteString("</figcaption>")
	}

	if standalone {
		_, _ = w.WriteString("</figure>\n")
	}

	return ast.WalkSkipChildren, nil
}

// isStandaloneImage returns true when the image node is the only child
// of its parent paragraph (no surrounding text, links, etc.).
func isStandaloneImage(n *ast.Image) bool {
	parent := n.Parent()
	if parent == nil {
		return false
	}
	if parent.Kind() != ast.KindParagraph {
		return false
	}
	// The paragraph must contain exactly one child: this image.
	return parent.ChildCount() == 1
}

// nodeText extracts the plain-text content of an inline node tree,
// which for images is the alt text built from text segments.
func nodeText(n ast.Node, source []byte) string {
	var buf []byte
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			buf = append(buf, t.Segment.Value(source)...)
		}
	}
	return string(buf)
}

// figureParagraphRenderer suppresses <p>…</p> wrapping for paragraphs
// that contain a single standalone image.  The <figure> tag produced by
// figureImageRenderer replaces the paragraph entirely.
type figureParagraphRenderer struct {
	html.Config
}

func newFigureParagraphRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &figureParagraphRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *figureParagraphRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindParagraph, r.renderParagraph)
}

func (r *figureParagraphRenderer) renderParagraph(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Check if this paragraph contains a single standalone image.
	if node.ChildCount() == 1 {
		if _, ok := node.FirstChild().(*ast.Image); ok {
			// Suppress <p> — the figureImageRenderer writes <figure> instead.
			return ast.WalkContinue, nil
		}
	}

	// Normal paragraph rendering.
	if entering {
		if node.PreviousSibling() != nil {
			_, _ = w.WriteString("\n")
		}
		_, _ = w.WriteString("<p>")
	} else {
		_, _ = w.WriteString("</p>\n")
	}
	return ast.WalkContinue, nil
}
