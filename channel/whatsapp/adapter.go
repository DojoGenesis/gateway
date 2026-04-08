package whatsapp

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
	"strconv"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

const (
	platform          = "whatsapp"
	signatureHeader   = "X-Hub-Signature-256"
	signaturePrefix   = "sha256="
)

// WhatsAppAdapter implements both channel.WebhookAdapter and channel.ActorAdapter
// for the WhatsApp Cloud API (On-Premises sunset Oct 2025).
//
// As a WebhookAdapter it handles webhook verification challenges (GET) and
// inbound message payloads (POST). As an ActorAdapter it manages persistent
// session state for supervised actor patterns (ADR-014).
//
// Construction: use NewWhatsAppAdapter — do not create the struct directly.
type WhatsAppAdapter struct {
	cfg        WhatsAppConfig
	httpClient *http.Client

	// actor state (protected by mu)
	mu          sync.RWMutex
	connected   bool
	msgHandler  func(*channel.ChannelMessage)
}

// NewWhatsAppAdapter returns a WhatsAppAdapter configured with the given config.
// The adapter uses http.DefaultClient for outbound API calls; use
// NewWhatsAppAdapterWithClient for test overrides.
func NewWhatsAppAdapter(cfg WhatsAppConfig) *WhatsAppAdapter {
	return &WhatsAppAdapter{
		cfg:        cfg,
		httpClient: http.DefaultClient,
	}
}

// NewWhatsAppAdapterWithClient returns a WhatsAppAdapter that uses the provided
// http.Client for outbound API calls. Intended for unit testing.
func NewWhatsAppAdapterWithClient(cfg WhatsAppConfig, client *http.Client) *WhatsAppAdapter {
	return &WhatsAppAdapter{
		cfg:        cfg,
		httpClient: client,
	}
}

// ---------------------------------------------------------------------------
// channel.ChannelAdapter
// ---------------------------------------------------------------------------

// Name returns the platform identifier "whatsapp".
func (a *WhatsAppAdapter) Name() string {
	return platform
}

// Capabilities returns the feature set supported by WhatsApp Cloud API.
func (a *WhatsAppAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     false,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       false,
		MaxMessageLength:    4096,
	}
}

// Normalize parses a raw WhatsApp webhook payload JSON and returns a
// ChannelMessage for the first inbound message found.
// Returns an error if the payload is malformed or contains no messages.
func (a *WhatsAppAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("whatsapp: normalize: invalid JSON: %w", err)
	}

	// Walk the nested structure to find the first message.
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				return a.normalizeMessage(msg, change.Value), nil
			}
		}
	}

	return nil, fmt.Errorf("whatsapp: normalize: payload contains no messages")
}

// normalizeMessage converts a WhatsApp Message into a ChannelMessage.
func (a *WhatsAppAdapter) normalizeMessage(msg Message, cv ChangeValue) *channel.ChannelMessage {
	ts := parseTimestamp(msg.Timestamp)

	cm := &channel.ChannelMessage{
		ID:        msg.ID,
		Platform:  platform,
		ChannelID: cv.Metadata.PhoneNumberID,
		UserID:    msg.From,
		Timestamp: ts,
		Metadata:  map[string]interface{}{},
	}

	// Resolve display name from contacts list.
	for _, c := range cv.Contacts {
		if c.WaID == msg.From {
			cm.UserName = c.Profile.Name
			break
		}
	}

	// Extract text body.
	if msg.Text != nil {
		cm.Text = msg.Text.Body
	}

	// Handle reply context.
	if msg.Context != nil {
		cm.ReplyTo = msg.Context.ID
	}

	// Collect attachments.
	switch msg.Type {
	case "image":
		if msg.Image != nil {
			att := channel.Attachment{
				Type:     "image",
				URL:      msg.Image.ID,
				MimeType: msg.Image.MimeType,
				Name:     msg.Image.Caption,
			}
			cm.Attachments = append(cm.Attachments, att)
		}
	case "document":
		if msg.Document != nil {
			att := channel.Attachment{
				Type:     "file",
				URL:      msg.Document.ID,
				MimeType: msg.Document.MimeType,
				Name:     msg.Document.Filename,
			}
			cm.Attachments = append(cm.Attachments, att)
		}
	}

	return cm
}

// Send delivers a ChannelMessage to a WhatsApp phone number via the Cloud API.
// The message.ChannelID is used as the recipient phone number (to field).
func (a *WhatsAppAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if msg == nil {
		return fmt.Errorf("whatsapp: send: nil message")
	}

	payload := sendMessageRequest{
		MessagingProduct: "whatsapp",
		To:               msg.ChannelID,
		Type:             "text",
		Text:             &sendTextBody{Body: msg.Text},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("whatsapp: send: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", a.cfg.apiURL(), a.cfg.PhoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp: send: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.AccessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send: HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp: send: API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// channel.WebhookAdapter
// ---------------------------------------------------------------------------

// VerifySignature validates the X-Hub-Signature-256 header using HMAC-SHA256
// of the raw request body and the configured AppSecret.
// If AppSecret is empty, verification is skipped and nil is returned.
func (a *WhatsAppAdapter) VerifySignature(r *http.Request) error {
	if a.cfg.AppSecret == "" {
		return nil
	}

	provided := r.Header.Get(signatureHeader)
	if provided == "" {
		return fmt.Errorf("whatsapp: missing %s header", signatureHeader)
	}

	// Read body bytes; the caller must have buffered the body or we read it once.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("whatsapp: read body for signature: %w", err)
	}
	// Restore the body so downstream handlers can re-read it.
	r.Body = io.NopCloser(bytes.NewReader(body))

	mac := hmac.New(sha256.New, []byte(a.cfg.AppSecret))
	mac.Write(body)
	expected := signaturePrefix + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return fmt.Errorf("whatsapp: invalid signature")
	}
	return nil
}

// HandleWebhook processes inbound WhatsApp webhook requests.
//
//   - GET  requests handle the webhook verification challenge:
//     validates hub.verify_token and responds with hub.challenge.
//   - POST requests handle inbound message payloads:
//     verifies signature, normalizes the payload, and invokes any
//     registered OnMessage handler.
func (a *WhatsAppAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleVerification(w, r)
	case http.MethodPost:
		a.handleMessage(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVerification responds to the GET webhook verification challenge.
func (a *WhatsAppAdapter) handleVerification(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" {
		http.Error(w, "invalid hub.mode", http.StatusForbidden)
		return
	}
	if token != a.cfg.VerifyToken {
		http.Error(w, "invalid verify token", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, challenge)
}

// handleMessage processes a POST webhook payload.
func (a *WhatsAppAdapter) handleMessage(w http.ResponseWriter, r *http.Request) {
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

	msg, err := a.Normalize(raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Dispatch to registered handler if present.
	a.mu.RLock()
	handler := a.msgHandler
	a.mu.RUnlock()

	if handler != nil {
		handler(msg)
	}

	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// channel.ActorAdapter
// ---------------------------------------------------------------------------

// Connect establishes a persistent session for the WhatsApp Cloud API.
// For the Cloud API (push-based), this stores the credentials and marks
// the adapter as connected for session lifecycle tracking.
func (a *WhatsAppAdapter) Connect(ctx context.Context, creds channel.Credentials) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Apply credentials from the Credentials envelope.
	if creds.Token != "" {
		a.cfg.AccessToken = creds.Token
	}
	if creds.Secret != "" {
		a.cfg.AppSecret = creds.Secret
	}
	if phoneID, ok := creds.Extra["phone_number_id"]; ok {
		a.cfg.PhoneNumberID = phoneID
	}

	a.connected = true
	return nil
}

// Disconnect clears the session state and marks the adapter as disconnected.
func (a *WhatsAppAdapter) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.connected = false
	a.msgHandler = nil
	return nil
}

// OnMessage registers a handler that is called for each inbound ChannelMessage.
// Replaces any previously registered handler. Thread-safe.
func (a *WhatsAppAdapter) OnMessage(handler func(*channel.ChannelMessage)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.msgHandler = handler
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// parseTimestamp converts a Unix timestamp string to time.Time (UTC).
// Returns the zero value if the string is empty or not a valid integer.
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	unix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}
