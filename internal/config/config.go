package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type LoggingConfig struct {
	Level  string `koanf:"level" yaml:"level"`
	Format string `koanf:"format" yaml:"format"`
}

// AIConfig holds settings for LLM-based summary generation via Kairos.
type AIConfig struct {
	// Provider is the LLM provider name: "gemini" (default), "anthropic",
	// "openai", "qwen", or "ollama".
	Provider string `koanf:"provider" yaml:"provider"`
	// Model is the model identifier (e.g. "gemini-3-flash-preview").
	// If empty, the provider's default model is used.
	Model string `koanf:"model" yaml:"model"`
	// APIKey is the API key for the provider. If empty, the provider's
	// default environment variable is used (e.g. GOOGLE_API_KEY for gemini).
	APIKey string `koanf:"api_key" yaml:"api_key"`
	// BaseURL overrides the provider's default API endpoint.
	// Mainly useful for ollama (e.g. "http://localhost:11434") or
	// custom OpenAI-compatible endpoints.
	BaseURL string `koanf:"base_url" yaml:"base_url"`
	// SystemPrompt is the system instruction sent to the LLM.
	// If empty a sensible default is used.
	SystemPrompt string `koanf:"system_prompt" yaml:"system_prompt"`
	// Timeout is the per-request timeout in seconds. Default: 30.
	Timeout int `koanf:"timeout" yaml:"timeout"`
	// Concurrency is the max number of parallel LLM requests. Default: 3.
	Concurrency int `koanf:"concurrency" yaml:"concurrency"`
}

type Config struct {
	BaseURL           string           `koanf:"base_url" yaml:"base_url"`
	SiteTitle         string           `koanf:"site_title" yaml:"site_title"`
	SiteDescription   string           `koanf:"site_description" yaml:"site_description"`
	Theme             string           `koanf:"theme" yaml:"theme"`
	ColorScheme       string           `koanf:"color_scheme" yaml:"color_scheme"`
	VaultPath         string           `koanf:"vault_path" yaml:"vault_path"`
	ContentDir        string           `koanf:"content_dir" yaml:"content_dir"`
	PublicDir         string           `koanf:"public_dir" yaml:"public_dir"`
	TemplatesDir      string           `koanf:"templates_dir" yaml:"templates_dir"`
	StaticDir         string           `koanf:"static_dir" yaml:"static_dir"`
	ThemesDir         string           `koanf:"themes_dir" yaml:"themes_dir"`
	PluginsDir        string           `koanf:"plugins_dir" yaml:"plugins_dir"`
	PluginsEnabled    []string         `koanf:"plugins_enabled" yaml:"plugins_enabled"`
	SassDir           string           `koanf:"sass_dir" yaml:"sass_dir"`
	ContentLayout     string           `koanf:"content_layout" yaml:"content_layout"`
	IncludeDrafts     bool             `koanf:"include_drafts" yaml:"include_drafts"`
	CompileSass       bool             `koanf:"compile_sass" yaml:"compile_sass"`
	TUIPrefix         string           `koanf:"tui_prefix" yaml:"tui_prefix"`
	TUIPrefixMs       int              `koanf:"tui_prefix_ms" yaml:"tui_prefix_ms"`
	ServeWatch        bool             `koanf:"serve_watch" yaml:"serve_watch"`
	ServeReload       bool             `koanf:"serve_live_reload" yaml:"serve_live_reload"`
	ServeDebounce     int              `koanf:"serve_debounce_ms" yaml:"serve_debounce_ms"`
	BuildIncremental  bool             `koanf:"build_incremental" yaml:"build_incremental"`
	BuildCacheDir     string           `koanf:"build_cache_dir" yaml:"build_cache_dir"`
	CleanPublic       bool             `koanf:"clean_public" yaml:"clean_public"`
	SummaryStrategy   string           `koanf:"summary_strategy" yaml:"summary_strategy"`
	SiteFeed          bool             `koanf:"site_feed" yaml:"site_feed"`
	SiteFeedLimit     int              `koanf:"site_feed_limit" yaml:"site_feed_limit"`
	ImageOptimization bool             `koanf:"image_optimization" yaml:"image_optimization"`
	ImageQuality      int              `koanf:"image_quality" yaml:"image_quality"`
	ImageWidths       []int            `koanf:"image_widths" yaml:"image_widths"`
	DefaultLanguage   string           `koanf:"default_language" yaml:"default_language"`
	DoctorProfile     string           `koanf:"doctor_profile" yaml:"doctor_profile"`
	AI                AIConfig         `koanf:"ai" yaml:"ai"`
	Logging           LoggingConfig    `koanf:"logging" yaml:"logging"`
	Taxonomies        []TaxonomyConfig `koanf:"taxonomies" yaml:"taxonomies"`
}

type TaxonomyConfig struct {
	Name         string `koanf:"name" yaml:"name"`
	PaginateBy   int    `koanf:"paginate_by" yaml:"paginate_by"`
	PaginatePath string `koanf:"paginate_path" yaml:"paginate_path"`
	Feed         bool   `koanf:"feed" yaml:"feed"`
	Render       bool   `koanf:"render" yaml:"render"`
}

func Default() Config {
	return Config{
		BaseURL:           "",
		SiteTitle:         "OSG",
		SiteDescription:   "",
		Theme:             "default",
		ColorScheme:       "auto",
		ContentDir:        "content",
		PublicDir:         "public",
		TemplatesDir:      "templates",
		StaticDir:         "static",
		ThemesDir:         "themes",
		PluginsDir:        "plugins",
		PluginsEnabled:    []string{},
		SassDir:           "sass",
		ContentLayout:     "{date}/{slug}",
		IncludeDrafts:     false,
		CompileSass:       false,
		TUIPrefix:         "space",
		TUIPrefixMs:       600,
		ServeWatch:        true,
		ServeReload:       true,
		ServeDebounce:     300,
		BuildIncremental:  true,
		BuildCacheDir:     ".osg/cache",
		CleanPublic:       true,
		SummaryStrategy:   "auto",
		SiteFeed:          true,
		SiteFeedLimit:     20,
		ImageOptimization: true,
		ImageQuality:      80,
		ImageWidths:       []int{640, 1200},
		DefaultLanguage:   "es",
		DoctorProfile:     "dev",
		AI: AIConfig{
			Provider:    "gemini",
			Model:       "gemini-3-flash-preview",
			Timeout:     30,
			Concurrency: 3,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	k := koanf.New(".")

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
				return cfg, fmt.Errorf("load config: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return cfg, fmt.Errorf("stat config: %w", err)
		}
	}

	if err := k.Load(env.Provider("OSG_", ".", envKeyTransform), nil); err != nil {
		return cfg, fmt.Errorf("load env: %w", err)
	}

	if len(k.Keys()) > 0 {
		if err := k.Unmarshal("", &cfg); err != nil {
			return cfg, fmt.Errorf("unmarshal config: %w", err)
		}
	}

	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = Default().Theme
	}

	// Normalise and validate color_scheme.
	cfg.ColorScheme = strings.ToLower(strings.TrimSpace(cfg.ColorScheme))
	switch cfg.ColorScheme {
	case "auto", "light", "dark":
		// valid
	case "":
		cfg.ColorScheme = "auto"
	default:
		return cfg, fmt.Errorf("invalid color_scheme %q: must be auto, light, or dark", cfg.ColorScheme)
	}

	// Normalise and validate summary_strategy.
	cfg.SummaryStrategy = strings.ToLower(strings.TrimSpace(cfg.SummaryStrategy))
	switch cfg.SummaryStrategy {
	case "auto", "manual", "ai":
		// valid
	case "":
		cfg.SummaryStrategy = "auto"
	default:
		return cfg, fmt.Errorf("invalid summary_strategy %q: must be auto, manual, or ai", cfg.SummaryStrategy)
	}

	// Normalise default_language.
	cfg.DefaultLanguage = strings.ToLower(strings.TrimSpace(cfg.DefaultLanguage))
	if cfg.DefaultLanguage == "" {
		cfg.DefaultLanguage = "es"
	}

	// Normalise and validate AI config.
	cfg.AI.Provider = strings.ToLower(strings.TrimSpace(cfg.AI.Provider))
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = "gemini"
	}
	switch cfg.AI.Provider {
	case "gemini", "anthropic", "openai", "qwen", "ollama":
		// valid
	default:
		return cfg, fmt.Errorf("invalid ai.provider %q: must be gemini, anthropic, openai, qwen, or ollama", cfg.AI.Provider)
	}
	if cfg.AI.Timeout <= 0 {
		cfg.AI.Timeout = 30
	}
	if cfg.AI.Concurrency <= 0 {
		cfg.AI.Concurrency = 3
	}

	return cfg, nil
}

func ResolveVaultPath(cfg Config) (string, error) {
	if cfg.VaultPath != "" {
		return cfg.VaultPath, nil
	}
	return "", fmt.Errorf("vault path not configured (use --vault-path)")
}

func DefaultConfigYAML() string {
	return strings.TrimSpace(`
# =============================================================================
# OSG — Obsidian Site Generator
# Full configuration reference. Every key shown here can also be set via
# environment variables with the OSG_ prefix (e.g. OSG_SITE_TITLE).
# For nested keys use double underscores (e.g. OSG_AI__PROVIDER=gemini).
# =============================================================================

# -----------------------------------------------------------------------------
# Site identity
# -----------------------------------------------------------------------------
# site_title: Displayed in the header, browser tab, and meta tags.
# site_description: Used in the HTML <meta name="description"> and OpenGraph.
# base_url: Absolute URL of the deployed site (e.g. "https://blog.example.com").
#           Leave empty for local/relative links.
# default_language: BCP-47 language code (e.g. "es", "en", "fr") used as the
#                   default for template translations and date localisation.
#                   Each page can override this via the "lang" front-matter field.
site_title: "OSG"
site_description: ""
base_url: ""
default_language: es

# -----------------------------------------------------------------------------
# Content
# -----------------------------------------------------------------------------
# vault_path: Path to the Obsidian vault to import from (used by "osg import").
# content_dir: Directory where Markdown content lives.
# content_layout: URL pattern for pages. Placeholders: {date}, {slug}, {title}.
# include_drafts: Render pages with "draft: true" in front-matter.
vault_path: ""
content_dir: content
content_layout: "{date}/{slug}"
include_drafts: false

# -----------------------------------------------------------------------------
# Theme & appearance
# -----------------------------------------------------------------------------
# theme: Active theme name (must match a subdirectory of themes_dir).
# themes_dir: Directory that contains installed themes.
# color_scheme: Color scheme for the default theme.
#   "auto"  — follows the visitor's OS preference (prefers-color-scheme).
#   "light" — always light mode.
#   "dark"  — always dark mode.
theme: default
themes_dir: themes
color_scheme: auto

# -----------------------------------------------------------------------------
# Output
# -----------------------------------------------------------------------------
# public_dir: Directory where the generated site is written.
# clean_public: Remove public_dir before a full build.
public_dir: public
clean_public: true

# -----------------------------------------------------------------------------
# Summaries
# -----------------------------------------------------------------------------
# summary_strategy: How to generate page summaries for listings.
#   "auto"   — auto-extract first sentences from markdown when frontmatter
#              has no summary/description/excerpt field (default).
#   "manual" — only use explicit frontmatter summaries.
#   "ai"     — generate via LLM using Kairos (see ai section below).
summary_strategy: auto

# -----------------------------------------------------------------------------
# AI summary generation (Kairos)
# -----------------------------------------------------------------------------
# Only used when summary_strategy is "ai".
#
# ai.provider: LLM provider to use. Supported: "gemini" (default),
#              "anthropic", "openai", "qwen", "ollama".
# ai.model: Model identifier. Defaults depend on the provider:
#            gemini -> "gemini-3-flash-preview"
#            anthropic -> "claude-haiku-4-20250514"
#            openai -> "gpt-5-mini"
#            qwen -> "qwen-turbo"
#            ollama -> set explicitly (e.g. "llama3.2")
# ai.api_key: API key. If empty the provider's default env var is used:
#             gemini -> GOOGLE_API_KEY or GEMINI_API_KEY
#             anthropic -> ANTHROPIC_API_KEY
#             openai -> OPENAI_API_KEY
#             qwen -> (required, no env var default)
#             ollama -> not needed
# ai.base_url: Override the API endpoint. Useful for ollama
#              ("http://localhost:11434") or custom proxies.
# ai.system_prompt: Custom system instruction for the LLM.
#                   Default: "Summarize the following blog post in 2-3
#                   concise sentences for use as a preview excerpt."
# ai.timeout: Per-request timeout in seconds (default: 30).
# ai.concurrency: Max parallel LLM requests (default: 3).
ai:
  provider: gemini
  model: "gemini-3-flash-preview"
  # api_key: ""
  # base_url: ""
  # system_prompt: ""
  timeout: 30
  concurrency: 3

# -----------------------------------------------------------------------------
# Site feed
# -----------------------------------------------------------------------------
# site_feed: Generate a site-wide RSS and Atom feed at the root (/atom.xml,
#            /rss.xml) containing the most recent pages across all sections.
# site_feed_limit: Maximum number of entries in the site feed (0 = all pages).
site_feed: true
site_feed_limit: 20

# -----------------------------------------------------------------------------
# Templates & static assets
# -----------------------------------------------------------------------------
# templates_dir: User-level template overrides (merged on top of the theme).
# static_dir: Extra static files copied as-is to public_dir.
templates_dir: templates
static_dir: static

# -----------------------------------------------------------------------------
# Plugins
# -----------------------------------------------------------------------------
# plugins_dir: Directory containing .wasm plugin files.
# plugins_enabled: List of plugin names to activate (without .wasm extension).
plugins_dir: plugins
plugins_enabled: []

# -----------------------------------------------------------------------------
# Sass
# -----------------------------------------------------------------------------
# sass_dir: Directory with .scss files to compile.
# compile_sass: Enable Sass compilation.
sass_dir: sass
compile_sass: false

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------
# build_incremental: Only re-render changed pages (speeds up rebuilds).
# build_cache_dir: Where build cache data is stored.
build_incremental: true
build_cache_dir: .osg/cache

# -----------------------------------------------------------------------------
# Image optimization
# -----------------------------------------------------------------------------
# image_optimization: Generate responsive image variants (resized JPEG + WebP).
#   When enabled, raster images (jpg, jpeg, png) in public/ are processed
#   after asset copy: resized to each configured width and, if the cwebp
#   binary is available, also converted to WebP.  Templates automatically
#   emit <picture> elements with srcset when variants exist.
# image_quality: Encoding quality for JPEG and WebP variants (1-100).
# image_widths: List of pixel widths to generate (images smaller than a width
#               are skipped — no upscaling).  The original is always kept.
image_optimization: true
image_quality: 80
image_widths: [640, 1200]

# -----------------------------------------------------------------------------
# Dev server (osg serve)
# -----------------------------------------------------------------------------
# serve_watch: Watch files for changes and rebuild automatically.
# serve_live_reload: Inject live-reload script into served pages.
# serve_debounce_ms: Milliseconds to wait before triggering a rebuild after
#                    a file change (avoids rapid successive builds).
serve_watch: true
serve_live_reload: true
serve_debounce_ms: 300

# -----------------------------------------------------------------------------
# TUI
# -----------------------------------------------------------------------------
# tui_prefix: Key used as prefix in the interactive TUI ("space" or "ctrl").
# tui_prefix_ms: Milliseconds to wait for a second key after prefix.
tui_prefix: space
tui_prefix_ms: 600

# -----------------------------------------------------------------------------
# Diagnostics
# -----------------------------------------------------------------------------
# doctor_profile: Profile for "osg doctor" checks ("dev" or "prod").
doctor_profile: dev

# -----------------------------------------------------------------------------
# Logging
# -----------------------------------------------------------------------------
# level: Minimum log level (debug, info, warn, error).
# format: Log output format ("json" or "text").
logging:
  level: info
  format: json

# -----------------------------------------------------------------------------
# Taxonomies
# -----------------------------------------------------------------------------
# Define content groupings derived from front-matter fields.
# Each taxonomy generates listing pages and per-term pages.
#
# taxonomies:
#   - name: tags            # front-matter field name
#     paginate_by: 10       # items per page (0 = no pagination)
#     paginate_path: page   # URL segment for paginated pages
#     feed: true            # generate RSS/Atom feed for this taxonomy
#     render: true          # generate HTML pages
#   - name: area
#     paginate_by: 10
#     paginate_path: page
#     feed: false
#     render: true
`) + "\n"
}

func envKeyTransform(key string) string {
	key = strings.TrimPrefix(key, "OSG_")
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "__", ".")
	return key
}
