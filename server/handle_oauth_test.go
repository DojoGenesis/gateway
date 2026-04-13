package server

import (
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

// ─── TestGenerateOAuthState ───────────────────────────────────────────────────

func TestGenerateOAuthState(t *testing.T) {
	t.Run("correct length", func(t *testing.T) {
		state, err := generateOAuthState()
		if err != nil {
			t.Fatalf("generateOAuthState() error: %v", err)
		}
		// 32 bytes → 64 hex characters
		if len(state) != 64 {
			t.Errorf("expected 64 hex chars, got %d: %q", len(state), state)
		}
		// Must be valid hex
		if _, err := hex.DecodeString(state); err != nil {
			t.Errorf("state is not valid hex: %v", err)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		const n = 20
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			s, err := generateOAuthState()
			if err != nil {
				t.Fatalf("generateOAuthState() iteration %d error: %v", i, err)
			}
			if _, dup := seen[s]; dup {
				t.Errorf("duplicate state token generated at iteration %d", i)
			}
			seen[s] = struct{}{}
		}
	})
}

// ─── TestHandleOAuthGitHubStart ───────────────────────────────────────────────

func TestHandleOAuthGitHubStart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_OAUTH_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GITHUB_OAUTH_REDIRECT_URI", "https://example.com/auth/github/callback")
	t.Setenv("GITHUB_OAUTH_ENABLED", "true")

	s := &Server{
		cfg: &ServerConfig{
			AccessTokenTTL:  24 * time.Hour,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}

	router := gin.New()
	router.GET("/auth/github", s.handleOAuthGitHubStart)

	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	result := w.Result()

	// Must redirect
	if result.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", result.StatusCode)
	}

	location := result.Header.Get("Location")
	if location == "" {
		t.Fatal("Location header is empty — expected redirect to GitHub")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Location header is not a valid URL: %v", err)
	}
	if !strings.Contains(parsed.Host, "github.com") {
		t.Errorf("expected redirect to github.com, got host %q", parsed.Host)
	}

	q := parsed.Query()

	if got := q.Get("client_id"); got != "test-client-id" {
		t.Errorf("client_id = %q, want %q", got, "test-client-id")
	}
	if got := q.Get("redirect_uri"); got != "https://example.com/auth/github/callback" {
		t.Errorf("redirect_uri = %q, want %q", got, "https://example.com/auth/github/callback")
	}
	if got := q.Get("scope"); !strings.Contains(got, "read:user") {
		t.Errorf("scope = %q, want it to contain 'read:user'", got)
	}

	state := q.Get("state")
	if state == "" {
		t.Error("state query parameter is empty")
	}
	if len(state) != 64 {
		t.Errorf("state length = %d, want 64", len(state))
	}

	// State must also be set in the oauth_state cookie.
	cookieFound := false
	for _, cookie := range result.Cookies() {
		if cookie.Name == "oauth_state" {
			cookieFound = true
			if cookie.Value != state {
				t.Errorf("cookie oauth_state = %q, want %q", cookie.Value, state)
			}
			if !cookie.HttpOnly {
				t.Error("oauth_state cookie must be HttpOnly")
			}
			if cookie.MaxAge <= 0 {
				t.Error("oauth_state cookie must have a positive MaxAge")
			}
		}
	}
	if !cookieFound {
		t.Error("oauth_state cookie not set in response")
	}
}

func TestHandleOAuthGitHubStart_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Setenv("GITHUB_OAUTH_ENABLED", "false")
	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "")

	s := &Server{
		cfg: &ServerConfig{
			AccessTokenTTL:  24 * time.Hour,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}

	router := gin.New()
	router.GET("/auth/github", s.handleOAuthGitHubStart)

	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 when OAuth disabled, got %d", w.Code)
	}
}

// ─── TestFindOrCreateOAuthUser ────────────────────────────────────────────────

// openTestDB opens an in-memory SQLite database with the minimal schema
// required by findOrCreateOAuthUser (local_users table).
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE local_users (
			id              TEXT PRIMARY KEY,
			user_type       TEXT DEFAULT 'guest',
			created_at      DATETIME NOT NULL,
			last_accessed_at DATETIME NOT NULL,
			cloud_user_id   TEXT,
			migration_status TEXT DEFAULT 'none',
			email           TEXT,
			password_hash   TEXT,
			display_name    TEXT,
			metadata        TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestFindOrCreateOAuthUser(t *testing.T) {
	t.Run("creates new user", func(t *testing.T) {
		db := openTestDB(t)

		id, err := findOrCreateOAuthUser(db, "alice@example.com", "Alice", "github", "123456")
		if err != nil {
			t.Fatalf("findOrCreateOAuthUser: %v", err)
		}
		if id == "" {
			t.Error("returned empty user ID")
		}

		// Verify the row exists.
		var email, displayName, userType string
		err = db.QueryRow(
			`SELECT email, display_name, user_type FROM local_users WHERE id = ?`, id,
		).Scan(&email, &displayName, &userType)
		if err != nil {
			t.Fatalf("row not found after insert: %v", err)
		}
		if email != "alice@example.com" {
			t.Errorf("email = %q, want %q", email, "alice@example.com")
		}
		if displayName != "Alice" {
			t.Errorf("display_name = %q, want %q", displayName, "Alice")
		}
		if userType != "authenticated" {
			t.Errorf("user_type = %q, want %q", userType, "authenticated")
		}
	})

	t.Run("idempotent lookup", func(t *testing.T) {
		db := openTestDB(t)

		id1, err := findOrCreateOAuthUser(db, "bob@example.com", "Bob", "github", "999")
		if err != nil {
			t.Fatalf("first call: %v", err)
		}

		id2, err := findOrCreateOAuthUser(db, "bob@example.com", "Bob Changed", "github", "999")
		if err != nil {
			t.Fatalf("second call: %v", err)
		}

		if id1 != id2 {
			t.Errorf("idempotency failed: first id=%q, second id=%q", id1, id2)
		}

		// Confirm only one row exists for this email.
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM local_users WHERE email = ?`, "bob@example.com",
		).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 row for email, got %d", count)
		}
	})

	t.Run("different emails create separate users", func(t *testing.T) {
		db := openTestDB(t)

		id1, err := findOrCreateOAuthUser(db, "user1@example.com", "User1", "github", "1")
		if err != nil {
			t.Fatalf("user1: %v", err)
		}
		id2, err := findOrCreateOAuthUser(db, "user2@example.com", "User2", "github", "2")
		if err != nil {
			t.Fatalf("user2: %v", err)
		}

		if id1 == id2 {
			t.Error("different emails should produce different user IDs")
		}
	})
}
