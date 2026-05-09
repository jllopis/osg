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

// LanguageConfig describes a secondary (non-default) language.  The default
// language is specified by DefaultLanguage and needs no entry here.
type LanguageConfig struct {
	// Code is the BCP-47 language code (e.g. "en", "fr", "de").
	Code string `koanf:"code" yaml:"code"`
	// Label is the human-readable name shown in the language switcher
	// (e.g. "English", "Francais").
	Label string `koanf:"label" yaml:"label"`
}

type Config struct {
	BaseURL            string                    `koanf:"base_url" yaml:"base_url"`
	SiteTitle          string                    `koanf:"site_title" yaml:"site_title"`
	SiteDescription    string                    `koanf:"site_description" yaml:"site_description"`
	Theme              string                    `koanf:"theme" yaml:"theme"`
	Logo               string                    `koanf:"logo" yaml:"logo"`
	Favicon            string                    `koanf:"favicon" yaml:"favicon"`
	ColorScheme        string                    `koanf:"color_scheme" yaml:"color_scheme"`
	VaultPath          string                    `koanf:"vault_path" yaml:"vault_path"`
	ContentDir         string                    `koanf:"content_dir" yaml:"content_dir"`
	PublicDir          string                    `koanf:"public_dir" yaml:"public_dir"`
	TemplatesDir       string                    `koanf:"templates_dir" yaml:"templates_dir"`
	StaticDir          string                    `koanf:"static_dir" yaml:"static_dir"`
	ThemesDir          string                    `koanf:"themes_dir" yaml:"themes_dir"`
	PluginsDir         string                    `koanf:"plugins_dir" yaml:"plugins_dir"`
	PluginsEnabled     []string                  `koanf:"plugins_enabled" yaml:"plugins_enabled"`
	PluginTimeout      int                       `koanf:"plugin_timeout" yaml:"plugin_timeout"`
	SassDir            string                    `koanf:"sass_dir" yaml:"sass_dir"`
	ContentLayout      string                    `koanf:"content_layout" yaml:"content_layout"`
	IncludeDrafts      bool                      `koanf:"include_drafts" yaml:"include_drafts"`
	CompileSass        bool                      `koanf:"compile_sass" yaml:"compile_sass"`
	TUIPrefix          string                    `koanf:"tui_prefix" yaml:"tui_prefix"`
	TUIPrefixMs        int                       `koanf:"tui_prefix_ms" yaml:"tui_prefix_ms"`
	TUILogModifier     string                    `koanf:"tui_log_modifier" yaml:"tui_log_modifier"`
	ServeWatch         bool                      `koanf:"serve_watch" yaml:"serve_watch"`
	ServeReload        bool                      `koanf:"serve_live_reload" yaml:"serve_live_reload"`
	ServeDebounce      int                       `koanf:"serve_debounce_ms" yaml:"serve_debounce_ms"`
	BuildIncremental   bool                      `koanf:"build_incremental" yaml:"build_incremental"`
	BuildCacheDir      string                    `koanf:"build_cache_dir" yaml:"build_cache_dir"`
	CleanPublic        bool                      `koanf:"clean_public" yaml:"clean_public"`
	SummaryStrategy    string                    `koanf:"summary_strategy" yaml:"summary_strategy"`
	SiteFeed           bool                      `koanf:"site_feed" yaml:"site_feed"`
	SiteFeedLimit      int                       `koanf:"site_feed_limit" yaml:"site_feed_limit"`
	SectionFeeds       bool                      `koanf:"section_feeds" yaml:"section_feeds"`
	PostsPerPage       int                       `koanf:"posts_per_page" yaml:"posts_per_page"`
	ImageOptimization  bool                      `koanf:"image_optimization" yaml:"image_optimization"`
	ImageQuality       int                       `koanf:"image_quality" yaml:"image_quality"`
	ImageWidths        []int                     `koanf:"image_widths" yaml:"image_widths"`
	Lightbox           bool                      `koanf:"lightbox" yaml:"lightbox"`
	Sharing            bool                      `koanf:"sharing" yaml:"sharing"`
	Breadcrumbs        bool                      `koanf:"breadcrumbs" yaml:"breadcrumbs"`
	Math               bool                      `koanf:"math" yaml:"math"`
	Minify             bool                      `koanf:"minify" yaml:"minify"`
	NavTaxonomy        string                    `koanf:"nav_taxonomy" yaml:"nav_taxonomy"`
	DefaultLanguage    string                    `koanf:"default_language" yaml:"default_language"`
	Languages          []LanguageConfig          `koanf:"languages" yaml:"languages"`
	DefaultEditor      string                    `koanf:"default_editor" yaml:"default_editor"`
	NewNotesDir        string                    `koanf:"new_notes_dir" yaml:"new_notes_dir"`
	DoctorProfile      string                    `koanf:"doctor_profile" yaml:"doctor_profile"`
	Author             string                    `koanf:"author" yaml:"author"`
	AuthorBio          string                    `koanf:"author_bio" yaml:"author_bio"`
	AuthorAvatar       string                    `koanf:"author_avatar" yaml:"author_avatar"`
	AuthorURL          string                    `koanf:"author_url" yaml:"author_url"`
	ThemeColorLight    string                    `koanf:"theme_color_light" yaml:"theme_color_light"`
	ThemeColorDark     string                    `koanf:"theme_color_dark" yaml:"theme_color_dark"`
	Organization       OrganizationConfig        `koanf:"organization" yaml:"organization"`
	Robots             RobotsConfig              `koanf:"robots" yaml:"robots"`
	Social             map[string]string         `koanf:"social" yaml:"social"`
	SidebarWidgets     []string                  `koanf:"sidebar_widgets" yaml:"sidebar_widgets"`
	NewsletterAction   string                    `koanf:"newsletter_action" yaml:"newsletter_action"`
	Copyright          string                    `koanf:"copyright" yaml:"copyright"`
	License            string                    `koanf:"license" yaml:"license"`
	AI                 AIConfig                  `koanf:"ai" yaml:"ai"`
	Logging            LoggingConfig             `koanf:"logging" yaml:"logging"`
	Taxonomies         []TaxonomyConfig          `koanf:"taxonomies" yaml:"taxonomies"`
	Deploy             DeployConfig              `koanf:"deploy" yaml:"deploy"`
	Interactions       InteractionsConfig        `koanf:"interactions" yaml:"interactions"`
	UI                 UIConfig                  `koanf:"ui" yaml:"ui"`
	Webhooks           []WebhookConfig           `koanf:"webhooks" yaml:"webhooks"`
	Analytics          bool                      `koanf:"analytics" yaml:"analytics"`
	AnalyticsProviders []AnalyticsProviderConfig `koanf:"analytics_providers" yaml:"analytics_providers"`
	HeadExtra          string                    `koanf:"head_extra" yaml:"head_extra"`
	BodyExtra          string                    `koanf:"body_extra" yaml:"body_extra"`
}

// OrganizationConfig describes the publishing organization for schema.org
// Organization JSON-LD on the homepage. When Name is empty no schema is emitted.
type OrganizationConfig struct {
	Name   string   `koanf:"name" yaml:"name"`
	URL    string   `koanf:"url" yaml:"url"`
	Logo   string   `koanf:"logo" yaml:"logo"`
	SameAs []string `koanf:"same_as" yaml:"same_as"`
}

// RobotsConfig holds configurable directives for the generated robots.txt.
// Disallow paths are emitted under the wildcard User-agent. Extra is appended
// verbatim to the file (e.g. additional User-agent rules).
type RobotsConfig struct {
	Disallow   []string `koanf:"disallow" yaml:"disallow"`
	CrawlDelay int      `koanf:"crawl_delay" yaml:"crawl_delay"`
	Extra      string   `koanf:"extra" yaml:"extra"`
}

// AnalyticsProviderConfig describes a third-party analytics provider.
type AnalyticsProviderConfig struct {
	// Provider name: "cloudflare", "google", "plausible", "fathom".
	Provider string `koanf:"provider" yaml:"provider"`
	// Token is the site token (Cloudflare Web Analytics, Fathom).
	Token string `koanf:"token" yaml:"token"`
	// TrackingID is the measurement ID (Google GA4: "G-XXXXXXX").
	TrackingID string `koanf:"tracking_id" yaml:"tracking_id"`
	// Domain is the site domain (Plausible).
	Domain string `koanf:"domain" yaml:"domain"`
}

// WebhookConfig defines a single webhook endpoint.
type WebhookConfig struct {
	URL    string   `koanf:"url" yaml:"url"`
	Events []string `koanf:"events" yaml:"events"`
	Secret string   `koanf:"secret" yaml:"secret"`
}

type TaxonomyConfig struct {
	Name         string   `koanf:"name" yaml:"name"`
	PaginateBy   int      `koanf:"paginate_by" yaml:"paginate_by"`
	PaginatePath string   `koanf:"paginate_path" yaml:"paginate_path"`
	Feed         bool     `koanf:"feed" yaml:"feed"`
	Render       bool     `koanf:"render" yaml:"render"`
	ExcludeTerms []string `koanf:"exclude_terms" yaml:"exclude_terms"`
}

// DeployConfig holds deployment settings.
type DeployConfig struct {
	// Provider is the deployment target: "cloudflare", "rsync", or "s3".
	Provider string `koanf:"provider" yaml:"provider"`
	// Cloudflare configures Cloudflare Pages/Workers deployment.
	Cloudflare map[string]any `koanf:"cloudflare" yaml:"cloudflare"`
	// Rsync configures rsync over SSH deployment.
	Rsync map[string]any `koanf:"rsync" yaml:"rsync"`
	// S3 configures S3-compatible storage deployment.
	S3 map[string]any `koanf:"s3" yaml:"s3"`
}

// InteractionsConfig holds settings for the page interactions API (views, likes).
type InteractionsConfig struct {
	// Enabled activates the interactions feature (view counts, likes/dislikes).
	Enabled bool `koanf:"enabled" yaml:"enabled"`
	// APIURL is the URL of the interactions API as seen by the browser.
	// For development with `osg serve --api` this is typically "" (same origin).
	// For production set the full URL (e.g. "https://api.mysite.com").
	APIURL string `koanf:"api_url" yaml:"api_url"`
	// Listen is the address for the standalone `osg api` server (default ":8090").
	Listen string `koanf:"listen" yaml:"listen"`
	// DBPath is the path to the SQLite database file.
	DBPath string `koanf:"db_path" yaml:"db_path"`
	// CORSOrigins is a list of allowed CORS origins for the API.
	CORSOrigins []string `koanf:"cors_origins" yaml:"cors_origins"`
	// ViewDedupHours is the dedup window for unique views per fingerprint
	// per page. Within this window the same fingerprint only counts once
	// as a unique view, though total views are always incremented.
	ViewDedupHours int `koanf:"view_dedup_hours" yaml:"view_dedup_hours"`
	// Comments holds settings for the comment system.
	Comments CommentsConfig `koanf:"comments" yaml:"comments"`
}

// CommentsConfig holds settings for the comment system.
type CommentsConfig struct {
	// Enabled activates the comment system.
	Enabled bool `koanf:"enabled" yaml:"enabled"`
	// DBPath is the path to the comments SQLite database file.
	// Separate from the interactions DB for future portability.
	DBPath string `koanf:"db_path" yaml:"db_path"`
	// AuthSessionDays is how long auth sessions last (in days).
	AuthSessionDays int `koanf:"auth_session_days" yaml:"auth_session_days"`
	// AuthCallbackURL is the base URL for OAuth callbacks
	// (e.g. "https://mysite.com"). If empty, derived from request Host.
	AuthCallbackURL string `koanf:"auth_callback_url" yaml:"auth_callback_url"`
	// Providers lists the OAuth2 providers available for login.
	Providers []AuthProviderConfig `koanf:"providers" yaml:"providers"`
}

// UIConfig holds settings for the `osg ui` web dashboard. The dashboard
// is intended for local use only; defaults to loopback with no auth.
type UIConfig struct {
	// Addr is the bind address for the dashboard (default ":1314").
	Addr string `koanf:"addr" yaml:"addr"`
}

// AuthProviderConfig describes a single OAuth2 provider.
type AuthProviderConfig struct {
	// Provider is the provider name: "github", "google".
	Provider string `koanf:"provider" yaml:"provider"`
	// ClientID is the OAuth2 client ID.
	ClientID string `koanf:"client_id" yaml:"client_id"`
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string `koanf:"client_secret" yaml:"client_secret"`
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
		PluginsEnabled:    []string{"search"},
		PluginTimeout:     5,
		SassDir:           "sass",
		ContentLayout:     "{date}/{slug}",
		IncludeDrafts:     false,
		CompileSass:       false,
		TUIPrefix:         "space",
		TUIPrefixMs:       600,
		TUILogModifier:    "shift",
		ServeWatch:        true,
		ServeReload:       true,
		ServeDebounce:     300,
		BuildIncremental:  true,
		BuildCacheDir:     ".osg/cache",
		CleanPublic:       true,
		SummaryStrategy:   "auto",
		SiteFeed:          true,
		SiteFeedLimit:     20,
		SectionFeeds:      true,
		PostsPerPage:      10,
		ImageOptimization: true,
		ImageQuality:      80,
		ImageWidths:       []int{640, 1200},
		Lightbox:          true,
		Sharing:           true,
		Breadcrumbs:       true,
		Math:              false,
		Minify:            true,
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
		UI: UIConfig{
			Addr: ":1314",
		},
		Interactions: InteractionsConfig{
			Enabled:        false,
			Listen:         ":8090",
			DBPath:         ".osg/interactions.db",
			ViewDedupHours: 24,
			Comments: CommentsConfig{
				Enabled:         false,
				DBPath:          ".osg/comments.db",
				AuthSessionDays: 30,
			},
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

	// Normalise and validate languages.
	for i, lang := range cfg.Languages {
		cfg.Languages[i].Code = strings.ToLower(strings.TrimSpace(lang.Code))
		cfg.Languages[i].Label = strings.TrimSpace(lang.Label)
		if cfg.Languages[i].Code == "" {
			return cfg, fmt.Errorf("languages[%d].code is required", i)
		}
		if cfg.Languages[i].Code == cfg.DefaultLanguage {
			return cfg, fmt.Errorf("languages[%d].code %q duplicates default_language", i, lang.Code)
		}
		if cfg.Languages[i].Label == "" {
			cfg.Languages[i].Label = cfg.Languages[i].Code
		}
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

	// Normalise interactions config.
	cfg.Interactions.Listen = strings.TrimSpace(cfg.Interactions.Listen)
	if cfg.Interactions.Listen == "" {
		cfg.Interactions.Listen = ":8090"
	}
	cfg.Interactions.DBPath = strings.TrimSpace(cfg.Interactions.DBPath)
	if cfg.Interactions.DBPath == "" {
		cfg.Interactions.DBPath = ".osg/interactions.db"
	}
	if cfg.Interactions.ViewDedupHours <= 0 {
		cfg.Interactions.ViewDedupHours = 24
	}
	cfg.Interactions.APIURL = strings.TrimSpace(cfg.Interactions.APIURL)

	// Normalise comments config.
	cfg.Interactions.Comments.DBPath = strings.TrimSpace(cfg.Interactions.Comments.DBPath)
	if cfg.Interactions.Comments.DBPath == "" {
		cfg.Interactions.Comments.DBPath = ".osg/comments.db"
	}
	if cfg.Interactions.Comments.AuthSessionDays <= 0 {
		cfg.Interactions.Comments.AuthSessionDays = 30
	}
	cfg.Interactions.Comments.AuthCallbackURL = strings.TrimSpace(cfg.Interactions.Comments.AuthCallbackURL)
	for i, p := range cfg.Interactions.Comments.Providers {
		cfg.Interactions.Comments.Providers[i].Provider = strings.ToLower(strings.TrimSpace(p.Provider))
		cfg.Interactions.Comments.Providers[i].ClientID = strings.TrimSpace(p.ClientID)
		cfg.Interactions.Comments.Providers[i].ClientSecret = strings.TrimSpace(p.ClientSecret)
		switch cfg.Interactions.Comments.Providers[i].Provider {
		case "github", "google":
			// valid
		default:
			return cfg, fmt.Errorf("interactions.comments.providers[%d].provider %q: must be github or google", i, p.Provider)
		}
		if cfg.Interactions.Comments.Providers[i].ClientID == "" {
			return cfg, fmt.Errorf("interactions.comments.providers[%d].client_id is required", i)
		}
		if cfg.Interactions.Comments.Providers[i].ClientSecret == "" {
			return cfg, fmt.Errorf("interactions.comments.providers[%d].client_secret is required", i)
		}
	}

	// Normalise and validate analytics providers.
	for i, ap := range cfg.AnalyticsProviders {
		cfg.AnalyticsProviders[i].Provider = strings.ToLower(strings.TrimSpace(ap.Provider))
		cfg.AnalyticsProviders[i].Token = strings.TrimSpace(ap.Token)
		cfg.AnalyticsProviders[i].TrackingID = strings.TrimSpace(ap.TrackingID)
		cfg.AnalyticsProviders[i].Domain = strings.TrimSpace(ap.Domain)
		switch cfg.AnalyticsProviders[i].Provider {
		case "cloudflare":
			if cfg.AnalyticsProviders[i].Token == "" {
				return cfg, fmt.Errorf("analytics_providers[%d] (cloudflare): token is required", i)
			}
		case "google":
			if cfg.AnalyticsProviders[i].TrackingID == "" {
				return cfg, fmt.Errorf("analytics_providers[%d] (google): tracking_id is required", i)
			}
		case "plausible":
			if cfg.AnalyticsProviders[i].Domain == "" {
				return cfg, fmt.Errorf("analytics_providers[%d] (plausible): domain is required", i)
			}
		case "fathom":
			if cfg.AnalyticsProviders[i].Token == "" {
				return cfg, fmt.Errorf("analytics_providers[%d] (fathom): token is required", i)
			}
		default:
			return cfg, fmt.Errorf("analytics_providers[%d].provider %q: must be cloudflare, google, plausible, or fathom", i, ap.Provider)
		}
	}

	// Normalise UI config.
	cfg.UI.Addr = strings.TrimSpace(cfg.UI.Addr)
	if cfg.UI.Addr == "" {
		cfg.UI.Addr = ":1314"
	}

	// Normalise sidebar_widgets: trim, drop empties, dedupe, validate names.
	if len(cfg.SidebarWidgets) > 0 {
		seen := map[string]bool{}
		out := make([]string, 0, len(cfg.SidebarWidgets))
		for _, name := range cfg.SidebarWidgets {
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "" || seen[name] {
				continue
			}
			switch name {
			case "author", "newsletter", "popular":
				// valid
			default:
				return cfg, fmt.Errorf("invalid sidebar_widgets entry %q: must be one of author, newsletter, popular", name)
			}
			seen[name] = true
			out = append(out, name)
		}
		cfg.SidebarWidgets = out
	}
	cfg.NewsletterAction = strings.TrimSpace(cfg.NewsletterAction)

	// Normalise and validate tui_log_modifier.
	cfg.TUILogModifier = strings.ToLower(strings.TrimSpace(cfg.TUILogModifier))
	switch cfg.TUILogModifier {
	case "alt", "shift":
		// valid
	case "":
		cfg.TUILogModifier = "shift"
	default:
		return cfg, fmt.Errorf("invalid tui_log_modifier %q: must be alt or shift", cfg.TUILogModifier)
	}

	return cfg, nil
}

// IsMultilingual returns true if secondary languages are configured.
func (c Config) IsMultilingual() bool {
	return len(c.Languages) > 0
}

// AllLanguages returns the default language followed by secondary language
// codes.  For a monolingual site it returns a single-element slice.
func (c Config) AllLanguages() []string {
	langs := []string{c.DefaultLanguage}
	for _, l := range c.Languages {
		langs = append(langs, l.Code)
	}
	return langs
}

// LanguageLabel returns the display label for a language code.
// Returns the code itself if no label is configured.
func (c Config) LanguageLabel(code string) string {
	if code == c.DefaultLanguage {
		return code // default language label is just the code
	}
	for _, l := range c.Languages {
		if l.Code == code {
			return l.Label
		}
	}
	return code
}

func ResolveVaultPath(cfg Config) (string, error) {
	if cfg.VaultPath != "" {
		return ExpandTilde(cfg.VaultPath), nil
	}
	return "", fmt.Errorf("vault path not configured (use --vault-path)")
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
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
# New Post (osg new)
# -----------------------------------------------------------------------------
# default_editor: Editor command to open new posts after creation with "osg new".
#                 If empty, falls back to the $EDITOR environment variable.
#                 Examples: "code", "vim", "nvim", "nano", "subl"
# default_editor: ""
# new_notes_dir: Subdirectory within the vault where "osg new" creates notes.
#                Relative to vault_path. If empty, notes are created at the
#                vault root (Obsidian default). The directory is auto-created
#                if it does not exist.
#                Examples: "02_Notes", "Posts", "drafts"
# new_notes_dir: ""

# -----------------------------------------------------------------------------
# Theme & appearance
# -----------------------------------------------------------------------------
# theme: Active theme name (must match a subdirectory of themes_dir).
# themes_dir: Directory that contains installed themes.
# color_scheme: Color scheme for the default theme.
#   "auto"  — follows the visitor's OS preference (prefers-color-scheme).
#   "light" — always light mode.
#   "dark"  — always dark mode.
# favicon: Path to a custom favicon (e.g. "/img/favicon.svg").
#          If empty and logo is set, the logo is used as favicon.
#          If both are empty, a built-in SVG favicon is used.
theme: default
themes_dir: themes
color_scheme: auto
# favicon: ""

# -----------------------------------------------------------------------------
# Navigation
# -----------------------------------------------------------------------------
# nav_taxonomy: Name of the taxonomy whose terms should appear as links in
#               the header navigation bar (e.g. "area", "tags").
#               When empty (default), the header shows taxonomy index links
#               (the taxonomy names themselves) instead of individual terms.
nav_taxonomy: ""

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
# section_feeds: Generate per-section RSS and Atom feeds (e.g. /blog/atom.xml).
# posts_per_page: Number of posts per page on the homepage (0 = no pagination).
site_feed: true
site_feed_limit: 20
section_feeds: true
posts_per_page: 10

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
#   The "search" plugin is bundled with OSG and enabled by default.
#   It generates a search index (search.json) and search page (search/index.html).
# plugin_timeout: Per-plugin call timeout in seconds. 0 = no timeout.
plugins_dir: plugins
plugins_enabled:
  - search
plugin_timeout: 5

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
# lightbox: Enable click-to-zoom lightbox for content images.
#   When enabled, standalone images in post content get a fullscreen overlay
#   with keyboard/touch navigation, captions from alt text, and automatic
#   gallery grouping for consecutive images.
image_optimization: true
image_quality: 80
image_widths: [640, 1200]
lightbox: true

# -----------------------------------------------------------------------------
# Sharing
# -----------------------------------------------------------------------------
# sharing: Show share buttons and copy-permalink icon on article pages.
#   When enabled, the article title gets a copy-link icon (hover on desktop,
#   always visible on mobile) and a share section appears at the bottom of each
#   article with buttons for X, LinkedIn, Bluesky, email, and copy permalink.
sharing: true

# -----------------------------------------------------------------------------
# Minification
# -----------------------------------------------------------------------------
# minify: Minify HTML, CSS, JS, JSON, SVG, and XML files in public/ after
#         rendering. Uses tdewolff/minify for fast, standards-compliant output.
#         Disable for easier debugging of generated output.
minify: true

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
# tui_log_modifier: Modifier key for log panel navigation ("alt" or "shift").
#   "shift" works everywhere; use "alt" if your terminal sends Meta for Option.
tui_prefix: space
tui_prefix_ms: 600
tui_log_modifier: shift

# -----------------------------------------------------------------------------
# Diagnostics
# -----------------------------------------------------------------------------
# doctor_profile: Profile for "osg doctor" checks ("dev" or "prod").
doctor_profile: dev

# -----------------------------------------------------------------------------
# SEO: theme-color, organization, robots
# -----------------------------------------------------------------------------
# theme_color_light / theme_color_dark: <meta name="theme-color"> for mobile
#   browser UI. When both are set, prefers-color-scheme variants are emitted.
# author / author_url: site-wide author defaults used by <meta name="author">
#   and the Person schema in JSON-LD when a page does not specify its own.
# organization: schema.org Organization emitted on the homepage when name is
#   set (publisher of the site for knowledge-graph signals).
# robots: customize generated robots.txt (Disallow paths, Crawl-delay, raw extra).
# author: ""
# author_url: ""
# theme_color_light: ""
# theme_color_dark: ""
# organization:
#   name: ""
#   url: ""
#   logo: ""
#   same_as: []
# robots:
#   disallow: []
#   crawl_delay: 0
#   extra: ""

# -----------------------------------------------------------------------------
# Social links, copyright & license
# -----------------------------------------------------------------------------
# social: Map of social network handles shown as icons in the footer.
#   Only configured networks appear. Supported keys:
#   x, github, mastodon, linkedin, bluesky, email
#   For email use the address directly (mailto: is added automatically).
# copyright: Copyright notice shown in the footer bar.
#   Use {year} as a placeholder for the current year.
#   If empty, defaults to "(c) {year} {site_title}".
# social:
#   x: "https://x.com/your_handle"
#   github: "https://github.com/your_user"
#   mastodon: "https://mastodon.social/@your_handle"
#   linkedin: "https://linkedin.com/in/your_profile"
#   bluesky: "https://bsky.app/profile/your_handle"
#   email: "you@example.com"
# copyright: ""
#
# license: Site content license displayed in the footer.
#   Supports markdown links: [text](url) for linking to the full license.
#   Leave empty to hide.
# license: ""

# -----------------------------------------------------------------------------
# Sidebar widgets (homepage)
# -----------------------------------------------------------------------------
# Activate the optional 3-column homepage layout. When sidebar_widgets is
# set the homepage renders with a right sidebar containing the listed
# widgets in the given order. The layout collapses to a single column
# below 1400px viewport — sidebars are desktop-only by design.
#
# Each widget self-hides when its data is missing, so it is safe to
# enable a widget you have not fully configured yet.
#
# Available widgets:
#   author      Avatar + name + bio + social, from author/author_bio/
#               author_avatar/author_url/social. Hidden if all blank.
#   newsletter  Email subscription form posting to newsletter_action with
#               an "email" field. Hidden if newsletter_action is empty.
#   popular     Top 5 pages by views from interactions.db at build time.
#               Hidden if interactions disabled or no view data yet.
#
# Order is rendering order. Unknown names fail validation.
#
# sidebar_widgets:
#   - author
#   - newsletter
#   - popular
#
# Newsletter form action URL. Provider-specific (Buttondown, Mailchimp,
# Substack, ConvertKit, etc.) — check your provider's docs.
# newsletter_action: ""

# -----------------------------------------------------------------------------
# Logging
# -----------------------------------------------------------------------------
# level: Minimum log level (debug, info, warn, error).
# format: Log output format ("json" or "text").
logging:
  level: info
  format: json

# -----------------------------------------------------------------------------
# Third-party analytics providers
# -----------------------------------------------------------------------------
# Inject tracking scripts from well-known analytics services.
# Supported: cloudflare (token), google (tracking_id), plausible (domain), fathom (token).
#
# analytics_providers:
#   - provider: cloudflare
#     token: "your-cloudflare-token"
#   - provider: google
#     tracking_id: "G-XXXXXXX"

# -----------------------------------------------------------------------------
# Custom code injection
# -----------------------------------------------------------------------------
# head_extra: Custom HTML/JS injected at the end of <head> on every page.
# body_extra: Custom HTML/JS injected before </body> on every page.
# head_extra: ""
# body_extra: ""

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
#   - name: type
#     paginate_by: 0
#     feed: false
#     render: true
#     exclude_terms:         # terms to exclude from this taxonomy
#       - fuente
#       - template
# -----------------------------------------------------------------------------
# UI dashboard (osg ui)
# -----------------------------------------------------------------------------
# Local web dashboard for the author. Surfaces vault/build state, plugin
# metadata, and lets you start/stop the serve and api services from a
# single browser tab. Loopback-only by default — non-loopback bind addresses
# are rejected.
#
# ui:
#   addr: ":1314"        # Bind address. Bare ":port" is normalised to
#                        # 127.0.0.1:port. Override with --addr at the CLI.
# -----------------------------------------------------------------------------
# Interactions (views, likes/dislikes)
# -----------------------------------------------------------------------------
# Enable page view counting and like/dislike buttons at the bottom of posts.
# Requires running the interactions API (standalone via "osg api" or embedded
# in the dev server via "osg serve --api").
#
# interactions:
#   enabled: true
#   api_url: ""              # Browser-visible API URL. Empty = same origin.
#                            # For production set full URL (e.g. "https://api.mysite.com").
#   listen: ":8090"          # Address for standalone "osg api" server.
#   db_path: ".osg/interactions.db"  # SQLite database file path.
#   cors_origins:            # Allowed CORS origins (only needed for cross-origin API).
#     - "https://mysite.com"
#   view_dedup_hours: 24     # Dedup window: same fingerprint counts as one
#                            # unique view per page within this period.
#
# --- Comments (nested under interactions) ---
# OAuth2-authenticated comment system with threaded replies.
# Uses a separate SQLite database for future portability.
#
# Visitors log in with their own GitHub/Google account to comment.
# The client_id and client_secret below identify YOUR SITE (not the
# commenter) to the OAuth provider. You obtain them by registering an
# OAuth application:
#
#   GitHub:
#     1. Go to https://github.com/settings/developers -> "OAuth Apps" -> "New OAuth App"
#     2. Set "Authorization callback URL" to:
#        https://yoursite.com/api/v1/auth/github/callback
#     3. Copy the Client ID and Client Secret into the config below.
#
#   Google:
#     1. Go to https://console.cloud.google.com/apis/credentials
#     2. Create an "OAuth 2.0 Client ID" (Web application).
#     3. Add to "Authorized redirect URIs":
#        https://yoursite.com/api/v1/auth/google/callback
#     4. Copy the Client ID and Client Secret into the config below.
#
# When a visitor clicks "Login with GitHub", they are redirected to
# GitHub where they authorize your app. GitHub sends back a temporary
# code that the server exchanges (using the client_secret) for the
# visitor's public profile. The visitor's password never touches your
# site.
#
#   comments:
#     enabled: true
#     db_path: ".osg/comments.db"     # Separate DB for comments.
#     auth_session_days: 30            # Login session lifetime in days.
#     auth_callback_url: ""            # Base URL for OAuth callbacks.
#                                      # Must match the callback URL registered
#                                      # with the provider (without the path).
#                                      # E.g. "https://yoursite.com"
#     providers:                       # OAuth2 providers for login.
#       - provider: github
#         client_id: "your_github_client_id"
#         client_secret: "your_github_client_secret"
#       - provider: google
#         client_id: "your_google_client_id"
#         client_secret: "your_google_client_secret"
`) + "\n"
}

func envKeyTransform(key string) string {
	key = strings.TrimPrefix(key, "OSG_")
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "__", ".")
	return key
}
