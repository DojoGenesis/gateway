package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// DGS-137: Meta confirms a WhatsApp webhook subscription with an unsigned GET
// (hub.mode, hub.verify_token, hub.challenge). WebhookGateway.ServeHTTP ran
// every request through VerifySignature first, and a GET carries no
// X-Hub-Signature-256, so the handshake was answered 401 by the gateway and
// never reached WhatsAppAdapter.handleVerification. The subscription could not
// be completed through the bridge at all.

func whatsappBridge(t *testing.T, verifyToken string) (http.Handler, *recordingBus) {
	t.Helper()
	clearChannelEnv(t)
	t.Setenv("DOJO_WHATSAPP_PHONE_NUMBER_ID", "123")
	t.Setenv("DOJO_WHATSAPP_ACCESS_TOKEN", "access")
	t.Setenv("DOJO_WHATSAPP_APP_SECRET", "wa-secret")
	t.Setenv("DOJO_WHATSAPP_VERIFY_TOKEN", verifyToken)
	return bridgeUnderTest(t)
}

func TestDGS137_WhatsAppHandshakeReachesTheHandler(t *testing.T) {
	h, bus := whatsappBridge(t, "verify-me")

	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=1158201444", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "1158201444" {
		t.Fatalf("handshake through the bridge: status %d body %q; want 200 and the challenge echoed",
			rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	if got := bus.published(); len(got) != 0 {
		t.Errorf("a handshake is not a message, but the bus got: %v", got)
	}
}

func TestDGS137_WhatsAppHandshakeWrongTokenRefused(t *testing.T) {
	h, _ := whatsappBridge(t, "verify-me")

	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=guess&hub.challenge=x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong verify token through the bridge: status %d, want 403", rec.Code)
	}
}

// With the routing fixed, the empty-token match is no longer shielded by the
// gateway's accidental 401 — this is the pair that must hold together.
func TestDGS137_WhatsAppHandshakeUnsetTokenRefused(t *testing.T) {
	h, _ := whatsappBridge(t, "")

	req := httptest.NewRequest(http.MethodGet,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.challenge=pwned", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK || rec.Body.String() == "pwned" {
		t.Fatalf("unset verify token through the bridge: status %d body %q; want a refusal", rec.Code, rec.Body.String())
	}
}

// Routing the GET past signature verification must not open anything else:
// an unsigned POST is still refused, and other platforms get no GET bypass.
func TestDGS137_HandshakeRoutingOpensNothingElse(t *testing.T) {
	clearChannelEnv(t)
	t.Setenv("DOJO_WHATSAPP_PHONE_NUMBER_ID", "123")
	t.Setenv("DOJO_WHATSAPP_ACCESS_TOKEN", "access")
	t.Setenv("DOJO_WHATSAPP_APP_SECRET", "wa-secret")
	t.Setenv("DOJO_WHATSAPP_VERIFY_TOKEN", "verify-me")
	t.Setenv("DOJO_TELEGRAM_BOT_TOKEN", "123:bot-token")
	t.Setenv("DOJO_TELEGRAM_SECRET_TOKEN", "tg-secret")
	h, bus := bridgeUnderTest(t)

	post := httptest.NewRequest(http.MethodPost,
		"/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=x",
		strings.NewReader(whatsappPayload))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, post)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unsigned WhatsApp POST carrying handshake params: status %d, want 401", rec.Code)
	}

	get := httptest.NewRequest(http.MethodGet, "/webhooks/telegram?hub.mode=subscribe", strings.NewReader(telegramUpdate))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, get)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unsigned Telegram GET: status %d, want 401", rec.Code)
	}

	if got := bus.published(); len(got) != 0 {
		t.Errorf("nothing here is authentic, but the bus got: %v", got)
	}
}
