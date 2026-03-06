package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"osg/internal/config"

	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// AuthProvider holds the OAuth2 config and user-info fetcher for a provider.
type AuthProvider struct {
	Name      string
	OAuthCfg  *oauth2.Config
	FetchUser func(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error)
}

// OAuthUserInfo is the normalised user info returned by a provider.
type OAuthUserInfo struct {
	ProviderID string
	Name       string
	Email      string
	AvatarURL  string
}

// providerLabel returns a human-friendly label for a provider name.
func providerLabel(name string) string {
	switch name {
	case "github":
		return "GitHub"
	case "google":
		return "Google"
	default:
		return name
	}
}

// BuildAuthProviders constructs the OAuth2 provider registry from config.
func BuildAuthProviders(commentsCfg config.CommentsConfig, baseCallbackURL string) map[string]*AuthProvider {
	providers := make(map[string]*AuthProvider, len(commentsCfg.Providers))

	for _, p := range commentsCfg.Providers {
		callbackURL := baseCallbackURL + "/api/v1/auth/" + p.Provider + "/callback"

		switch p.Provider {
		case "github":
			providers["github"] = &AuthProvider{
				Name: "github",
				OAuthCfg: &oauth2.Config{
					ClientID:     p.ClientID,
					ClientSecret: p.ClientSecret,
					Endpoint:     oauthgithub.Endpoint,
					RedirectURL:  callbackURL,
					Scopes:       []string{"read:user"},
				},
				FetchUser: fetchGitHubUser,
			}
		case "google":
			providers["google"] = &AuthProvider{
				Name: "google",
				OAuthCfg: &oauth2.Config{
					ClientID:     p.ClientID,
					ClientSecret: p.ClientSecret,
					Endpoint:     google.Endpoint,
					RedirectURL:  callbackURL,
					Scopes:       []string{"openid", "profile", "email"},
				},
				FetchUser: fetchGoogleUser,
			}
		}
	}

	return providers
}

// --- Provider user-info fetchers ---

func fetchGitHubUser(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("github user API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user API %d: %s", resp.StatusCode, body)
	}

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode github user: %w", err)
	}

	name := data.Name
	if name == "" {
		name = data.Login
	}

	return &OAuthUserInfo{
		ProviderID: fmt.Sprintf("%d", data.ID),
		Name:       name,
		Email:      data.Email,
		AvatarURL:  data.AvatarURL,
	}, nil
}

func fetchGoogleUser(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google userinfo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo API %d: %s", resp.StatusCode, body)
	}

	var data struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode google user: %w", err)
	}

	return &OAuthUserInfo{
		ProviderID: data.ID,
		Name:       data.Name,
		Email:      data.Email,
		AvatarURL:  data.Picture,
	}, nil
}

// --- Auth HTTP handlers ---

// AuthHandlers groups the HTTP handlers for authentication.
type AuthHandlers struct {
	store         *CommentStore
	providers     map[string]*AuthProvider
	logger        *slog.Logger
	secureCookies bool
}

// NewAuthHandlers creates the authentication handler group.
func NewAuthHandlers(store *CommentStore, providers map[string]*AuthProvider, logger *slog.Logger, secureCookies bool) *AuthHandlers {
	return &AuthHandlers{
		store:         store,
		providers:     providers,
		logger:        logger,
		secureCookies: secureCookies,
	}
}

// HandleLogin starts the OAuth2 flow for a provider.
// GET /api/v1/auth/{provider}?return_to=/some/page/
func (h *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		http.Error(w, "unknown provider", http.StatusBadRequest)
		return
	}

	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/"
	}

	// Generate random state.
	state, err := generateToken()
	if err != nil {
		h.logger.Error("generate oauth state", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Store state|return_to in a cookie (httpOnly, 10 min TTL).
	http.SetCookie(w, &http.Cookie{
		Name:     "osg_auth_state",
		Value:    state + "|" + returnTo,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	url := provider.OAuthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// HandleCallback processes the OAuth2 callback.
// GET /api/v1/auth/{provider}/callback?code=...&state=...
func (h *AuthHandlers) HandleCallback(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		http.Error(w, "unknown provider", http.StatusBadRequest)
		return
	}

	// Verify state from cookie.
	cookie, err := r.Cookie("osg_auth_state")
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(cookie.Value, "|", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid state cookie", http.StatusBadRequest)
		return
	}
	expectedState, returnTo := parts[0], parts[1]

	if r.URL.Query().Get("state") != expectedState {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	// Clear the state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "osg_auth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	// Check for OAuth error response.
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		h.logger.Warn("oauth error", "provider", providerName, "error", errMsg,
			"description", r.URL.Query().Get("error_description"))
		http.Redirect(w, r, returnTo, http.StatusFound)
		return
	}

	// Exchange code for token.
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	code := r.URL.Query().Get("code")
	token, err := provider.OAuthCfg.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("oauth exchange", "provider", providerName, "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Fetch user info from provider.
	userInfo, err := provider.FetchUser(ctx, token)
	if err != nil {
		h.logger.Error("fetch user info", "provider", providerName, "error", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	// Upsert user in our DB.
	user, err := h.store.UpsertUser(providerName, userInfo.ProviderID, userInfo.Name, userInfo.Email, userInfo.AvatarURL)
	if err != nil {
		h.logger.Error("upsert user", "provider", providerName, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Create session.
	sessionToken, err := h.store.CreateSession(user.ID)
	if err != nil {
		h.logger.Error("create session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Set session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "osg_session",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   h.store.authSessionDays * 86400,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("user authenticated", "provider", providerName, "user", user.Name, "id", user.ID)
	http.Redirect(w, r, returnTo, http.StatusFound)
}

// HandleMe returns the currently authenticated user, or 401.
// GET /api/v1/auth/me
func (h *AuthHandlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r, h.store)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
		"provider":   user.Provider,
	})
}

// HandleLogout destroys the session.
// POST /api/v1/auth/logout
func (h *AuthHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("osg_session")
	if err == nil && cookie.Value != "" {
		h.store.DeleteSession(cookie.Value)
	}

	// Clear the session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "osg_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// getUserFromRequest extracts the authenticated user from the session cookie.
// Returns nil if not authenticated.
func getUserFromRequest(r *http.Request, store *CommentStore) *User {
	cookie, err := r.Cookie("osg_session")
	if err != nil || cookie.Value == "" {
		return nil
	}

	user, err := store.ValidateSession(cookie.Value)
	if err != nil {
		return nil
	}
	return user
}
