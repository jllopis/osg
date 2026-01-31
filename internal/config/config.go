package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	VaultPath     string        `koanf:"vault_path" yaml:"vault_path"`
	VaultBase     string        `koanf:"vault_base" yaml:"vault_base"`
	Vault         string        `koanf:"vault" yaml:"vault"`
	ContentDir    string        `koanf:"content_dir" yaml:"content_dir"`
	PublicDir     string        `koanf:"public_dir" yaml:"public_dir"`
	TemplatesDir  string        `koanf:"templates_dir" yaml:"templates_dir"`
	StaticDir     string        `koanf:"static_dir" yaml:"static_dir"`
	ThemesDir     string        `koanf:"themes_dir" yaml:"themes_dir"`
	PluginsDir    string        `koanf:"plugins_dir" yaml:"plugins_dir"`
	SassDir       string        `koanf:"sass_dir" yaml:"sass_dir"`
	ContentLayout string        `koanf:"content_layout" yaml:"content_layout"`
	IncludeDrafts bool          `koanf:"include_drafts" yaml:"include_drafts"`
	Logging       LoggingConfig `koanf:"logging" yaml:"logging"`
}

func Default() Config {
	return Config{
		ContentDir:    "content",
		PublicDir:     "public",
		TemplatesDir:  "templates",
		StaticDir:     "static",
		ThemesDir:     "themes",
		PluginsDir:    "plugins",
		SassDir:       "sass",
		ContentLayout: "{date}/{slug}",
		IncludeDrafts: false,
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

	return cfg, nil
}

func ResolveVaultPath(cfg Config) (string, error) {
	if cfg.VaultPath != "" {
		return cfg.VaultPath, nil
	}
	if cfg.VaultBase != "" && cfg.Vault != "" {
		return filepath.Join(cfg.VaultBase, cfg.Vault), nil
	}
	return "", fmt.Errorf("vault path not configured (use --vault-path or --obsidian-vault-base + --vault)")
}

func DefaultConfigYAML() string {
	return strings.TrimSpace(`vault_path: ""
# vault_base: ""
# vault: ""
content_dir: content
public_dir: public
templates_dir: templates
static_dir: static
themes_dir: themes
plugins_dir: plugins
sass_dir: sass
content_layout: "{date}/{slug}"
include_drafts: false
logging:
  level: info
  format: json
`) + "\n"
}

func envKeyTransform(key string) string {
	key = strings.TrimPrefix(key, "OSG_")
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "__", ".")
	return key
}
