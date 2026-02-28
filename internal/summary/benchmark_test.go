package summary

import (
	"context"
	"strings"
	"testing"
)

var sampleBody = strings.Repeat(`This is a paragraph with **bold** and *italic* text. It has
[links](https://example.com) and `+"`code spans`"+`. The quick brown fox jumps
over the lazy dog. Here is another sentence for good measure.

## Heading Two

More content follows the heading. Lists are common:

- First item in the list
- Second item with **emphasis**
- Third item with a [link](https://example.com)

`, 5)

func BenchmarkPlainText(b *testing.B) {
	for b.Loop() {
		PlainText(sampleBody)
	}
}

func BenchmarkExtractProvider_Summarize(b *testing.B) {
	provider := ExtractProvider{}
	ctx := context.Background()
	for b.Loop() {
		_, _ = provider.Summarize(ctx, "Sample Title", sampleBody)
	}
}

func BenchmarkTruncateSentence(b *testing.B) {
	plain := PlainText(sampleBody)
	for b.Loop() {
		truncateSentence(plain, 160)
	}
}
