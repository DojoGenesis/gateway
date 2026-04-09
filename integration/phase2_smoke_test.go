package integration

// Phase 2 smoke tests — Email, SMS, WhatsApp, Teams, WebChat adapters.
//
// Each test verifies the critical path: adapter registers with WebhookGateway,
// inbound HTTP request is routed, signature verification passes (empty-secret
// bypass where applicable), payload normalizes to ChannelMessage, and NATS
// publish is called.
//
// Teams uses a direct Normalize call because JWT JWKS verification requires
// a running Microsoft endpoint or a mock server (tested in channel/teams).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
	"github.com/DojoGenesis/gateway/channel/email"
	"github.com/DojoGenesis/gateway/channel/sms"
	"github.com/DojoGenesis/gateway/channel/teams"
	"github.com/DojoGenesis/gateway/channel/webchat"
	"github.com/DojoGenesis/gateway/channel/whatsapp"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newSmokeGateway creates a WebhookGateway backed by a memPublisher NATSBus.
// Returns the gateway and a function that reports all NATS-published subjects.
func newSmokeGateway(t *testing.T) (*channel.WebhookGateway, func() []string) {
	t.Helper()
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub, channel.WithNATSSubscriber(pub))

	var received []string
	bus.Subscribe(func(subject string, _ channel.Event) {
		received = append(received, subject)
	})

	gw := channel.NewWebhookGateway(bus, nil)
	return gw, func() []string { return received }
}

// postToGateway sends a POST to path on the given test server and returns
// the HTTP response code.
func postToGateway(srv *httptest.Server, path, contentType string, body []byte, headers map[string]string) int {
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(body))
	if err != nil {
		panic(fmt.Sprintf("build request: %v", err))
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(fmt.Sprintf("do request: %v", err))
	}
	resp.Body.Close()
	return resp.StatusCode
}

// ---------------------------------------------------------------------------
// Email (SendGrid Inbound Parse) — empty WebhookSecret bypasses signature check
// ---------------------------------------------------------------------------

func TestPhase2_Email_Smoke(t *testing.T) {
	gw, publishedSubjects := newSmokeGateway(t)

	// Empty WebhookSecret → signature verification skipped.
	adapter := email.New(email.EmailConfig{
		WebhookSecret:  "",
		SendGridAPIKey: "test-key",
		FromAddress:    "noreply@test.example",
		FromName:       "Test",
	})
	gw.Register("email", adapter)

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// The email adapter's Normalize expects a JSON-encoded InboundEmail.
	// (When deployed with real SendGrid, the HandleWebhook method parses
	// multipart and serializes to JSON before calling Normalize. Through the
	// WebhookGateway path, the raw body must already be JSON.)
	emailPayload := map[string]string{
		"from":     "alice@example.com",
		"to":       "bot@dojo.example",
		"subject":  "Hello from integration test",
		"text":     "this is a smoke test email",
		"envelope": `{"from":"alice@example.com","to":["bot@dojo.example"]}`,
	}
	body, _ := json.Marshal(emailPayload)

	status := postToGateway(srv, "/webhooks/email", "application/json", body, nil)

	if status != http.StatusOK {
		t.Errorf("email smoke: status = %d, want 200", status)
	}

	// Gateway should have published a dojo.channel.message.email event.
	time.Sleep(10 * time.Millisecond) // let async publish flush
	subjects := publishedSubjects()
	if len(subjects) == 0 {
		t.Error("email smoke: expected at least one NATS publish, got zero")
	}
	for _, s := range subjects {
		if strings.Contains(s, "email") {
			return // found it
		}
	}
	t.Errorf("email smoke: no dojo.channel.message.email subject in published: %v", subjects)
}

// ---------------------------------------------------------------------------
// SMS (Twilio) — empty AuthToken bypasses HMAC-SHA1 signature check
// ---------------------------------------------------------------------------

func TestPhase2_SMS_Smoke(t *testing.T) {
	gw, publishedSubjects := newSmokeGateway(t)

	// Empty AuthToken → signature verification skipped.
	adapter := sms.NewSMSAdapter(sms.SMSConfig{
		AccountSID: "ACtest123",
		AuthToken:  "", // bypass
		FromNumber: "+15550000000",
	})
	gw.Register("sms", adapter)

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Twilio sends form-encoded POSTs.
	form := url.Values{}
	form.Set("MessageSid", "SMtest123")
	form.Set("From", "+15551234567")
	form.Set("To", "+15550000000")
	form.Set("Body", "hello from integration smoke test")
	form.Set("AccountSid", "ACtest123")

	status := postToGateway(srv, "/webhooks/sms",
		"application/x-www-form-urlencoded",
		[]byte(form.Encode()),
		nil,
	)

	if status != http.StatusOK {
		t.Errorf("sms smoke: status = %d, want 200", status)
	}

	time.Sleep(10 * time.Millisecond)
	subjects := publishedSubjects()
	if len(subjects) == 0 {
		t.Error("sms smoke: expected at least one NATS publish, got zero")
	}
	for _, s := range subjects {
		if strings.Contains(s, "sms") {
			return
		}
	}
	t.Errorf("sms smoke: no dojo.channel.message.sms subject in published: %v", subjects)
}

// ---------------------------------------------------------------------------
// WhatsApp (Meta Cloud API) — empty AppSecret bypasses HMAC-SHA256 check
// ---------------------------------------------------------------------------

func TestPhase2_WhatsApp_Smoke(t *testing.T) {
	gw, publishedSubjects := newSmokeGateway(t)

	// Empty AppSecret → signature verification skipped.
	adapter := whatsapp.NewWhatsAppAdapter(whatsapp.WhatsAppConfig{
		PhoneNumberID: "123456789",
		AccessToken:   "test-access-token",
		VerifyToken:   "test-verify-token",
		AppSecret:     "", // bypass
	})
	gw.Register("whatsapp", adapter)

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Minimal WhatsApp Cloud API webhook payload.
	payload := map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "123456",
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": map[string]interface{}{
							"messaging_product": "whatsapp",
							"metadata": map[string]string{
								"display_phone_number": "15551234567",
								"phone_number_id":      "123456789",
							},
							"contacts": []map[string]interface{}{
								{"wa_id": "15559876543", "profile": map[string]string{"name": "Alice"}},
							},
							"messages": []map[string]interface{}{
								{
									"from":      "15559876543",
									"id":        "wamid.test001",
									"timestamp": "1712332800",
									"type":      "text",
									"text":      map[string]string{"body": "hello whatsapp smoke test"},
								},
							},
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	status := postToGateway(srv, "/webhooks/whatsapp", "application/json", body, nil)

	if status != http.StatusOK {
		t.Errorf("whatsapp smoke: status = %d, want 200", status)
	}

	time.Sleep(10 * time.Millisecond)
	subjects := publishedSubjects()
	if len(subjects) == 0 {
		t.Error("whatsapp smoke: expected at least one NATS publish, got zero")
	}
	for _, s := range subjects {
		if strings.Contains(s, "whatsapp") {
			return
		}
	}
	t.Errorf("whatsapp smoke: no dojo.channel.message.whatsapp subject in published: %v", subjects)
}

// ---------------------------------------------------------------------------
// WebChat (embedded widget) — empty token bypasses Bearer check
// ---------------------------------------------------------------------------

func TestPhase2_WebChat_Smoke(t *testing.T) {
	gw, publishedSubjects := newSmokeGateway(t)

	// Empty token → no auth required.
	adapter := webchat.NewWebChatAdapter("")
	gw.Register("webchat", adapter)

	srv := httptest.NewServer(gw)
	defer srv.Close()

	payload := `{"text":"hello webchat smoke test","user_id":"u_test","session_id":"sess_test"}`
	status := postToGateway(srv, "/webhooks/webchat", "application/json",
		[]byte(payload), nil)

	if status != http.StatusOK {
		t.Errorf("webchat smoke: status = %d, want 200", status)
	}

	time.Sleep(10 * time.Millisecond)
	subjects := publishedSubjects()
	if len(subjects) == 0 {
		t.Error("webchat smoke: expected at least one NATS publish, got zero")
	}
	for _, s := range subjects {
		if strings.Contains(s, "webchat") {
			return
		}
	}
	t.Errorf("webchat smoke: no dojo.channel.message.webchat subject in published: %v", subjects)
}

// ---------------------------------------------------------------------------
// Teams (Bot Framework v3) — Normalize tested directly
//
// JWT JWKS verification requires an external Microsoft endpoint or a mock
// JWKS server (tested in channel/teams package tests). Here we verify that
// a Bot Framework Activity JSON body normalizes correctly to a ChannelMessage.
// ---------------------------------------------------------------------------

func TestPhase2_Teams_Normalize(t *testing.T) {
	adapter := teams.NewTeamsAdapter("test-bot-token", "test-app-id")

	if adapter.Name() != "teams" {
		t.Errorf("teams Name() = %q, want %q", adapter.Name(), "teams")
	}

	activity := map[string]interface{}{
		"type": "message",
		"id":   "activity-001",
		"text": "hello teams smoke test",
		"from": map[string]string{
			"id":   "user-001",
			"name": "Alice",
		},
		"conversation": map[string]string{
			"id":   "conv-001",
			"name": "General",
		},
		"recipient": map[string]string{
			"id":   "bot-001",
			"name": "Dojo",
		},
		"serviceUrl": "https://smba.trafficmanager.net/amer/",
		"channelId":  "msteams",
	}
	body, _ := json.Marshal(activity)

	msg, err := adapter.Normalize(body)
	if err != nil {
		t.Fatalf("teams Normalize: %v", err)
	}

	if msg.Text != "hello teams smoke test" {
		t.Errorf("teams Normalize Text = %q, want %q", msg.Text, "hello teams smoke test")
	}
	if msg.Platform != "teams" {
		t.Errorf("teams Normalize Platform = %q, want %q", msg.Platform, "teams")
	}
	if msg.UserID == "" {
		t.Error("teams Normalize UserID is empty")
	}
	if msg.ChannelID == "" {
		t.Error("teams Normalize ChannelID is empty")
	}
}
