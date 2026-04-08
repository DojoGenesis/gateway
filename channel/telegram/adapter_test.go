package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// 1. TestTelegramAdapter_Name
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Name(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")
	if got := a.Name(); got != "telegram" {
		t.Errorf("Name() = %q, want %q", got, "telegram")
	}
}

// ---------------------------------------------------------------------------
// 2. TestTelegramAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Capabilities(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")
	caps := a.Capabilities()

	if caps.SupportsThreads {
		t.Error("SupportsThreads should be false for Telegram")
	}
	if !caps.SupportsReactions {
		t.Error("SupportsReactions should be true for Telegram")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true for Telegram")
	}
	if !caps.SupportsEdits {
		t.Error("SupportsEdits should be true for Telegram")
	}
	if caps.MaxMessageLength != 4096 {
		t.Errorf("MaxMessageLength = %d, want 4096", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestTelegramAdapter_Normalize_TextMessage
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_TextMessage(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")

	raw := mustMarshal(t, Update{
		UpdateID: 100,
		Message: &Message{
			MessageID: 42,
			From:      &User{ID: 111, Username: "alice", FirstName: "Alice"},
			Chat:      &Chat{ID: 999, Type: "private"},
			Date:      1712332800, // 2024-04-05 12:00:00 UTC
			Text:      "hello telegram",
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "telegram")
	assertField(t, "ID", msg.ID, "42")
	assertField(t, "ChannelID", msg.ChannelID, "999")
	assertField(t, "UserID", msg.UserID, "111")
	assertField(t, "UserName", msg.UserName, "alice")
	assertField(t, "Text", msg.Text, "hello telegram")
	assertField(t, "ReplyTo", msg.ReplyTo, "")

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(msg.Attachments))
	}
}

// ---------------------------------------------------------------------------
// 4. TestTelegramAdapter_Normalize_Reply
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_Reply(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")

	raw := mustMarshal(t, Update{
		UpdateID: 200,
		Message: &Message{
			MessageID: 55,
			From:      &User{ID: 222, Username: "bob", FirstName: "Bob"},
			Chat:      &Chat{ID: 888, Type: "group"},
			Date:      1712332800,
			Text:      "replying now",
			ReplyToMessage: &Message{
				MessageID: 50,
				Chat:      &Chat{ID: 888, Type: "group"},
				Date:      1712332700,
				Text:      "original message",
			},
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "ReplyTo", msg.ReplyTo, "50")
	assertField(t, "Text", msg.Text, "replying now")
}

// ---------------------------------------------------------------------------
// 5. TestTelegramAdapter_Normalize_WithDocument
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_WithDocument(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")

	raw := mustMarshal(t, Update{
		UpdateID: 300,
		Message: &Message{
			MessageID: 66,
			From:      &User{ID: 333, Username: "charlie", FirstName: "Charlie"},
			Chat:      &Chat{ID: 777, Type: "private"},
			Date:      1712332800,
			Document: &Document{
				FileID:       "BQACAgIAAxkB",
				FileUniqueID: "unique-doc-1",
				FileName:     "report.pdf",
				MimeType:     "application/pdf",
				FileSize:     204800,
			},
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	assertField(t, "Attachment.Type", att.Type, "file")
	assertField(t, "Attachment.Name", att.Name, "report.pdf")
	assertField(t, "Attachment.MimeType", att.MimeType, "application/pdf")
	assertField(t, "Attachment.URL", att.URL, "BQACAgIAAxkB")

	if att.Size != 204800 {
		t.Errorf("Attachment.Size = %d, want 204800", att.Size)
	}
}

// ---------------------------------------------------------------------------
// 6. TestTelegramAdapter_Normalize_WithPhoto
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_WithPhoto(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")

	raw := mustMarshal(t, Update{
		UpdateID: 400,
		Message: &Message{
			MessageID: 77,
			From:      &User{ID: 444, Username: "diana", FirstName: "Diana"},
			Chat:      &Chat{ID: 666, Type: "private"},
			Date:      1712332800,
			Photo: []PhotoSize{
				{FileID: "small-id", FileUniqueID: "u1", Width: 90, Height: 90, FileSize: 1024},
				{FileID: "medium-id", FileUniqueID: "u2", Width: 320, Height: 320, FileSize: 8192},
				{FileID: "large-id", FileUniqueID: "u3", Width: 800, Height: 800, FileSize: 65536},
			},
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("expected 1 attachment (largest photo), got %d", len(msg.Attachments))
	}

	att := msg.Attachments[0]
	assertField(t, "Attachment.Type", att.Type, "image")
	// Should be the largest (last) photo size.
	assertField(t, "Attachment.URL", att.URL, "large-id")

	if att.Size != 65536 {
		t.Errorf("Attachment.Size = %d, want 65536", att.Size)
	}
}

// ---------------------------------------------------------------------------
// 7. TestTelegramAdapter_VerifySignature_Valid
// ---------------------------------------------------------------------------

func TestTelegramAdapter_VerifySignature_Valid(t *testing.T) {
	a := NewTelegramAdapter("test-token", "my-webhook-secret")

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set(secretTokenHeader, "my-webhook-secret")

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("expected valid signature to pass, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 8. TestTelegramAdapter_VerifySignature_Invalid
// ---------------------------------------------------------------------------

func TestTelegramAdapter_VerifySignature_Invalid(t *testing.T) {
	a := NewTelegramAdapter("test-token", "correct-secret")

	t.Run("wrong_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.Header.Set(secretTokenHeader, "wrong-secret")

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for wrong token, got nil")
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing header, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 9. TestTelegramAdapter_HandleWebhook — full request cycle
// ---------------------------------------------------------------------------

func TestTelegramAdapter_HandleWebhook(t *testing.T) {
	a := NewTelegramAdapter("test-token", "hook-secret")

	raw := mustMarshal(t, Update{
		UpdateID: 500,
		Message: &Message{
			MessageID: 88,
			From:      &User{ID: 555, Username: "eve", FirstName: "Eve"},
			Chat:      &Chat{ID: 123, Type: "private"},
			Date:      1712332800,
			Text:      "webhook round trip",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(secretTokenHeader, "hook-secret")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook status = %d, want 200", rec.Code)
	}
}

func TestTelegramAdapter_HandleWebhook_InvalidSignature(t *testing.T) {
	a := NewTelegramAdapter("test-token", "correct-secret")

	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram", strings.NewReader("{}"))
	req.Header.Set(secretTokenHeader, "wrong-secret")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestTelegramAdapter_HandleWebhook_BadJSON(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTelegramAdapter_HandleWebhook_NoMessage(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{UpdateID: 999})
	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewReader(raw))
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for update without message, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 10. TestTelegramAdapter_Send — mock HTTP server, verify POST to sendMessage
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Send(t *testing.T) {
	var capturedBody []byte
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := &TelegramAdapter{
		token:  "mytoken",
		secret: "",
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{
				base:       http.DefaultTransport,
				serverAddr: srv.URL,
			},
		},
	}

	msg := &channel.ChannelMessage{
		ID:        "88",
		Platform:  "telegram",
		ChannelID: "123456",
		Text:      "hello from send test",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	expectedPath := "/botmytoken/sendMessage"
	if capturedPath != expectedPath {
		t.Errorf("path = %q, want %q", capturedPath, expectedPath)
	}

	var payload sendMessageRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if payload.ChatID != "123456" {
		t.Errorf("chat_id = %q, want %q", payload.ChatID, "123456")
	}
	if payload.Text != "hello from send test" {
		t.Errorf("text = %q, want %q", payload.Text, "hello from send test")
	}
	if payload.ReplyToMessageID != 0 {
		t.Errorf("reply_to_message_id = %d, want 0", payload.ReplyToMessageID)
	}
}

func TestTelegramAdapter_Send_WithReply(t *testing.T) {
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := &TelegramAdapter{
		token:  "mytoken",
		secret: "",
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{
				base:       http.DefaultTransport,
				serverAddr: srv.URL,
			},
		},
	}

	msg := &channel.ChannelMessage{
		ID:        "89",
		Platform:  "telegram",
		ChannelID: "123456",
		Text:      "replying",
		ReplyTo:   "50",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	var payload sendMessageRequest
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ReplyToMessageID != 50 {
		t.Errorf("reply_to_message_id = %d, want 50", payload.ReplyToMessageID)
	}
}

func TestTelegramAdapter_Send_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"description":"Bad Request"}`))
	}))
	defer srv.Close()

	a := &TelegramAdapter{
		token:  "mytoken",
		secret: "",
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{
				base:       http.DefaultTransport,
				serverAddr: srv.URL,
			},
		},
	}

	msg := &channel.ChannelMessage{
		ChannelID: "123",
		Text:      "test",
	}

	err := a.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for API 400 response, got nil")
	}
}

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

// urlRewriteTransport redirects all requests to the given mock server address
// while preserving the request path. This allows testing Send() against a
// local httptest.Server without modifying the adapter's URL construction.
type urlRewriteTransport struct {
	base       http.RoundTripper
	serverAddr string
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original.
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(t.serverAddr, "http://")
	return t.base.RoundTrip(cloned)
}
