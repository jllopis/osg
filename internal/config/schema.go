package config

// FieldType describes the kind of value a config field holds.
type FieldType int

const (
	FieldString FieldType = iota
	FieldBool
	FieldInt
	FieldStringList
	FieldIntList
	FieldStringMap
	FieldStruct
	FieldStructList
)

// ConfigField describes a single editable config field.
type ConfigField struct {
	// Key is the YAML key name (e.g. "site_title", "ai.provider").
	Key string
	// Label is the human-readable name shown in the TUI.
	Label string
	// Description explains what the field does (shown as help text).
	Description string
	// Type indicates the value type for the editor.
	Type FieldType
	// Options lists valid choices for enum-like fields (e.g. ["auto","light","dark"]).
	// Empty for free-form fields.
	Options []string
	// Default is the default value as a string (for display purposes).
	Default string
	// Nested holds sub-fields for FieldStruct and FieldStructList types.
	Nested []ConfigField
	// Sensitive marks fields that should be masked in the UI (e.g. api_key, client_secret).
	Sensitive bool
}

// ConfigSection groups related config fields.
type ConfigSection struct {
	Name        string
	Description string
	Fields      []ConfigField
}

// ConfigSchema returns the complete schema for the OSG configuration,
// organised into sections. This is the single source of truth for the
// TUI config editor.
func ConfigSchema() []ConfigSection {
	return []ConfigSection{
		{
			Name:        "Site Identity",
			Description: "Basic site metadata: title, description, URL, language.",
			Fields: []ConfigField{
				{Key: "site_title", Label: "Site Title", Description: "Displayed in the header, browser tab, and meta tags.", Type: FieldString, Default: "OSG"},
				{Key: "site_description", Label: "Site Description", Description: "HTML <meta name=\"description\"> and OpenGraph.", Type: FieldString},
				{Key: "base_url", Label: "Base URL", Description: "Absolute URL of the deployed site (e.g. \"https://blog.example.com\").", Type: FieldString},
				{Key: "default_language", Label: "Default Language", Description: "BCP-47 language code for template translations and date localisation.", Type: FieldString, Default: "es"},
				{Key: "copyright", Label: "Copyright", Description: "Footer copyright notice. Use {year} for current year.", Type: FieldString},
			},
		},
		{
			Name:        "Theme & Appearance",
			Description: "Active theme, color scheme, logo, and favicon.",
			Fields: []ConfigField{
				{Key: "theme", Label: "Theme", Description: "Active theme name (subdirectory of themes_dir).", Type: FieldString, Default: "default"},
				{Key: "themes_dir", Label: "Themes Directory", Description: "Directory containing installed themes.", Type: FieldString, Default: "themes"},
				{Key: "color_scheme", Label: "Color Scheme", Description: "Color scheme: follows OS preference, always light, or always dark.", Type: FieldString, Options: []string{"auto", "light", "dark"}, Default: "auto"},
				{Key: "logo", Label: "Logo", Description: "Path to site logo image.", Type: FieldString},
				{Key: "favicon", Label: "Favicon", Description: "Path to custom favicon. Falls back to logo, then built-in SVG.", Type: FieldString},
			},
		},
		{
			Name:        "Content",
			Description: "Vault path, content directory, URL layout, drafts.",
			Fields: []ConfigField{
				{Key: "vault_path", Label: "Vault Path", Description: "Path to the Obsidian vault to import from.", Type: FieldString},
				{Key: "content_dir", Label: "Content Directory", Description: "Directory where Markdown content lives.", Type: FieldString, Default: "content"},
				{Key: "content_layout", Label: "Content Layout", Description: "URL pattern for pages. Placeholders: {date}, {year}, {month}, {day}, {slug}, {title}.", Type: FieldString, Default: "{date}/{slug}"},
				{Key: "include_drafts", Label: "Include Drafts", Description: "Render pages with draft: true in front-matter.", Type: FieldBool, Default: "false"},
			},
		},
		{
			Name:        "Editor",
			Description: "Editor for opening new posts after creation.",
			Fields: []ConfigField{
				{Key: "default_editor", Label: "Default Editor", Description: "Editor command to open new posts (falls back to $EDITOR env var).", Type: FieldString},
			},
		},
		{
			Name:        "Output",
			Description: "Public directory and build cleaning.",
			Fields: []ConfigField{
				{Key: "public_dir", Label: "Public Directory", Description: "Directory where the generated site is written.", Type: FieldString, Default: "public"},
				{Key: "clean_public", Label: "Clean Public", Description: "Remove public_dir before a full build.", Type: FieldBool, Default: "true"},
				{Key: "minify", Label: "Minify", Description: "Minify HTML, CSS, JS, JSON, SVG, and XML files after rendering.", Type: FieldBool, Default: "true"},
			},
		},
		{
			Name:        "Navigation",
			Description: "Header navigation taxonomy.",
			Fields: []ConfigField{
				{Key: "nav_taxonomy", Label: "Nav Taxonomy", Description: "Taxonomy whose terms appear as links in the header nav bar.", Type: FieldString},
			},
		},
		{
			Name:        "Summaries",
			Description: "Page summary generation strategy.",
			Fields: []ConfigField{
				{Key: "summary_strategy", Label: "Summary Strategy", Description: "How to generate page summaries: auto-extract, manual only, or AI.", Type: FieldString, Options: []string{"auto", "manual", "ai"}, Default: "auto"},
			},
		},
		{
			Name:        "AI",
			Description: "LLM-based summary generation via Kairos.",
			Fields: []ConfigField{
				{Key: "ai.provider", Label: "Provider", Description: "LLM provider: gemini, anthropic, openai, qwen, or ollama.", Type: FieldString, Options: []string{"gemini", "anthropic", "openai", "qwen", "ollama"}, Default: "gemini"},
				{Key: "ai.model", Label: "Model", Description: "Model identifier (e.g. \"gemini-3-flash-preview\").", Type: FieldString, Default: "gemini-3-flash-preview"},
				{Key: "ai.api_key", Label: "API Key", Description: "API key for the provider. If empty, uses the provider's default env var.", Type: FieldString, Sensitive: true},
				{Key: "ai.base_url", Label: "Base URL", Description: "Override the API endpoint (useful for ollama or custom proxies).", Type: FieldString},
				{Key: "ai.system_prompt", Label: "System Prompt", Description: "Custom system instruction for the LLM.", Type: FieldString},
				{Key: "ai.timeout", Label: "Timeout", Description: "Per-request timeout in seconds.", Type: FieldInt, Default: "30"},
				{Key: "ai.concurrency", Label: "Concurrency", Description: "Max parallel LLM requests.", Type: FieldInt, Default: "3"},
			},
		},
		{
			Name:        "Feeds",
			Description: "Site-wide RSS and Atom feed settings.",
			Fields: []ConfigField{
				{Key: "site_feed", Label: "Site Feed", Description: "Generate site-wide RSS and Atom feeds.", Type: FieldBool, Default: "true"},
				{Key: "site_feed_limit", Label: "Feed Limit", Description: "Maximum entries in the site feed (0 = all pages).", Type: FieldInt, Default: "20"},
			},
		},
		{
			Name:        "Templates & Static",
			Description: "Template overrides and static asset directories.",
			Fields: []ConfigField{
				{Key: "templates_dir", Label: "Templates Directory", Description: "User-level template overrides (merged on top of theme).", Type: FieldString, Default: "templates"},
				{Key: "static_dir", Label: "Static Directory", Description: "Extra static files copied as-is to public_dir.", Type: FieldString, Default: "static"},
			},
		},
		{
			Name:        "Plugins",
			Description: "WASM plugin directory and enabled plugins.",
			Fields: []ConfigField{
				{Key: "plugins_dir", Label: "Plugins Directory", Description: "Directory containing .wasm plugin files.", Type: FieldString, Default: "plugins"},
				{Key: "plugins_enabled", Label: "Enabled Plugins", Description: "List of plugin names to activate (without .wasm extension).", Type: FieldStringList, Default: "search"},
				{Key: "plugin_timeout", Label: "Plugin Timeout", Description: "Per-plugin call timeout in seconds (0 = no timeout).", Type: FieldInt, Default: "5"},
			},
		},
		{
			Name:        "Sass",
			Description: "Sass compilation settings.",
			Fields: []ConfigField{
				{Key: "sass_dir", Label: "Sass Directory", Description: "Directory with .scss files to compile.", Type: FieldString, Default: "sass"},
				{Key: "compile_sass", Label: "Compile Sass", Description: "Enable Sass compilation.", Type: FieldBool, Default: "false"},
			},
		},
		{
			Name:        "Images",
			Description: "Image optimization and lightbox settings.",
			Fields: []ConfigField{
				{Key: "image_optimization", Label: "Image Optimization", Description: "Generate responsive image variants (resized JPEG + WebP).", Type: FieldBool, Default: "true"},
				{Key: "image_quality", Label: "Image Quality", Description: "Encoding quality for JPEG and WebP variants (1-100).", Type: FieldInt, Default: "80"},
				{Key: "image_widths", Label: "Image Widths", Description: "Pixel widths to generate (no upscaling). Original always kept.", Type: FieldIntList, Default: "640, 1200"},
				{Key: "lightbox", Label: "Lightbox", Description: "Enable click-to-zoom lightbox for content images.", Type: FieldBool, Default: "true"},
			},
		},
		{
			Name:        "Sharing",
			Description: "Social sharing buttons on articles.",
			Fields: []ConfigField{
				{Key: "sharing", Label: "Sharing", Description: "Show share buttons on article pages (X, LinkedIn, Bluesky, email, copy link).", Type: FieldBool, Default: "true"},
			},
		},
		{
			Name:        "Build",
			Description: "Incremental build and cache settings.",
			Fields: []ConfigField{
				{Key: "build_incremental", Label: "Incremental Build", Description: "Only re-render changed pages (speeds up rebuilds).", Type: FieldBool, Default: "true"},
				{Key: "build_cache_dir", Label: "Cache Directory", Description: "Where build cache data is stored.", Type: FieldString, Default: ".osg/cache"},
			},
		},
		{
			Name:        "Dev Server",
			Description: "Settings for osg serve.",
			Fields: []ConfigField{
				{Key: "serve_watch", Label: "Watch", Description: "Watch files for changes and rebuild automatically.", Type: FieldBool, Default: "true"},
				{Key: "serve_live_reload", Label: "Live Reload", Description: "Inject live-reload script into served pages.", Type: FieldBool, Default: "true"},
				{Key: "serve_debounce_ms", Label: "Debounce (ms)", Description: "Milliseconds to wait before triggering a rebuild after a file change.", Type: FieldInt, Default: "300"},
			},
		},
		{
			Name:        "Logging",
			Description: "Log level and format.",
			Fields: []ConfigField{
				{Key: "logging.level", Label: "Level", Description: "Minimum log level.", Type: FieldString, Options: []string{"debug", "info", "warn", "error"}, Default: "info"},
				{Key: "logging.format", Label: "Format", Description: "Log output format.", Type: FieldString, Options: []string{"json", "text"}, Default: "json"},
			},
		},
		{
			Name:        "Social Links",
			Description: "Social network handles shown as icons in the footer.",
			Fields: []ConfigField{
				{Key: "social", Label: "Social Links", Description: "Map of social network handles. Supported: x, github, mastodon, linkedin, bluesky, email.", Type: FieldStringMap},
			},
		},
		{
			Name:        "Languages",
			Description: "Multi-language configuration for secondary languages.",
			Fields: []ConfigField{
				{Key: "languages", Label: "Languages", Description: "Secondary languages (the default language needs no entry here).", Type: FieldStructList, Nested: []ConfigField{
					{Key: "code", Label: "Code", Description: "BCP-47 language code (e.g. \"en\", \"fr\").", Type: FieldString},
					{Key: "label", Label: "Label", Description: "Human-readable name for the language switcher.", Type: FieldString},
				}},
			},
		},
		{
			Name:        "Taxonomies",
			Description: "Content groupings derived from front-matter fields.",
			Fields: []ConfigField{
				{Key: "taxonomies", Label: "Taxonomies", Description: "Define content groupings (tags, categories, etc.).", Type: FieldStructList, Nested: []ConfigField{
					{Key: "name", Label: "Name", Description: "Front-matter field name.", Type: FieldString},
					{Key: "paginate_by", Label: "Paginate By", Description: "Items per page (0 = no pagination).", Type: FieldInt, Default: "0"},
					{Key: "paginate_path", Label: "Paginate Path", Description: "URL segment for paginated pages.", Type: FieldString, Default: "page"},
					{Key: "feed", Label: "Feed", Description: "Generate RSS/Atom feed for this taxonomy.", Type: FieldBool, Default: "false"},
					{Key: "render", Label: "Render", Description: "Generate HTML pages for this taxonomy.", Type: FieldBool, Default: "true"},
					{Key: "exclude_terms", Label: "Exclude Terms", Description: "Terms to exclude from this taxonomy.", Type: FieldStringList},
				}},
			},
		},
		{
			Name:        "Interactions",
			Description: "Page view counting and like/dislike buttons.",
			Fields: []ConfigField{
				{Key: "interactions.enabled", Label: "Enabled", Description: "Activate the interactions feature (views, likes).", Type: FieldBool, Default: "false"},
				{Key: "interactions.api_url", Label: "API URL", Description: "Browser-visible API URL. Empty = same origin.", Type: FieldString},
				{Key: "interactions.listen", Label: "Listen", Description: "Address for standalone osg api server.", Type: FieldString, Default: ":8090"},
				{Key: "interactions.db_path", Label: "DB Path", Description: "SQLite database file path for interactions.", Type: FieldString, Default: ".osg/interactions.db"},
				{Key: "interactions.cors_origins", Label: "CORS Origins", Description: "Allowed CORS origins for the API.", Type: FieldStringList},
				{Key: "interactions.view_dedup_hours", Label: "View Dedup Hours", Description: "Dedup window for unique views per fingerprint per page.", Type: FieldInt, Default: "24"},
			},
		},
		{
			Name:        "Comments",
			Description: "OAuth2-authenticated comment system with threaded replies.",
			Fields: []ConfigField{
				{Key: "interactions.comments.enabled", Label: "Enabled", Description: "Activate the comment system.", Type: FieldBool, Default: "false"},
				{Key: "interactions.comments.db_path", Label: "DB Path", Description: "SQLite database file path for comments.", Type: FieldString, Default: ".osg/comments.db"},
				{Key: "interactions.comments.auth_session_days", Label: "Session Days", Description: "How long auth sessions last (in days).", Type: FieldInt, Default: "30"},
				{Key: "interactions.comments.auth_callback_url", Label: "Callback URL", Description: "Base URL for OAuth callbacks (e.g. \"https://yoursite.com\").", Type: FieldString},
				{Key: "interactions.comments.providers", Label: "Auth Providers", Description: "OAuth2 providers for login.", Type: FieldStructList, Nested: []ConfigField{
					{Key: "provider", Label: "Provider", Description: "Provider name: github or google.", Type: FieldString, Options: []string{"github", "google"}},
					{Key: "client_id", Label: "Client ID", Description: "OAuth2 client ID.", Type: FieldString, Sensitive: true},
					{Key: "client_secret", Label: "Client Secret", Description: "OAuth2 client secret.", Type: FieldString, Sensitive: true},
				}},
			},
		},
		{
			Name:        "Deploy",
			Description: "Deployment target settings.",
			Fields: []ConfigField{
				{Key: "deploy.provider", Label: "Provider", Description: "Deployment target: cloudflare, rsync, or s3.", Type: FieldString, Options: []string{"cloudflare", "rsync", "s3"}},
			},
		},
		{
			Name:        "Diagnostics",
			Description: "Doctor profile for checks.",
			Fields: []ConfigField{
				{Key: "doctor_profile", Label: "Doctor Profile", Description: "Profile for osg doctor checks.", Type: FieldString, Options: []string{"dev", "prod"}, Default: "dev"},
			},
		},
		{
			Name:        "TUI",
			Description: "Terminal UI settings.",
			Fields: []ConfigField{
				{Key: "tui_prefix", Label: "Prefix Key", Description: "Key used as prefix in the interactive TUI.", Type: FieldString, Options: []string{"space", "ctrl"}, Default: "space"},
				{Key: "tui_prefix_ms", Label: "Prefix Timeout (ms)", Description: "Milliseconds to wait for a second key after prefix.", Type: FieldInt, Default: "600"},
				{Key: "tui_log_modifier", Label: "Log Modifier Key", Description: "Modifier key for log panel navigation (scroll, tab switch). Use \"alt\" if your terminal sends Meta for Option.", Type: FieldString, Options: []string{"alt", "shift"}, Default: "shift"},
			},
		},
	}
}

// AllFields returns a flat list of all config fields across all sections.
// Nested struct fields are not flattened; use field.Nested for those.
func AllFields() []ConfigField {
	var fields []ConfigField
	for _, section := range ConfigSchema() {
		fields = append(fields, section.Fields...)
	}
	return fields
}

// FindField returns the ConfigField with the given key, searching all sections.
// Returns the field and true if found, or zero value and false if not.
func FindField(key string) (ConfigField, bool) {
	for _, section := range ConfigSchema() {
		for _, f := range section.Fields {
			if f.Key == key {
				return f, true
			}
		}
	}
	return ConfigField{}, false
}
