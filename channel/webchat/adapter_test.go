package webchat

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. TestWebChatAdapter_Name
// ---------------------------------------------------------------------------

func TestWebChatAdapter_Name(t *testing.T) {
	a := NewWebChatAdapter("test-token")
	if got := a.Name(); got != "webchat" {
		t.Errorf("Name() = %q, want %q", got, "webchat")
	}
}

// ---------------------------------------------------------------------------
// 2. TestWebChatAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestWebChatAdapter_Capabilities(t *testing.T) {
	a := NewWebChatAdapter("test-token")
	caps := a.Capabilities()

	if !caps.SupportsThreads {
		t.Error("SupportsThreads should be true for WebChat")
	}
	if caps.SupportsReactions {
		t.Error("SupportsReactions should be false for WebChat")
	}
	if caps.SupportsAttachments {
		t.Error("SupportsAttachments should be false for WebChat")
	}
	if caps.SupportsEdits {
		t.Error("SupportsEdits should be false for WebChat")
	}
	if caps.MaxMessageLength != 10000 {
		t.Errorf("MaxMessageLength = %d, want 10000", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestWebChatAdapter_Normalize
// ---------------------------------------------------------------------------

func TestWebChatAdapter_Normalize(t *testing.T) {
	a := NewWebChatAdapter("")

	raw := []byte(`{"text":"hello webchat","user_id":"user-abc","session_id":"sess-xyz"}`)

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "webchat")
	assertField(t, "ChannelID", msg.ChannelID, "sess-xyz")
	assertField(t, "UserID", msg.UserID, "user-abc")
	assertField(t, "Text", msg.Text, "hello webchat")

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// ---------------------------------------------------------------------------
// 4. TestWebChatAdapter_VerifySignature_Valid
// ---------------------------------------------------------------------------

func TestWebChatAdapter_VerifySignature_Valid(t *testing.T) {
	a := NewWebChatAdapter("my-secret-token")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("expected valid signature to pass, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5. TestWebChatAdapter_VerifySignature_Invalid
// ---------------------------------------------------------------------------

func TestWebChatAdapter_VerifySignature_Invalid(t *testing.T) {
	a := NewWebChatAdapter("correct-token")

	t.Run("wrong_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for wrong token, got nil")
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", nil)

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing header, got nil")
		}
	})

	t.Run("not_bearer_scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", nil)
		req.Header.Set("Authorization", "Basic correct-token")

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for non-Bearer scheme, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 6. TestWebChatAdapter_HandleWebhook
// ---------------------------------------------------------------------------

func TestWebChatAdapter_HandleWebhook(t *testing.T) {
	a := NewWebChatAdapter("test-token")

	body := bytes.NewBufferString(`{"text":"hello","user_id":"u1","session_id":"s1"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/webchat", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook status = %d, want 200", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 7. TestWidget_ContainsHTML
// ---------------------------------------------------------------------------

func TestWidget_ContainsHTML(t *testing.T) {
	html := Widget("https://gateway.example.com")

	checks := []string{
		"<!DOCTYPE html>",
		"dojo-chat-bubble",
		"dojo-chat-panel",
		"dojo-chat-messages",
		"dojo-chat-input",
		"dojo-chat-send",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("Widget HTML should contain %q", check)
		}
	}
}

// ---------------------------------------------------------------------------
// 8. TestWidget_ContainsGatewayURL
// ---------------------------------------------------------------------------

func TestWidget_ContainsGatewayURL(t *testing.T) {
	gatewayURL := "https://gateway.example.com"
	html := Widget(gatewayURL)

	expectedEndpoint := gatewayURL + "/webhooks/webchat"
	if !strings.Contains(html, expectedEndpoint) {
		t.Errorf("Widget HTML should contain endpoint %q", expectedEndpoint)
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
