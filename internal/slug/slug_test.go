package slug

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello, World!", "hello-world"},
		{"  Leading Trailing  ", "leading-trailing"},
		{"Múltiple áccéntös", "múltiple-áccéntös"},
		{"", "untitled"},
		{"   ", "untitled"},
		{"!!!---!!!", "untitled"},
		{"a--b", "a-b"},
		{"hello_world", "hello-world"},
		{"one two  three", "one-two-three"},
		{"CamelCase", "camelcase"},
		{"123 numbers", "123-numbers"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := Slugify(tc.input); got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDeriveUsesSlug(t *testing.T) {
	fm := map[string]any{"slug": "Custom Slug"}
	if got := Derive(fm, "file.md"); got != "custom-slug" {
		t.Fatalf("expected custom-slug, got %s", got)
	}
}

func TestDeriveUsesTitle(t *testing.T) {
	fm := map[string]any{"title": "My Great Post"}
	if got := Derive(fm, "file.md"); got != "my-great-post" {
		t.Fatalf("expected my-great-post, got %q", got)
	}
}

func TestDeriveSlugTakesPrecedenceOverTitle(t *testing.T) {
	fm := map[string]any{"slug": "explicit", "title": "Title"}
	if got := Derive(fm, "file.md"); got != "explicit" {
		t.Fatalf("expected slug to win over title, got %q", got)
	}
}

func TestDeriveFallsBackToFilename(t *testing.T) {
	fm := map[string]any{"other": "value"}
	if got := Derive(fm, "My Post.md"); got != "my-post" {
		t.Fatalf("expected my-post from filename, got %q", got)
	}
}

func TestDeriveNilFrontmatter(t *testing.T) {
	if got := Derive(nil, "Post Title.md"); got != "post-title" {
		t.Fatalf("expected post-title, got %q", got)
	}
}

func TestStringFrom(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		key  string
		want string
	}{
		{"found", map[string]any{"slug": "hello"}, "slug", "hello"},
		{"trimmed", map[string]any{"slug": "  hello  "}, "slug", "hello"},
		{"missing key", map[string]any{"title": "x"}, "slug", ""},
		{"non-string value", map[string]any{"slug": 42}, "slug", ""},
		{"nil map", nil, "slug", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stringFrom(tc.fm, tc.key); got != tc.want {
				t.Errorf("stringFrom(%v, %q) = %q, want %q", tc.fm, tc.key, got, tc.want)
			}
		})
	}
}
