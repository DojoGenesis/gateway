package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newAdapter returns a default EmailAdapter suitable for most tests.
func newAdapter() *EmailAdapter {
	return New(EmailConfig{
		WebhookSecret:  "test-secret",
		SendGridAPIKey: "SG.test-key",
		FromAddress:    "bot@example.com",
		FromName:       "Dojo Bot",
	})
}

// buildInboundJSON serialises an InboundEmail to JSON bytes for Normalize.
func buildInboundJSON(e InboundEmail) []byte {
	b, err := json.Marshal(e)
	if err != nil {
		panic(fmt.Sprintf("buildInboundJSON: %v", err))
	}
	return b
}

// buildMultipartRequest constructs an *http.Request with a multipart/form-data
// body from the provided fields map, adding the X-Webhook-Secret header.
func buildMultipartRequest(fields map[string]string, secret string) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/webhook/email", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	return req
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestEmailAdapter_Name(t *testing.T) {
	a := newAdapter()
	if got := a.Name(); got != "email" {
		t.Errorf("Name() = %q; want %q", got, "email")
	}
}

func TestEmailAdapter_Capabilities(t *testing.T) {
	a := newAdapter()
	caps := a.Capabilities()

	if !caps.SupportsThreads {
		t.Error("Capabilities().SupportsThreads = false; want true")
	}
	if caps.SupportsReactions {
		t.Error("Capabilities().SupportsReactions = true; want false")
	}
	if !caps.SupportsAttachments {
		t.Error("Capabilities().SupportsAttachments = false; want true")
	}
	if caps.SupportsEdits {
		t.Error("Capabilities().SupportsEdits = true; want false")
	}
	if caps.MaxMessageLength != 0 {
		t.Errorf("Capabilities().MaxMessageLength = %d; want 0", caps.MaxMessageLength)
	}
}

func TestEmailAdapter_Normalize_SimpleEmail(t *testing.T) {
	a := newAdapter()

	raw := buildInboundJSON(InboundEmail{
		From:    "sender@example.com",
		To:      "inbox@dojo.ai",
		Subject: "Hello world",
		Text:    "Hi there!",
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.Platform != "email" {
		t.Errorf("Platform = %q; want %q", msg.Platform, "email")
	}
	if msg.ChannelID != "inbox@dojo.ai" {
		t.Errorf("ChannelID = %q; want %q", msg.ChannelID, "inbox@dojo.ai")
	}
	if msg.Text != "Hi there!" {
		t.Errorf("Text = %q; want %q", msg.Text, "Hi there!")
	}
	if subj, ok := msg.Metadata["subject"].(string); !ok || subj != "Hello world" {
		t.Errorf("Metadata[subject] = %v; want %q", msg.Metadata["subject"], "Hello world")
	}
}

func TestEmailAdapter_Normalize_WithReplyTo(t *testing.T) {
	a := newAdapter()

	headers := strings.Join([]string{
		"Message-ID: <abc123@mail.example.com>",
		"In-Reply-To: <original-msg-id@mail.example.com>",
	}, "\n")

	raw := buildInboundJSON(InboundEmail{
		From:    "reply@example.com",
		To:      "inbox@dojo.ai",
		Subject: "Re: Test",
		Text:    "This is a reply.",
		Headers: headers,
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.ThreadID != "original-msg-id@mail.example.com" {
		t.Errorf("ThreadID = %q; want %q", msg.ThreadID, "original-msg-id@mail.example.com")
	}
	if msg.ID != "abc123@mail.example.com" {
		t.Errorf("ID = %q; want %q", msg.ID, "abc123@mail.example.com")
	}
}

func TestEmailAdapter_Normalize_ParseFromAddress(t *testing.T) {
	a := newAdapter()

	raw := buildInboundJSON(InboundEmail{
		From:    "John Doe <john@example.com>",
		To:      "inbox@dojo.ai",
		Subject: "Greetings",
		Text:    "Hello.",
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.UserName != "John Doe" {
		t.Errorf("UserName = %q; want %q", msg.UserName, "John Doe")
	}
	if msg.UserID != "john@example.com" {
		t.Errorf("UserID = %q; want %q", msg.UserID, "john@example.com")
	}
}

func TestEmailAdapter_Normalize_HTMLFallback(t *testing.T) {
	a := newAdapter()

	raw := buildInboundJSON(InboundEmail{
		From:    "sender@example.com",
		To:      "inbox@dojo.ai",
		Subject: "HTML only",
		Text:    "", // intentionally empty
		HTML:    "<p>Hello HTML world</p>",
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.Text != "<p>Hello HTML world</p>" {
		t.Errorf("Text = %q; want %q", msg.Text, "<p>Hello HTML world</p>")
	}
}

func TestEmailAdapter_VerifySignature_Valid(t *testing.T) {
	a := newAdapter()

	req := httptest.NewRequest(http.MethodPost, "/webhook/email", nil)
	req.Header.Set("X-Webhook-Secret", "test-secret")

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("VerifySignature() unexpected error: %v", err)
	}
}

func TestEmailAdapter_VerifySignature_Invalid(t *testing.T) {
	a := newAdapter()

	req := httptest.NewRequest(http.MethodPost, "/webhook/email", nil)
	req.Header.Set("X-Webhook-Secret", "wrong-secret")

	if err := a.VerifySignature(req); err == nil {
		t.Error("VerifySignature() expected error for wrong secret; got nil")
	}
}

func TestEmailAdapter_HandleWebhook_FormData(t *testing.T) {
	a := newAdapter()

	fields := map[string]string{
		"from":    "Alice <alice@example.com>",
		"to":      "inbox@dojo.ai",
		"subject": "Webhook test",
		"text":    "Testing webhook.",
	}

	req := buildMultipartRequest(fields, "test-secret")
	rr := httptest.NewRecorder()

	a.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("HandleWebhook() status = %d; want %d", rr.Code, http.StatusOK)
	}

	var msg channel.ChannelMessage
	if err := json.NewDecoder(rr.Body).Decode(&msg); err != nil {
		t.Fatalf("HandleWebhook() response decode error: %v", err)
	}

	if msg.Platform != "email" {
		t.Errorf("msg.Platform = %q; want %q", msg.Platform, "email")
	}
	if msg.UserName != "Alice" {
		t.Errorf("msg.UserName = %q; want %q", msg.UserName, "Alice")
	}
	if msg.Text != "Testing webhook." {
		t.Errorf("msg.Text = %q; want %q", msg.Text, "Testing webhook.")
	}
}

func TestEmailAdapter_Send(t *testing.T) {
	// Start a mock HTTP server that records the SendGrid request.
	var capturedBody []byte
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted) // SendGrid returns 202
	}))
	defer srv.Close()

	a := New(EmailConfig{
		SendGridAPIKey: "SG.testkey",
		FromAddress:    "bot@example.com",
		FromName:       "Dojo Bot",
	})
	a.sendURL = srv.URL // override endpoint

	msg := &channel.ChannelMessage{
		Platform:  "email",
		ChannelID: "recipient@example.com",
		Text:      "Hello from Dojo!",
		Metadata: map[string]interface{}{
			"subject": "Test subject",
		},
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if capturedAuth != "Bearer SG.testkey" {
		t.Errorf("Authorization header = %q; want %q", capturedAuth, "Bearer SG.testkey")
	}

	// Verify the body contains the recipient address.
	if !bytes.Contains(capturedBody, []byte("recipient@example.com")) {
		t.Errorf("request body does not contain recipient address; body = %s", capturedBody)
	}

	// Verify the subject is included.
	if !bytes.Contains(capturedBody, []byte("Test subject")) {
		t.Errorf("request body does not contain subject; body = %s", capturedBody)
	}
}
