package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/channel"
	"github.com/google/uuid"
	slackgo "github.com/slack-go/slack"
)

const (
	// platformName is the identifier returned by Name().
	platformName = "slack"

	// signatureVersion is the prefix Slack prepends to HMAC signatures.
	signatureVersion = "v0"

	// replayWindow is the maximum age of a valid webhook timestamp.
	replayWindow = 5 * time.Minute

	// slackMaxMessageLength is the documented maximum message size for Slack.
	slackMaxMessageLength = 40000
)

// slackSender is the minimal interface from slack-go that SlackAdapter calls.
// Declaring it here lets tests inject a mock without importing slack-go.
type slackSender interface {
	PostMessage(channelID string, options ...slackgo.MsgOption) (string, string, error)
}

// SlackAdapter implements channel.WebhookAdapter for Slack.
// It handles both HTTP webhook mode and normalizes Slack event payloads into
// the platform-agnostic channel.ChannelMessage envelope.
//
// Construct with New() to wire the slack-go client. In tests, inject a
// slackSender mock via NewWithSender().
type SlackAdapter struct {
	cfg    SlackConfig
	client slackSender
}

// New constructs a SlackAdapter from config, wiring up a real slack-go client.
func New(cfg SlackConfig) *SlackAdapter {
	return &SlackAdapter{
		cfg:    cfg,
		client: slackgo.New(cfg.BotToken),
	}
}

// NewWithSender constructs a SlackAdapter with an injected slackSender.
// Used by tests to avoid real Slack API calls.
func NewWithSender(cfg SlackConfig, sender slackSender) *SlackAdapter {
	return &SlackAdapter{
		cfg:    cfg,
		client: sender,
	}
}

// Name returns the platform identifier.
func (a *SlackAdapter) Name() string {
	return platformName
}

// Capabilities returns the feature set supported by Slack.
func (a *SlackAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       true,
		MaxMessageLength:    slackMaxMessageLength,
	}
}

// VerifySignature validates the Slack request signature using HMAC-SHA256.
//
// Slack signs webhook payloads using:
//
//	sig = HMAC-SHA256(signing_secret, "v0:{timestamp}:{body}")
//	X-Slack-Signature: v0={sig_hex}
//
// The X-Slack-Request-Timestamp header must be within 5 minutes of the
// current time to prevent replay attacks (Slack documentation requirement).
func (a *SlackAdapter) VerifySignature(r *http.Request) error {
	timestampHeader := r.Header.Get("X-Slack-Request-Timestamp")
	if timestampHeader == "" {
		return fmt.Errorf("slack: missing X-Slack-Request-Timestamp header")
	}

	ts, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("slack: invalid X-Slack-Request-Timestamp value: %w", err)
	}

	age := time.Since(time.Unix(ts, 0))
	if age > replayWindow || age < -replayWindow {
		return fmt.Errorf("slack: request timestamp is outside the %s replay window", replayWindow)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("slack: failed to read request body for signature verification: %w", err)
	}
	// Restore body so downstream handlers can re-read it.
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	sigBase := fmt.Sprintf("%s:%s:%s", signatureVersion, timestampHeader, string(body))
	mac := hmac.New(sha256.New, []byte(a.cfg.SigningSecret))
	mac.Write([]byte(sigBase))
	expected := signatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))

	provided := r.Header.Get("X-Slack-Signature")
	if provided == "" {
		return fmt.Errorf("slack: missing X-Slack-Signature header")
	}

	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return fmt.Errorf("slack: signature mismatch")
	}

	return nil
}

// HandleWebhook processes inbound Slack HTTP requests. It handles two cases:
//
//  1. URL verification challenge (type = "url_verification") — responds with
//     the challenge string so Slack can confirm the endpoint is live.
//  2. Event callbacks (type = "event_callback") — normalizes the inner event
//     to a channel.ChannelMessage and writes 200 OK per ADR-018 pattern:
//     verify → return 200 immediately → enqueue to NATS for async processing.
//     (Enqueuing is outside the scope of this adapter.)
func (a *SlackAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	var envelope struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	switch envelope.Type {
	case "url_verification":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(map[string]string{"challenge": envelope.Challenge})
		w.Write(resp)

	case "event_callback":
		// Normalize to ChannelMessage. The caller is responsible for
		// publishing to NATS; the adapter returns 200 immediately.
		_, _ = a.Normalize(body)
		w.WriteHeader(http.StatusOK)

	default:
		// Unknown event types are acknowledged with 200 to avoid Slack retries.
		w.WriteHeader(http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Normalize — Slack event JSON → channel.ChannelMessage
// ---------------------------------------------------------------------------

// slackFile mirrors the relevant fields of a Slack file object in event payloads.
type slackFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Mimetype string `json:"mimetype"`
	FileSize int64  `json:"size"`
	URLPrivate string `json:"url_private"`
}

// slackEvent mirrors the inner "event" object of a Slack event_callback payload.
type slackEvent struct {
	Type     string      `json:"type"`
	User     string      `json:"user"`
	Text     string      `json:"text"`
	Channel  string      `json:"channel"`
	Ts       string      `json:"ts"`
	ThreadTs string      `json:"thread_ts"`
	Files    []slackFile `json:"files"`
}

// slackEventCallback mirrors the outer envelope of a Slack event_callback.
type slackEventCallback struct {
	Type      string     `json:"type"`
	Challenge string     `json:"challenge"`
	Event     slackEvent `json:"event"`
}

// Normalize converts a raw Slack event_callback JSON payload into a
// channel.ChannelMessage. It also handles the url_verification type so callers
// can pass any Slack webhook payload without pre-inspection.
func (a *SlackAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var envelope slackEventCallback
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("slack: normalize failed to parse payload: %w", err)
	}

	if envelope.Type == "url_verification" {
		// URL verification carries no message content.
		return nil, fmt.Errorf("slack: normalize called on url_verification payload (not a message event)")
	}

	if envelope.Type != "event_callback" {
		return nil, fmt.Errorf("slack: normalize: unsupported envelope type %q", envelope.Type)
	}

	evt := envelope.Event

	// Parse the Slack timestamp (Unix seconds with microsecond decimal).
	ts, err := parseSlackTS(evt.Ts)
	if err != nil {
		// Fall back to now so normalization still succeeds.
		ts = time.Now().UTC()
	}

	msg := &channel.ChannelMessage{
		ID:        uuid.New().String(),
		Platform:  platformName,
		ChannelID: evt.Channel,
		UserID:    evt.User,
		Text:      evt.Text,
		Timestamp: ts,
		ThreadID:  evt.ThreadTs,
		Metadata: map[string]interface{}{
			"slack_ts": evt.Ts,
		},
	}

	for _, f := range evt.Files {
		kind := classifyMime(f.Mimetype)
		msg.Attachments = append(msg.Attachments, channel.Attachment{
			Type:     kind,
			URL:      f.URLPrivate,
			Name:     f.Name,
			Size:     f.FileSize,
			MimeType: f.Mimetype,
		})
	}

	return msg, nil
}

// Send delivers a ChannelMessage to Slack via PostMessage. If msg.ThreadID is
// set the reply is posted as a thread reply using MsgOptionTS.
func (a *SlackAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if msg == nil {
		return fmt.Errorf("slack: send called with nil message")
	}

	opts := []slackgo.MsgOption{
		slackgo.MsgOptionText(msg.Text, false),
	}

	if msg.ThreadID != "" {
		opts = append(opts, slackgo.MsgOptionTS(msg.ThreadID))
	}

	_, _, err := a.client.PostMessage(msg.ChannelID, opts...)
	if err != nil {
		return fmt.Errorf("slack: PostMessage failed: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// parseSlackTS converts a Slack timestamp string (e.g. "1609459200.000001")
// into a time.Time with microsecond precision.
func parseSlackTS(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, fmt.Errorf("slack: empty timestamp")
	}

	parts := strings.SplitN(ts, ".", 2)
	secs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("slack: invalid timestamp seconds %q: %w", ts, err)
	}

	var micros int64
	if len(parts) == 2 {
		// Pad or truncate to 6 digits for microseconds.
		frac := parts[1]
		for len(frac) < 6 {
			frac += "0"
		}
		frac = frac[:6]
		micros, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			micros = 0
		}
	}

	return time.Unix(secs, micros*1000).UTC(), nil
}

// classifyMime maps a MIME type string to an Attachment.Type category.
func classifyMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	default:
		return "file"
	}
}
