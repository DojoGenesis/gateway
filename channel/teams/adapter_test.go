package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/channel"
)

// ---------------------------------------------------------------------------
// 1. TestTeamsAdapter_Name
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Name(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")
	if got := a.Name(); got != "teams" {
		t.Errorf("Name() = %q, want %q", got, "teams")
	}
}

// ---------------------------------------------------------------------------
// 2. TestTeamsAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Capabilities(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")
	caps := a.Capabilities()

	if !caps.SupportsThreads {
		t.Error("SupportsThreads should be true for Teams")
	}
	if !caps.SupportsReactions {
		t.Error("SupportsReactions should be true for Teams")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true for Teams")
	}
	if caps.SupportsEdits {
		t.Error("SupportsEdits should be false for Teams")
	}
	if caps.MaxMessageLength != 28000 {
		t.Errorf("MaxMessageLength = %d, want 28000", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestTeamsAdapter_Normalize_Message
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Normalize_Message(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")

	act := Activity{
		Type: "message",
		ID:   "msg-001",
		From: ChannelAccount{ID: "user-abc", Name: "Alice"},
		Conversation: ConversationAccount{ID: "conv-xyz", Name: "General"},
		Text:       "hello teams",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
		Timestamp:  "2024-04-05T12:00:00Z",
	}

	raw, _ := json.Marshal(act)
	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "teams")
	assertField(t, "ID", msg.ID, "msg-001")
	assertField(t, "ChannelID", msg.ChannelID, "conv-xyz")
	assertField(t, "UserID", msg.UserID, "user-abc")
	assertField(t, "UserName", msg.UserName, "Alice")
	assertField(t, "Text", msg.Text, "hello teams")
	assertField(t, "ThreadID", msg.ThreadID, "")

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(msg.Attachments))
	}
}

// ---------------------------------------------------------------------------
// 4. TestTeamsAdapter_Normalize_Reply
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Normalize_Reply(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")

	act := Activity{
		Type:      "message",
		ID:        "msg-002",
		From:      ChannelAccount{ID: "user-bob", Name: "Bob"},
		Conversation: ConversationAccount{ID: "conv-xyz"},
		Text:      "replying now",
		ReplyToID: "msg-001",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
	}

	raw, _ := json.Marshal(act)
	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "ThreadID", msg.ThreadID, "msg-001")
	assertField(t, "Text", msg.Text, "replying now")
}

// ---------------------------------------------------------------------------
// 5. TestTeamsAdapter_VerifySignature
// ---------------------------------------------------------------------------

func TestTeamsAdapter_VerifySignature(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")

	// Build a minimal valid JWT structure: three base64url-encoded parts.
	header  := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
	payload := "eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QiLCJpYXQiOjE1MTYyMzkwMjJ9"
	sig     := "c2lnbmF0dXJl" // "signature" in base64url
	token   := header + "." + payload + "." + sig

	t.Run("valid_structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		// Phase 0: structural check only — should not error on well-formed JWT.
		if err := a.VerifySignature(req); err != nil {
			t.Errorf("expected valid JWT structure to pass, got error: %v", err)
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing Authorization header, got nil")
		}
	})

	t.Run("malformed_jwt", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer notajwt")

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for malformed JWT, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 6. TestTeamsAdapter_HandleWebhook
// ---------------------------------------------------------------------------

func TestTeamsAdapter_HandleWebhook(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")

	act := Activity{
		Type: "message",
		ID:   "msg-003",
		From: ChannelAccount{ID: "user-charlie", Name: "Charlie"},
		Conversation: ConversationAccount{ID: "conv-abc"},
		Text:       "webhook round trip",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
	}

	raw, _ := json.Marshal(act)

	// Build valid JWT token for the header.
	header  := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
	payload := "eyJzdWIiOiJ0ZXN0IiwiaWF0IjoxNTE2MjM5MDIyfQ"
	sig     := "c2lnbmF0dXJl"
	token   := header + "." + payload + "." + sig

	req := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook status = %d, want 200", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 7. TestTeamsAdapter_Send
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Send(t *testing.T) {
	var capturedBody []byte
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"reply-001"}`))
	}))
	defer srv.Close()

	a := NewTeamsAdapterWithClient("my-bot-token", &http.Client{
		Transport: &urlRewriteTransport{
			base:       http.DefaultTransport,
			serverAddr: srv.URL,
		},
	})

	msg := &channel.ChannelMessage{
		Platform:  "teams",
		ChannelID: "conv-xyz",
		Text:      "hello from send test",
		Metadata: map[string]interface{}{
			"service_url": srv.URL,
		},
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Verify Bearer auth was set.
	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("Authorization header = %q, expected Bearer token", capturedAuth)
	}

	// Verify the activity body.
	var act Activity
	if err := json.Unmarshal(capturedBody, &act); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if act.Text != "hello from send test" {
		t.Errorf("activity.text = %q, want %q", act.Text, "hello from send test")
	}
	if act.Type != "message" {
		t.Errorf("activity.type = %q, want %q", act.Type, "message")
	}
}

// ---------------------------------------------------------------------------
// 8. TestTeamsAdapter_Send_MissingServiceURL
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Send_MissingServiceURL(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token")

	msg := &channel.ChannelMessage{
		Platform:  "teams",
		ChannelID: "conv-xyz",
		Text:      "no service url",
	}

	if err := a.Send(context.Background(), msg); err == nil {
		t.Error("expected error when service_url missing from metadata, got nil")
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertField(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// urlRewriteTransport redirects all requests to the given mock server.
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
