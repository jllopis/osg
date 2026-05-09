// Package deploy provides deployment targets for OSG static sites.
//
// Supported providers: cloudflare, rsync, s3.
// Configuration goes in the "deploy" section of config.yaml.
package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// logWriterKey is the unexported context key used to thread a
// preferred output destination through to the deploy subprocess. The
// /vault flow drawer (and any other caller that already has a writer
// it wants subprocess output mirrored to) passes its writer via
// WithLogWriter; runCommand reads it back.
type logWriterKey struct{}

// WithLogWriter returns a derived context that carries w as the
// destination for subprocess stdout/stderr. A nil writer is a no-op.
// Callers in internal/app use this so wrangler/rsync output reaches
// the osg ui flow drawer without losing the terminal stream — the
// caller can pass an io.MultiWriter when both are wanted.
func WithLogWriter(ctx context.Context, w io.Writer) context.Context {
	if w == nil {
		return ctx
	}
	return context.WithValue(ctx, logWriterKey{}, w)
}

// logWriterFromContext returns the writer registered with WithLogWriter,
// or nil when none is set. runCommand falls back to os.Stdout/Stderr
// in that case so the CLI behaviour is unchanged.
func logWriterFromContext(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(logWriterKey{}).(io.Writer); ok {
		return w
	}
	return nil
}

// logf writes a formatted status line to the ctx-registered writer
// (same one runCommand uses for subprocess stdout/stderr) or to
// os.Stdout when none is set. Providers use this so their
// "Deploying to X…" / "Deployed to X" lines reach the flow drawer
// alongside the subprocess output, instead of going only to the
// terminal.
func logf(ctx context.Context, format string, args ...any) {
	w := logWriterFromContext(ctx)
	if w == nil {
		w = os.Stdout
	}
	_, _ = fmt.Fprintf(w, format, args...)
}

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

// runCommand executes a shell command, streaming output to the writer
// registered with WithLogWriter on ctx, or os.Stdout/os.Stderr when
// none is set.
func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	wireCommandOutput(ctx, cmd)
	cmd.Env = os.Environ()
	return cmd.Run()
}

// wireCommandOutput points cmd's stdout/stderr at the ctx-registered
// log writer, falling back to os.Stdout/os.Stderr. Providers that
// build their own exec.Cmd (because they need cmd.Env or cmd.Dir
// customisation runCommand doesn't expose) call this so subprocess
// output reaches the same destination as runCommand-style calls.
func wireCommandOutput(ctx context.Context, cmd *exec.Cmd) {
	if w := logWriterFromContext(ctx); w != nil {
		cmd.Stdout = w
		cmd.Stderr = w
		return
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
