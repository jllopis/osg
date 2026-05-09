package ui

import (
	"net/http"
	"path/filepath"
	"strings"

	"osg/internal/build"
	"osg/internal/site"
)

// handleSummaryInvalidate drops the AI summary cache entry for a
// single page so the next build re-calls the LLM. Form input:
//   - source: path to the markdown file relative to cfg.ContentDir
//
// The path is sanitized (Clean + relativity check) so a malicious or
// misspelled value cannot escape the vault. On success redirects back
// to /vault.
func (s *Server) handleSummaryInvalidate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	source := strings.TrimSpace(r.PostFormValue("source"))
	if source == "" {
		http.Error(w, "missing source", http.StatusBadRequest)
		return
	}
	contentDir := strings.TrimSpace(s.opts.Cfg.ContentDir)
	if contentDir == "" {
		http.Error(w, "content_dir not configured", http.StatusServiceUnavailable)
		return
	}

	clean := filepath.Clean(source)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		http.Error(w, "invalid source path", http.StatusBadRequest)
		return
	}
	abs := filepath.Join(contentDir, clean)
	rel, err := filepath.Rel(contentDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "source escapes content dir", http.StatusBadRequest)
		return
	}

	page, _, err := site.ParseFile(contentDir, s.opts.Cfg.BaseURL, abs)
	if err != nil || page == nil {
		http.Error(w, "parse failed: "+errOrEmpty(err), http.StatusNotFound)
		return
	}
	hash := build.ContentHash(page.RawContent)
	removed, err := build.InvalidateAISummary(s.opts.Cfg, hash, s.opts.Logger)
	if err != nil {
		http.Error(w, "invalidate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if s.opts.Logger != nil {
		s.opts.Logger.Info("ai summary invalidated", "source", clean, "hash", hash, "removed", removed)
	}

	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/vault"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func errOrEmpty(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
