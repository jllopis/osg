package i18n

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	b := New("")
	if b.DefaultLang() != "es" {
		t.Errorf("expected default lang 'es', got %q", b.DefaultLang())
	}

	b = New("EN")
	if b.DefaultLang() != "en" {
		t.Errorf("expected default lang 'en', got %q", b.DefaultLang())
	}
}

func TestTransFallbackToKey(t *testing.T) {
	b := New("en")
	got := b.Trans("unknown_key")
	if got != "unknown_key" {
		t.Errorf("expected key as fallback, got %q", got)
	}
}

func TestLoadDirAndTrans(t *testing.T) {
	dir := t.TempDir()

	enData := `
featured: Featured
min_read: min read
recent_posts: Recent posts
`
	esData := `
featured: Destacado
min_read: min de lectura
recent_posts: Publicaciones recientes
`

	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), []byte(enData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "es.yaml"), []byte(esData), 0o644); err != nil {
		t.Fatal(err)
	}

	b := New("es")
	if err := b.LoadDir(dir); err != nil {
		t.Fatal(err)
	}

	// Default lang (es) lookup.
	got := b.Trans("featured")
	if got != "Destacado" {
		t.Errorf("Trans(featured) = %q, want %q", got, "Destacado")
	}

	// Explicit lang override.
	got = b.Trans("featured", "en")
	if got != "Featured" {
		t.Errorf("Trans(featured, en) = %q, want %q", got, "Featured")
	}

	// Fallback to default lang when key missing in requested lang.
	got = b.Trans("recent_posts", "fr")
	if got != "Publicaciones recientes" {
		t.Errorf("Trans(recent_posts, fr) = %q, want %q", got, "Publicaciones recientes")
	}

	// Fallback to key when missing everywhere.
	got = b.Trans("nonexistent", "en")
	if got != "nonexistent" {
		t.Errorf("Trans(nonexistent) = %q, want %q", got, "nonexistent")
	}
}

func TestLoadDirOverride(t *testing.T) {
	themeDir := t.TempDir()
	userDir := t.TempDir()

	themeData := `
featured: Theme Featured
back_to_home: Theme Home
`
	userData := `
featured: User Featured
`

	if err := os.WriteFile(filepath.Join(themeDir, "en.yaml"), []byte(themeData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "en.yaml"), []byte(userData), 0o644); err != nil {
		t.Fatal(err)
	}

	b := New("en")
	if err := b.LoadDir(themeDir); err != nil {
		t.Fatal(err)
	}
	if err := b.LoadDir(userDir); err != nil {
		t.Fatal(err)
	}

	// User overrides theme.
	got := b.Trans("featured", "en")
	if got != "User Featured" {
		t.Errorf("Trans(featured) = %q, want %q", got, "User Featured")
	}

	// Theme key not overridden by user is preserved.
	got = b.Trans("back_to_home", "en")
	if got != "Theme Home" {
		t.Errorf("Trans(back_to_home) = %q, want %q", got, "Theme Home")
	}
}

func TestLoadDirMissing(t *testing.T) {
	b := New("en")
	err := b.LoadDir("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("LoadDir on missing dir should not error, got %v", err)
	}
}

func TestLoadDirEmpty(t *testing.T) {
	b := New("en")
	err := b.LoadDir("")
	if err != nil {
		t.Errorf("LoadDir with empty string should not error, got %v", err)
	}
}

func TestHasLang(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), []byte("foo: bar"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := New("en")
	if err := b.LoadDir(dir); err != nil {
		t.Fatal(err)
	}

	if !b.HasLang("en") {
		t.Error("expected HasLang(en) = true")
	}
	if b.HasLang("fr") {
		t.Error("expected HasLang(fr) = false")
	}
}

func TestLanguages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.yaml"), []byte("a: b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "es.yaml"), []byte("a: b"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := New("en")
	if err := b.LoadDir(dir); err != nil {
		t.Fatal(err)
	}

	langs := b.Languages()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

func TestDateFormatEnglish(t *testing.T) {
	d := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "January 2, 2006", "en")
	want := "March 15, 2025"
	if got != want {
		t.Errorf("DateFormat(en) = %q, want %q", got, want)
	}
}

func TestDateFormatSpanish(t *testing.T) {
	d := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "January 2, 2006", "es")
	want := "marzo 15, 2025"
	if got != want {
		t.Errorf("DateFormat(es) = %q, want %q", got, want)
	}
}

func TestDateFormatShortMonth(t *testing.T) {
	d := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "Jan 2, 2006", "es")
	want := "mar 15, 2025"
	if got != want {
		t.Errorf("DateFormat short(es) = %q, want %q", got, want)
	}
}

func TestDateFormatEmptyLang(t *testing.T) {
	d := time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "Jan 2, 2006", "")
	want := "Jun 1, 2025"
	if got != want {
		t.Errorf("DateFormat('') = %q, want %q", got, want)
	}
}

func TestDateFormatUnknownLang(t *testing.T) {
	d := time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "January 2, 2006", "xx")
	want := "January 10, 2025"
	if got != want {
		t.Errorf("DateFormat(xx) = %q, want %q", got, want)
	}
}

func TestDateFormatNumericOnly(t *testing.T) {
	d := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
	got := DateFormat(d, "2006-01-02", "es")
	want := "2025-03-15"
	if got != want {
		t.Errorf("DateFormat numeric(es) = %q, want %q", got, want)
	}
}
