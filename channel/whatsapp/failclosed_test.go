package whatsapp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// DGS-115: an adapter with no app secret configured must refuse, not accept.

func TestDGS115_WhatsApp_NoAppSecretConfiguredRefuses(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{PhoneNumberID: "p", AccessToken: "t"})
	body := []byte(`{"object":"whatsapp_business_account"}`)

	// No header, and a header computed with the empty key — which is what an
	// attacker would send if an empty secret were ever used as the HMAC key.
	for name, sig := range map[string]string{
		"no header":      "",
		"empty-key hmac": hmacSig("", body),
	} {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", bytes.NewReader(body))
		if sig != "" {
			req.Header.Set(signatureHeader, sig)
		}
		if err := a.VerifySignature(req); err == nil {
			t.Errorf("%s: VerifySignature returned nil with no app secret configured; want a refusal", name)
		}
	}
}

func TestDGS115_WhatsApp_HandleWebhookPOSTNoSecretIs401(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{PhoneNumberID: "p"})
	called := false
	a.OnMessage(func(_ *channel.ChannelMessage) { called = true })

	raw := mustMarshal(t, sampleTextPayload("p", "1555", "wamid.1", "forged"))
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST with no app secret configured: status %d, want 401", rec.Code)
	}
	if called {
		t.Error("message handler ran for an unverifiable POST")
	}
}
