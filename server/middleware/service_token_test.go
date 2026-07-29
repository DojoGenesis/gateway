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

// signClaims signs arbitrary claims with the gateway's own secret. Used to
// forge the malformed / expired / over-long tokens that IssueServiceToken
// refuses to produce, so the VALIDATION side is proven independently of the
// issuance side.
func signClaims(t *testing.T, claims GatewayClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	require.NoError(t, err)
	return s
}

// serviceClaims builds a well-formed set of service claims that callers mutate
// to isolate one defect at a time.
func serviceClaims(name string, ttl time.Duration) GatewayClaims {
	now := time.Now()
	return GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   ServiceSubjectPrefix + name,
			ID:        "11111111-2222-3333-4444-555555555555",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Role: ServiceRole,
	}
}

// cleanRevocations guarantees a test starts and ends with an empty denylist,
// since it is process-global state.
func cleanRevocations(t *testing.T) {
	t.Helper()
	ResetServiceTokenRevocations()
	t.Cleanup(ResetServiceTokenRevocations)
}

// ─── Acceptance + identity attribution ──────────────────────────────────────

func TestServiceToken_AcceptedByAuthMiddleware_AndAttributed(t *testing.T) {
	cleanRevocations(t)

	token, jti, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)
	require.NotEmpty(t, jti)

	router := setupAuthTestRouter()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":      c.GetString("user_id"),
			"auth_type":    c.GetString("auth_type"),
			"service_name": c.GetString("service_name"),
			"token_id":     c.GetString("token_id"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "a valid service token must authenticate; body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"user_id":"service:pdi"`,
		"the subject must name the client so gateway logs attribute the traffic")
	assert.Contains(t, w.Body.String(), `"auth_type":"service"`)
	assert.Contains(t, w.Body.String(), `"service_name":"pdi"`)
	assert.Contains(t, w.Body.String(), jti, "the jti must reach the handler context for log attribution")
}

func TestServiceToken_HumanSessionStillAttributedAsUser(t *testing.T) {
	cleanRevocations(t)

	userToken := signClaims(t, GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "human-user-id",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "user",
	})

	router := setupAuthTestRouter()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":   c.GetString("user_id"),
			"auth_type": c.GetString("auth_type"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"auth_type":"user"`,
		"human sessions must remain distinguishable from machine traffic")
}

// ─── Least privilege ────────────────────────────────────────────────────────

func TestServiceToken_RejectedByAdminMiddleware(t *testing.T) {
	cleanRevocations(t)

	token, _, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	router := setupAuthTestRouter()
	router.GET("/admin/thing", AdminAuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action performed"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/thing", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code,
		"a service token must NOT satisfy AdminAuthMiddleware; got %d body=%s", w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Admin privileges required")
	assert.NotContains(t, w.Body.String(), "admin action performed",
		"the admin handler must never run for a machine credential")
}

// A token that claims role=admin while carrying a service subject must still be
// refused. This is the belt to the role check's braces: it holds even if the
// role test is ever widened.
func TestServiceToken_AdminRoleOnServiceSubject_StillRejectedByAdmin(t *testing.T) {
	cleanRevocations(t)

	claims := serviceClaims("pdi", time.Hour)
	claims.Role = "admin"
	token := signClaims(t, claims)

	router := setupAuthTestRouter()
	router.GET("/admin/thing", AdminAuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action performed"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/thing", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"the service: subject namespace must never be admin, whatever the role claim says")
	assert.NotContains(t, w.Body.String(), "admin action performed")
}

// A genuine admin token must still work — proof the least-privilege additions
// did not break admin auth outright.
func TestAdminTokenStillSatisfiesAdminMiddleware(t *testing.T) {
	cleanRevocations(t)

	adminToken := signClaims(t, GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "a-real-human-admin",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "admin",
	})

	router := setupAuthTestRouter()
	router.GET("/admin/thing", AdminAuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action performed"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/thing", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "admin action performed")
}

// ─── Rejection paths ────────────────────────────────────────────────────────

func TestServiceToken_RejectedCases(t *testing.T) {
	cleanRevocations(t)

	expired := serviceClaims("pdi", time.Hour)
	expired.IssuedAt = jwt.NewNumericDate(time.Now().Add(-48 * time.Hour))
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-24 * time.Hour))

	noJTI := serviceClaims("pdi", time.Hour)
	noJTI.ID = ""

	noExpiry := serviceClaims("pdi", time.Hour)
	noExpiry.ExpiresAt = nil

	noIssuedAt := serviceClaims("pdi", time.Hour)
	noIssuedAt.IssuedAt = nil

	tooLong := serviceClaims("pdi", MaxServiceTokenTTL+24*time.Hour)

	badSubject := serviceClaims("pdi", time.Hour)
	badSubject.Subject = "pdi" // not namespaced

	emptyName := serviceClaims("pdi", time.Hour)
	emptyName.Subject = ServiceSubjectPrefix // "service:" with no client name

	cases := []struct {
		name  string
		token string
		why   string
	}{
		{"expired", signClaims(t, expired), "expiry is enforced by the standard JWT claim validation"},
		{"missing jti", signClaims(t, noJTI), "without a jti there is no revocation handle"},
		{"missing expiry", signClaims(t, noExpiry), "infinite lifetime is not permitted"},
		{"missing issued-at", signClaims(t, noIssuedAt), "lifetime cannot be bounded without iat"},
		{"lifetime over maximum", signClaims(t, tooLong), "the ceiling is enforced at validation, not only at issuance"},
		{"subject not namespaced", signClaims(t, badSubject), "a service token must name its client"},
		{"empty client name", signClaims(t, emptyName), "\"service:\" alone identifies nobody"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subject, claims, err := validateTokenWithClaims(tc.token)
			assert.Error(t, err, "must be rejected: %s", tc.why)
			assert.Empty(t, subject)
			assert.Nil(t, claims)
		})
	}
}

func TestServiceToken_Revoked_ByID_Rejected(t *testing.T) {
	cleanRevocations(t)

	token, jti, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	// Valid before revocation.
	subject, _, err := validateTokenWithClaims(token)
	require.NoError(t, err)
	require.Equal(t, "service:pdi", subject)

	RevokeServiceTokenID(jti)

	subject, claims, err := validateTokenWithClaims(token)
	assert.ErrorIs(t, err, ErrServiceTokenRevoked)
	assert.Empty(t, subject)
	assert.Nil(t, claims)
}

func TestServiceToken_Revoked_BySubject_Rejected(t *testing.T) {
	cleanRevocations(t)

	tokenA, _, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)
	tokenB, _, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)
	otherService, _, err := IssueServiceToken("someone-else", 0)
	require.NoError(t, err)

	RevokeServiceSubject("service:pdi")

	_, _, err = validateTokenWithClaims(tokenA)
	assert.ErrorIs(t, err, ErrServiceTokenRevoked, "subject revocation must kill every token for that service")
	_, _, err = validateTokenWithClaims(tokenB)
	assert.ErrorIs(t, err, ErrServiceTokenRevoked)

	// Blast radius: no other service, and no human session, is affected.
	subject, _, err := validateTokenWithClaims(otherService)
	assert.NoError(t, err, "revoking one service must not touch another")
	assert.Equal(t, "service:someone-else", subject)

	human := signClaims(t, GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "a-human",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "user",
	})
	humanSubject, _, err := validateTokenWithClaims(human)
	assert.NoError(t, err, "service revocation must never invalidate a human session")
	assert.Equal(t, "a-human", humanSubject)
}

func TestServiceToken_RevocationSurvivesEnvLoad(t *testing.T) {
	cleanRevocations(t)

	token, jti, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	t.Setenv(EnvRevokedTokenIDs, "some-other-id, "+jti+" ,another-id")
	LoadServiceTokenRevocationsFromEnv()

	_, _, err = validateTokenWithClaims(token)
	assert.ErrorIs(t, err, ErrServiceTokenRevoked,
		"the denylist must be seedable from the environment so revocations survive a restart")
}

// ─── Signature integrity: the token must not bypass validateToken ───────────

func TestServiceToken_TamperedOrForged_Rejected(t *testing.T) {
	cleanRevocations(t)

	valid, _, err := IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	// 1. Payload mutated in place — signature no longer matches.
	tampered := []byte(valid)
	// Flip a character inside the payload segment (after the first '.').
	dot := 0
	for i, ch := range tampered {
		if ch == '.' {
			dot = i
			break
		}
	}
	require.NotZero(t, dot)
	if tampered[dot+3] == 'A' {
		tampered[dot+3] = 'B'
	} else {
		tampered[dot+3] = 'A'
	}

	// 2. Re-signed by an attacker who does NOT hold JWT_SECRET.
	foreign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, serviceClaims("pdi", time.Hour)).
		SignedString([]byte("an-attacker-controlled-secret"))
	require.NoError(t, err)

	// 3. alg=none — the classic signature-stripping forgery.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, serviceClaims("pdi", time.Hour)).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// 4. Not a JWT at all — the shape a naive static-string branch would admit.
	cases := map[string]string{
		"tampered payload":      string(tampered),
		"signed by wrong key":   foreign,
		"alg=none":              unsigned,
		"raw service subject":   "service:pdi",
		"bare service name":     "pdi",
		"service-looking guess": "service-token",
	}

	router := setupAuthTestRouter()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should never be reached"})
	})

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := validateTokenWithClaims(token)
			assert.Error(t, err, "%s must fail signature verification", name)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code, "%s must be 401 at the middleware", name)
			assert.NotContains(t, w.Body.String(), "should never be reached")
		})
	}
}

// ─── Issuance constraints ───────────────────────────────────────────────────

func TestIssueServiceToken_Constraints(t *testing.T) {
	cleanRevocations(t)

	t.Run("rejects an invalid client name", func(t *testing.T) {
		for _, bad := range []string{"", "PDI", "pdi:admin", "pdi service", "-pdi", "../etc", "pdi\n"} {
			_, _, err := IssueServiceToken(bad, 0)
			assert.ErrorIs(t, err, ErrServiceTokenInvalidName, "name %q must be rejected", bad)
		}
	})

	t.Run("rejects a TTL over the maximum", func(t *testing.T) {
		_, _, err := IssueServiceToken("pdi", MaxServiceTokenTTL+time.Hour)
		assert.ErrorIs(t, err, ErrServiceTokenTTLTooLong)
	})

	t.Run("defaults a non-positive TTL rather than minting an eternal token", func(t *testing.T) {
		token, _, err := IssueServiceToken("pdi", 0)
		require.NoError(t, err)

		claims := &GatewayClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) { return jwtSecret, nil })
		require.NoError(t, err)
		require.NotNil(t, claims.ExpiresAt)

		lifetime := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
		assert.InDelta(t, DefaultServiceTokenTTL.Seconds(), lifetime.Seconds(), 5)
		assert.LessOrEqual(t, lifetime, MaxServiceTokenTTL, "no token may exceed the hard ceiling")
	})

	t.Run("issues a distinct jti per token", func(t *testing.T) {
		_, first, err := IssueServiceToken("pdi", time.Hour)
		require.NoError(t, err)
		_, second, err := IssueServiceToken("pdi", time.Hour)
		require.NoError(t, err)
		assert.NotEqual(t, first, second, "each token needs its own revocation handle")
	})
}

// ─── The development backdoor must be untouched ─────────────────────────────

func TestDevelopmentBackdoorUnchanged(t *testing.T) {
	cleanRevocations(t)

	t.Run("still accepted outside production", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "development")
		for token, want := range map[string]string{
			"test-token":  "test-user",
			"user-12345":  "user-12345",
			"admin-alice": "admin-alice",
		} {
			got, claims, err := validateTokenWithClaims(token)
			assert.NoError(t, err, "legacy dev token %q must still work", token)
			assert.Equal(t, want, got)
			assert.Nil(t, claims, "legacy shim tokens are not JWTs and carry no claims")
		}
	})

	t.Run("still off in production", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		for _, token := range []string{"test-token", "user-12345", "admin-alice"} {
			_, _, err := validateTokenWithClaims(token)
			assert.Error(t, err, "legacy dev token %q must not be accepted in production", token)
		}
	})

	t.Run("service tokens work in production", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		token, _, err := IssueServiceToken("pdi", 0)
		require.NoError(t, err)

		subject, claims, err := validateTokenWithClaims(token)
		require.NoError(t, err, "service tokens must not depend on the development shim")
		assert.Equal(t, "service:pdi", subject)
		require.NotNil(t, claims)
		assert.Equal(t, ServiceRole, claims.Role)
	})
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func TestSubjectHelpers(t *testing.T) {
	assert.True(t, IsServiceSubject("service:pdi"))
	assert.False(t, IsServiceSubject("service:"), "an empty client name identifies nobody")
	assert.False(t, IsServiceSubject("services:pdi"))
	assert.False(t, IsServiceSubject("a-human"))

	assert.Equal(t, "pdi", ServiceNameFromSubject("service:pdi"))
	assert.Equal(t, "", ServiceNameFromSubject("a-human"))
}

func TestContainsConstantTime(t *testing.T) {
	list := []string{"alpha", "beta", "gamma"}
	assert.True(t, containsConstantTime(list, "beta"))
	assert.False(t, containsConstantTime(list, "delta"))
	assert.False(t, containsConstantTime(list, "bet"), "prefixes must not match")
	assert.False(t, containsConstantTime(nil, "alpha"))
	assert.False(t, containsConstantTime(list, ""))
}

func TestSplitAndTrim(t *testing.T) {
	assert.Nil(t, splitAndTrim(""))
	assert.Nil(t, splitAndTrim("   "))
	assert.Nil(t, splitAndTrim(",,,"))
	assert.Equal(t, []string{"a", "b", "c"}, splitAndTrim(" a , b ,c "))
}
