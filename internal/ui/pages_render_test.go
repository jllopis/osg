package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pagesRenderHTTPServer starts an httptest.Server over the Server's mux and
// registers cleanup. File-unique helper (not shared via testsupport).
func pagesRenderHTTPServer(t *testing.T, s *Server) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(s.mux)
	t.Cleanup(srv.Close)
	return srv
}

// noRedirectClient returns an *http.Client that does not follow redirects,
// so callers can inspect the 3xx response directly.
func pagesRenderNoRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// pagesRenderGet issues a GET against srvURL+path using the supplied client
// and returns the response together with its fully-read body.
func pagesRenderGet(t *testing.T, client *http.Client, url string) (*http.Response, string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read body for %s: %v", url, err)
	}
	return resp, string(body)
}

// TestHTMLPagesRenderOK drives the full router via httptest and confirms each
// SSR page renders: status 200, non-empty body, and an HTML content type.
func TestHTMLPagesRenderOK(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))

	// Give /vault and /assets something to enumerate.
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	contentDir := filepath.Join(root, s.opts.Cfg.ContentDir)
	writeTempMarkdown(t, contentDir, "posts/hello.md", "---\ntitle: Hello\n---\n\nbody text\n")
	staticDir := filepath.Join(root, s.opts.Cfg.StaticDir)
	if err := os.WriteFile(filepath.Join(staticDir, "logo.png"), []byte("\x89PNG\r\n\x1a\n"), 0o644); err != nil {
		t.Fatalf("write static asset: %v", err)
	}

	srv := pagesRenderHTTPServer(t, s)

	paths := []string{
		"/", "/actions", "/history", "/vault", "/plugins", "/assets",
		"/scheduler", "/import", "/themes", "/audit", "/services",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			resp, body := pagesRenderGet(t, srv.Client(), srv.URL+p)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", p, resp.StatusCode)
			}
			if strings.TrimSpace(body) == "" {
				t.Fatalf("GET %s returned empty body", p)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Fatalf("GET %s Content-Type = %q, want text/html", p, ct)
			}
		})
	}
}

// TestHealthEndpoint confirms /health returns the literal "ok" body.
func TestHealthEndpoint(t *testing.T) {
	srv := pagesRenderHTTPServer(t, newTestServer(t, newTestRunner(t)))
	resp, body := pagesRenderGet(t, srv.Client(), srv.URL+"/health")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

// TestFaviconRedirect confirms /favicon.ico 302-redirects to the embedded SVG.
func TestFaviconRedirect(t *testing.T) {
	srv := pagesRenderHTTPServer(t, newTestServer(t, newTestRunner(t)))
	client := pagesRenderNoRedirectClient()
	resp, _ := pagesRenderGet(t, client, srv.URL+"/favicon.ico")
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/assets/favicon.svg" {
		t.Fatalf("Location = %q, want /assets/favicon.svg", loc)
	}
}

// TestEmbeddedAssetServed confirms an embedded static asset is served 200.
func TestEmbeddedAssetServed(t *testing.T) {
	srv := pagesRenderHTTPServer(t, newTestServer(t, newTestRunner(t)))
	resp, body := pagesRenderGet(t, srv.Client(), srv.URL+"/assets/style.css")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if strings.TrimSpace(body) == "" {
		t.Fatalf("style.css body is empty")
	}
}

// TestDashboardNotFound confirms handleDashboard 404s any path other than "/".
func TestDashboardNotFound(t *testing.T) {
	srv := pagesRenderHTTPServer(t, newTestServer(t, newTestRunner(t)))
	resp, _ := pagesRenderGet(t, srv.Client(), srv.URL+"/does-not-exist")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// TestHistoryWithFilters exercises the history filter path (?status=ok&kind=task).
func TestHistoryWithFilters(t *testing.T) {
	srv := pagesRenderHTTPServer(t, newTestServer(t, newTestRunner(t)))
	resp, body := pagesRenderGet(t, srv.Client(), srv.URL+"/history?status=ok&kind=task")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if strings.TrimSpace(body) == "" {
		t.Fatalf("history body is empty")
	}
}

// TestAuditRendersFindings points PublicDir at a temp dir holding a tiny HTML
// file and confirms audit.Run executes and the audit page renders 200.
func TestAuditRendersFindings(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	publicDir := filepath.Join(root, s.opts.Cfg.PublicDir)
	// A minimal page lacking <title>/meta description triggers audit findings,
	// exercising the findings table render path.
	html := "<!doctype html><html><body><p>hi</p></body></html>"
	if err := os.WriteFile(filepath.Join(publicDir, "index.html"), []byte(html), 0o644); err != nil {
		t.Fatalf("write public html: %v", err)
	}

	srv := pagesRenderHTTPServer(t, s)
	resp, body := pagesRenderGet(t, srv.Client(), srv.URL+"/audit")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if strings.TrimSpace(body) == "" {
		t.Fatalf("audit body is empty")
	}
}
