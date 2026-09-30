package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const webchatPayload = `{"text":"forged","user_id":"u_mallory","session_id":"s_1"}`

func postWebchat(t *testing.T, h http.Handler, auth string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", strings.NewReader(webchatPayload))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// TestDGS142_BridgeRefusesAnonymousWebchatWhenNoTokenConfigured is the
// production shape: the bridge registered webchat unconditionally, and with
// no DOJO_WEBCHAT_TOKEN the adapter skipped verification, so an anonymous
// POST was accepted and published to the bus. It must now be refused, and
// nothing may reach the bus.
func TestDGS142_BridgeRefusesAnonymousWebchatWhenNoTokenConfigured(t *testing.T) {
	clearChannelEnv(t)
	h, bus := bridgeUnderTest(t)

	for _, auth := range []string{"", "Bearer ", "Bearer anything"} {
		if code := postWebchat(t, h, auth); code == http.StatusOK {
			t.Errorf("webchat POST (Authorization %q) with no token configured: status 200, want a refusal", auth)
		}
	}
	if got := bus.published(); len(got) != 0 {
		t.Errorf("anonymous webchat POST reached the bus: %v", got)
	}
}

// TestDGS142_BridgeAcceptsWebchatWithTheConfiguredBearer: with a token set,
// the right bearer is accepted and published; a wrong one is refused.
func TestDGS142_BridgeAcceptsWebchatWithTheConfiguredBearer(t *testing.T) {
	clearChannelEnv(t)
	t.Setenv("DOJO_WEBCHAT_TOKEN", "wc-secret")
	h, bus := bridgeUnderTest(t)

	if code := postWebchat(t, h, "Bearer wrong"); code != http.StatusUnauthorized {
		t.Errorf("wrong bearer: status %d, want 401", code)
	}
	if got := bus.published(); len(got) != 0 {
		t.Fatalf("wrong-bearer webchat POST reached the bus: %v", got)
	}

	if code := postWebchat(t, h, "Bearer wc-secret"); code != http.StatusOK {
		t.Errorf("correct bearer: status %d, want 200", code)
	}
	found := false
	for _, s := range bus.published() {
		if strings.Contains(s, "webchat") {
			found = true
		}
	}
	if !found {
		t.Errorf("correct-bearer webchat POST was not published: %v", bus.published())
	}
}
