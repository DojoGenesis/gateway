package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The registration switch is the only thing standing between the public
// internet and a role=user JWT that can reach /v1. Until recently it was
// hardcoded true in two places and the config key that was supposed to control
// it did not exist on the config struct, so `registration_enabled: false` in a
// deployed file did nothing at all. These tests assert the switch is real in
// both positions, at the HTTP layer, on the route that is actually registered
// (server/router.go: auth.POST("/register", s.handleAuthRegister) — the gin
// path; the workflow ServeMux path serves no auth routes).

// validRegistration is a body that passes every other check in the handler, so
// a 403 can only come from the registration switch and a 201 proves the whole
// path works.
func validRegistration(email string) map[string]string {
	return map[string]string{
		"email":        email,
		"password":     "correcthorsebattery",
		"display_name": "Test User",
	}
}

func TestRegister_DisabledByConfig_IsRefused(t *testing.T) {
	s, router := newAuthTestServer(t)
	s.cfg.RegistrationEnabled = false

	w := postJSON(router, "/auth/register", validRegistration("closed@lab.edu"))

	require.Equal(t, http.StatusForbidden, w.Code,
		"a disabled registration endpoint must refuse, not fall through to validation: %s", w.Body.String())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok, "error object expected: %s", w.Body.String())
	assert.Equal(t, "registration_disabled", errObj["code"])

	// No token may be minted on the refused path.
	assert.NotContains(t, w.Body.String(), "access_token")
}

// TestRegister_DisabledByConfig_RefusesBeforeValidation pins the ordering. The
// live symptom that exposed the bug was POST /auth/register returning 400
// "Email and display_name are required" — a validation error, which is only
// reachable once the switch has already let the request through. A closed
// endpoint must answer 403 even for a request that is otherwise malformed.
func TestRegister_DisabledByConfig_RefusesBeforeValidation(t *testing.T) {
	s, router := newAuthTestServer(t)
	s.cfg.RegistrationEnabled = false

	w := postJSON(router, "/auth/register", map[string]string{})

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an empty body must still be refused with 403, not answered with a validation error")
}

func TestRegister_EnabledByConfig_Succeeds(t *testing.T) {
	s, router := newAuthTestServer(t)
	s.cfg.RegistrationEnabled = true

	w := postJSON(router, "/auth/register", validRegistration("open@lab.edu"))

	require.Equal(t, http.StatusCreated, w.Code, "open registration must still work: %s", w.Body.String())

	var resp authTokenResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.UserID)
	assert.NotEmpty(t, resp.AccessToken)
}

// TestRegister_SwitchIsReadPerRequest guards against the switch being captured
// at route-registration time. Both answers must come from one running server.
func TestRegister_SwitchIsReadPerRequest(t *testing.T) {
	s, router := newAuthTestServer(t)

	s.cfg.RegistrationEnabled = false
	assert.Equal(t, http.StatusForbidden,
		postJSON(router, "/auth/register", validRegistration("a@lab.edu")).Code)

	s.cfg.RegistrationEnabled = true
	assert.Equal(t, http.StatusCreated,
		postJSON(router, "/auth/register", validRegistration("b@lab.edu")).Code)
}
