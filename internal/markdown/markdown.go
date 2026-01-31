package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func Render(input []byte) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert(input, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderString(input string) (string, error) {
	return Render([]byte(input))
}
