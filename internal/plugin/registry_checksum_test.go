package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newReleaseServer serves a GitHub "latest" release for owner/repo whose
// single .wasm asset has the given bytes and digest, plus the asset bytes.
func newReleaseServer(t *testing.T, owner, repo, tag string, body []byte, digest string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	rel := GitHubRelease{
		TagName: tag,
		Assets: []GitHubAsset{{
			Name:               repo + ".wasm",
			BrowserDownloadURL: srv.URL + "/dl/" + repo + ".wasm",
			Size:               len(body),
			Digest:             digest,
		}},
	}
	mux.HandleFunc("/repos/"+owner+"/"+repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/dl/"+repo+".wasm", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	})
	return srv
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestInstallFromGitHub_ChecksumVerification(t *testing.T) {
	body := []byte("\x00asm-good-plugin-bytes")
	good := sha256Hex(body)

	t.Run("digest match installs and records sha256", func(t *testing.T) {
		srv := newReleaseServer(t, "acme", "plug", "v1.0.0", body, "sha256:"+good)
		orig := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = orig }()

		base := t.TempDir()
		pluginsDir := filepath.Join(base, "plugins")
		name, err := InstallFromGitHub(context.Background(), "acme", "plug", "", pluginsDir)
		if err != nil {
			t.Fatalf("install: %v", err)
		}
		if _, err := os.Stat(filepath.Join(pluginsDir, "plug.wasm")); err != nil {
			t.Fatalf("plugin not installed: %v", err)
		}
		// No leftover temp file.
		if _, err := os.Stat(filepath.Join(pluginsDir, "plug.wasm.download")); !os.IsNotExist(err) {
			t.Errorf("temp download file was left behind")
		}
		lock, _ := LoadLockFile(base)
		entry, ok := lock.Get(name)
		if !ok || entry.SHA256 != good {
			t.Errorf("lock sha256 = %q (ok=%v), want %q", entry.SHA256, ok, good)
		}
	})

	t.Run("digest mismatch refuses and leaves nothing", func(t *testing.T) {
		srv := newReleaseServer(t, "acme", "plug", "v1.0.0", body, "sha256:"+sha256Hex([]byte("different")))
		orig := githubAPIBase
		githubAPIBase = srv.URL
		defer func() { githubAPIBase = orig }()

		pluginsDir := filepath.Join(t.TempDir(), "plugins")
		if _, err := InstallFromGitHub(context.Background(), "acme", "plug", "", pluginsDir); err == nil {
			t.Fatal("expected checksum-mismatch error, got nil")
		}
		if _, err := os.Stat(filepath.Join(pluginsDir, "plug.wasm")); !os.IsNotExist(err) {
			t.Errorf("tampered plugin should not be installed")
		}
	})

	t.Run("re-tagged release with same version is refused", func(t *testing.T) {
		base := t.TempDir()
		pluginsDir := filepath.Join(base, "plugins")

		srv1 := newReleaseServer(t, "acme", "plug", "v1.0.0", body, "sha256:"+good)
		orig := githubAPIBase
		githubAPIBase = srv1.URL
		if _, err := InstallFromGitHub(context.Background(), "acme", "plug", "", pluginsDir); err != nil {
			githubAPIBase = orig
			t.Fatalf("first install: %v", err)
		}

		// Same tag, different bytes (digest matches the new bytes, so the
		// transit check passes, but the recorded hash differs).
		body2 := []byte("\x00asm-re-tagged-bytes")
		srv2 := newReleaseServer(t, "acme", "plug", "v1.0.0", body2, "sha256:"+sha256Hex(body2))
		githubAPIBase = srv2.URL
		defer func() { githubAPIBase = orig }()

		if _, err := InstallFromGitHub(context.Background(), "acme", "plug", "", pluginsDir); err == nil {
			t.Fatal("expected re-tag refusal, got nil")
		}
		// The original good bytes must remain in place.
		got, _ := os.ReadFile(filepath.Join(pluginsDir, "plug.wasm"))
		if string(got) != string(body) {
			t.Errorf("original plugin was overwritten by re-tagged release")
		}
	})
}
