package teams

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

const (
	platform         = "teams"
	maxMessageLength = 28000
)

// TeamsAdapter implements channel.WebhookAdapter for Microsoft Teams via the
// Bot Framework v3 Bot Connector REST API. It verifies JWT Bearer tokens,
// normalizes Bot Framework Activity JSON into ChannelMessage envelopes, and
// delivers outbound messages by POSTing Activity objects back to the
// serviceUrl provided in the inbound Activity.
//
// Phase 0 note: full JWKS verification is intentionally deferred. The adapter
// validates JWT structure (three base64-decodable parts) and logs a warning.
// See ADR-018 for the rationale.
//
// Construction: use NewTeamsAdapter — do not create the struct directly.
type TeamsAdapter struct {
	// botToken is the Bearer token used for outbound API calls.
	botToken string

	// httpClient is used for all outbound API calls.
	httpClient *http.Client
}

// NewTeamsAdapter returns a TeamsAdapter configured with the given bot token.
func NewTeamsAdapter(botToken string) *TeamsAdapter {
	return &TeamsAdapter{
		botToken:   botToken,
		httpClient: http.DefaultClient,
	}
}

// NewTeamsAdapterWithClient returns a TeamsAdapter that uses the provided
// http.Client for outbound API calls. Intended for unit testing.
func NewTeamsAdapterWithClient(botToken string, client *http.Client) *TeamsAdapter {
	return &TeamsAdapter{
		botToken:   botToken,
		httpClient: client,
	}
}

// Name returns the platform identifier "teams".
func (a *TeamsAdapter) Name() string {
	return platform
}

// Capabilities returns the feature set supported by Microsoft Teams.
func (a *TeamsAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       false,
		MaxMessageLength:    maxMessageLength,
	}
}

// VerifySignature validates the JWT Bearer token in the Authorization header.
// Phase 0: validates token structure (3 parts, each base64-decodable) but
// skips full JWKS verification. Logs a warning when skipping JWKS check.
func (a *TeamsAdapter) VerifySignature(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("teams: missing Authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fmt.Errorf("teams: Authorization header must use Bearer scheme")
	}

	token := parts[1]
	if err := validateJWTStructure(token); err != nil {
		return fmt.Errorf("teams: invalid JWT: %w", err)
	}

	// Phase 0: full JWKS verification deferred.
	log.Println("teams: WARNING: JWKS signature verification not yet implemented (Phase 0). Token structure validated only.")
	return nil
}

// HandleWebhook processes an inbound Teams Bot Framework Activity POST. It
// verifies the token, normalizes the payload, and writes 200 OK. On error it
// writes the appropriate HTTP status.
func (a *TeamsAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
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

// Normalize parses a Bot Framework Activity JSON payload and returns a
// ChannelMessage. Returns an error if the payload is malformed or missing
// required fields.
func (a *TeamsAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	var act Activity
	if err := json.Unmarshal(raw, &act); err != nil {
		return nil, fmt.Errorf("teams: normalize: invalid JSON: %w", err)
	}

	if act.Conversation.ID == "" {
		return nil, fmt.Errorf("teams: normalize: missing conversation.id")
	}

	cm := &channel.ChannelMessage{
		ID:        act.ID,
		Platform:  platform,
		ChannelID: act.Conversation.ID,
		UserID:    act.From.ID,
		UserName:  act.From.Name,
		Text:      act.Text,
		ThreadID:  act.ReplyToID,
		Timestamp: time.Now().UTC(),
	}

	// Parse timestamp if provided.
	if act.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, act.Timestamp); err == nil {
			cm.Timestamp = ts.UTC()
		}
	}

	// Map Bot Framework attachments.
	for _, att := range act.Attachments {
		attType := contentTypeToAttachmentType(att.ContentType)
		cm.Attachments = append(cm.Attachments, channel.Attachment{
			Type:     attType,
			URL:      att.ContentURL,
			Name:     att.Name,
			MimeType: att.ContentType,
		})
	}

	return cm, nil
}

// Send delivers a ChannelMessage to Microsoft Teams by posting a reply
// Activity to the Bot Connector endpoint derived from the original Activity's
// serviceUrl field. The ChannelID is used as the conversation ID.
func (a *TeamsAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if msg == nil {
		return fmt.Errorf("teams: send: nil message")
	}

	// Metadata must carry the serviceUrl set during Normalize/HandleWebhook.
	serviceURL, _ := msg.Metadata["service_url"].(string)
	if serviceURL == "" {
		return fmt.Errorf("teams: send: service_url missing from message metadata")
	}

	reply := Activity{
		Type:         "message",
		Text:         msg.Text,
		Conversation: ConversationAccount{ID: msg.ChannelID},
	}

	body, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("teams: send: marshal activity: %w", err)
	}

	apiURL := fmt.Sprintf("%s/v3/conversations/%s/activities",
		strings.TrimRight(serviceURL, "/"),
		msg.ChannelID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("teams: send: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.botToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("teams: send: HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("teams: send: Bot Connector returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// validateJWTStructure checks that the token consists of exactly three
// dot-separated parts, each of which is base64url-decodable. This is the
// Phase 0 structural check — full JWKS verification is deferred.
func validateJWTStructure(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("expected 3 parts, got %d", len(parts))
	}
	for i, part := range parts {
		if _, err := base64.RawURLEncoding.DecodeString(part); err != nil {
			return fmt.Errorf("part %d is not valid base64url: %w", i, err)
		}
	}
	return nil
}

// contentTypeToAttachmentType maps a Bot Framework content type to an
// Attachment.Type string.
func contentTypeToAttachmentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	default:
		return "file"
	}
}
