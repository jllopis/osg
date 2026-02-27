package markdown

import (
	"strings"
	"testing"
)

func TestFigureRendering(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:  "standalone image gets figure wrapper",
			input: "![A sunset](sunset.jpg)",
			contains: []string{
				`<figure data-lightbox>`,
				`<img src="sunset.jpg"`,
				`alt="A sunset"`,
				`loading="lazy"`,
				`<figcaption>A sunset</figcaption>`,
				`</figure>`,
			},
			notContains: []string{
				"<p>",
			},
		},
		{
			name:  "standalone image without alt text: no figcaption",
			input: "![](photo.png)",
			contains: []string{
				`<figure data-lightbox>`,
				`<img src="photo.png"`,
				`alt=""`,
				`</figure>`,
			},
			notContains: []string{
				"<figcaption>",
				"<p>",
			},
		},
		{
			name:  "inline image stays in paragraph",
			input: "Check out this ![icon](icon.png) image inline.",
			contains: []string{
				"<p>",
				`<img src="icon.png"`,
				`alt="icon"`,
			},
			notContains: []string{
				"<figure",
				"<figcaption>",
			},
		},
		{
			name:  "image with title attribute",
			input: `![Mountain](mountain.jpg "View from the top")`,
			contains: []string{
				`<figure data-lightbox>`,
				`title="View from the top"`,
				`alt="Mountain"`,
				`<figcaption>Mountain</figcaption>`,
			},
		},
		{
			name:  "multiple standalone images produce separate figures",
			input: "![First](first.jpg)\n\n![Second](second.jpg)",
			contains: []string{
				`<figure data-lightbox><img src="first.jpg"`,
				`<figure data-lightbox><img src="second.jpg"`,
				`<figcaption>First</figcaption>`,
				`<figcaption>Second</figcaption>`,
			},
		},
		{
			name:  "image in link stays inline (not standalone)",
			input: "[![Click me](thumb.jpg)](page.html)",
			contains: []string{
				"<a href=",
				`<img src="thumb.jpg"`,
			},
			// Image inside <a> is not a direct child of <p>, so figure
			// should not wrap it — it's inside a link node.
		},
		{
			name:  "text then image on next paragraph",
			input: "Some text here.\n\n![Photo](photo.jpg)",
			contains: []string{
				"<p>Some text here.</p>",
				`<figure data-lightbox><img src="photo.jpg"`,
			},
		},
		{
			name:  "image URL with special characters is escaped",
			input: `![Art](path/to/image%20file.jpg)`,
			contains: []string{
				`<figure data-lightbox>`,
				`src="path/to/image%20file.jpg"`,
			},
		},
		{
			name:  "image alt with HTML entities is escaped",
			input: `![A <b>bold</b> image](test.jpg)`,
			contains: []string{
				`alt="A bold image"`,
				// figcaption should have escaped HTML
				`<figcaption>A bold image</figcaption>`,
			},
			notContains: []string{
				`<figcaption>A <b>bold</b>`, // should be escaped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q\ngot:\n%s", expected, output)
				}
			}
			for _, unexpected := range tt.notContains {
				if strings.Contains(output, unexpected) {
					t.Errorf("expected output NOT to contain %q\ngot:\n%s", unexpected, output)
				}
			}
		})
	}
}

func TestFigureDoesNotBreakNormalParagraphs(t *testing.T) {
	// Ensure normal paragraphs (no images) still render correctly.
	input := "First paragraph.\n\nSecond paragraph with **bold**."
	output, err := Render([]byte(input))
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(output, "<p>First paragraph.</p>") {
		t.Errorf("expected first paragraph, got:\n%s", output)
	}
	if !strings.Contains(output, "<p>Second paragraph with <strong>bold</strong>.</p>") {
		t.Errorf("expected second paragraph with bold, got:\n%s", output)
	}
}
