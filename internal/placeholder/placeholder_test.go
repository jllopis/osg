package placeholder

import (
	"strings"
	"testing"
)

func TestGenerate_Deterministic(t *testing.T) {
	title := "Hello World"
	svg1 := Generate(title)
	svg2 := Generate(title)
	if svg1 != svg2 {
		t.Fatal("Generate should be deterministic for the same title")
	}
}

func TestGenerate_DifferentTitles(t *testing.T) {
	svg1 := Generate("Title A")
	svg2 := Generate("Title B")
	if svg1 == svg2 {
		t.Fatal("different titles should produce different SVGs")
	}
}

func TestGenerate_ValidSVG(t *testing.T) {
	svg := Generate("Test Post")
	if !strings.HasPrefix(svg, "<svg") {
		t.Fatal("SVG should start with <svg")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Fatal("SVG should end with </svg>")
	}
	if !strings.Contains(svg, `width="1200"`) {
		t.Fatal("SVG should have width 1200")
	}
	if !strings.Contains(svg, `height="630"`) {
		t.Fatal("SVG should have height 630")
	}
}

func TestGenerate_ContainsShapes(t *testing.T) {
	svg := Generate("Shapes Test")
	hasCircle := strings.Contains(svg, "<circle")
	hasEllipse := strings.Contains(svg, "<ellipse")
	hasRect := strings.Count(svg, "<rect") > 1 // background rect + possible shape rects
	if !hasCircle && !hasEllipse && !hasRect {
		t.Fatal("SVG should contain at least one shape beyond background")
	}
}

func TestFilename_Deterministic(t *testing.T) {
	f1 := Filename("Title")
	f2 := Filename("Title")
	if f1 != f2 {
		t.Fatal("Filename should be deterministic")
	}
}

func TestFilename_Format(t *testing.T) {
	f := Filename("Test")
	if !strings.HasPrefix(f, "placeholder-") {
		t.Fatal("filename should start with placeholder-")
	}
	if !strings.HasSuffix(f, ".svg") {
		t.Fatal("filename should end with .svg")
	}
}
