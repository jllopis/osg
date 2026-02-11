package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
)

type doctorCounters struct {
	warn  int
	error int
}

func RunDoctor(ctx context.Context, opts CLIOptions) error {
	_ = ctx
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)
	counts := doctorCounters{}

	profile := normalizeProfile(cfg.DoctorProfile)
	checkInfo(logger, "doctor starting", "profile", profile)

	if strings.TrimSpace(cfg.BaseURL) == "" {
		if profile == "prod" {
			checkError(logger, &counts, "base_url is empty")
		} else {
			checkWarn(logger, &counts, "base_url is empty")
		}
	} else if err := validateBaseURL(cfg.BaseURL); err != nil {
		if profile == "prod" {
			checkError(logger, &counts, "base_url invalid", "error", err.Error())
		} else {
			checkWarn(logger, &counts, "base_url invalid", "error", err.Error())
		}
	}

	checkPath(logger, &counts, "vault_path", cfg.VaultPath, profile == "prod")
	checkPath(logger, &counts, "content_dir", cfg.ContentDir, profile == "prod")
	checkPath(logger, &counts, "public_dir", cfg.PublicDir, profile == "prod")
	checkPath(logger, &counts, "templates_dir", cfg.TemplatesDir, profile == "prod")
	checkPath(logger, &counts, "static_dir", cfg.StaticDir, profile == "prod")
	checkPath(logger, &counts, "themes_dir", cfg.ThemesDir, profile == "prod")
	checkPath(logger, &counts, "plugins_dir", cfg.PluginsDir, false)
	checkPath(logger, &counts, "sass_dir", cfg.SassDir, cfg.CompileSass && profile == "prod")

	themePath := filepath.Join(cfg.ThemesDir, cfg.Theme)
	if strings.TrimSpace(cfg.Theme) == "" {
		if profile == "prod" {
			checkError(logger, &counts, "theme is empty")
		} else {
			checkWarn(logger, &counts, "theme is empty")
		}
	} else if !pathExists(themePath) {
		if profile == "prod" {
			checkError(logger, &counts, "theme not found", "theme", cfg.Theme, "path", themePath)
		} else {
			checkWarn(logger, &counts, "theme not found", "theme", cfg.Theme, "path", themePath)
		}
	}

	checkTaxonomies(logger, &counts, cfg.Taxonomies)
	checkPlugins(logger, &counts, cfg.PluginsDir, cfg.PluginsEnabled)
	checkServeConfig(logger, &counts, cfg)
	checkSass(logger, &counts, cfg, profile)
	checkThemeTemplates(logger, &counts, cfg, profile)

	logger.Info("doctor summary", "warnings", counts.warn, "errors", counts.error)
	if counts.error > 0 {
		return fmt.Errorf("doctor found %d error(s)", counts.error)
	}
	return nil
}

func checkInfo(logger *slog.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

func checkWarn(logger *slog.Logger, counts *doctorCounters, msg string, args ...any) {
	counts.warn++
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

func checkError(logger *slog.Logger, counts *doctorCounters, msg string, args ...any) {
	counts.error++
	if logger != nil {
		logger.Error(msg, args...)
	}
}

func checkPath(logger *slog.Logger, counts *doctorCounters, label string, value string, required bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			checkWarn(logger, counts, fmt.Sprintf("%s is empty", label))
		}
		return
	}
	if !pathExists(value) {
		if required {
			checkError(logger, counts, fmt.Sprintf("%s does not exist", label), "path", value)
		} else {
			checkWarn(logger, counts, fmt.Sprintf("%s does not exist", label), "path", value)
		}
	}
}

func checkTaxonomies(logger *slog.Logger, counts *doctorCounters, taxonomies []config.TaxonomyConfig) {
	if len(taxonomies) == 0 {
		checkWarn(logger, counts, "no taxonomies configured")
		return
	}

	seen := map[string]int{}
	for _, tax := range taxonomies {
		name := strings.TrimSpace(tax.Name)
		if name == "" {
			checkWarn(logger, counts, "taxonomy name is empty")
			continue
		}
		seen[name]++
		if tax.PaginateBy <= 0 {
			checkWarn(logger, counts, "taxonomy paginate_by should be > 0", "name", name)
		}
		if strings.TrimSpace(tax.PaginatePath) == "" {
			checkWarn(logger, counts, "taxonomy paginate_path is empty", "name", name)
		}
	}
	dups := []string{}
	for name, count := range seen {
		if count > 1 {
			dups = append(dups, name)
		}
	}
	sort.Strings(dups)
	if len(dups) > 0 {
		checkWarn(logger, counts, "duplicate taxonomies", "names", dups)
	}
}

func checkPlugins(logger *slog.Logger, counts *doctorCounters, dir string, enabled []string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		if len(enabled) > 0 {
			checkWarn(logger, counts, "plugins_enabled set but plugins_dir is empty")
		}
		return
	}
	if !pathExists(dir) {
		if len(enabled) > 0 {
			checkWarn(logger, counts, "plugins_enabled set but plugins_dir missing", "path", dir)
		}
		return
	}

	for _, name := range enabled {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		path := filepath.Join(dir, trimmed+".wasm")
		if !pathExists(path) {
			checkWarn(logger, counts, "plugin enabled but not installed", "plugin", trimmed, "path", path)
		}
	}
}

func checkThemeTemplates(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	if strings.TrimSpace(cfg.Theme) == "" {
		return
	}
	templatesDir := filepath.Join(cfg.ThemesDir, cfg.Theme, "templates")
	if !pathExists(templatesDir) {
		if profile == "prod" {
			checkError(logger, counts, "theme templates not found", "path", templatesDir)
		} else {
			checkWarn(logger, counts, "theme templates not found", "path", templatesDir)
		}
	}
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func normalizeProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile == "" {
		return "dev"
	}
	if profile != "dev" && profile != "prod" {
		return "dev"
	}
	return profile
}

func validateBaseURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("base_url should include scheme and host")
	}
	return nil
}

func checkServeConfig(logger *slog.Logger, counts *doctorCounters, cfg config.Config) {
	if cfg.ServeReload && !cfg.ServeWatch {
		checkWarn(logger, counts, "serve_live_reload enabled but serve_watch is false")
	}
	if cfg.ServeWatch && cfg.ServeDebounce <= 0 {
		checkWarn(logger, counts, "serve_debounce_ms should be > 0")
	}
}

func checkSass(logger *slog.Logger, counts *doctorCounters, cfg config.Config, profile string) {
	if !cfg.CompileSass {
		return
	}
	if _, err := exec.LookPath("sass"); err != nil {
		if profile == "prod" {
			checkError(logger, counts, "sass binary not found in PATH")
		} else {
			checkWarn(logger, counts, "sass binary not found in PATH")
		}
	}
}
