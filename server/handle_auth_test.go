package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/server/middleware"

	_ "modernc.org/sqlite"
)

// newAuthTestServer creates a minimal Server with an in-memory SQLite DB
// that has the required schema for portal auth tests.
func newAuthTestServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create schema_migrations table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL,
		description TEXT
	)`)
	require.NoError(t, err)

	// Create local_users table (v0.0.30 base)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS local_users (
		id TEXT PRIMARY KEY,
		user_type TEXT DEFAULT 'guest',
		created_at DATETIME NOT NULL,
		last_accessed_at DATETIME NOT NULL,
		cloud_user_id TEXT,
		migration_status TEXT DEFAULT 'none',
		metadata TEXT
	)`)
	require.NoError(t, err)

	// Apply portal auth migration columns
	_, err = db.Exec(`ALTER TABLE local_users ADD COLUMN email TEXT`)
	require.NoError(t, err)
	_, err = db.Exec(`ALTER TABLE local_users ADD COLUMN password_hash TEXT`)
	require.NoError(t, err)
	_, err = db.Exec(`ALTER TABLE local_users ADD COLUMN display_name TEXT`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_local_users_email ON local_users(email) WHERE email IS NOT NULL`)
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	s := &Server{
		router: router,
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		authDB:         db,
		orchestrations: NewOrchestrationStore(),
	}

	router.POST("/auth/register", s.handleAuthRegister)
	router.POST("/auth/login", s.handleAuthLogin)
	router.POST("/auth/refresh", s.handleAuthRefresh)

	t.Cleanup(func() {
		db.Close()
	})

	return s, router
}

func postJSON(router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// registerUser is a test helper that registers a user and returns the parsed token response.
func registerUser(t *testing.T, router *gin.Engine, email, password, displayName string) authTokenResponse {
	t.Helper()
	w := postJSON(router, "/auth/register", map[string]string{
		"email":        email,
		"password":     password,
		"display_name": displayName,
	})
	require.Equal(t, http.StatusCreated, w.Code, "register should return 201: %s", w.Body.String())

	var resp authTokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	return resp
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	_, router := newAuthTestServer(t)

	w := postJSON(router, "/auth/register", map[string]string{
		"email":        "researcher@lab.edu",
		"password":     "securepass123",
		"display_name": "Jane Doe",
	})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp authTokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.UserID)
	assert.Equal(t, "Jane Doe", resp.DisplayName)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, 86400, resp.ExpiresIn)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	_, router := newAuthTestServer(t)

	registerUser(t, router, "dup@lab.edu", "password123", "First User")

	// Try to register again with the same email
	w := postJSON(router, "/auth/register", map[string]string{
		"email":        "dup@lab.edu",
		"password":     "otherpass123",
		"display_name": "Second User",
	})

	assert.Equal(t, http.StatusConflict, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "email_taken", errObj["code"])
}

func TestRegister_InvalidPassword(t *testing.T) {
	_, router := newAuthTestServer(t)

	w := postJSON(router, "/auth/register", map[string]string{
		"email":        "short@lab.edu",
		"password":     "short",
		"display_name": "Short Pass",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "invalid_request", errObj["code"])
}

func TestRegister_MissingFields(t *testing.T) {
	_, router := newAuthTestServer(t)

	// Missing email
	w := postJSON(router, "/auth/register", map[string]string{
		"password":     "password123",
		"display_name": "No Email",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Missing display_name
	w = postJSON(router, "/auth/register", map[string]string{
		"email":    "noemail@lab.edu",
		"password": "password123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	_, router := newAuthTestServer(t)

	registerUser(t, router, "login@lab.edu", "password123", "Login User")

	w := postJSON(router, "/auth/login", map[string]string{
		"email":    "login@lab.edu",
		"password": "password123",
	})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authTokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.UserID)
	assert.Equal(t, "Login User", resp.DisplayName)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, 86400, resp.ExpiresIn)
}

func TestLogin_WrongPassword(t *testing.T) {
	_, router := newAuthTestServer(t)

	registerUser(t, router, "wrong@lab.edu", "password123", "Wrong Pass")

	w := postJSON(router, "/auth/login", map[string]string{
		"email":    "wrong@lab.edu",
		"password": "wrongpassword",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "invalid_credentials", errObj["code"])
}

func TestLogin_UnknownEmail(t *testing.T) {
	_, router := newAuthTestServer(t)

	w := postJSON(router, "/auth/login", map[string]string{
		"email":    "nobody@lab.edu",
		"password": "password123",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "invalid_credentials", errObj["code"])
}

func TestRefresh_Success(t *testing.T) {
	_, router := newAuthTestServer(t)

	regResp := registerUser(t, router, "refresh@lab.edu", "password123", "Refresh User")

	w := postJSON(router, "/auth/refresh", map[string]string{
		"refresh_token": regResp.RefreshToken,
	})

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.NotEmpty(t, body["access_token"])
	assert.Equal(t, float64(86400), body["expires_in"])
}

func TestRefresh_ExpiredToken(t *testing.T) {
	_, router := newAuthTestServer(t)

	regResp := registerUser(t, router, "expired@lab.edu", "password123", "Expired User")

	// Create an expired refresh token manually
	claims := middleware.GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   regResp.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-8 * 24 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired
		},
		Role: "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString(middleware.GetJWTSecret())
	require.NoError(t, err)

	w := postJSON(router, "/auth/refresh", map[string]string{
		"refresh_token": expiredToken,
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "invalid_token", errObj["code"])
}

func TestRefresh_AccessTokenNotRefresh(t *testing.T) {
	_, router := newAuthTestServer(t)

	regResp := registerUser(t, router, "access@lab.edu", "password123", "Access User")

	// Use the access token (role=user) as a refresh token — should be rejected
	w := postJSON(router, "/auth/refresh", map[string]string{
		"refresh_token": regResp.AccessToken,
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "invalid_token", errObj["code"])
}

// ─── Additional verification tests ─────────────────────────────────────────

// TestAccessToken_PassesAuthMiddleware verifies that issued access tokens are accepted
// by the existing AuthMiddleware on protected routes (success criterion #10).
func TestAccessToken_PassesAuthMiddleware(t *testing.T) {
	_, router := newAuthTestServer(t)

	// Add a protected test endpoint behind AuthMiddleware
	router.GET("/protected", middleware.AuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	regResp := registerUser(t, router, "protected@lab.edu", "password123", "Protected User")

	// Use the access token to hit the protected endpoint
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+regResp.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "access token should pass AuthMiddleware")

	var respBody map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, regResp.UserID, respBody["user_id"], "AuthMiddleware should extract the correct user ID from the token")
}

// TestTokenRoleClaims verifies that access tokens have role=user and refresh tokens
// have role=refresh in their JWT claims (success criterion #11).
func TestTokenRoleClaims(t *testing.T) {
	_, router := newAuthTestServer(t)

	regResp := registerUser(t, router, "roles@lab.edu", "password123", "Role User")

	// Parse access token claims
	accessClaims := &middleware.GatewayClaims{}
	_, err := jwt.ParseWithClaims(regResp.AccessToken, accessClaims, func(token *jwt.Token) (interface{}, error) {
		return middleware.GetJWTSecret(), nil
	})
	require.NoError(t, err)
	assert.Equal(t, "user", accessClaims.Role, "access token must have role=user")
	assert.Equal(t, regResp.UserID, accessClaims.Subject, "access token subject must be user ID")

	// Parse refresh token claims
	refreshClaims := &middleware.GatewayClaims{}
	_, err = jwt.ParseWithClaims(regResp.RefreshToken, refreshClaims, func(token *jwt.Token) (interface{}, error) {
		return middleware.GetJWTSecret(), nil
	})
	require.NoError(t, err)
	assert.Equal(t, "refresh", refreshClaims.Role, "refresh token must have role=refresh")
	assert.Equal(t, regResp.UserID, refreshClaims.Subject, "refresh token subject must be user ID")
}
