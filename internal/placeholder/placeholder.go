// Package placeholder generates deterministic SVG placeholder images
// using Nord-palette geometric patterns, seeded by a SHA-256 hash of
// the post title. Dimensions are 1200×630 (OG-friendly aspect ratio).
package placeholder

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	Width  = 1200
	Height = 630
)

// Nord palette groups used for generation.
var (
	// Dark gradient backgrounds (Polar Night).
	bgColors = [2]string{"#2e3440", "#3b4252"}

	// Shapes use Frost + Aurora at low opacity.
	shapeColors = []string{
		"#8fbcbb", "#88c0d0", "#81a1c1", "#5e81ac", // Frost
		"#bf616a", "#d08770", "#ebcb8b", "#a3be8c", "#b48ead", // Aurora
	}
)

// Generate returns an SVG string for the given title, deterministic and
// reproducible. The SVG has no embedded text—title display is handled
// by HTML.
func Generate(title string) string {
	h := sha256.Sum256([]byte(title))
	// We use the 32-byte hash as a pool of randomness.
	// idx tracks how many bytes we've consumed.
	idx := 0
	nextByte := func() byte {
		b := h[idx%32]
		idx++
		return b
	}
	nextFloat := func() float64 {
		return float64(nextByte()) / 255.0
	}
	nextRange := func(min, max float64) float64 {
		return min + nextFloat()*(max-min)
	}
	nextInt := func(max int) int {
		return int(binary.BigEndian.Uint16([]byte{nextByte(), nextByte()})) % max
	}

	var b strings.Builder
	b.Grow(4096)

	// SVG header.
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, Width, Height, Width, Height)
	b.WriteString("\n")

	// Gradient background.
	gradAngle := nextFloat() * 360
	rad := gradAngle * math.Pi / 180
	x1 := 50 - math.Cos(rad)*50
	y1 := 50 - math.Sin(rad)*50
	x2 := 50 + math.Cos(rad)*50
	y2 := 50 + math.Sin(rad)*50
	fmt.Fprintf(&b, `<defs><linearGradient id="bg" x1="%.1f%%" y1="%.1f%%" x2="%.1f%%" y2="%.1f%%">`, x1, y1, x2, y2)
	fmt.Fprintf(&b, `<stop offset="0%%" stop-color="%s"/>`, bgColors[0])
	fmt.Fprintf(&b, `<stop offset="100%%" stop-color="%s"/>`, bgColors[1])
	b.WriteString(`</linearGradient></defs>`)
	b.WriteString("\n")
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="url(#bg)"/>`, Width, Height)
	b.WriteString("\n")

	// Overlapping geometric shapes (circles, ellipses, rounded rects).
	numShapes := 5 + nextInt(5) // 5–9 shapes
	for i := 0; i < numShapes; i++ {
		color := shapeColors[nextInt(len(shapeColors))]
		opacity := nextRange(0.06, 0.22)
		cx := nextRange(-100, float64(Width)+100)
		cy := nextRange(-80, float64(Height)+80)

		shapeType := nextInt(3)
		switch shapeType {
		case 0: // circle
			r := nextRange(60, 280)
			fmt.Fprintf(&b, `<circle cx="%.0f" cy="%.0f" r="%.0f" fill="%s" opacity="%.2f"/>`, cx, cy, r, color, opacity)
		case 1: // ellipse
			rx := nextRange(80, 320)
			ry := nextRange(50, 200)
			fmt.Fprintf(&b, `<ellipse cx="%.0f" cy="%.0f" rx="%.0f" ry="%.0f" fill="%s" opacity="%.2f"/>`, cx, cy, rx, ry, color, opacity)
		case 2: // rounded rect
			w := nextRange(120, 400)
			rh := nextRange(80, 300)
			rx := nextRange(10, 40)
			fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="%.0f" height="%.0f" rx="%.0f" fill="%s" opacity="%.2f"/>`, cx-w/2, cy-rh/2, w, rh, rx, color, opacity)
		}
		b.WriteString("\n")
	}

	// Subtle noise overlay via small translucent dots.
	numDots := 8 + nextInt(8) // 8–15
	for i := 0; i < numDots; i++ {
		dx := nextRange(0, float64(Width))
		dy := nextRange(0, float64(Height))
		dr := nextRange(2, 8)
		op := nextRange(0.05, 0.15)
		color := shapeColors[nextInt(len(shapeColors))]
		fmt.Fprintf(&b, `<circle cx="%.0f" cy="%.0f" r="%.0f" fill="%s" opacity="%.2f"/>`, dx, dy, dr, color, op)
		b.WriteString("\n")
	}

	b.WriteString("</svg>")
	return b.String()
}

// Filename returns a deterministic filename for the placeholder SVG
// based on the title hash, suitable for use in the public dir.
func Filename(title string) string {
	h := sha256.Sum256([]byte(title))
	return fmt.Sprintf("placeholder-%x.svg", h[:8])
}
