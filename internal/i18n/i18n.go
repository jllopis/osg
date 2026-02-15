// Package i18n provides translation loading and lookup for OSG template strings.
//
// Translations are flat key-value YAML files stored in i18n/{lang}.yaml.
// Files are loaded in layers: theme dir first, then user dir (user overrides theme).
// The Bundle holds all loaded languages and provides a Trans(key, lang) method
// that falls back to the default language, then to returning the key itself.
package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Bundle holds all loaded translations keyed by language code.
type Bundle struct {
	defaultLang  string
	translations map[string]map[string]string
}

// New creates an empty Bundle with the given default language.
func New(defaultLang string) *Bundle {
	defaultLang = strings.TrimSpace(strings.ToLower(defaultLang))
	if defaultLang == "" {
		defaultLang = "es"
	}
	return &Bundle{
		defaultLang:  defaultLang,
		translations: make(map[string]map[string]string),
	}
}

// LoadDir loads all .yaml files from dir into the bundle.
// Each file should be named {lang}.yaml and contain a flat string-to-string map.
// Files loaded later (e.g. user dir after theme dir) override existing keys.
func (b *Bundle) LoadDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Missing dir is fine — no translations to load.
		}
		return fmt.Errorf("i18n stat dir: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("i18n read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		lang := strings.TrimSuffix(name, ext)
		lang = strings.ToLower(strings.TrimSpace(lang))
		if lang == "" {
			continue
		}

		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("i18n read %s: %w", path, err)
		}

		var translations map[string]string
		if err := yaml.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("i18n parse %s: %w", path, err)
		}

		if translations == nil {
			continue
		}

		if b.translations[lang] == nil {
			b.translations[lang] = make(map[string]string, len(translations))
		}
		for k, v := range translations {
			b.translations[lang][k] = v
		}
	}

	return nil
}

// Trans looks up a translation key for the given language.
// Fallback order: lang -> defaultLang -> key itself.
// If lang is empty, the default language is used directly.
func (b *Bundle) Trans(key string, lang ...string) string {
	l := ""
	if len(lang) > 0 {
		l = strings.TrimSpace(strings.ToLower(lang[0]))
	}

	// Try requested language first.
	if l != "" {
		if translations, ok := b.translations[l]; ok {
			if val, ok := translations[key]; ok {
				return val
			}
		}
	}

	// Fall back to default language.
	if l != b.defaultLang {
		if translations, ok := b.translations[b.defaultLang]; ok {
			if val, ok := translations[key]; ok {
				return val
			}
		}
	}

	// Last resort: return the key itself.
	return key
}

// DefaultLang returns the default language code.
func (b *Bundle) DefaultLang() string {
	return b.defaultLang
}

// HasLang reports whether the bundle has any translations for the given language.
func (b *Bundle) HasLang(lang string) bool {
	lang = strings.TrimSpace(strings.ToLower(lang))
	_, ok := b.translations[lang]
	return ok
}

// Languages returns the list of loaded language codes.
func (b *Bundle) Languages() []string {
	langs := make([]string, 0, len(b.translations))
	for lang := range b.translations {
		langs = append(langs, lang)
	}
	return langs
}

// Localized month names, keyed by language code.
// Only languages that differ from Go's English default need entries.
var localizedMonths = map[string][12]string{
	"es": {
		"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	},
	"fr": {
		"janvier", "f\u00e9vrier", "mars", "avril", "mai", "juin",
		"juillet", "ao\u00fbt", "septembre", "octobre", "novembre", "d\u00e9cembre",
	},
	"de": {
		"Januar", "Februar", "M\u00e4rz", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember",
	},
	"pt": {
		"janeiro", "fevereiro", "mar\u00e7o", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
	},
	"it": {
		"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno",
		"luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre",
	},
	"ca": {
		"gener", "febrer", "mar\u00e7", "abril", "maig", "juny",
		"juliol", "agost", "setembre", "octubre", "novembre", "desembre",
	},
}

var localizedShortMonths = map[string][12]string{
	"es": {"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"},
	"fr": {"janv", "f\u00e9vr", "mars", "avr", "mai", "juin", "juil", "ao\u00fbt", "sept", "oct", "nov", "d\u00e9c"},
	"de": {"Jan", "Feb", "M\u00e4r", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"},
	"pt": {"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"},
	"it": {"gen", "feb", "mar", "apr", "mag", "giu", "lug", "ago", "set", "ott", "nov", "dic"},
	"ca": {"gen", "feb", "mar\u00e7", "abr", "maig", "juny", "jul", "ag", "set", "oct", "nov", "des"},
}

// DateFormat formats a time.Time using a Go layout string, replacing English
// month names with locale-aware equivalents. For "en" or unknown languages,
// it simply uses Go's time.Format.
func DateFormat(t time.Time, layout string, lang string) string {
	formatted := t.Format(layout)
	lang = strings.TrimSpace(strings.ToLower(lang))

	if lang == "" || lang == "en" {
		return formatted
	}

	month := t.Month()

	// Replace long month name first, then short (order matters to avoid
	// partial replacements like "January" -> "enerouary").
	if months, ok := localizedMonths[lang]; ok {
		enLong := t.Format("January")
		localLong := months[month-1]
		formatted = strings.ReplaceAll(formatted, enLong, localLong)
	}

	if months, ok := localizedShortMonths[lang]; ok {
		enShort := t.Format("Jan")
		localShort := months[month-1]
		formatted = strings.ReplaceAll(formatted, enShort, localShort)
	}

	return formatted
}
