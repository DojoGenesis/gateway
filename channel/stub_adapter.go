package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StubAdapter implements WebhookAdapter for testing. It normalizes any JSON
// payload containing a "text" field into a ChannelMessage and records all
// sent messages in an inspectable slice.
type StubAdapter struct {
	mu   sync.Mutex
	sent []*ChannelMessage
}

// Name returns "stub".
func (s *StubAdapter) Name() string {
	return "stub"
}

// Normalize parses raw JSON into a ChannelMessage. The payload must contain
// at least a "text" field. Optional fields: "user_id", "user_name",
// "channel_id", "reply_to", "thread_id".
func (s *StubAdapter) Normalize(raw []byte) (*ChannelMessage, error) {
	var payload struct {
		Text      string `json:"text"`
		UserID    string `json:"user_id"`
		UserName  string `json:"user_name"`
		ChannelID string `json:"channel_id"`
		ReplyTo   string `json:"reply_to"`
		ThreadID  string `json:"thread_id"`
	}

	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("channel: stub normalize failed: %w", err)
	}

	if payload.Text == "" {
		return nil, fmt.Errorf("channel: stub normalize: missing text field")
	}

	return &ChannelMessage{
		ID:        uuid.New().String(),
		Platform:  "stub",
		ChannelID: payload.ChannelID,
		UserID:    payload.UserID,
		UserName:  payload.UserName,
		Text:      payload.Text,
		Timestamp: time.Now().UTC(),
		ReplyTo:   payload.ReplyTo,
		ThreadID:  payload.ThreadID,
	}, nil
}

// Send records the message in the internal sent slice for test inspection.
func (s *StubAdapter) Send(_ context.Context, msg *ChannelMessage) error {
	s.mu.Lock()
	s.sent = append(s.sent, msg)
	s.mu.Unlock()
	return nil
}

// Capabilities returns a minimal capability set for stub testing.
func (s *StubAdapter) Capabilities() AdapterCapabilities {
	return AdapterCapabilities{
		SupportsThreads:     false,
		SupportsReactions:   false,
		SupportsAttachments: false,
		SupportsEdits:       false,
		MaxMessageLength:    4096,
	}
}

// HandleWebhook is a no-op for the stub — the gateway drives normalization.
func (s *StubAdapter) HandleWebhook(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// VerifySignature always returns nil for the stub adapter.
func (s *StubAdapter) VerifySignature(_ *http.Request) error {
	return nil
}

// Sent returns a copy of all messages recorded by Send.
func (s *StubAdapter) Sent() []*ChannelMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*ChannelMessage, len(s.sent))
	copy(out, s.sent)
	return out
}
