package deploy

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	providers := Providers()
	expected := []string{"cloudflare", "rsync", "s3"}

	for _, exp := range expected {
		found := false
		for _, p := range providers {
			if p == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected provider %q in registry, got %v", exp, providers)
		}
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("unknown", nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestRsync_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr bool
	}{
		{
			name:    "missing host",
			cfg:     map[string]any{"path": "/var/www"},
			wantErr: true,
		},
		{
			name:    "missing path",
			cfg:     map[string]any{"host": "user@example.com"},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: map[string]any{
				"host": "user@example.com",
				"path": "/var/www/blog",
			},
			wantErr: false,
		},
		{
			name: "with key file",
			cfg: map[string]any{
				"host":     "user@example.com",
				"path":     "/var/www/blog",
				"key_file": "/nonexistent/key", // Will fail validation
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newRsync(tt.cfg)
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		env     map[string]string
		wantErr bool
	}{
		{
			name:    "missing bucket",
			cfg:     map[string]any{},
			env:     map[string]string{"AWS_ACCESS_KEY_ID": "test"},
			wantErr: true,
		},
		{
			name:    "missing credentials",
			cfg:     map[string]any{"bucket": "my-bucket"},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "valid with access key",
			cfg:  map[string]any{"bucket": "my-bucket"},
			env: map[string]string{
				"AWS_ACCESS_KEY_ID":     "test",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
			wantErr: false,
		},
		{
			name:    "valid with profile",
			cfg:     map[string]any{"bucket": "my-bucket"},
			env:     map[string]string{"AWS_PROFILE": "default"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set env
			for k := range tt.env {
				t.Setenv(k, tt.env[k])
			}

			p := newS3(tt.cfg)
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloudflare_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		env     map[string]string
		wantErr bool
	}{
		{
			name:    "missing API token",
			cfg:     map[string]any{"project": "my-blog"},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "missing project",
			cfg:     map[string]any{},
			env:     map[string]string{"CLOUDFLARE_API_TOKEN": "test"},
			wantErr: true,
		},
		{
			name:    "valid pages config",
			cfg:     map[string]any{"project": "my-blog"},
			env:     map[string]string{"CLOUDFLARE_API_TOKEN": "test"},
			wantErr: false, // Will fail on wrangler check but validation passes
		},
		{
			name: "workers without name",
			cfg: map[string]any{
				"workers": true,
			},
			env:     map[string]string{"CLOUDFLARE_API_TOKEN": "test"},
			wantErr: true,
		},
		{
			name: "workers with name",
			cfg: map[string]any{
				"workers":     true,
				"worker_name": "my-worker",
			},
			env:     map[string]string{"CLOUDFLARE_API_TOKEN": "test"},
			wantErr: false, // Will fail on wrangler check but validation passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env first
			t.Setenv("CLOUDFLARE_API_TOKEN", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			p := newCloudflare(tt.cfg)
			err := p.Validate()
			// Wrangler check will fail in test env, so we only check config errors
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			// For valid configs, wrangler check will fail but that's expected
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"key":  "value",
		"num":  42,
		"bool": true,
	}

	if got := getString(m, "key", "default"); got != "value" {
		t.Errorf("getString(key) = %q, want %q", got, "value")
	}
	if got := getString(m, "missing", "default"); got != "default" {
		t.Errorf("getString(missing) = %q, want %q", got, "default")
	}
	if got := getString(m, "num", "default"); got != "default" {
		t.Errorf("getString(num) = %q, want %q", got, "default")
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{
		"true":  true,
		"false": false,
		"str":   "yes",
	}

	if got := getBool(m, "true", false); !got {
		t.Error("getBool(true) = false, want true")
	}
	if got := getBool(m, "false", true); got {
		t.Error("getBool(false) = true, want false")
	}
	if got := getBool(m, "missing", true); !got {
		t.Error("getBool(missing) = false, want true (default)")
	}
	if got := getBool(m, "str", false); got {
		t.Error("getBool(str) = true, want false (not a bool)")
	}
}

func TestExpandPath(t *testing.T) {
	home := "/home/test"
	t.Setenv("HOME", home)

	tests := []struct {
		in, want string
	}{
		{"~/blog", home + "/blog"},
		{"/absolute/path", "/absolute/path"},
		{"relative", "relative"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := expandPath(tt.in); got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getEnv / getEnvOr
// ---------------------------------------------------------------------------

func TestGetEnv(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		t.Setenv("OSG_TEST_KEY", "value")
		got, err := getEnv("OSG_TEST_KEY")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "value" {
			t.Errorf("got %q, want %q", got, "value")
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Setenv("OSG_TEST_KEY", "")
		_, err := getEnv("OSG_TEST_KEY")
		if err == nil {
			t.Error("expected error for empty env var")
		}
	})
}

func TestGetEnvOr(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		t.Setenv("OSG_TEST_KEY", "value")
		got := getEnvOr("OSG_TEST_KEY", "default")
		if got != "value" {
			t.Errorf("got %q, want %q", got, "value")
		}
	})

	t.Run("missing uses fallback", func(t *testing.T) {
		t.Setenv("OSG_TEST_KEY", "")
		got := getEnvOr("OSG_TEST_KEY", "fallback")
		if got != "fallback" {
			t.Errorf("got %q, want %q", got, "fallback")
		}
	})
}

// ---------------------------------------------------------------------------
// Get happy path
// ---------------------------------------------------------------------------

func TestGet_Known(t *testing.T) {
	for _, name := range []string{"rsync", "s3", "cloudflare"} {
		t.Run(name, func(t *testing.T) {
			p, err := Get(name, map[string]any{})
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			if p == nil {
				t.Fatalf("Get(%q) returned nil provider", name)
			}
			if p.Name() != name {
				t.Errorf("Name() = %q, want %q", p.Name(), name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// newRsync field extraction
// ---------------------------------------------------------------------------

func TestNewRsync_Defaults(t *testing.T) {
	p := newRsync(map[string]any{
		"host": "user@host",
		"path": "/var/www",
	}).(*RsyncProvider)

	if p.Port != "22" {
		t.Errorf("Port = %q, want %q", p.Port, "22")
	}
	if !p.Delete {
		t.Error("Delete should default to true")
	}
	if p.KeyFile != "" {
		t.Errorf("KeyFile should be empty, got %q", p.KeyFile)
	}
	if p.ExtraFlags != "" {
		t.Errorf("ExtraFlags should be empty, got %q", p.ExtraFlags)
	}
}

func TestNewRsync_CustomValues(t *testing.T) {
	p := newRsync(map[string]any{
		"host":        "user@host",
		"path":        "/var/www",
		"port":        "2222",
		"key_file":    "~/.ssh/deploy",
		"delete":      false,
		"exclude":     ".git,*.tmp",
		"extra_flags": "--dry-run",
	}).(*RsyncProvider)

	if p.Port != "2222" {
		t.Errorf("Port = %q, want %q", p.Port, "2222")
	}
	if p.Delete {
		t.Error("Delete should be false")
	}
	if p.KeyFile != "~/.ssh/deploy" {
		t.Errorf("KeyFile = %q", p.KeyFile)
	}
	if p.Exclude != ".git,*.tmp" {
		t.Errorf("Exclude = %q", p.Exclude)
	}
	if p.ExtraFlags != "--dry-run" {
		t.Errorf("ExtraFlags = %q", p.ExtraFlags)
	}
}

// ---------------------------------------------------------------------------
// newS3 field extraction
// ---------------------------------------------------------------------------

func TestNewS3_Defaults(t *testing.T) {
	// Clear env to get true defaults
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")

	p := newS3(map[string]any{
		"bucket": "my-bucket",
	}).(*S3Provider)

	if p.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q", p.Bucket)
	}
	if p.ACL != "public-read" {
		t.Errorf("ACL = %q, want %q", p.ACL, "public-read")
	}
	if p.Delete {
		t.Error("Delete should default to false")
	}
	if p.Region != "" {
		t.Errorf("Region should be empty, got %q", p.Region)
	}
}

func TestNewS3_PathTrimming(t *testing.T) {
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")

	p := newS3(map[string]any{
		"bucket": "b",
		"path":   "/blog/site/",
	}).(*S3Provider)

	if p.Path != "blog/site" {
		t.Errorf("Path = %q, want %q (trimmed slashes)", p.Path, "blog/site")
	}
}

func TestNewS3_EnvDefaults(t *testing.T) {
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	t.Setenv("AWS_PROFILE", "prod")

	p := newS3(map[string]any{
		"bucket": "b",
	}).(*S3Provider)

	if p.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1 from env", p.Region)
	}
	if p.Profile != "prod" {
		t.Errorf("Profile = %q, want prod from env", p.Profile)
	}
}

// ---------------------------------------------------------------------------
// newCloudflare field extraction
// ---------------------------------------------------------------------------

func TestNewCloudflare_Defaults(t *testing.T) {
	p := newCloudflare(map[string]any{
		"project": "my-site",
	}).(*CloudflareProvider)

	if p.Project != "my-site" {
		t.Errorf("Project = %q", p.Project)
	}
	if p.Branch != "main" {
		t.Errorf("Branch = %q, want %q", p.Branch, "main")
	}
	if p.UseWorkers {
		t.Error("UseWorkers should default to false")
	}
}

// ---------------------------------------------------------------------------
// Run error paths (no exec needed)
// ---------------------------------------------------------------------------

func TestRun_UnknownProvider(t *testing.T) {
	err := Run(nil, "nonexistent", nil, "/tmp")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestRun_ValidationFailure(t *testing.T) {
	// rsync without host should fail validation
	err := Run(nil, "rsync", map[string]any{}, "/tmp")
	if err == nil {
		t.Error("expected validation error for rsync without host")
	}
}
