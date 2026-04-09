package teams

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

const (
	platform         = "teams"
	maxMessageLength = 28000

	// botFrameworkIssuer is the only accepted issuer for Bot Framework v3
	// tokens sent to bots. AAD-backed issuers (sts.windows.net,
	// login.microsoftonline.com) are not used by the Bot Connector itself.
	botFrameworkIssuer = "https://api.botframework.com"

	// maxTokenBytes bounds memory allocated before we even begin parsing.
	// A legitimate Bot Framework JWT is well under 2 KiB.
	maxTokenBytes = 8192
)

// TeamsAdapter implements channel.WebhookAdapter for Microsoft Teams via the
// Bot Framework v3 Bot Connector REST API. It verifies JWT Bearer tokens
// against Microsoft's public JWKS endpoint, normalizes Bot Framework Activity
// JSON into ChannelMessage envelopes, and delivers outbound messages by
// POSTing Activity objects back to the serviceUrl provided in the inbound
// Activity.
//
// Construction: use NewTeamsAdapter — do not create the struct directly.
type TeamsAdapter struct {
	// botToken is the Bearer token used for outbound API calls.
	botToken string

	// appID is the Microsoft App ID of this bot. It is checked against the
	// aud claim of every inbound JWT.
	appID string

	// httpClient is used for all outbound API calls and JWKS fetches.
	httpClient *http.Client

	// jwks caches Microsoft's public signing keys.
	jwks *jwksCache
}

// NewTeamsAdapter returns a TeamsAdapter configured with the given bot token
// and Microsoft App ID. The App ID is validated against the aud claim of
// every inbound JWT.
func NewTeamsAdapter(botToken, appID string) *TeamsAdapter {
	return newTeamsAdapterInternal(botToken, appID, defaultJWKSURL, http.DefaultClient)
}

// NewTeamsAdapterWithClient returns a TeamsAdapter that uses the provided
// http.Client for outbound API calls and JWKS fetches. Intended for unit
// testing.
func NewTeamsAdapterWithClient(botToken, appID string, client *http.Client) *TeamsAdapter {
	return newTeamsAdapterInternal(botToken, appID, defaultJWKSURL, client)
}

// newTeamsAdapterInternal is the canonical constructor. jwksURL is exposed
// so tests can point to a mock JWKS server.
func newTeamsAdapterInternal(botToken, appID, jwksURL string, client *http.Client) *TeamsAdapter {
	return &TeamsAdapter{
		botToken:   botToken,
		appID:      appID,
		httpClient: client,
		jwks:       newJWKSCache(jwksURL),
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

// VerifySignature validates the JWT Bearer token in the Authorization header
// against Microsoft's Bot Framework JWKS endpoint. It enforces:
//   - RS256 algorithm
//   - Signature valid against a current Microsoft public key
//   - iss == "https://api.botframework.com"
//   - aud == the configured bot App ID
//   - Token is not expired (with 5-minute clock-skew tolerance)
//   - Token is past its nbf (with 5-minute clock-skew tolerance)
func (a *TeamsAdapter) VerifySignature(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("teams: missing Authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fmt.Errorf("teams: Authorization header must use Bearer scheme")
	}

	return a.verifyJWT(r.Context(), parts[1])
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

// verifyJWT validates a raw JWT string. Order follows RFC 7519 §7.2:
// structure and algorithm first, then signature (the gate that authenticates
// the sender), then claim validation (exp, nbf, iss, aud).
func (a *TeamsAdapter) verifyJWT(ctx context.Context, token string) error {
	if len(token) > maxTokenBytes {
		return fmt.Errorf("teams: jwt: token length %d exceeds %d-byte limit", len(token), maxTokenBytes)
	}

	// --- Structure ---
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("teams: jwt: expected 3 parts, got %d", len(parts))
	}

	// --- Parse header ---
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("teams: jwt: invalid header encoding: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return fmt.Errorf("teams: jwt: invalid header JSON: %w", err)
	}
	if header.Alg != "RS256" {
		return fmt.Errorf("teams: jwt: unsupported algorithm %q, expected RS256", header.Alg)
	}
	if header.Kid == "" {
		return fmt.Errorf("teams: jwt: missing kid in header")
	}

	// --- Decode signature bytes ---
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("teams: jwt: invalid signature encoding: %w", err)
	}

	// --- Fetch public key and verify signature ---
	// Do this before trusting any payload claim values.
	pub, err := a.jwks.getOrRefresh(ctx, a.httpClient, header.Kid)
	if err != nil {
		return err
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sigBytes); err != nil {
		return fmt.Errorf("teams: jwt: signature verification failed: %w", err)
	}

	// --- Parse payload (claims are now authenticated) ---
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("teams: jwt: invalid payload encoding: %w", err)
	}
	var claims struct {
		Iss string  `json:"iss"`
		Aud string  `json:"aud"`
		Exp float64 `json:"exp"`
		Nbf float64 `json:"nbf"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return fmt.Errorf("teams: jwt: invalid payload JSON: %w", err)
	}

	// --- Validate claims ---
	now := time.Now()

	if claims.Exp == 0 {
		return fmt.Errorf("teams: jwt: missing exp claim")
	}
	expiry := time.Unix(int64(claims.Exp), 0)
	if now.After(expiry.Add(clockSkewTolerance)) {
		return fmt.Errorf("teams: jwt: token expired at %s", expiry.UTC().Format(time.RFC3339))
	}

	if claims.Nbf != 0 {
		notBefore := time.Unix(int64(claims.Nbf), 0)
		if now.Before(notBefore.Add(-clockSkewTolerance)) {
			return fmt.Errorf("teams: jwt: token not yet valid until %s", notBefore.UTC().Format(time.RFC3339))
		}
	}

	if claims.Iss != botFrameworkIssuer {
		return fmt.Errorf("teams: jwt: unexpected issuer %q", claims.Iss)
	}
	if claims.Aud != a.appID {
		return fmt.Errorf("teams: jwt: unexpected audience %q", claims.Aud)
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
