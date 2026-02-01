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

type Config struct {
	BaseURL          string           `koanf:"base_url" yaml:"base_url"`
	Theme            string           `koanf:"theme" yaml:"theme"`
	VaultPath        string           `koanf:"vault_path" yaml:"vault_path"`
	ContentDir       string           `koanf:"content_dir" yaml:"content_dir"`
	PublicDir        string           `koanf:"public_dir" yaml:"public_dir"`
	TemplatesDir     string           `koanf:"templates_dir" yaml:"templates_dir"`
	StaticDir        string           `koanf:"static_dir" yaml:"static_dir"`
	ThemesDir        string           `koanf:"themes_dir" yaml:"themes_dir"`
	PluginsDir       string           `koanf:"plugins_dir" yaml:"plugins_dir"`
	SassDir          string           `koanf:"sass_dir" yaml:"sass_dir"`
	ContentLayout    string           `koanf:"content_layout" yaml:"content_layout"`
	IncludeDrafts    bool             `koanf:"include_drafts" yaml:"include_drafts"`
	CompileSass      bool             `koanf:"compile_sass" yaml:"compile_sass"`
	TUIPrefix        string           `koanf:"tui_prefix" yaml:"tui_prefix"`
	TUIPrefixMs      int              `koanf:"tui_prefix_ms" yaml:"tui_prefix_ms"`
	ServeWatch       bool             `koanf:"serve_watch" yaml:"serve_watch"`
	ServeReload      bool             `koanf:"serve_live_reload" yaml:"serve_live_reload"`
	ServeDebounce    int              `koanf:"serve_debounce_ms" yaml:"serve_debounce_ms"`
	BuildIncremental bool             `koanf:"build_incremental" yaml:"build_incremental"`
	BuildCacheDir    string           `koanf:"build_cache_dir" yaml:"build_cache_dir"`
	Logging          LoggingConfig    `koanf:"logging" yaml:"logging"`
	Taxonomies       []TaxonomyConfig `koanf:"taxonomies" yaml:"taxonomies"`
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
		BaseURL:          "",
		Theme:            "default",
		ContentDir:       "content",
		PublicDir:        "public",
		TemplatesDir:     "templates",
		StaticDir:        "static",
		ThemesDir:        "themes",
		PluginsDir:       "plugins",
		SassDir:          "sass",
		ContentLayout:    "{date}/{slug}",
		IncludeDrafts:    false,
		CompileSass:      false,
		TUIPrefix:        "space",
		TUIPrefixMs:      600,
		ServeWatch:       true,
		ServeReload:      true,
		ServeDebounce:    300,
		BuildIncremental: true,
		BuildCacheDir:    ".osg/cache",
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

	return cfg, nil
}

func ResolveVaultPath(cfg Config) (string, error) {
	if cfg.VaultPath != "" {
		return cfg.VaultPath, nil
	}
	return "", fmt.Errorf("vault path not configured (use --vault-path)")
}

func DefaultConfigYAML() string {
	return strings.TrimSpace(`vault_path: ""
base_url: ""
theme: default
content_dir: content
public_dir: public
templates_dir: templates
static_dir: static
themes_dir: themes
plugins_dir: plugins
sass_dir: sass
content_layout: "{date}/{slug}"
include_drafts: false
compile_sass: false
tui_prefix: space
tui_prefix_ms: 600
serve_watch: true
serve_live_reload: true
serve_debounce_ms: 300
build_incremental: true
build_cache_dir: .osg/cache
logging:
  level: info
  format: json
# taxonomies:
#   - name: tags
#     paginate_by: 10
#     paginate_path: page
#     feed: true
#     render: true
#   - name: area
#     paginate_by: 10
#     paginate_path: page
#     feed: false
#     render: true
#   - name: type
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
