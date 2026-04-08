package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

const (
	apiBaseURL        = "https://api.telegram.org"
	secretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"
	platform          = "telegram"
)

// TelegramAdapter implements channel.WebhookAdapter for the Telegram Bot API.
// It verifies inbound webhook payloads using the secret_token mechanism,
// normalizes Telegram Update objects into ChannelMessage envelopes, and
// delivers outbound messages via the sendMessage endpoint.
//
// Construction: use NewTelegramAdapter — do not create the struct directly.
type TelegramAdapter struct {
	// token is the Telegram Bot API token (e.g. "123456:ABC-DEF...").
	token string

	// secret is the value registered with setWebhook's secret_token field.
	// Inbound requests must carry this value in X-Telegram-Bot-Api-Secret-Token.
	secret string

	// httpClient is used for all outbound API calls. Defaults to
	// http.DefaultClient; override in tests via NewTelegramAdapterWithClient.
	httpClient *http.Client
}

// NewTelegramAdapter returns a TelegramAdapter configured with the given bot
// token and webhook secret. The token must not be empty; the secret may be
// empty (signature verification is skipped when no secret is configured).
func NewTelegramAdapter(token, secret string) *TelegramAdapter {
	return &TelegramAdapter{
		token:      token,
		secret:     secret,
		httpClient: http.DefaultClient,
	}
}

// NewTelegramAdapterWithClient returns a TelegramAdapter that uses the
// provided http.Client for outbound API calls. Intended for unit testing.
func NewTelegramAdapterWithClient(token, secret string, client *http.Client) *TelegramAdapter {
	return &TelegramAdapter{
		token:      token,
		secret:     secret,
		httpClient: client,
	}
}

// Name returns the platform identifier "telegram".
func (a *TelegramAdapter) Name() string {
	return platform
}

// Capabilities returns the feature set supported by Telegram.
func (a *TelegramAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     false,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       true,
		MaxMessageLength:    4096,
	}
}

// VerifySignature validates the X-Telegram-Bot-Api-Secret-Token header against
// the configured secret. If no secret is configured, verification is skipped
// and nil is returned. Returns an error if the header is absent or mismatched.
func (a *TelegramAdapter) VerifySignature(r *http.Request) error {
	if a.secret == "" {
		return nil
	}
	provided := r.Header.Get(secretTokenHeader)
	if provided == "" {
		return fmt.Errorf("telegram: missing %s header", secretTokenHeader)
	}
	if provided != a.secret {
		return fmt.Errorf("telegram: invalid secret token")
	}
	return nil
}

// HandleWebhook processes a single inbound Telegram webhook POST. It verifies
// the signature, normalizes the payload, and writes 200 OK. On error it writes
// the appropriate HTTP status.
func (a *TelegramAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
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

// Normalize parses raw Telegram Update JSON and returns a ChannelMessage.
// Returns an error if the payload is malformed or contains no message.
func (a *TelegramAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var update Update
	if err := json.Unmarshal(raw, &update); err != nil {
		return nil, fmt.Errorf("telegram: normalize: invalid JSON: %w", err)
	}

	msg := update.Message
	if msg == nil {
		return nil, fmt.Errorf("telegram: normalize: update contains no message")
	}
	if msg.Chat == nil {
		return nil, fmt.Errorf("telegram: normalize: message missing chat")
	}

	cm := &channel.ChannelMessage{
		ID:        strconv.Itoa(msg.MessageID),
		Platform:  platform,
		ChannelID: strconv.FormatInt(msg.Chat.ID, 10),
		Text:      msg.Text,
		Timestamp: time.Unix(msg.Date, 0).UTC(),
	}

	if msg.From != nil {
		cm.UserID = strconv.FormatInt(msg.From.ID, 10)
		cm.UserName = msg.From.Username
		if cm.UserName == "" {
			cm.UserName = msg.From.FirstName
		}
	}

	if msg.ReplyToMessage != nil {
		cm.ReplyTo = strconv.Itoa(msg.ReplyToMessage.MessageID)
	}

	// Collect attachments.
	if msg.Document != nil {
		att := channel.Attachment{
			Type:     "file",
			Name:     msg.Document.FileName,
			Size:     msg.Document.FileSize,
			MimeType: msg.Document.MimeType,
			// URL is the file_id; callers resolve via getFile API as needed.
			URL: msg.Document.FileID,
		}
		cm.Attachments = append(cm.Attachments, att)
	}

	// Use the largest photo size (last element in the array, per Telegram docs).
	if len(msg.Photo) > 0 {
		largest := msg.Photo[len(msg.Photo)-1]
		att := channel.Attachment{
			Type: "image",
			URL:  largest.FileID,
			Size: largest.FileSize,
		}
		cm.Attachments = append(cm.Attachments, att)
	}

	return cm, nil
}

// Send delivers a ChannelMessage to the Telegram chat identified by
// msg.ChannelID using the sendMessage API. If msg.ReplyTo is set, the
// outbound message will quote the referenced message.
func (a *TelegramAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if msg == nil {
		return fmt.Errorf("telegram: send: nil message")
	}

	payload := sendMessageRequest{
		ChatID: msg.ChannelID,
		Text:   msg.Text,
	}

	if msg.ReplyTo != "" {
		replyID, err := strconv.Atoi(msg.ReplyTo)
		if err != nil {
			return fmt.Errorf("telegram: send: invalid reply_to %q: %w", msg.ReplyTo, err)
		}
		payload.ReplyToMessageID = replyID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: send: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, a.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: send: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: send: API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
