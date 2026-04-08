package whatsapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mustMarshal marshals v to JSON, fatally failing the test on error.
func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return b
}

// assertField is a generic string-equality helper for test assertions.
func assertField(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// hmacSig computes the X-Hub-Signature-256 value for the given secret and body.
func hmacSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// sampleTextPayload returns a minimal WhatsApp text message webhook payload.
func sampleTextPayload(phoneID, from, msgID, text string) WebhookPayload {
	return WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []Entry{
			{
				ID: "123456",
				Changes: []Change{
					{
						Field: "messages",
						Value: ChangeValue{
							MessagingProduct: "whatsapp",
							Metadata: Metadata{
								DisplayPhoneNumber: "15551234567",
								PhoneNumberID:      phoneID,
							},
							Contacts: []Contact{
								{WaID: from, Profile: ContactProfile{Name: "Alice"}},
							},
							Messages: []Message{
								{
									From:      from,
									ID:        msgID,
									Timestamp: "1712332800",
									Type:      "text",
									Text:      &TextBody{Body: text},
								},
							},
						},
					},
				},
			},
		},
	}
}

// urlRewriteTransport redirects all requests to the given mock server address
// while preserving the request path.
type urlRewriteTransport struct {
	base       http.RoundTripper
	serverAddr string
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(t.serverAddr, "http://")
	return t.base.RoundTrip(cloned)
}

// ---------------------------------------------------------------------------
// 1. TestWhatsAppAdapter_Name
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Name(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})
	if got := a.Name(); got != "whatsapp" {
		t.Errorf("Name() = %q, want %q", got, "whatsapp")
	}
}

// ---------------------------------------------------------------------------
// 2. TestWhatsAppAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Capabilities(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})
	caps := a.Capabilities()

	if caps.SupportsThreads {
		t.Error("SupportsThreads should be false for WhatsApp")
	}
	if !caps.SupportsReactions {
		t.Error("SupportsReactions should be true for WhatsApp")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true for WhatsApp")
	}
	if caps.SupportsEdits {
		t.Error("SupportsEdits should be false for WhatsApp")
	}
	if caps.MaxMessageLength != 4096 {
		t.Errorf("MaxMessageLength = %d, want 4096", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestWhatsAppAdapter_Normalize_TextMessage
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Normalize_TextMessage(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})
	raw := mustMarshal(t, sampleTextPayload("phone-001", "15559876543", "wamid.abc123", "hello whatsapp"))

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "whatsapp")
	assertField(t, "ID", msg.ID, "wamid.abc123")
	assertField(t, "ChannelID", msg.ChannelID, "phone-001")
	assertField(t, "UserID", msg.UserID, "15559876543")
	assertField(t, "UserName", msg.UserName, "Alice")
	assertField(t, "Text", msg.Text, "hello whatsapp")

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(msg.Attachments))
	}
}

// ---------------------------------------------------------------------------
// 4. TestWhatsAppAdapter_Normalize_ImageMessage
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Normalize_ImageMessage(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})

	payload := WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []Entry{
			{
				ID: "entry-1",
				Changes: []Change{
					{
						Field: "messages",
						Value: ChangeValue{
							MessagingProduct: "whatsapp",
							Metadata: Metadata{
								DisplayPhoneNumber: "15551234567",
								PhoneNumberID:      "phone-002",
							},
							Contacts: []Contact{
								{WaID: "15559998888", Profile: ContactProfile{Name: "Bob"}},
							},
							Messages: []Message{
								{
									From:      "15559998888",
									ID:        "wamid.img001",
									Timestamp: "1712332800",
									Type:      "image",
									Image: &Media{
										ID:       "img-media-id",
										MimeType: "image/jpeg",
										Caption:  "check this out",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	raw := mustMarshal(t, payload)
	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize image: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "whatsapp")
	assertField(t, "UserName", msg.UserName, "Bob")

	if len(msg.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	assertField(t, "Attachment.Type", att.Type, "image")
	assertField(t, "Attachment.URL", att.URL, "img-media-id")
	assertField(t, "Attachment.MimeType", att.MimeType, "image/jpeg")
}

// ---------------------------------------------------------------------------
// 5. TestWhatsAppAdapter_VerifySignature_Valid
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_VerifySignature_Valid(t *testing.T) {
	secret := "my-app-secret"
	a := NewWhatsAppAdapter(WhatsAppConfig{AppSecret: secret})

	body := []byte(`{"object":"whatsapp_business_account"}`)
	sig := hmacSig(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(signatureHeader, sig)

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("expected valid signature, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 6. TestWhatsAppAdapter_VerifySignature_Invalid
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_VerifySignature_Invalid(t *testing.T) {
	secret := "correct-secret"
	a := NewWhatsAppAdapter(WhatsAppConfig{AppSecret: secret})

	body := []byte(`{"object":"whatsapp_business_account"}`)

	t.Run("wrong_signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set(signatureHeader, "sha256=badhash")
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for wrong signature, got nil")
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing header, got nil")
		}
	})

	t.Run("no_secret_skips_verification", func(t *testing.T) {
		noSecret := NewWhatsAppAdapter(WhatsAppConfig{})
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		// No header set — should still pass because AppSecret is empty.
		if err := noSecret.VerifySignature(req); err != nil {
			t.Errorf("expected nil when AppSecret is empty, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// 7. TestWhatsAppAdapter_HandleWebhook_Verification (GET challenge)
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_HandleWebhook_Verification(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{VerifyToken: "my-verify-token"})

	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge123", nil)
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "challenge123" {
		t.Errorf("body = %q, want %q", got, "challenge123")
	}
}

func TestWhatsAppAdapter_HandleWebhook_Verification_WrongToken(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{VerifyToken: "correct-token"})

	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=xyz", nil)
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 8. TestWhatsAppAdapter_HandleWebhook_Message (POST)
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_HandleWebhook_Message(t *testing.T) {
	// Use no AppSecret so signature verification is skipped.
	a := NewWhatsAppAdapter(WhatsAppConfig{PhoneNumberID: "phone-001"})

	raw := mustMarshal(t, sampleTextPayload("phone-001", "15551112222", "wamid.post001", "webhook message test"))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook POST status = %d, want 200", rec.Code)
	}
}

func TestWhatsAppAdapter_HandleWebhook_Message_WithHandler(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})

	var received *channel.ChannelMessage
	a.OnMessage(func(msg *channel.ChannelMessage) {
		received = msg
	})

	raw := mustMarshal(t, sampleTextPayload("phone-003", "15553334444", "wamid.handler001", "dispatched to handler"))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if received == nil {
		t.Fatal("expected message handler to be called, got nil")
	}
	assertField(t, "Text", received.Text, "dispatched to handler")
}

// ---------------------------------------------------------------------------
// 9. TestWhatsAppAdapter_Send (mock HTTP)
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Send(t *testing.T) {
	var capturedBody []byte
	var capturedPath string
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messages":[{"id":"wamid.resp001"}]}`))
	}))
	defer srv.Close()

	cfg := WhatsAppConfig{
		PhoneNumberID: "phone-send-01",
		AccessToken:   "test-access-token",
	}
	a := NewWhatsAppAdapterWithClient(cfg, &http.Client{
		Transport: &urlRewriteTransport{
			base:       http.DefaultTransport,
			serverAddr: srv.URL,
		},
	})

	msg := &channel.ChannelMessage{
		ID:        "out-001",
		Platform:  "whatsapp",
		ChannelID: "15557778888",
		Text:      "hello from send test",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// The adapter builds the URL as {APIURL}/{PhoneNumberID}/messages.
	// The default APIURL is "https://graph.facebook.com/v21.0", so the path
	// in the request becomes "/v21.0/{PhoneNumberID}/messages".
	expectedPath := "/v21.0/phone-send-01/messages"
	if capturedPath != expectedPath {
		t.Errorf("path = %q, want %q", capturedPath, expectedPath)
	}

	if capturedAuth != "Bearer test-access-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer test-access-token")
	}

	var payload sendMessageRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if payload.MessagingProduct != "whatsapp" {
		t.Errorf("messaging_product = %q, want %q", payload.MessagingProduct, "whatsapp")
	}
	if payload.To != "15557778888" {
		t.Errorf("to = %q, want %q", payload.To, "15557778888")
	}
	if payload.Type != "text" {
		t.Errorf("type = %q, want %q", payload.Type, "text")
	}
	if payload.Text == nil || payload.Text.Body != "hello from send test" {
		t.Errorf("text.body = %q, want %q", payload.Text, "hello from send test")
	}
}

func TestWhatsAppAdapter_Send_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid access token"}}`))
	}))
	defer srv.Close()

	cfg := WhatsAppConfig{PhoneNumberID: "phone-err-01", AccessToken: "bad-token"}
	a := NewWhatsAppAdapterWithClient(cfg, &http.Client{
		Transport: &urlRewriteTransport{
			base:       http.DefaultTransport,
			serverAddr: srv.URL,
		},
	})

	err := a.Send(context.Background(), &channel.ChannelMessage{ChannelID: "15551234567", Text: "test"})
	if err == nil {
		t.Fatal("expected error for API 401 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// 10. TestWhatsAppAdapter_Connect
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Connect(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})

	creds := channel.Credentials{
		Platform: "whatsapp",
		Token:    "new-access-token",
		Secret:   "new-app-secret",
		Extra: map[string]string{
			"phone_number_id": "phone-connect-01",
		},
	}

	if err := a.Connect(context.Background(), creds); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.connected {
		t.Error("expected connected = true after Connect()")
	}
	if a.cfg.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want %q", a.cfg.AccessToken, "new-access-token")
	}
	if a.cfg.AppSecret != "new-app-secret" {
		t.Errorf("AppSecret = %q, want %q", a.cfg.AppSecret, "new-app-secret")
	}
	if a.cfg.PhoneNumberID != "phone-connect-01" {
		t.Errorf("PhoneNumberID = %q, want %q", a.cfg.PhoneNumberID, "phone-connect-01")
	}
}

func TestWhatsAppAdapter_Connect_CancelledContext(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	if err := a.Connect(ctx, channel.Credentials{}); err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// 11. TestWhatsAppAdapter_Disconnect
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_Disconnect(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})
	// First connect.
	if err := a.Connect(context.Background(), channel.Credentials{Token: "tok"}); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Register a handler.
	a.OnMessage(func(msg *channel.ChannelMessage) {})

	// Now disconnect.
	if err := a.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.connected {
		t.Error("expected connected = false after Disconnect()")
	}
	if a.msgHandler != nil {
		t.Error("expected msgHandler = nil after Disconnect()")
	}
}

// ---------------------------------------------------------------------------
// 12. TestWhatsAppAdapter_OnMessage
// ---------------------------------------------------------------------------

func TestWhatsAppAdapter_OnMessage(t *testing.T) {
	a := NewWhatsAppAdapter(WhatsAppConfig{})

	var called bool
	var receivedText string

	a.OnMessage(func(msg *channel.ChannelMessage) {
		called = true
		receivedText = msg.Text
	})

	// Simulate receiving a message via the webhook handler.
	raw := mustMarshal(t, sampleTextPayload("phone-on-msg", "15556667777", "wamid.onmsg001", "on message test"))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if !called {
		t.Error("expected OnMessage handler to be called")
	}
	if receivedText != "on message test" {
		t.Errorf("received text = %q, want %q", receivedText, "on message test")
	}
}
