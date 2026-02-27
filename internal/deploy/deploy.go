// Package deploy provides deployment targets for OSG static sites.
//
// Supported providers: cloudflare, rsync, s3.
// Configuration goes in the "deploy" section of config.yaml.
package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Provider deploys a static site to a remote destination.
type Provider interface {
	// Name returns the provider identifier (cloudflare, rsync, s3).
	Name() string

	// Deploy uploads the contents of publicDir to the remote destination.
	Deploy(ctx context.Context, publicDir string) error

	// Validate checks that required configuration and credentials are present.
	Validate() error
}

// Registry holds registered providers.
var registry = make(map[string]func(map[string]any) Provider)
var registryMu sync.RWMutex

// Register adds a provider factory to the registry.
func Register(name string, factory func(map[string]any) Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get returns a provider by name, or an error if not found.
func Get(name string, cfg map[string]any) (Provider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("deploy: unknown provider %q", name)
	}
	return factory(cfg), nil
}

// Providers returns the list of registered provider names.
func Providers() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Run executes the deployment for the given provider.
func Run(ctx context.Context, provider string, cfg map[string]any, publicDir string) error {
	p, err := Get(provider, cfg)
	if err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return fmt.Errorf("deploy %s: %w", provider, err)
	}
	return p.Deploy(ctx, publicDir)
}

// runCommand executes a shell command, streaming output to stdout/stderr.
func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

// getEnv returns an environment variable or an error if missing.
func getEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("missing environment variable %s", key)
	}
	return v, nil
}

// getEnvOr returns an environment variable or a fallback value.
func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getString extracts a string from a map with a default.
func getString(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// getBool extracts a bool from a map with a default.
func getBool(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
