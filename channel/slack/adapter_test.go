package slack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/channel"
	slackgo "github.com/slack-go/slack"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testSigningSecret = "test-signing-secret-abc123"

func newTestAdapter() *SlackAdapter {
	return NewWithSender(SlackConfig{
		BotToken:      "xoxb-test",
		SigningSecret: testSigningSecret,
		Mode:          "http",
	}, &mockSlackSender{})
}

// buildSignedRequest constructs an *http.Request with a valid Slack signature.
func buildSignedRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sigBase := fmt.Sprintf("%s:%s:%s", signatureVersion, ts, string(body))
	mac := hmac.New(sha256.New, []byte(testSigningSecret))
	mac.Write([]byte(sigBase))
	sig := signatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))

	r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", bytes.NewReader(body))
	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", sig)
	r.Header.Set("Content-Type", "application/json")
	return r
}

// mockSlackSender records all PostMessage calls for assertion.
type mockSlackSender struct {
	calls []mockCall
	err   error // if non-nil, returned by PostMessage
}

type mockCall struct {
	channelID string
	options   []slackgo.MsgOption
}

func (m *mockSlackSender) PostMessage(channelID string, options ...slackgo.MsgOption) (string, string, error) {
	m.calls = append(m.calls, mockCall{channelID: channelID, options: options})
	if m.err != nil {
		return "", "", m.err
	}
	return channelID, "1609459200.000001", nil
}

// ---------------------------------------------------------------------------
// Test: Name
// ---------------------------------------------------------------------------

func TestSlackAdapter_Name(t *testing.T) {
	a := newTestAdapter()
	if got := a.Name(); got != "slack" {
		t.Errorf("Name() = %q, want %q", got, "slack")
	}
}

// ---------------------------------------------------------------------------
// Test: Capabilities
// ---------------------------------------------------------------------------

func TestSlackAdapter_Capabilities(t *testing.T) {
	a := newTestAdapter()
	caps := a.Capabilities()

	if !caps.SupportsThreads {
		t.Error("SupportsThreads should be true")
	}
	if !caps.SupportsReactions {
		t.Error("SupportsReactions should be true")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true")
	}
	if !caps.SupportsEdits {
		t.Error("SupportsEdits should be true")
	}
	if caps.MaxMessageLength != slackMaxMessageLength {
		t.Errorf("MaxMessageLength = %d, want %d", caps.MaxMessageLength, slackMaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — plain message
// ---------------------------------------------------------------------------

func TestSlackAdapter_Normalize_Message(t *testing.T) {
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U12345",
			"text": "hello world",
			"channel": "C98765",
			"ts": "1609459200.000001"
		}
	}`

	a := newTestAdapter()
	msg, err := a.Normalize([]byte(payload))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.Platform != "slack" {
		t.Errorf("Platform = %q, want %q", msg.Platform, "slack")
	}
	if msg.UserID != "U12345" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "U12345")
	}
	if msg.Text != "hello world" {
		t.Errorf("Text = %q, want %q", msg.Text, "hello world")
	}
	if msg.ChannelID != "C98765" {
		t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "C98765")
	}
	if msg.ThreadID != "" {
		t.Errorf("ThreadID should be empty for non-threaded message, got %q", msg.ThreadID)
	}
	if msg.ID == "" {
		t.Error("ID should not be empty")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	// Verify timestamp is parsed from the Slack ts field.
	wantTime := time.Unix(1609459200, 1000).UTC()
	if !msg.Timestamp.Equal(wantTime) {
		t.Errorf("Timestamp = %v, want %v", msg.Timestamp, wantTime)
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — threaded reply
// ---------------------------------------------------------------------------

func TestSlackAdapter_Normalize_ThreadReply(t *testing.T) {
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U99999",
			"text": "thread reply",
			"channel": "C11111",
			"ts": "1609459210.000002",
			"thread_ts": "1609459200.000001"
		}
	}`

	a := newTestAdapter()
	msg, err := a.Normalize([]byte(payload))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if msg.ThreadID != "1609459200.000001" {
		t.Errorf("ThreadID = %q, want %q", msg.ThreadID, "1609459200.000001")
	}
	if msg.UserID != "U99999" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "U99999")
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — message with file attachment
// ---------------------------------------------------------------------------

func TestSlackAdapter_Normalize_WithAttachment(t *testing.T) {
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U55555",
			"text": "check this file",
			"channel": "C22222",
			"ts": "1609459300.000001",
			"files": [
				{
					"id": "F001",
					"name": "report.pdf",
					"mimetype": "application/pdf",
					"size": 204800,
					"url_private": "https://files.slack.com/files-pri/T-F001/report.pdf"
				},
				{
					"id": "F002",
					"name": "screenshot.png",
					"mimetype": "image/png",
					"size": 51200,
					"url_private": "https://files.slack.com/files-pri/T-F002/screenshot.png"
				}
			]
		}
	}`

	a := newTestAdapter()
	msg, err := a.Normalize([]byte(payload))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if len(msg.Attachments) != 2 {
		t.Fatalf("len(Attachments) = %d, want 2", len(msg.Attachments))
	}

	pdf := msg.Attachments[0]
	if pdf.Type != "file" {
		t.Errorf("Attachments[0].Type = %q, want %q", pdf.Type, "file")
	}
	if pdf.Name != "report.pdf" {
		t.Errorf("Attachments[0].Name = %q, want %q", pdf.Name, "report.pdf")
	}
	if pdf.Size != 204800 {
		t.Errorf("Attachments[0].Size = %d, want %d", pdf.Size, 204800)
	}
	if pdf.MimeType != "application/pdf" {
		t.Errorf("Attachments[0].MimeType = %q, want %q", pdf.MimeType, "application/pdf")
	}

	img := msg.Attachments[1]
	if img.Type != "image" {
		t.Errorf("Attachments[1].Type = %q, want %q", img.Type, "image")
	}
	if img.Name != "screenshot.png" {
		t.Errorf("Attachments[1].Name = %q, want %q", img.Name, "screenshot.png")
	}
}

// ---------------------------------------------------------------------------
// Test: VerifySignature — valid
// ---------------------------------------------------------------------------

func TestSlackAdapter_VerifySignature_Valid(t *testing.T) {
	a := newTestAdapter()
	body := []byte(`{"type":"event_callback","event":{"text":"hi"}}`)
	r := buildSignedRequest(t, body)

	if err := a.VerifySignature(r); err != nil {
		t.Errorf("VerifySignature() returned unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: VerifySignature — wrong signature
// ---------------------------------------------------------------------------

func TestSlackAdapter_VerifySignature_Invalid(t *testing.T) {
	a := newTestAdapter()
	body := []byte(`{"type":"event_callback"}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())

	r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", bytes.NewReader(body))
	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", "v0=deadbeefdeadbeefdeadbeefdeadbeef")

	err := a.VerifySignature(r)
	if err == nil {
		t.Error("VerifySignature() expected error for invalid signature, got nil")
	}
	if !strings.Contains(err.Error(), "signature mismatch") {
		t.Errorf("expected 'signature mismatch' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: VerifySignature — expired timestamp
// ---------------------------------------------------------------------------

func TestSlackAdapter_VerifySignature_Expired(t *testing.T) {
	a := newTestAdapter()
	body := []byte(`{"type":"event_callback"}`)

	// Timestamp more than 5 minutes in the past.
	oldTS := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	sigBase := fmt.Sprintf("%s:%s:%s", signatureVersion, oldTS, string(body))
	mac := hmac.New(sha256.New, []byte(testSigningSecret))
	mac.Write([]byte(sigBase))
	sig := signatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))

	r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", bytes.NewReader(body))
	r.Header.Set("X-Slack-Request-Timestamp", oldTS)
	r.Header.Set("X-Slack-Signature", sig)

	err := a.VerifySignature(r)
	if err == nil {
		t.Error("VerifySignature() expected replay-window error, got nil")
	}
	if !strings.Contains(err.Error(), "replay window") {
		t.Errorf("expected 'replay window' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: HandleWebhook — URL verification challenge
// ---------------------------------------------------------------------------

func TestSlackAdapter_HandleWebhook_URLVerification(t *testing.T) {
	a := newTestAdapter()

	payload := map[string]string{
		"type":      "url_verification",
		"challenge": "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
	}
	body, _ := json.Marshal(payload)

	r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	a.HandleWebhook(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got map[string]string
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &got); err != nil {
		t.Fatalf("could not parse response body: %v", err)
	}
	if got["challenge"] != "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P" {
		t.Errorf("challenge = %q, want original challenge value", got["challenge"])
	}
}

// ---------------------------------------------------------------------------
// Test: HandleWebhook — event callback
// ---------------------------------------------------------------------------

func TestSlackAdapter_HandleWebhook_EventCallback(t *testing.T) {
	a := newTestAdapter()

	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U12345",
			"text": "hello from webhook",
			"channel": "C98765",
			"ts": "1609459200.000001"
		}
	}`

	r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	a.HandleWebhook(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Test: Send — plain message
// ---------------------------------------------------------------------------

func TestSlackAdapter_Send_PlainMessage(t *testing.T) {
	mock := &mockSlackSender{}
	a := NewWithSender(SlackConfig{BotToken: "xoxb-test", SigningSecret: testSigningSecret}, mock)

	msg := &channel.ChannelMessage{
		Platform:  "slack",
		ChannelID: "C12345",
		Text:      "hello from dojo",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 PostMessage call, got %d", len(mock.calls))
	}
	if mock.calls[0].channelID != "C12345" {
		t.Errorf("channelID = %q, want %q", mock.calls[0].channelID, "C12345")
	}
}

// ---------------------------------------------------------------------------
// Test: Send — thread reply
// ---------------------------------------------------------------------------

func TestSlackAdapter_Send_ThreadReply(t *testing.T) {
	mock := &mockSlackSender{}
	a := NewWithSender(SlackConfig{BotToken: "xoxb-test", SigningSecret: testSigningSecret}, mock)

	msg := &channel.ChannelMessage{
		Platform:  "slack",
		ChannelID: "C12345",
		Text:      "thread reply",
		ThreadID:  "1609459200.000001",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 PostMessage call, got %d", len(mock.calls))
	}
	// The test verifies that two MsgOptions are passed (MsgOptionText + MsgOptionTS).
	if len(mock.calls[0].options) != 2 {
		t.Errorf("expected 2 MsgOptions for thread reply, got %d", len(mock.calls[0].options))
	}
}

// ---------------------------------------------------------------------------
// Test: Send — PostMessage error propagation
// ---------------------------------------------------------------------------

func TestSlackAdapter_Send_Error(t *testing.T) {
	mock := &mockSlackSender{err: fmt.Errorf("rate limited")}
	a := NewWithSender(SlackConfig{BotToken: "xoxb-test", SigningSecret: testSigningSecret}, mock)

	msg := &channel.ChannelMessage{
		Platform:  "slack",
		ChannelID: "C12345",
		Text:      "hi",
	}

	err := a.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("Send() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected 'rate limited' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: interface compliance (compile-time check)
// ---------------------------------------------------------------------------

var _ channel.WebhookAdapter = (*SlackAdapter)(nil)
