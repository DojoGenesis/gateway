package telegram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// DGS-115: an adapter with no secret configured must refuse, not accept.

func TestDGS115_Telegram_NoSecretConfiguredRefuses(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	for name, header := range map[string]string{
		"no header":    "",
		"empty header": " ",
		"any header":   "guess",
	} {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram", strings.NewReader(`{}`))
		if header != "" {
			req.Header.Set(secretTokenHeader, header)
		}
		if err := a.VerifySignature(req); err == nil {
			t.Errorf("%s: VerifySignature returned nil with no secret configured; want a refusal", name)
		}
	}
}

func TestDGS115_Telegram_HandleWebhookNoSecretIs401(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")
	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram",
		strings.NewReader(`{"update_id":1,"message":{"message_id":1,"date":1,"chat":{"id":1,"type":"private"},"text":"x"}}`))
	rec := httptest.NewRecorder()
	a.HandleWebhook(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("HandleWebhook with no secret configured: status %d, want 401", rec.Code)
	}
}

func TestDGS115_Telegram_WhitespaceSecretCountsAsUnset(t *testing.T) {
	// A secret of only whitespace is a config mistake, not a secret — and the
	// header value " " would otherwise match it.
	a := NewTelegramAdapter("test-token", "  ")
	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram", nil)
	req.Header.Set(secretTokenHeader, "  ")
	if err := a.VerifySignature(req); err == nil {
		t.Fatal("whitespace-only secret was accepted as a match")
	}
}

func TestDGS115_Telegram_CorrectSecretStillAccepted(t *testing.T) {
	a := NewTelegramAdapter("test-token", "s3cret")
	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram", nil)
	req.Header.Set(secretTokenHeader, "s3cret")
	if err := a.VerifySignature(req); err != nil {
		t.Fatalf("correct secret rejected: %v", err)
	}
	req.Header.Set(secretTokenHeader, "s3cre")
	if err := a.VerifySignature(req); err == nil {
		t.Fatal("prefix of the secret was accepted")
	}
}
