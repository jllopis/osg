package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/build"
	"osg/internal/frontmatter"
	"osg/internal/site"
	"osg/internal/summary"
)

// resolveVaultSource validates a user-supplied path that should point
// to a markdown file inside cfg.ContentDir. It rejects absolute paths,
// `..` traversal, and paths that escape the vault root. Returns the
// cleaned relative path (forward-slashes) and the absolute path.
func (s *Server) resolveVaultSource(source string) (rel string, abs string, err error) {
	contentDir := strings.TrimSpace(s.opts.Cfg.ContentDir)
	if contentDir == "" {
		return "", "", fmt.Errorf("content_dir not configured")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", fmt.Errorf("missing source")
	}
	clean := filepath.Clean(source)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", "", fmt.Errorf("invalid source path")
	}
	abs = filepath.Join(contentDir, clean)
	rel2, err := filepath.Rel(contentDir, abs)
	if err != nil || strings.HasPrefix(rel2, "..") {
		return "", "", fmt.Errorf("source escapes content dir")
	}
	return filepath.ToSlash(rel2), abs, nil
}

// handleSummaryInvalidate drops the AI summary cache entry for a
// single page so the next build re-calls the LLM. Form input is the
// markdown file path relative to cfg.ContentDir.
func (s *Server) handleSummaryInvalidate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	rel, abs, err := s.resolveVaultSource(r.PostFormValue("source"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	page, _, err := site.ParseFile(s.opts.Cfg.ContentDir, s.opts.Cfg.BaseURL, abs)
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
		s.opts.Logger.Info("ai summary invalidated", "source", rel, "hash", hash, "removed", removed)
	}
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/vault"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// PageEditView is the data passed to the page-edit template. It
// carries enough context that the editor can show the user what the
// site currently displays as the summary, where it came from
// (frontmatter, AI cache or extract), and let them write a value to
// `osg.summary` to override it.
type PageEditView struct {
	Source           string
	Title            string
	Section          string
	DateLabel        string
	Draft            bool
	CurrentSummary   string // value already used by build (osg.summary, top-level summary, or AI/extract fallback)
	OverrideSummary  string // current osg.summary if any (the value that survives across rebuilds)
	HasOverride      bool
	AICachedSummary  string // value present in .osg/cache/ai-summaries.json, if any
	HasAICached      bool
	HasFrontmatterFM bool // whether the file has a YAML frontmatter block at all
}

// handleVaultPage renders the per-page editor at /vault/page?source=…
// The page is read-only except for the osg.summary field which can be
// edited and persisted via /summary/save.
func (s *Server) handleVaultPage(w http.ResponseWriter, r *http.Request) {
	rel, abs, err := s.resolveVaultSource(r.URL.Query().Get("source"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, "read failed: "+err.Error(), http.StatusNotFound)
		return
	}
	fm, _, hasFM, err := frontmatter.SplitFrontmatter(data)
	if err != nil {
		http.Error(w, "parse frontmatter: "+err.Error(), http.StatusInternalServerError)
		return
	}
	page, _, parseErr := site.ParseFile(s.opts.Cfg.ContentDir, s.opts.Cfg.BaseURL, abs)
	if parseErr != nil || page == nil {
		http.Error(w, "parse failed: "+errOrEmpty(parseErr), http.StatusInternalServerError)
		return
	}

	override, _ := readOSGSummary(fm)
	view := viewData{
		Title:   "Edit page",
		Active:  "vault",
		Version: s.opts.Version,
		State:   State{ContentDir: s.opts.Cfg.ContentDir},
		PageEdit: &PageEditView{
			Source:           rel,
			Title:            page.Title,
			Section:          sectionOf(page.Path),
			Draft:            page.Draft,
			CurrentSummary:   page.Summary,
			OverrideSummary:  override,
			HasOverride:      override != "",
			HasFrontmatterFM: hasFM,
		},
	}
	if !page.Date.IsZero() {
		view.PageEdit.DateLabel = page.Date.Format("2006-01-02")
	}
	hash := build.ContentHash(page.RawContent)
	if cached, ok := build.LookupAISummary(s.opts.Cfg, hash, s.opts.Logger); ok {
		view.PageEdit.AICachedSummary = cached
		view.PageEdit.HasAICached = true
	}
	s.render(w, "page", view)
}

// handleSummarySave writes the submitted summary into osg.summary in
// the page's frontmatter and, when build/deploy form fields are set,
// triggers the corresponding pipeline operations through the runner
// (fire-and-forget; the user lands on /actions to watch progress in
// the flow drawer).
//
// Form fields:
//   - source: relative markdown path
//   - summary: new value for osg.summary (empty deletes the override)
//   - build:  "1" to trigger build after saving
//   - deploy: "1" to also chain deploy after build
func (s *Server) handleSummarySave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	rel, abs, err := s.resolveVaultSource(r.PostFormValue("source"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	summary := strings.TrimSpace(r.PostFormValue("summary"))
	doBuild := r.PostFormValue("build") == "1"
	doDeploy := r.PostFormValue("deploy") == "1"
	if doDeploy {
		// Deploy without a fresh build would push stale HTML; treat
		// the "Save, build & deploy" button as build+deploy.
		doBuild = true
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, "read failed: "+err.Error(), http.StatusNotFound)
		return
	}
	out, err := frontmatter.UpdateField(data, "osg.summary", summary)
	if err != nil {
		http.Error(w, "update frontmatter: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := writeAtomic(abs, out); err != nil {
		http.Error(w, "write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if s.opts.Logger != nil {
		s.opts.Logger.Info("frontmatter updated",
			"source", rel,
			"key", "osg.summary",
			"action", overrideAction(summary),
			"build", doBuild,
			"deploy", doDeploy)
	}

	dest := "/vault/page?source=" + rel
	if doBuild && s.opts.Runner != nil {
		seq := []string{"build"}
		if doDeploy {
			seq = append(seq, "deploy")
		}
		go func(steps []string) {
			failed, runErr := s.opts.Runner.RunFlow(steps)
			if runErr != nil && s.opts.Logger != nil {
				s.opts.Logger.Warn("post-save flow aborted",
					"failed_at", failed, "error", runErr)
			}
		}(seq)
		dest = "/actions"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// generateSummaryNow opens a fresh kairos provider and runs a single
// blocking Summarize call against the page at abs. The returned
// string is the same one the build pipeline would have produced — it
// is shared by the regenerate (persists the result) and suggest
// (preview only) handlers. Honours cfg.AI.Timeout.
func (s *Server) generateSummaryNow(ctx context.Context, abs string) (page *site.Page, summaryText string, hash string, err error) {
	if !strings.EqualFold(s.opts.Cfg.SummaryStrategy, "ai") && s.opts.Cfg.AI.Provider == "" {
		return nil, "", "", fmt.Errorf("AI summary provider not configured")
	}
	page, _, err = site.ParseFile(s.opts.Cfg.ContentDir, s.opts.Cfg.BaseURL, abs)
	if err != nil || page == nil {
		return nil, "", "", fmt.Errorf("parse page: %w", err)
	}
	aiCfg := summary.AIConfig{
		Provider:     s.opts.Cfg.AI.Provider,
		Model:        s.opts.Cfg.AI.Model,
		APIKey:       s.opts.Cfg.AI.APIKey,
		BaseURL:      s.opts.Cfg.AI.BaseURL,
		SystemPrompt: s.opts.Cfg.AI.SystemPrompt,
		Language:     s.opts.Cfg.DefaultLanguage,
	}
	provider, err := summary.NewKairosProvider(ctx, aiCfg)
	if err != nil {
		return nil, "", "", fmt.Errorf("init AI provider: %w", err)
	}
	timeout := time.Duration(s.opts.Cfg.AI.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	text, err := provider.Summarize(callCtx, page.Title, page.RawContent)
	if err != nil {
		return nil, "", "", err
	}
	return page, text, build.ContentHash(page.RawContent), nil
}

// handleSummaryRegenerate runs a single AI call for the named page,
// stores the result in the summary cache, and redirects back to the
// vault. Synchronous because the user is waiting for the row to
// refresh; the kairos retry policy bounds the worst-case latency.
func (s *Server) handleSummaryRegenerate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	rel, abs, err := s.resolveVaultSource(r.PostFormValue("source"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, text, hash, err := s.generateSummaryNow(r.Context(), abs)
	if err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("summary regenerate failed", "source", rel, "error", err)
		}
		http.Error(w, "regenerate failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := build.UpsertAISummary(s.opts.Cfg, hash, text, s.opts.Cfg.AI.Provider, s.opts.Cfg.AI.Model, s.opts.Logger); err != nil {
		http.Error(w, "store summary: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if s.opts.Logger != nil {
		s.opts.Logger.Info("ai summary regenerated", "source", rel, "hash", hash, "chars", len(text))
	}
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/vault"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// handleSummarySuggest returns the AI-generated summary for the page
// as JSON without persisting it. Used by the editor's "AI Suggestion"
// button to fill the textarea so the user can review before saving.
func (s *Server) handleSummarySuggest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	source := r.PostFormValue("source")
	if source == "" {
		source = r.URL.Query().Get("source")
	}
	_, abs, err := s.resolveVaultSource(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, text, _, err := s.generateSummaryNow(r.Context(), abs)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"suggestion": text})
}

// overrideAction labels what the save did for log output. Useful when
// scanning logs to see whether a save deleted or replaced the value.
func overrideAction(value string) string {
	if value == "" {
		return "deleted"
	}
	return "set"
}

// writeAtomic writes data to path via a temp file in the same dir,
// then renames. This avoids leaving the markdown file half-written if
// the process crashes mid-write — important because the editor is
// persisting user content into the vault.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".osg-save-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// readOSGSummary extracts osg.summary (or the legacy osg.abstract
// alias) from the parsed frontmatter map. Returns the value and a
// flag indicating which key was used.
func readOSGSummary(fm map[string]any) (string, string) {
	osg, _ := fm["osg"].(map[string]any)
	if osg == nil {
		return "", ""
	}
	for _, key := range []string{"summary", "abstract"} {
		if v, ok := osg[key].(string); ok && strings.TrimSpace(v) != "" {
			return v, key
		}
	}
	return "", ""
}

func errOrEmpty(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
