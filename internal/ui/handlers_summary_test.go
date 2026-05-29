package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"osg/internal/frontmatter"
)

// summaryWritePage writes a markdown file with the given relative path
// (relative to the server's ContentDir) and body, returning its
// absolute path. Helper local to this file to avoid clashing with the
// shared writeTempMarkdown signature.
func summaryWritePage(t *testing.T, s *Server, rel, body string) string {
	t.Helper()
	abs := filepath.Join(s.opts.Cfg.ContentDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	return abs
}

const summaryFixtureBody = `---
title: Hello World
osg:
  summary: An overriding summary line.
---

This is the body of the post with enough words to be parsed.
`

func TestResolveVaultSource(t *testing.T) {
	s := newTestServer(t, nil)
	contentDir := s.opts.Cfg.ContentDir

	t.Run("valid and traversal cases", func(t *testing.T) {
		cases := []struct {
			name    string
			source  string
			wantErr bool
			wantRel string
		}{
			{"simple file", "post.md", false, "post.md"},
			{"nested file", "blog/post.md", false, "blog/post.md"},
			{"absolute path", "/etc/passwd", true, ""},
			{"single dotdot", "../escape.md", true, ""},
			{"double dotdot", "../../x", true, ""},
			{"empty source", "", true, ""},
			{"whitespace only", "   ", true, ""},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rel, abs, err := s.resolveVaultSource(tc.source)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("resolveVaultSource(%q): want error, got rel=%q abs=%q", tc.source, rel, abs)
					}
					return
				}
				if err != nil {
					t.Fatalf("resolveVaultSource(%q): unexpected error: %v", tc.source, err)
				}
				if rel != tc.wantRel {
					t.Fatalf("resolveVaultSource(%q): rel = %q, want %q", tc.source, rel, tc.wantRel)
				}
				// abs must stay inside the content dir.
				cleanContent := filepath.Clean(contentDir)
				if abs != filepath.Join(cleanContent, filepath.FromSlash(tc.wantRel)) {
					t.Fatalf("abs %q not the expected join of content dir %q + rel %q", abs, cleanContent, tc.wantRel)
				}
				r, err := filepath.Rel(cleanContent, abs)
				if err != nil || strings.HasPrefix(r, "..") {
					t.Fatalf("abs %q escapes content dir %q (rel %q, err %v)", abs, cleanContent, r, err)
				}
			})
		}
	})

	t.Run("content_dir not configured", func(t *testing.T) {
		s.opts.Cfg.ContentDir = ""
		_, _, err := s.resolveVaultSource("post.md")
		if err == nil {
			t.Fatal("want error when ContentDir unset")
		}
		if !strings.Contains(err.Error(), "content_dir") {
			t.Fatalf("error %q does not mention content_dir", err.Error())
		}
	})
}

func TestHandleVaultPage(t *testing.T) {
	s := newTestServer(t, nil)
	summaryWritePage(t, s, "post.md", summaryFixtureBody)

	t.Run("ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/vault/page?source=post.md", nil)
		rec := httptest.NewRecorder()
		s.handleVaultPage(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Hello World") {
			t.Fatalf("response missing page title; body: %s", body)
		}
		if !strings.Contains(body, "An overriding summary line.") {
			t.Fatalf("response missing summary; body: %s", body)
		}
	})

	t.Run("bad source absolute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/vault/page?source=/etc/passwd", nil)
		rec := httptest.NewRecorder()
		s.handleVaultPage(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("absolute source: status = %d, want 400", rec.Code)
		}
	})

	t.Run("bad source traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/vault/page?source=../escape.md", nil)
		rec := httptest.NewRecorder()
		s.handleVaultPage(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("traversal source: status = %d, want 400", rec.Code)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/vault/page?source=missing.md", nil)
		rec := httptest.NewRecorder()
		s.handleVaultPage(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("missing file: status = %d, want 404", rec.Code)
		}
	})
}

func TestHandleSummarySave(t *testing.T) {
	t.Run("update override and redirect", func(t *testing.T) {
		s := newTestServer(t, nil)
		abs := summaryWritePage(t, s, "post.md", summaryFixtureBody)

		form := url.Values{}
		form.Set("source", "post.md")
		form.Set("summary", "A freshly saved summary.")
		req := httptest.NewRequest(http.MethodPost, "/summary/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummarySave(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want 303; body: %s", rec.Code, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc != "/vault/page?source=post.md" {
			t.Fatalf("Location = %q, want /vault/page?source=post.md", loc)
		}

		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		fm, _, _, err := frontmatter.SplitFrontmatter(data)
		if err != nil {
			t.Fatalf("split frontmatter: %v", err)
		}
		got, _ := readOSGSummary(fm)
		if got != "A freshly saved summary." {
			t.Fatalf("osg.summary = %q, want %q", got, "A freshly saved summary.")
		}
	})

	t.Run("empty summary deletes override", func(t *testing.T) {
		s := newTestServer(t, nil)
		abs := summaryWritePage(t, s, "post.md", summaryFixtureBody)

		form := url.Values{}
		form.Set("source", "post.md")
		form.Set("summary", "")
		req := httptest.NewRequest(http.MethodPost, "/summary/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummarySave(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want 303", rec.Code)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		fm, _, _, err := frontmatter.SplitFrontmatter(data)
		if err != nil {
			t.Fatalf("split frontmatter: %v", err)
		}
		if got, _ := readOSGSummary(fm); got != "" {
			t.Fatalf("override should be deleted, got %q", got)
		}
	})

	t.Run("build=1 redirects to actions", func(t *testing.T) {
		s := newTestServer(t, newTestRunner(t))
		summaryWritePage(t, s, "post.md", summaryFixtureBody)

		form := url.Values{}
		form.Set("source", "post.md")
		form.Set("summary", "Build me.")
		form.Set("build", "1")
		req := httptest.NewRequest(http.MethodPost, "/summary/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummarySave(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want 303", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/actions" {
			t.Fatalf("Location = %q, want /actions", loc)
		}
	})

	t.Run("bad source", func(t *testing.T) {
		s := newTestServer(t, nil)
		form := url.Values{}
		form.Set("source", "../escape.md")
		form.Set("summary", "x")
		req := httptest.NewRequest(http.MethodPost, "/summary/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummarySave(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestHandleSummaryInvalidate(t *testing.T) {
	t.Run("html redirect", func(t *testing.T) {
		s := newTestServer(t, nil)
		summaryWritePage(t, s, "post.md", summaryFixtureBody)

		form := url.Values{}
		form.Set("source", "post.md")
		req := httptest.NewRequest(http.MethodPost, "/summary/invalidate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummaryInvalidate(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want 303; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("json response", func(t *testing.T) {
		s := newTestServer(t, nil)
		summaryWritePage(t, s, "post.md", summaryFixtureBody)

		form := url.Values{}
		form.Set("source", "post.md")
		req := httptest.NewRequest(http.MethodPost, "/summary/invalidate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		s.handleSummaryInvalidate(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "\"ok\"") {
			t.Fatalf("json body missing ok: %s", body)
		}
		if !strings.Contains(body, "\"removed\"") {
			t.Fatalf("json body missing removed: %s", body)
		}
	})

	t.Run("bad source", func(t *testing.T) {
		s := newTestServer(t, nil)
		form := url.Values{}
		form.Set("source", "/etc/passwd")
		req := httptest.NewRequest(http.MethodPost, "/summary/invalidate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.handleSummaryInvalidate(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestHandleSummaryRegenerateNoProvider(t *testing.T) {
	s := newTestServer(t, nil)
	// Force the "AI summary provider not configured" branch.
	s.opts.Cfg.SummaryStrategy = "auto"
	s.opts.Cfg.AI.Provider = ""
	summaryWritePage(t, s, "post.md", summaryFixtureBody)

	form := url.Values{}
	form.Set("source", "post.md")
	req := httptest.NewRequest(http.MethodPost, "/summary/regenerate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleSummaryRegenerate(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not configured") {
		t.Fatalf("body missing provider error: %s", rec.Body.String())
	}
}

func TestHandleSummarySuggestNoProvider(t *testing.T) {
	s := newTestServer(t, nil)
	s.opts.Cfg.SummaryStrategy = "auto"
	s.opts.Cfg.AI.Provider = ""
	summaryWritePage(t, s, "post.md", summaryFixtureBody)

	form := url.Values{}
	form.Set("source", "post.md")
	req := httptest.NewRequest(http.MethodPost, "/summary/suggest", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleSummarySuggest(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "\"error\"") {
		t.Fatalf("json body missing error key: %s", body)
	}
	if !strings.Contains(body, "not configured") {
		t.Fatalf("body missing provider error: %s", body)
	}
}

func TestOverrideAction(t *testing.T) {
	if got := overrideAction(""); got != "deleted" {
		t.Fatalf("overrideAction(\"\") = %q, want deleted", got)
	}
	if got := overrideAction("x"); got != "set" {
		t.Fatalf("overrideAction(\"x\") = %q, want set", got)
	}
}

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.md")
	if err := writeAtomic(target, []byte("hello world")); err != nil {
		t.Fatalf("writeAtomic: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("content = %q, want %q", got, "hello world")
	}
	// No temp files should linger in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".osg-save-") {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one file, got %d", len(entries))
	}
}

func TestReadOSGSummary(t *testing.T) {
	cases := []struct {
		name    string
		fm      map[string]any
		wantVal string
		wantKey string
	}{
		{
			name:    "summary key",
			fm:      map[string]any{"osg": map[string]any{"summary": "from summary"}},
			wantVal: "from summary",
			wantKey: "summary",
		},
		{
			name:    "legacy abstract alias",
			fm:      map[string]any{"osg": map[string]any{"abstract": "from abstract"}},
			wantVal: "from abstract",
			wantKey: "abstract",
		},
		{
			name:    "summary preferred over abstract",
			fm:      map[string]any{"osg": map[string]any{"summary": "s", "abstract": "a"}},
			wantVal: "s",
			wantKey: "summary",
		},
		{
			name:    "no osg block",
			fm:      map[string]any{"title": "x"},
			wantVal: "",
			wantKey: "",
		},
		{
			name:    "empty osg block",
			fm:      map[string]any{"osg": map[string]any{}},
			wantVal: "",
			wantKey: "",
		},
		{
			name:    "whitespace value ignored",
			fm:      map[string]any{"osg": map[string]any{"summary": "   "}},
			wantVal: "",
			wantKey: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, key := readOSGSummary(tc.fm)
			if val != tc.wantVal || key != tc.wantKey {
				t.Fatalf("readOSGSummary = (%q, %q), want (%q, %q)", val, key, tc.wantVal, tc.wantKey)
			}
		})
	}
}

func TestErrOrEmpty(t *testing.T) {
	if got := errOrEmpty(nil); got != "" {
		t.Fatalf("errOrEmpty(nil) = %q, want \"\"", got)
	}
	if got := errOrEmpty(os.ErrNotExist); got != os.ErrNotExist.Error() {
		t.Fatalf("errOrEmpty(err) = %q, want %q", got, os.ErrNotExist.Error())
	}
}

func TestAppendFlash(t *testing.T) {
	cases := []struct {
		name   string
		rawURL string
		key    string
		value  string
		want   string
	}{
		{"no existing query", "/vault", "summary", "ok", "/vault?summary=ok"},
		{"existing query", "/vault?page=1", "summary", "ok", "/vault?page=1&summary=ok"},
		{"escapes value", "/vault", "summary", "regenerated:a/b c", "/vault?summary=" + url.QueryEscape("regenerated:a/b c")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := appendFlash(tc.rawURL, tc.key, tc.value); got != tc.want {
				t.Fatalf("appendFlash = %q, want %q", got, tc.want)
			}
		})
	}
}
