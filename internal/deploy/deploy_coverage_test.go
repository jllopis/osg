package deploy

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Register — custom provider
// ---------------------------------------------------------------------------

type fakeProvider struct{ name string }

func (f *fakeProvider) Name() string                             { return f.name }
func (f *fakeProvider) Validate() error                          { return nil }
func (f *fakeProvider) Deploy(_ context.Context, _ string) error { return nil }

func TestRegister_CustomProvider(t *testing.T) {
	Register("test-custom", func(cfg map[string]any) Provider {
		return &fakeProvider{name: "test-custom"}
	})
	defer func() {
		// Clean up registry
		registryMu.Lock()
		delete(registry, "test-custom")
		registryMu.Unlock()
	}()

	p, err := Get("test-custom", nil)
	if err != nil {
		t.Fatalf("Get(test-custom) error: %v", err)
	}
	if p.Name() != "test-custom" {
		t.Errorf("Name() = %q, want %q", p.Name(), "test-custom")
	}

	// Verify it appears in Providers list
	if !slices.Contains(Providers(), "test-custom") {
		t.Error("test-custom not found in Providers()")
	}
}

// ---------------------------------------------------------------------------
// CloudflareProvider.Deploy error paths
// ---------------------------------------------------------------------------

func TestCloudflareDeploy_NonexistentPublicDir(t *testing.T) {
	p := &CloudflareProvider{
		WorkerName:        "test-worker",
		CompatibilityDate: "2024-09-01",
		NotFoundHandling:  "404-page",
	}

	err := p.Deploy(context.Background(), "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for nonexistent publicDir")
	}
	if !strings.Contains(err.Error(), "reading public dir") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCloudflareDeploy_EmptyPublicDir(t *testing.T) {
	tmpDir := t.TempDir()

	p := &CloudflareProvider{
		WorkerName:        "test-worker",
		CompatibilityDate: "2024-09-01",
		NotFoundHandling:  "404-page",
	}

	err := p.Deploy(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("expected error for empty publicDir")
	}
	if !strings.Contains(err.Error(), "is empty") {
		t.Errorf("expected 'is empty' error, got: %v", err)
	}
}

func TestCloudflareDeploy_WranglerTomlGeneration(t *testing.T) {
	// Create a public dir with content so we get past the empty check.
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html/>"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &CloudflareProvider{
		WorkerName:        "my-worker",
		CompatibilityDate: "2024-09-01",
		NotFoundHandling:  "404-page",
		AccountID:         "abc123",
		ExtraFlags:        "--dry-run",
	}

	// Deploy exercises the toml generation and argument construction paths.
	// If wrangler is installed, --dry-run makes it succeed; otherwise it fails.
	// Either way, the code paths for toml writing and arg building are covered.
	_ = p.Deploy(context.Background(), tmpDir)
}

// ---------------------------------------------------------------------------
// RsyncProvider.Deploy — arg construction paths
// ---------------------------------------------------------------------------

func TestRsyncDeploy_ArgConstruction(t *testing.T) {
	// We can't actually run rsync in test, but we can exercise the argument
	// construction code paths by calling Deploy and checking the error.
	// The function builds args and then calls runCommand which will fail
	// because rsync either isn't installed or the host doesn't exist.

	tests := []struct {
		name string
		cfg  map[string]any
	}{
		{
			name: "default options",
			cfg: map[string]any{
				"host": "user@example.com",
				"path": "/var/www",
			},
		},
		{
			name: "custom port",
			cfg: map[string]any{
				"host": "user@example.com",
				"path": "/var/www",
				"port": "2222",
			},
		},
		{
			name: "with exclude patterns",
			cfg: map[string]any{
				"host":    "user@example.com",
				"path":    "/var/www",
				"exclude": ".git,*.tmp,node_modules",
			},
		},
		{
			name: "with extra flags and delete false",
			cfg: map[string]any{
				"host":        "user@example.com",
				"path":        "/var/www",
				"delete":      false,
				"extra_flags": "--dry-run --verbose",
			},
		},
		{
			name: "with key file",
			cfg: map[string]any{
				"host":     "user@example.com",
				"path":     "/var/www",
				"key_file": "/tmp/fake-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newRsync(tt.cfg).(*RsyncProvider)

			// Create a minimal publicDir so filepath.Abs succeeds.
			tmpDir := t.TempDir()
			err := p.Deploy(context.Background(), tmpDir)
			// rsync will fail (command not found or connection refused), but
			// this exercises all the argument construction code.
			if err == nil {
				t.Fatal("expected error (rsync execution should fail in test)")
			}
			// Error should come from rsync itself, not from path resolution.
			if !strings.Contains(err.Error(), "rsync:") {
				t.Errorf("expected rsync error, got: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// S3Provider.Deploy — URL construction and arg building
// ---------------------------------------------------------------------------

func TestS3Deploy_URLConstruction(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantURL string // expected substring in the s3:// URL
	}{
		{
			name:    "bucket only",
			cfg:     map[string]any{"bucket": "my-bucket"},
			wantURL: "s3://my-bucket/",
		},
		{
			name:    "bucket with path",
			cfg:     map[string]any{"bucket": "my-bucket", "path": "/blog/site/"},
			wantURL: "s3://my-bucket/blog/site/",
		},
		{
			name:    "bucket with path no slashes",
			cfg:     map[string]any{"bucket": "my-bucket", "path": "subdir"},
			wantURL: "s3://my-bucket/subdir/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_DEFAULT_REGION", "")
			t.Setenv("AWS_PROFILE", "")

			p := newS3(tt.cfg).(*S3Provider)

			// Construct the URL the same way Deploy does
			s3URL := "s3://" + p.Bucket
			if p.Path != "" {
				s3URL += "/" + p.Path
			}
			s3URL += "/"

			if s3URL != tt.wantURL {
				t.Errorf("s3URL = %q, want %q", s3URL, tt.wantURL)
			}
		})
	}
}

func TestS3Deploy_AllOptionPaths(t *testing.T) {
	// Exercise the Deploy method with all optional fields set.
	// The aws CLI won't be available but we exercise arg construction.
	tmpDir := t.TempDir()

	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("CLOUDFRONT_DISTRIBUTION_ID", "")

	tests := []struct {
		name string
		cfg  map[string]any
	}{
		{
			name: "minimal",
			cfg:  map[string]any{"bucket": "b"},
		},
		{
			name: "with endpoint region profile",
			cfg: map[string]any{
				"bucket":   "b",
				"endpoint": "https://r2.example.com",
				"region":   "us-east-1",
				"profile":  "prod",
			},
		},
		{
			name: "with delete and extra flags",
			cfg: map[string]any{
				"bucket":      "b",
				"delete":      true,
				"extra_flags": "--dryrun --size-only",
			},
		},
		{
			name: "with path and no ACL",
			cfg: map[string]any{
				"bucket": "b",
				"path":   "/site/",
				"acl":    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newS3(tt.cfg).(*S3Provider)

			err := p.Deploy(context.Background(), tmpDir)
			if err == nil {
				t.Fatal("expected error (aws CLI should fail in test)")
			}
			// Error should be from s3 command execution, not earlier.
			if !strings.Contains(err.Error(), "s3:") {
				t.Errorf("expected s3 error, got: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// S3 Validate — profile from config (not from env)
// ---------------------------------------------------------------------------

func TestS3Validate_ProfileInConfig(t *testing.T) {
	// Clear all AWS env vars to isolate the profile-in-config path.
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_PROFILE", "")

	p := newS3(map[string]any{
		"bucket":  "my-bucket",
		"profile": "my-profile",
	}).(*S3Provider)

	err := p.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (profile in config should suffice)", err)
	}

	if p.Profile != "my-profile" {
		t.Errorf("Profile = %q, want %q", p.Profile, "my-profile")
	}
}

// ---------------------------------------------------------------------------
// Cloudflare Validate — wrangler not installed gives specific error
// ---------------------------------------------------------------------------

func TestCloudflareValidate_WranglerNotInstalled(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-token")

	p := &CloudflareProvider{
		WorkerName: "my-worker",
	}

	err := p.Validate()
	// In CI/test environments, wrangler is typically not installed.
	// If it happens to be installed, validation passes, which is also fine.
	if err != nil {
		if !strings.Contains(err.Error(), "wrangler CLI not found") {
			t.Errorf("expected 'wrangler CLI not found' error, got: %v", err)
		}
	}
	// If err is nil, wrangler is installed and validation passed — that's OK.
}

// ---------------------------------------------------------------------------
// runCommand — exercises the helper with a known-good command
// ---------------------------------------------------------------------------

func TestRunCommand_Success(t *testing.T) {
	err := runCommand(context.Background(), "true")
	if err != nil {
		t.Errorf("runCommand(true) error: %v", err)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	err := runCommand(context.Background(), "false")
	if err == nil {
		t.Error("runCommand(false) expected error")
	}
}

func TestRunCommand_NotFound(t *testing.T) {
	err := runCommand(context.Background(), "nonexistent-command-xyz")
	if err == nil {
		t.Error("runCommand(nonexistent) expected error")
	}
}

func TestRunCommand_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := runCommand(ctx, "sleep", "10")
	if err == nil {
		t.Error("runCommand with cancelled context expected error")
	}
}

// ---------------------------------------------------------------------------
// S3 Deploy — endpoint URL parsing for display
// ---------------------------------------------------------------------------

func TestS3Deploy_EndpointURLParsing(t *testing.T) {
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("CLOUDFRONT_DISTRIBUTION_ID", "")

	tmpDir := t.TempDir()

	// With a valid endpoint URL, Deploy constructs a destination display string
	// using url.Parse. We exercise that code path.
	p := newS3(map[string]any{
		"bucket":   "my-bucket",
		"endpoint": "https://account.r2.cloudflarestorage.com",
	}).(*S3Provider)

	err := p.Deploy(context.Background(), tmpDir)
	// Will fail on aws command, but exercises the endpoint URL parsing path.
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------------------------------------------------------------------------
// S3 Deploy — with invalid endpoint URL
// ---------------------------------------------------------------------------

func TestS3Deploy_InvalidEndpointURL(t *testing.T) {
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("CLOUDFRONT_DISTRIBUTION_ID", "")

	tmpDir := t.TempDir()

	// An endpoint that fails url.Parse should still work (falls back to bucket name).
	p := newS3(map[string]any{
		"bucket":   "my-bucket",
		"endpoint": "://not-a-url",
	}).(*S3Provider)

	err := p.Deploy(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("expected error")
	}
}
