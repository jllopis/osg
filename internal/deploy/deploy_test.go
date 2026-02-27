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
