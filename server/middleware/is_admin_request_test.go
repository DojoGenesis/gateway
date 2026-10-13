package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func roleClaims(role, subject string) GatewayClaims {
	now := time.Now()
	return GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Role: role,
	}
}

// isAdminBehindAuth sends token through AuthMiddleware and reports what
// IsAdminRequest said inside the handler, which is the only place it is used.
func isAdminBehindAuth(t *testing.T, token string) (admin bool, status int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", AuthMiddleware(), func(c *gin.Context) {
		admin = IsAdminRequest(c)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return admin, w.Code
}

func TestIsAdminRequest(t *testing.T) {
	ResetServiceTokenRevocations()
	t.Cleanup(ResetServiceTokenRevocations)

	svcToken, _, err := IssueServiceToken("is-admin-test", 0)
	require.NoError(t, err)

	// A service-role token that also claims role=admin on a service subject:
	// forged, so the signature is valid but the shape is not one the gateway
	// issues. It must never read as an admin.
	forgedService := serviceClaims("is-admin-forged", time.Hour)
	forgedService.Role = "admin"

	cases := []struct {
		name       string
		token      string
		wantAdmin  bool
		wantStatus int
	}{
		{"admin session", signClaims(t, roleClaims("admin", "alice")), true, http.StatusOK},
		{"user session", signClaims(t, roleClaims("user", "bob")), false, http.StatusOK},
		{"no role claim", signClaims(t, roleClaims("", "carol")), false, http.StatusOK},
		{"service token", svcToken, false, http.StatusOK},
		{"admin role on a service subject", signClaims(t, forgedService), false, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			admin, status := isAdminBehindAuth(t, tc.token)
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantAdmin, admin)
		})
	}
}

// Outside AuthMiddleware there is no verified token on the context, and the
// answer must be "not admin" rather than a panic or a guess.
func TestIsAdminRequest_FailsClosedWithoutAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	assert.False(t, IsAdminRequest(c))

	// An unverified token placed on the context by something other than
	// AuthMiddleware still has to pass the same signature check.
	c.Set("user_id", "mallory")
	c.Set("token", "not-a-jwt")
	assert.False(t, IsAdminRequest(c))
}
