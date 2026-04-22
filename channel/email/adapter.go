package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/channel"
	"github.com/google/uuid"
)

// sendGridMailSendURL is the SendGrid v3 transactional mail endpoint.
const sendGridMailSendURL = "https://api.sendgrid.com/v3/mail/send"

// EmailAdapter implements channel.WebhookAdapter for SendGrid Inbound Parse.
// Each inbound email arrives as a multipart/form-data POST to the registered
// webhook URL. Outbound replies are delivered via the SendGrid v3 mail/send API.
type EmailAdapter struct {
	cfg        EmailConfig
	httpClient *http.Client
	// sendURL allows tests to override the SendGrid endpoint.
	sendURL string
}

// New returns an EmailAdapter configured with cfg. If cfg.SendGridAPIKey is
// empty, Send will return an error on every call.
func New(cfg EmailConfig) *EmailAdapter {
	return &EmailAdapter{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		sendURL:    sendGridMailSendURL,
	}
}

// Name satisfies channel.ChannelAdapter. Returns "email".
func (a *EmailAdapter) Name() string {
	return "email"
}

// Capabilities satisfies channel.ChannelAdapter.
// Email supports threads (via In-Reply-To) and attachments, but not
// reactions or message edits. There is no maximum message length.
func (a *EmailAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   false,
		SupportsAttachments: true,
		SupportsEdits:       false,
		MaxMessageLength:    0,
	}
}

// VerifySignature satisfies channel.WebhookAdapter. It checks for the
// X-Webhook-Secret header and compares it to the configured WebhookSecret.
// If WebhookSecret is empty the check is skipped (useful for development).
func (a *EmailAdapter) VerifySignature(r *http.Request) error {
	if a.cfg.WebhookSecret == "" {
		return nil
	}
	got := r.Header.Get("X-Webhook-Secret")
	if got == "" {
		return fmt.Errorf("email: missing X-Webhook-Secret header")
	}
	if got != a.cfg.WebhookSecret {
		return fmt.Errorf("email: invalid webhook secret")
	}
	return nil
}

// HandleWebhook satisfies channel.WebhookAdapter. It parses the multipart
// form from SendGrid Inbound Parse, normalizes it into a ChannelMessage, and
// writes a 200 OK (SendGrid retries on any non-2xx response).
//
// The normalized message is embedded in the response body as JSON so that
// a caller (e.g. the WebhookGateway) can read and route it downstream.
func (a *EmailAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if err := a.VerifySignature(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Limit the request body to 1 MiB before parsing to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	// ParseMultipartForm reads the body up to the given limit.
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, fmt.Sprintf("email: failed to parse multipart form: %v", err), http.StatusBadRequest)
		return
	}

	raw, err := buildRawFromForm(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("email: %v", err), http.StatusBadRequest)
		return
	}

	msg, err := a.Normalize(raw)
	if err != nil {
		http.Error(w, fmt.Sprintf("email: normalize failed: %v", err), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(msg)
}

// Normalize satisfies channel.ChannelAdapter. It accepts a JSON-encoded
// InboundEmail (as produced by buildRawFromForm) and converts it into a
// ChannelMessage.
//
// Field mapping:
//   - from          → UserID (raw address), UserName (display name)
//   - to            → ChannelID
//   - subject       → Metadata["subject"]
//   - text          → Text  (preferred; falls back to html when empty)
//   - envelope      → Metadata["envelope"]
//   - Message-ID    → ID     (parsed from headers; UUID fallback)
//   - In-Reply-To   → ThreadID
func (a *EmailAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var email InboundEmail
	if err := json.Unmarshal(raw, &email); err != nil {
		return nil, fmt.Errorf("email: normalize: unmarshal failed: %w", err)
	}

	// Parse "Name <addr>" → display name + address.
	userName, userID, err := parseAddress(email.From)
	if err != nil {
		// Fall back to using the raw string if RFC 5322 parse fails.
		userID = email.From
		userName = email.From
	}

	// Prefer plain text; fall back to HTML when text is absent.
	text := email.Text
	if text == "" {
		text = email.HTML
	}

	// Extract Message-ID and In-Reply-To from the raw headers block.
	messageID, inReplyTo := parseHeaders(email.Headers)
	if messageID == "" {
		messageID = uuid.New().String()
	}

	metadata := map[string]interface{}{
		"subject": email.Subject,
	}
	if email.Envelope != "" {
		metadata["envelope"] = email.Envelope
	}
	if email.HTML != "" {
		metadata["html"] = email.HTML
	}

	return &channel.ChannelMessage{
		ID:        messageID,
		Platform:  "email",
		ChannelID: email.To,
		UserID:    userID,
		UserName:  userName,
		Text:      text,
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
		ThreadID:  inReplyTo,
	}, nil
}

// Send satisfies channel.ChannelAdapter. It delivers msg via the SendGrid v3
// mail/send API. ChannelID is used as the recipient address; the subject is
// read from Metadata["subject"] when present.
func (a *EmailAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if a.cfg.SendGridAPIKey == "" {
		return fmt.Errorf("email: SendGridAPIKey is not configured")
	}

	subject := "Re: (no subject)"
	if v, ok := msg.Metadata["subject"].(string); ok && v != "" {
		subject = v
	}

	type emailAddress struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}
	type content struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type personalization struct {
		To []emailAddress `json:"to"`
	}
	type mailBody struct {
		Personalizations []personalization `json:"personalizations"`
		From             emailAddress      `json:"from"`
		Subject          string            `json:"subject"`
		Content          []content         `json:"content"`
	}

	body := mailBody{
		Personalizations: []personalization{
			{To: []emailAddress{{Email: msg.ChannelID}}},
		},
		From: emailAddress{
			Email: a.cfg.FromAddress,
			Name:  a.cfg.FromName,
		},
		Subject: subject,
		Content: []content{
			{Type: "text/plain", Value: msg.Text},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("email: send: marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.sendURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("email: send: build request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.SendGridAPIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: send: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email: send: SendGrid returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildRawFromForm converts the parsed multipart form values from an
// http.Request into a JSON-encoded InboundEmail.
func buildRawFromForm(r *http.Request) ([]byte, error) {
	email := InboundEmail{
		From:     r.FormValue("from"),
		To:       r.FormValue("to"),
		Subject:  r.FormValue("subject"),
		Text:     r.FormValue("text"),
		HTML:     r.FormValue("html"),
		Envelope: r.FormValue("envelope"),
		Headers:  r.FormValue("headers"),
	}
	return json.Marshal(email)
}

// parseAddress parses an RFC 5322 address string of the form
// "Display Name <addr@example.com>" and returns (displayName, address).
// If no display name is present, the address is returned for both values.
func parseAddress(raw string) (name, addr string, err error) {
	parsed, parseErr := mail.ParseAddress(raw)
	if parseErr != nil {
		return "", "", parseErr
	}
	name = parsed.Name
	addr = parsed.Address
	if name == "" {
		name = addr
	}
	return name, addr, nil
}

// parseHeaders scans the raw email headers block (newline-separated
// "Key: Value" pairs) and returns the Message-ID and In-Reply-To values.
func parseHeaders(raw string) (messageID, inReplyTo string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		switch strings.ToLower(key) {
		case "message-id":
			messageID = strings.Trim(val, "<>")
		case "in-reply-to":
			inReplyTo = strings.Trim(val, "<>")
		}
	}
	return messageID, inReplyTo
}
