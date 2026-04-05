package discord

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestAdapter returns a DiscordAdapter configured with a freshly generated
// Ed25519 key pair. The private key is returned so tests can sign payloads.
func newTestAdapter(t *testing.T) (*DiscordAdapter, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	a, err := New(DiscordConfig{
		PublicKey: hex.EncodeToString(pub),
		AppID:     "test-app",
		// BotToken intentionally empty — no real Discord session needed.
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return a, priv
}

// signedRequest creates an *http.Request whose X-Signature-Ed25519 and
// X-Signature-Timestamp headers are valid for the given body and private key.
func signedRequest(t *testing.T, body []byte, priv ed25519.PrivateKey) *http.Request {
	t.Helper()

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	message := append([]byte(timestamp), body...)
	sig := ed25519.Sign(priv, message)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	req.Header.Set("X-Signature-Timestamp", timestamp)
	return req
}

// ---------------------------------------------------------------------------
// 1. TestDiscordAdapter_Name
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Name(t *testing.T) {
	a, _ := newTestAdapter(t)
	if got := a.Name(); got != "discord" {
		t.Errorf("Name() = %q, want %q", got, "discord")
	}
}

// ---------------------------------------------------------------------------
// 2. TestDiscordAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Capabilities(t *testing.T) {
	a, _ := newTestAdapter(t)
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
	if caps.MaxMessageLength != 2000 {
		t.Errorf("MaxMessageLength = %d, want 2000", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestDiscordAdapter_Normalize_Message
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Normalize_Message(t *testing.T) {
	a, _ := newTestAdapter(t)

	raw := []byte(`{
		"id": "111222333",
		"channel_id": "CHAN-001",
		"content": "hello from discord",
		"timestamp": "2026-04-05T12:00:00Z",
		"author": {
			"id": "USER-42",
			"username": "alice"
		}
	}`)

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if msg.ID != "111222333" {
		t.Errorf("ID = %q, want %q", msg.ID, "111222333")
	}
	if msg.Platform != "discord" {
		t.Errorf("Platform = %q, want %q", msg.Platform, "discord")
	}
	if msg.ChannelID != "CHAN-001" {
		t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "CHAN-001")
	}
	if msg.UserID != "USER-42" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "USER-42")
	}
	if msg.UserName != "alice" {
		t.Errorf("UserName = %q, want %q", msg.UserName, "alice")
	}
	if msg.Text != "hello from discord" {
		t.Errorf("Text = %q, want %q", msg.Text, "hello from discord")
	}
	if msg.ThreadID != "" {
		t.Errorf("ThreadID should be empty, got %q", msg.ThreadID)
	}
}

// ---------------------------------------------------------------------------
// 4. TestDiscordAdapter_Normalize_Reply
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Normalize_Reply(t *testing.T) {
	a, _ := newTestAdapter(t)

	raw := []byte(`{
		"id": "999",
		"channel_id": "CHAN-002",
		"content": "this is a reply",
		"timestamp": "2026-04-05T13:00:00Z",
		"author": {"id": "U2", "username": "bob"},
		"message_reference": {
			"message_id": "PARENT-MSG-ID"
		}
	}`)

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize reply: %v", err)
	}

	if msg.ThreadID != "PARENT-MSG-ID" {
		t.Errorf("ThreadID = %q, want %q", msg.ThreadID, "PARENT-MSG-ID")
	}
	if msg.Text != "this is a reply" {
		t.Errorf("Text = %q, want %q", msg.Text, "this is a reply")
	}
}

// ---------------------------------------------------------------------------
// 5. TestDiscordAdapter_Normalize_WithAttachment
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Normalize_WithAttachment(t *testing.T) {
	a, _ := newTestAdapter(t)

	raw := []byte(`{
		"id": "777",
		"channel_id": "CHAN-003",
		"content": "check this file",
		"timestamp": "2026-04-05T14:00:00Z",
		"author": {"id": "U3", "username": "charlie"},
		"attachments": [
			{
				"url": "https://cdn.discord.com/files/report.pdf",
				"filename": "report.pdf",
				"size": 204800,
				"content_type": "application/pdf"
			},
			{
				"url": "https://cdn.discord.com/files/photo.png",
				"filename": "photo.png",
				"size": 1024,
				"content_type": "image/png"
			}
		]
	}`)

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize attachment: %v", err)
	}

	if len(msg.Attachments) != 2 {
		t.Fatalf("Attachments length = %d, want 2", len(msg.Attachments))
	}

	pdf := msg.Attachments[0]
	if pdf.URL != "https://cdn.discord.com/files/report.pdf" {
		t.Errorf("pdf URL = %q", pdf.URL)
	}
	if pdf.Name != "report.pdf" {
		t.Errorf("pdf Name = %q, want %q", pdf.Name, "report.pdf")
	}
	if pdf.Size != 204800 {
		t.Errorf("pdf Size = %d, want 204800", pdf.Size)
	}
	if pdf.MimeType != "application/pdf" {
		t.Errorf("pdf MimeType = %q, want %q", pdf.MimeType, "application/pdf")
	}
	if pdf.Type != "file" {
		t.Errorf("pdf Type = %q, want %q", pdf.Type, "file")
	}

	img := msg.Attachments[1]
	if img.Type != "image" {
		t.Errorf("img Type = %q, want %q", img.Type, "image")
	}
}

// ---------------------------------------------------------------------------
// 6. TestDiscordAdapter_VerifySignature_Valid
// ---------------------------------------------------------------------------

func TestDiscordAdapter_VerifySignature_Valid(t *testing.T) {
	a, priv := newTestAdapter(t)

	body := []byte(`{"type":1}`)
	req := signedRequest(t, body, priv)

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("VerifySignature valid: unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. TestDiscordAdapter_VerifySignature_Invalid
// ---------------------------------------------------------------------------

func TestDiscordAdapter_VerifySignature_Invalid(t *testing.T) {
	a, _ := newTestAdapter(t) // adapter has its own public key

	// Sign with a *different* private key — the adapter should reject it.
	_, wrongPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}

	body := []byte(`{"type":1}`)
	req := signedRequest(t, body, wrongPriv)

	if err := a.VerifySignature(req); err == nil {
		t.Error("VerifySignature invalid: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// 8. TestDiscordAdapter_HandleWebhook_Ping
// ---------------------------------------------------------------------------

func TestDiscordAdapter_HandleWebhook_Ping(t *testing.T) {
	a, priv := newTestAdapter(t)

	pingBody, _ := json.Marshal(map[string]int{"type": 1})
	req := signedRequest(t, pingBody, priv)
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleWebhook PING: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode PING response: %v", err)
	}
	if resp["type"] != 1 {
		t.Errorf("PING response type = %d, want 1", resp["type"])
	}
}

// ---------------------------------------------------------------------------
// 9. TestDiscordAdapter_HandleWebhook_InvalidSignature
// ---------------------------------------------------------------------------

func TestDiscordAdapter_HandleWebhook_InvalidSignature(t *testing.T) {
	a, _ := newTestAdapter(t)

	// Request with no signature headers at all.
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"type":1}`))
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 10. TestAttachmentType helper
// ---------------------------------------------------------------------------

func TestAttachmentType(t *testing.T) {
	cases := []struct {
		mime string
		want string
	}{
		{"image/png", "image"},
		{"video/mp4", "video"},
		{"audio/mpeg", "audio"},
		{"application/pdf", "file"},
		{"", "file"},
	}
	for _, c := range cases {
		got := attachmentType(c.mime)
		if got != c.want {
			t.Errorf("attachmentType(%q) = %q, want %q", c.mime, got, c.want)
		}
	}
}

// Ensure io is used (needed for io.Reader in signedRequest helper).
var _ io.Reader = (*byteSliceReader)(nil)
