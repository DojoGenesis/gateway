package webchat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/channel"
)

const (
	platform         = "webchat"
	maxMessageLength = 10000
)

// inboundPayload is the JSON body sent by the web chat widget.
type inboundPayload struct {
	Text      string `json:"text"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// WebChatAdapter implements channel.WebhookAdapter for the embedded web chat
// widget. It verifies inbound requests via Bearer token, normalizes JSON
// payloads into ChannelMessage envelopes, and uses a no-op Send since
// responses are delivered synchronously in HandleWebhook or via WebSocket.
//
// Construction: use NewWebChatAdapter — do not create the struct directly.
type WebChatAdapter struct {
	// token is the shared Bearer token checked on every inbound request.
	// If empty, signature verification is skipped (not recommended in
	// production).
	token string
}

// NewWebChatAdapter returns a WebChatAdapter configured with the given
// Bearer token. The token may be empty to disable verification.
func NewWebChatAdapter(token string) *WebChatAdapter {
	return &WebChatAdapter{token: token}
}

// Name returns the platform identifier "webchat".
func (a *WebChatAdapter) Name() string {
	return platform
}

// Capabilities returns the feature set supported by the web chat widget.
func (a *WebChatAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   false,
		SupportsAttachments: false,
		SupportsEdits:       false,
		MaxMessageLength:    maxMessageLength,
	}
}

// VerifySignature checks that the Authorization header contains the expected
// Bearer token. If no token is configured, verification is skipped.
func (a *WebChatAdapter) VerifySignature(r *http.Request) error {
	if a.token == "" {
		return nil
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("webchat: missing Authorization header")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fmt.Errorf("webchat: Authorization header must use Bearer scheme")
	}
	if parts[1] != a.token {
		return fmt.Errorf("webchat: invalid Bearer token")
	}
	return nil
}

// HandleWebhook processes an inbound web chat POST. It verifies the token,
// normalizes the JSON payload, and writes 200 OK. On error it writes the
// appropriate HTTP status.
func (a *WebChatAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if err := a.VerifySignature(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if _, err := a.Normalize(raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Normalize parses a JSON web chat payload and returns a ChannelMessage.
// The payload must contain at least a session_id field.
func (a *WebChatAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var p inboundPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("webchat: normalize: invalid JSON: %w", err)
	}
	if p.SessionID == "" {
		return nil, fmt.Errorf("webchat: normalize: missing session_id")
	}

	cm := &channel.ChannelMessage{
		Platform:  platform,
		ChannelID: p.SessionID,
		UserID:    p.UserID,
		Text:      p.Text,
		Timestamp: time.Now().UTC(),
	}
	return cm, nil
}

// Send is a no-op for the web chat adapter. Responses are delivered
// synchronously within HandleWebhook or via a WebSocket connection managed
// outside this adapter.
func (a *WebChatAdapter) Send(_ context.Context, _ *channel.ChannelMessage) error {
	return nil
}
