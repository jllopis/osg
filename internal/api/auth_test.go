package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

// --- BuildAuthProviders ---

func TestBuildAuthProviders_GitHub(t *testing.T) {
	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github", ClientID: "gh-id", ClientSecret: "gh-secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")

	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	gh, ok := providers["github"]
	if !ok {
		t.Fatal("expected github provider")
	}
	if gh.OAuthCfg.ClientID != "gh-id" {
		t.Errorf("client_id = %q, want gh-id", gh.OAuthCfg.ClientID)
	}
	if gh.OAuthCfg.RedirectURL != "https://example.com/api/v1/auth/github/callback" {
		t.Errorf("redirect_url = %q", gh.OAuthCfg.RedirectURL)
	}
	if len(gh.OAuthCfg.Scopes) != 1 || gh.OAuthCfg.Scopes[0] != "read:user" {
		t.Errorf("scopes = %v, want [read:user]", gh.OAuthCfg.Scopes)
	}
}

func TestBuildAuthProviders_Google(t *testing.T) {
	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "google", ClientID: "g-id", ClientSecret: "g-secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://mysite.dev")

	g, ok := providers["google"]
	if !ok {
		t.Fatal("expected google provider")
	}
	if g.OAuthCfg.ClientID != "g-id" {
		t.Errorf("client_id = %q, want g-id", g.OAuthCfg.ClientID)
	}
	if g.OAuthCfg.RedirectURL != "https://mysite.dev/api/v1/auth/google/callback" {
		t.Errorf("redirect_url = %q", g.OAuthCfg.RedirectURL)
	}
	if len(g.OAuthCfg.Scopes) != 3 {
		t.Errorf("scopes = %v, want 3 scopes", g.OAuthCfg.Scopes)
	}
}

func TestBuildAuthProviders_MultipleMixed(t *testing.T) {
	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github", ClientID: "gh-id", ClientSecret: "gh-secret"},
			{Provider: "google", ClientID: "g-id", ClientSecret: "g-secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestBuildAuthProviders_UnknownSkipped(t *testing.T) {
	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "unknown", ClientID: "x", ClientSecret: "y"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")
	if len(providers) != 0 {
		t.Errorf("expected 0 providers for unknown, got %d", len(providers))
	}
}

func TestBuildAuthProviders_Empty(t *testing.T) {
	cfg := config.CommentsConfig{}
	providers := BuildAuthProviders(cfg, "https://example.com")
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

// --- providerLabel ---

func TestProviderLabel(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"github", "GitHub"},
		{"google", "Google"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := providerLabel(tt.name)
		if got != tt.want {
			t.Errorf("providerLabel(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// --- Auth handler helpers ---

func testAuthHandlers(t *testing.T) (*AuthHandlers, *CommentStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, err := NewCommentStore(dbPath, 30)
	if err != nil {
		t.Fatalf("NewCommentStore: %v", err)
	}
	t.Cleanup(func() { cs.Close() })

	providers := map[string]*AuthProvider{
		"github": {Name: "github"},
	}
	h := NewAuthHandlers(cs, providers, slog.Default(), false)
	return h, cs
}

// --- HandleLogin ---

func TestHandleLogin_UnknownProvider(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/unknown", nil)
	req.SetPathValue("provider", "unknown")
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// Note: HandleLogin with a valid provider would redirect to the OAuth URL.
// We can test that it sets the state cookie and returns 302.
func TestHandleLogin_SetsCookieAndRedirects(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github", ClientID: "test-id", ClientSecret: "test-secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")
	h := NewAuthHandlers(cs, providers, slog.Default(), false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github?return_to=/posts/hello/", nil)
	req.SetPathValue("provider", "github")
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}

	// Check state cookie was set.
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "osg_auth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("expected osg_auth_state cookie")
	}
	if stateCookie.HttpOnly != true {
		t.Error("expected HttpOnly cookie")
	}
	if stateCookie.MaxAge != 600 {
		t.Errorf("MaxAge = %d, want 600", stateCookie.MaxAge)
	}

	// Redirect should go to GitHub.
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}
}

func TestHandleLogin_DefaultReturnTo(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github", ClientID: "id", ClientSecret: "secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")
	h := NewAuthHandlers(cs, providers, slog.Default(), false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github", nil)
	req.SetPathValue("provider", "github")
	rec := httptest.NewRecorder()
	h.HandleLogin(rec, req)

	// Should still redirect (return_to defaults to "/").
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
}

// --- HandleCallback ---

func TestHandleCallback_MissingStateCookie(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state=xyz", nil)
	req.SetPathValue("provider", "github")
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallback_StateMismatch(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state=wrong", nil)
	req.SetPathValue("provider", "github")
	req.AddCookie(&http.Cookie{Name: "osg_auth_state", Value: "correct|/posts/"})
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallback_InvalidStateCookie(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state=xyz", nil)
	req.SetPathValue("provider", "github")
	req.AddCookie(&http.Cookie{Name: "osg_auth_state", Value: "no-pipe-separator"})
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallback_UnknownProvider(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/unknown/callback?code=abc&state=xyz", nil)
	req.SetPathValue("provider", "unknown")
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallback_OAuthError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	cfg := config.CommentsConfig{
		Providers: []config.AuthProviderConfig{
			{Provider: "github", ClientID: "id", ClientSecret: "secret"},
		},
	}
	providers := BuildAuthProviders(cfg, "https://example.com")
	h := NewAuthHandlers(cs, providers, slog.Default(), false)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/auth/github/callback?error=access_denied&error_description=user+denied&state=mystate", nil)
	req.SetPathValue("provider", "github")
	req.AddCookie(&http.Cookie{Name: "osg_auth_state", Value: "mystate|/return/"})
	rec := httptest.NewRecorder()
	h.HandleCallback(rec, req)

	// Should redirect to return_to without error.
	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/return/" {
		t.Errorf("Location = %q, want /return/", loc)
	}
}

// --- HandleMe ---

func TestHandleMe_NotAuthenticated(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.HandleMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleMe_Authenticated(t *testing.T) {
	h, cs := testAuthHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	token, _ := cs.CreateSession(user.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: token})
	rec := httptest.NewRecorder()
	h.HandleMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", resp["name"])
	}
	if resp["provider"] != "github" {
		t.Errorf("provider = %v, want github", resp["provider"])
	}
}

// --- HandleLogout ---

func TestHandleLogout_WithSession(t *testing.T) {
	h, cs := testAuthHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	token, _ := cs.CreateSession(user.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: token})
	rec := httptest.NewRecorder()
	h.HandleLogout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Session should be deleted.
	u, _ := cs.ValidateSession(token)
	if u != nil {
		t.Error("expected session to be deleted after logout")
	}

	// Cookie should be cleared.
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "osg_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.MaxAge != -1 {
		t.Error("expected osg_session cookie to be cleared (MaxAge -1)")
	}
}

func TestHandleLogout_NoSession(t *testing.T) {
	h, _ := testAuthHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.HandleLogout(rec, req)

	// Should still return OK even without a session.
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// --- getUserFromRequest ---

func TestGetUserFromRequest_NoCookie(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	u := getUserFromRequest(req, cs)
	if u != nil {
		t.Error("expected nil without cookie")
	}
}

func TestGetUserFromRequest_EmptyCookie(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: ""})
	u := getUserFromRequest(req, cs)
	if u != nil {
		t.Error("expected nil with empty cookie")
	}
}

func TestGetUserFromRequest_ValidSession(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, _ := NewCommentStore(dbPath, 30)
	defer cs.Close()

	user, _ := cs.UpsertUser("github", "1", "Alice", "", "")
	token, _ := cs.CreateSession(user.ID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: token})
	u := getUserFromRequest(req, cs)
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.Name != "Alice" {
		t.Errorf("name = %q, want Alice", u.Name)
	}
}
