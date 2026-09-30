package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// useResolvedEnvironment sets the process's resolved environment for one test
// and clears it afterwards, so other tests keep the env-var fallback.
func useResolvedEnvironment(t *testing.T, env string) {
	t.Helper()
	SetResolvedEnvironment(env)
	t.Cleanup(func() { SetResolvedEnvironment("") })
}

// A host whose production setting lives only in config.yaml
// (`environment: production`) with ENVIRONMENT unset. The JWT startup gate
// already read the resolved config and called this production; the dev-token
// refusal read only the env var and did not, so an explicit
// GATEWAY_DEV_TOKENS=true on such a host handed out unsigned admin tokens.
func TestProductionCheck_YAMLOnlyProductionRefusesDevTokens(t *testing.T) {
	t.Setenv(envEnvironment, "")
	t.Setenv(EnvDevTokens, "true")
	useResolvedEnvironment(t, "production")

	assert.True(t, IsProductionEnvironment(), "resolved config says production")
	_, _, err := validateTokenWithClaims("admin-alice")
	assert.Error(t, err, "a YAML-configured production host must refuse unsigned tokens even when opted in")
}

// The env var still outranks YAML (the documented config order), so a
// resolved "development" is development, whatever the raw env var says.
func TestProductionCheck_ResolvedValueIsTheOneAnswer(t *testing.T) {
	t.Setenv(envEnvironment, "production")
	useResolvedEnvironment(t, "development")
	assert.False(t, IsProductionEnvironment(), "the resolved value, not a second read of the env var, decides")
}

// With nothing resolved (unit tests, other binaries), the env var is used, as
// before.
func TestProductionCheck_FallsBackToTheEnvVarWhenNothingResolved(t *testing.T) {
	useResolvedEnvironment(t, "")
	t.Setenv(envEnvironment, " Production ")
	assert.True(t, IsProductionEnvironment())
	t.Setenv(envEnvironment, "")
	assert.False(t, IsProductionEnvironment())
}

// HSTS used an exact == "production", so a host set to "Production" was
// production for auth and not for HSTS.
func TestProductionCheck_HSTSUsesTheSharedPredicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, env := range []string{"production", "Production", " PRODUCTION "} {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("environment", env); c.Next() })
		r.Use(SecurityHeadersMiddleware())
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"), "environment %q must send HSTS", env)
	}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("environment", "development"); c.Next() })
	r.Use(SecurityHeadersMiddleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"), "development must not send HSTS")
}
