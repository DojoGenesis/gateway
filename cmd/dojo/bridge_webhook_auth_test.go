package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// recordingBus is a channel.EventPublisher that remembers what reached the bus.
// Publishing is the side effect that matters: a forged webhook that is
// published has been accepted, whatever status code came back.
type recordingBus struct {
	mu       sync.Mutex
	subjects []string
}

func (b *recordingBus) Publish(subject string, _ channel.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subjects = append(b.subjects, subject)
	return nil
}

func (b *recordingBus) published() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.subjects...)
}

// clearChannelEnv blanks every credential the bridge reads so a developer's
// shell cannot change what gets registered.
func clearChannelEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DOJO_SLACK_TOKEN", "DOJO_SLACK_SIGNINGSECRET", "DOJO_SLACK_APPTOKEN",
		"DOJO_DISCORD_BOT_TOKEN", "DOJO_DISCORD_PUBLIC_KEY",
		"DOJO_TELEGRAM_BOT_TOKEN", "DOJO_TELEGRAM_SECRET_TOKEN",
		"DOJO_EMAIL_WEBHOOK_SECRET", "DOJO_EMAIL_SENDGRID_API_KEY",
		"DOJO_SMS_ACCOUNT_SID", "DOJO_SMS_AUTH_TOKEN",
		"DOJO_WHATSAPP_PHONE_NUMBER_ID", "DOJO_WHATSAPP_ACCESS_TOKEN",
		"DOJO_WHATSAPP_VERIFY_TOKEN", "DOJO_WHATSAPP_APP_SECRET",
		"DOJO_TEAMS_BOT_TOKEN", "DOJO_TEAMS_APP_ID",
		"DOJO_WEBCHAT_TOKEN",
	} {
		t.Setenv(k, "")
	}
}

// bridgeUnderTest wires a WebhookGateway exactly as runBridgeCommand does —
// same credential store, same buildAndRegisterAdapters — over a recording bus.
func bridgeUnderTest(t *testing.T) (http.Handler, *recordingBus) {
	t.Helper()
	bus := &recordingBus{}
	creds := channel.NewEnvCredentialStore()
	gw := channel.NewWebhookGateway(bus, creds)
	buildAndRegisterAdapters(gw, creds)
	mux := http.NewServeMux()
	mux.Handle("/webhooks/", gw)
	return mux, bus
}

const telegramUpdate = `{"update_id":1,"message":{"message_id":7,"date":1712332800,` +
	`"chat":{"id":42,"type":"private"},"from":{"id":9,"username":"mallory"},"text":"forged"}}`

const whatsappPayload = `{"object":"whatsapp_business_account","entry":[{"id":"1","changes":[{"field":"messages",` +
	`"value":{"messaging_product":"whatsapp","metadata":{"phone_number_id":"123"},` +
	`"messages":[{"from":"15550001111","id":"wamid.x","timestamp":"1712332800","type":"text","text":{"body":"forged"}}]}}]}]}`

// TestDGS115_BridgeRefusesUnsignedTelegramWhenNoSecretConfigured: a bridge
// holding a Telegram bot token but no SECRET_TOKEN used to register the adapter
// with verification switched off, so an anonymous POST was accepted and
// published. It must now refuse, and nothing may reach the bus.
func TestDGS115_BridgeRefusesUnsignedTelegramWhenNoSecretConfigured(t *testing.T) {
	clearChannelEnv(t)
	t.Setenv("DOJO_TELEGRAM_BOT_TOKEN", "123:bot-token")

	h, bus := bridgeUnderTest(t)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/telegram", strings.NewReader(telegramUpdate))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("unsigned Telegram webhook with no secret configured: status 200, want a refusal")
	}
	if got := bus.published(); len(got) != 0 {
		t.Errorf("unsigned Telegram webhook reached the bus: %v", got)
	}
}

// TestDGS115_BridgeRefusesUnsignedWhatsAppWhenNoAppSecretConfigured: same
// shape for WhatsApp — PHONE_NUMBER_ID + ACCESS_TOKEN registered the adapter
// even with no APP_SECRET, and VerifySignature then returned nil for anything.
func TestDGS115_BridgeRefusesUnsignedWhatsAppWhenNoAppSecretConfigured(t *testing.T) {
	clearChannelEnv(t)
	t.Setenv("DOJO_WHATSAPP_PHONE_NUMBER_ID", "123")
	t.Setenv("DOJO_WHATSAPP_ACCESS_TOKEN", "access")

	h, bus := bridgeUnderTest(t)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", strings.NewReader(whatsappPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("unsigned WhatsApp webhook with no app secret configured: status 200, want a refusal")
	}
	if got := bus.published(); len(got) != 0 {
		t.Errorf("unsigned WhatsApp webhook reached the bus: %v", got)
	}
}

// TestDGS115_BridgeStillAcceptsCorrectlySignedWebhooks guards the other
// direction: with the secrets configured, genuine platform traffic still flows.
func TestDGS115_BridgeStillAcceptsCorrectlySignedWebhooks(t *testing.T) {
	clearChannelEnv(t)
	t.Setenv("DOJO_TELEGRAM_BOT_TOKEN", "123:bot-token")
	t.Setenv("DOJO_TELEGRAM_SECRET_TOKEN", "tg-secret")
	t.Setenv("DOJO_WHATSAPP_PHONE_NUMBER_ID", "123")
	t.Setenv("DOJO_WHATSAPP_ACCESS_TOKEN", "access")
	t.Setenv("DOJO_WHATSAPP_APP_SECRET", "wa-secret")

	h, bus := bridgeUnderTest(t)

	tg := httptest.NewRequest(http.MethodPost, "/webhooks/telegram", strings.NewReader(telegramUpdate))
	tg.Header.Set("X-Telegram-Bot-Api-Secret-Token", "tg-secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, tg)
	if rec.Code != http.StatusOK {
		t.Errorf("signed Telegram webhook: status %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	body := []byte(whatsappPayload)
	mac := hmac.New(sha256.New, []byte("wa-secret"))
	mac.Write(body)
	wa := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", bytes.NewReader(body))
	wa.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, wa)
	if rec.Code != http.StatusOK {
		t.Errorf("signed WhatsApp webhook: status %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	if got := bus.published(); len(got) != 2 {
		t.Errorf("published subjects = %v, want one telegram and one whatsapp", got)
	}
}
