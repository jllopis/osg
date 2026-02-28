package slug

import "testing"

func BenchmarkSlugify_ASCII(b *testing.B) {
	for b.Loop() {
		Slugify("A Simple English Title With Words")
	}
}

func BenchmarkSlugify_Unicode(b *testing.B) {
	for b.Loop() {
		Slugify("Artículo sobre Optimización de Rendimiento")
	}
}

func BenchmarkSlugify_Special(b *testing.B) {
	for b.Loop() {
		Slugify("Title with --- dashes & special! chars? (yes)")
	}
}

func BenchmarkDerive_FromTitle(b *testing.B) {
	fm := map[string]any{"title": "My Blog Post About Performance"}
	for b.Loop() {
		Derive(fm, "some-file.md")
	}
}

func BenchmarkDerive_FromFilename(b *testing.B) {
	for b.Loop() {
		Derive(nil, "my-existing-slug-file.md")
	}
}
