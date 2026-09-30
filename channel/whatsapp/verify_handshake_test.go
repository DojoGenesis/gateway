package whatsapp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// DGS-137: the GET subscription handshake compared hub.verify_token with ==
// against the configured VerifyToken. With VerifyToken unset, a request that
// omits hub.verify_token (or sends it empty) matched "" == "" and was answered
// with the caller's challenge — a subscription confirmed by nobody.

func TestDGS137_EmptyVerifyTokenNeverMatches(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{AppSecret: "s", VerifyToken: ""})

	for _, q := range []string{
		"hub.mode=subscribe&hub.challenge=pwned",                   // param absent
		"hub.mode=subscribe&hub.verify_token=&hub.challenge=pwned", // param empty
	} {
		req := httptest.NewRequest(http.MethodGet, "/webhooks/whatsapp?"+q, nil)
		rec := httptest.NewRecorder()
		a.HandleWebhook(rec, req)

		if rec.Code == http.StatusOK || rec.Body.String() == "pwned" {
			t.Errorf("%s: status %d body %q — an unset verify token matched an empty one", q, rec.Code, rec.Body.String())
		}
	}
}

func TestDGS137_WhitespaceVerifyTokenCountsAsUnset(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{AppSecret: "s", VerifyToken: "  "})
	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=%20%20&hub.challenge=pwned", nil)
	rec := httptest.NewRecorder()
	a.HandleWebhook(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("whitespace verify token matched: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestDGS137_ConfiguredVerifyTokenStillWorks(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{AppSecret: "s", VerifyToken: "tok-123"})

	ok := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=tok-123&hub.challenge=4242", nil)
	rec := httptest.NewRecorder()
	a.HandleWebhook(rec, ok)
	if rec.Code != http.StatusOK || rec.Body.String() != "4242" {
		t.Fatalf("correct token: status %d body %q, want 200 \"4242\"", rec.Code, rec.Body.String())
	}

	bad := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=tok-12&hub.challenge=4242", nil)
	rec = httptest.NewRecorder()
	a.HandleWebhook(rec, bad)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("prefix of the token: status %d, want 403", rec.Code)
	}
}
