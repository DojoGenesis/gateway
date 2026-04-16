package server

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─── OAuth Config ────────────────────────────────────────────────────────────

// OAuthConfig holds configuration for OAuth2 providers.
type OAuthConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURI  string
	Enabled            bool
	AllowedUsers       []string // if non-empty, only these GitHub logins are permitted
}

// loadOAuthConfig reads GitHub OAuth configuration from environment variables.
//
//	GITHUB_OAUTH_CLIENT_ID          — OAuth app client ID
//	GITHUB_OAUTH_CLIENT_SECRET      — OAuth app client secret
//	GITHUB_OAUTH_REDIRECT_URI       — Callback URI (default: https://pdi.trespies.dev/auth/github/callback)
//	GITHUB_OAUTH_ENABLED            — Set to "true" to enable the flow
//	GITHUB_OAUTH_ALLOWED_USERS      — Comma-separated list of GitHub logins permitted to log in (empty = all)
func loadOAuthConfig() OAuthConfig {
	redirectURI := os.Getenv("GITHUB_OAUTH_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "https://pdi.trespies.dev/auth/github/callback"
	}

	allowedRaw := os.Getenv("GITHUB_OAUTH_ALLOWED_USERS") // comma-separated
	var allowedUsers []string
	for _, u := range strings.Split(allowedRaw, ",") {
		if u = strings.TrimSpace(u); u != "" {
			allowedUsers = append(allowedUsers, u)
		}
	}

	return OAuthConfig{
		GitHubClientID:     os.Getenv("GITHUB_OAUTH_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_OAUTH_CLIENT_SECRET"),
		GitHubRedirectURI:  redirectURI,
		Enabled:            os.Getenv("GITHUB_OAUTH_ENABLED") == "true",
		AllowedUsers:       allowedUsers,
	}
}

// ─── GitHub API response types ───────────────────────────────────────────────

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// handleOAuthGitHubStart handles GET /auth/github.
// Generates a random CSRF state token, stores it in a secure cookie,
// and redirects the browser to GitHub's authorization endpoint.
func (s *Server) handleOAuthGitHubStart(c *gin.Context) {
	cfg := loadOAuthConfig()
	if !cfg.Enabled || cfg.GitHubClientID == "" {
		s.errorResponse(c, http.StatusNotImplemented, "oauth_disabled", "GitHub OAuth is not enabled on this server")
		return
	}

	state, err := generateOAuthState()
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to generate OAuth state")
		return
	}

	// Store state in a short-lived httpOnly cookie (10 minutes).
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	params := url.Values{}
	params.Set("client_id", cfg.GitHubClientID)
	params.Set("redirect_uri", cfg.GitHubRedirectURI)
	params.Set("scope", "read:user user:email")
	params.Set("state", state)

	authURL := "https://github.com/login/oauth/authorize?" + params.Encode()
	c.Redirect(http.StatusFound, authURL)
}

// handleOAuthGitHubCallback handles GET /auth/github/callback.
// Validates the state parameter, exchanges the code for a GitHub access token,
// fetches the user's profile, finds or creates a portal account, then issues
// JWT tokens using the same shape as handleAuthLogin.
func (s *Server) handleOAuthGitHubCallback(c *gin.Context) {
	cfg := loadOAuthConfig()
	if !cfg.Enabled || cfg.GitHubClientID == "" {
		s.errorResponse(c, http.StatusNotImplemented, "oauth_disabled", "GitHub OAuth is not enabled on this server")
		return
	}

	// Validate state to prevent CSRF.
	state := c.Query("state")
	cookieState, err := c.Cookie("oauth_state")
	if err != nil || state == "" || state != cookieState {
		s.errorResponse(c, http.StatusBadRequest, "invalid_state", "OAuth state mismatch — possible CSRF attempt")
		return
	}

	// Clear the state cookie immediately.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})

	code := c.Query("code")
	if code == "" {
		errParam := c.Query("error")
		if errParam == "" {
			errParam = "no_code"
		}
		s.errorResponse(c, http.StatusBadRequest, "oauth_error", "GitHub OAuth returned no authorization code: "+errParam)
		return
	}

	// Exchange authorization code for access token.
	accessToken, err := exchangeGitHubCode(cfg.GitHubClientID, cfg.GitHubClientSecret, code)
	if err != nil {
		s.errorResponse(c, http.StatusBadGateway, "github_token_error", "Failed to exchange GitHub authorization code")
		return
	}

	// Fetch GitHub user profile.
	ghUser, err := fetchGitHubUser(accessToken)
	if err != nil {
		s.errorResponse(c, http.StatusBadGateway, "github_api_error", "Failed to fetch GitHub user profile")
		return
	}

	// Prefer profile email; fall back to /user/emails endpoint.
	email := strings.TrimSpace(ghUser.Email)
	if email == "" {
		email, err = fetchGitHubEmail(accessToken)
		if err != nil || email == "" {
			s.errorResponse(c, http.StatusBadGateway, "github_email_error", "Could not retrieve a verified email from GitHub")
			return
		}
	}

	// Derive display name: prefer the profile Name, fall back to Login.
	displayName := strings.TrimSpace(ghUser.Name)
	if displayName == "" {
		displayName = ghUser.Login
	}

	// Enforce allowlist: if AllowedUsers is non-empty, reject logins not in the list.
	if len(cfg.AllowedUsers) > 0 {
		allowed := false
		for _, u := range cfg.AllowedUsers {
			if strings.EqualFold(u, ghUser.Login) {
				allowed = true
				break
			}
		}
		if !allowed {
			slog.Warn("OAuth login rejected: user not in allowlist",
				"github_login", ghUser.Login)
			c.Redirect(http.StatusFound, "/chat?error=access_denied")
			return
		}
	}

	providerID := fmt.Sprintf("%d", ghUser.ID)

	// Find or create a portal user keyed on email.
	userID, err := findOrCreateOAuthUser(s.authDB, email, displayName, "github", providerID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to find or create user account")
		return
	}

	// Issue JWT access and refresh tokens (same as handleAuthLogin).
	accessJWT, err := issueToken(userID, "user", s.cfg.AccessTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue access token")
		return
	}
	refreshJWT, err := issueToken(userID, "refresh", s.cfg.RefreshTokenTTL)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to issue refresh token")
		return
	}

	resp := authTokenResponse{
		UserID:       userID,
		DisplayName:  displayName,
		AccessToken:  accessJWT,
		RefreshToken: refreshJWT,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}

	// Check Accept header: JSON clients receive the token payload directly;
	// browser clients are redirected to /chat with tokens in httpOnly cookies.
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, resp)
		return
	}

	// Redirect to the chat SPA with tokens in the URL fragment so the SPA can
	// persist them to localStorage (httpOnly cookies are not readable by JS).
	q := url.Values{}
	q.Set("access_token", accessJWT)
	q.Set("refresh_token", refreshJWT)
	q.Set("user_id", userID)
	q.Set("display_name", displayName)
	c.Redirect(http.StatusFound, "/chat?"+q.Encode())
}

// ─── Helper functions ────────────────────────────────────────────────────────

// generateOAuthState returns a cryptographically random hex-encoded state token
// (32 bytes → 64 hex characters) suitable for CSRF protection.
func generateOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateOAuthState: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// exchangeGitHubCode POSTs to GitHub's token endpoint and returns the OAuth
// access token on success.
func exchangeGitHubCode(clientID, clientSecret, code string) (string, error) {
	body := url.Values{}
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("code", code)

	req, err := http.NewRequest(http.MethodPost,
		"https://github.com/login/oauth/access_token",
		bytes.NewBufferString(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("exchangeGitHubCode: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchangeGitHubCode: POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exchangeGitHubCode: GitHub returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("exchangeGitHubCode: decode response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("exchangeGitHubCode: GitHub error: %s", result.Error)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("exchangeGitHubCode: empty access token in response")
	}

	return result.AccessToken, nil
}

// fetchGitHubUser calls GET https://api.github.com/user with the given access token
// and returns the parsed user profile.
func fetchGitHubUser(accessToken string) (*githubUser, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("fetchGitHubUser: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchGitHubUser: GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetchGitHubUser: GitHub returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("fetchGitHubUser: decode response: %w", err)
	}
	return &user, nil
}

// fetchGitHubEmail calls GET https://api.github.com/user/emails and returns the
// primary verified email address. Returns an error if none is found.
func fetchGitHubEmail(accessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("fetchGitHubEmail: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetchGitHubEmail: GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetchGitHubEmail: GitHub returned HTTP %d", resp.StatusCode)
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("fetchGitHubEmail: decode response: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	// Fall back to any verified email if the primary is not verified.
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("fetchGitHubEmail: no verified email found")
}

// findOrCreateOAuthUser performs a SELECT-then-INSERT against the authDB to
// return a stable portal user ID keyed on email.
//
// If a user with this email already exists (created via any auth method), the
// existing ID is returned unchanged.  Otherwise a new local_users row is
// inserted with user_type = 'authenticated'.
//
// The provider and providerID parameters are stored as JSON in the metadata
// column so future sign-ins can surface the linked provider.
func findOrCreateOAuthUser(db *sql.DB, email, displayName, provider, providerID string) (string, error) {
	// Try to find an existing user by email (any user_type).
	var existingID string
	err := db.QueryRow(
		`SELECT id FROM local_users WHERE email = ? LIMIT 1`,
		email,
	).Scan(&existingID)

	if err == nil {
		// User already exists; return their ID.
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("findOrCreateOAuthUser: lookup: %w", err)
	}

	// No existing user — create one.
	newID := uuid.New().String()
	now := time.Now()

	// Store provider info as JSON metadata.
	meta := fmt.Sprintf(`{"oauth_provider":%q,"oauth_provider_id":%q}`, provider, providerID)

	_, err = db.Exec(
		`INSERT INTO local_users (id, user_type, created_at, last_accessed_at, email, display_name, metadata)
		 VALUES (?, 'authenticated', ?, ?, ?, ?, ?)`,
		newID, now, now, email, displayName, meta,
	)
	if err != nil {
		return "", fmt.Errorf("findOrCreateOAuthUser: insert: %w", err)
	}

	return newID, nil
}
