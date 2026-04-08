package sms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Twilio mandates HMAC-SHA1 for webhook signatures
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

const (
	platform          = "sms"
	twilioAPIBase     = "https://api.twilio.com/2010-04-01/Accounts"
	sigHeader         = "X-Twilio-Signature"
	maxMessageLength  = 1600
)

// SMSAdapter implements channel.WebhookAdapter for the Twilio SMS/MMS API.
// It verifies inbound Twilio webhook signatures using HMAC-SHA1, normalizes
// Twilio form-encoded payloads into ChannelMessage envelopes, and sends
// outbound messages via the Twilio Messages REST API.
//
// Construction: use NewSMSAdapter — do not create the struct directly.
type SMSAdapter struct {
	cfg        SMSConfig
	httpClient *http.Client
}

// NewSMSAdapter returns an SMSAdapter configured with the given SMSConfig.
// The AccountSID must not be empty. AuthToken is required for signature
// verification; it may be empty to skip verification (not recommended in
// production).
func NewSMSAdapter(cfg SMSConfig) *SMSAdapter {
	return &SMSAdapter{
		cfg:        cfg,
		httpClient: http.DefaultClient,
	}
}

// NewSMSAdapterWithClient returns an SMSAdapter that uses the provided
// http.Client for outbound API calls. Intended for unit testing.
func NewSMSAdapterWithClient(cfg SMSConfig, client *http.Client) *SMSAdapter {
	return &SMSAdapter{
		cfg:        cfg,
		httpClient: client,
	}
}

// Name returns the platform identifier "sms".
func (a *SMSAdapter) Name() string {
	return platform
}

// Capabilities returns the feature set supported by SMS/MMS.
func (a *SMSAdapter) Capabilities() channel.AdapterCapabilities {
	return channel.AdapterCapabilities{
		SupportsThreads:     false,
		SupportsReactions:   false,
		SupportsAttachments: true, // MMS
		SupportsEdits:       false,
		MaxMessageLength:    maxMessageLength,
	}
}

// VerifySignature validates the Twilio webhook signature using HMAC-SHA1.
// Twilio computes: HMAC-SHA1(authToken, fullURL + sorted POST params).
// See https://www.twilio.com/docs/usage/webhooks/webhooks-security
func (a *SMSAdapter) VerifySignature(r *http.Request) error {
	if a.cfg.AuthToken == "" {
		return nil
	}

	provided := r.Header.Get(sigHeader)
	if provided == "" {
		return fmt.Errorf("sms: missing %s header", sigHeader)
	}

	// Parse form to get POST parameters (needed for signature computation).
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("sms: failed to parse form for signature: %w", err)
	}

	// Build the full URL string.
	fullURL := buildFullURL(r)

	// Sort POST parameters alphabetically and concatenate key+value pairs.
	var paramStr strings.Builder
	keys := make([]string, 0, len(r.PostForm))
	for k := range r.PostForm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		paramStr.WriteString(k)
		paramStr.WriteString(r.PostForm.Get(k))
	}

	// Compute HMAC-SHA1.
	sigData := fullURL + paramStr.String()
	mac := hmac.New(sha1.New, []byte(a.cfg.AuthToken)) //nolint:gosec
	mac.Write([]byte(sigData))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return fmt.Errorf("sms: invalid Twilio signature")
	}
	return nil
}

// HandleWebhook processes an inbound Twilio webhook POST. It verifies the
// signature, normalizes the payload, and writes 200 OK. On error it writes
// the appropriate HTTP status.
func (a *SMSAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if err := a.VerifySignature(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Ensure form is parsed (VerifySignature may have already done this).
	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Encode form values as raw bytes for Normalize.
	raw := []byte(r.PostForm.Encode())

	if _, err := a.Normalize(raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Normalize parses a URL-encoded Twilio webhook payload and returns a
// ChannelMessage. The payload must contain at least From and To fields.
func (a *SMSAdapter) Normalize(raw []byte) (*channel.ChannelMessage, error) {
	values, err := url.ParseQuery(string(raw))
	if err != nil {
		return nil, fmt.Errorf("sms: normalize: invalid form data: %w", err)
	}

	from := values.Get("From")
	to := values.Get("To")
	if from == "" {
		return nil, fmt.Errorf("sms: normalize: missing From field")
	}
	if to == "" {
		return nil, fmt.Errorf("sms: normalize: missing To field")
	}

	cm := &channel.ChannelMessage{
		ID:        values.Get("MessageSid"),
		Platform:  platform,
		ChannelID: to,
		UserID:    from,
		Text:      values.Get("Body"),
		Timestamp: time.Now().UTC(),
	}

	// Collect MMS media attachments.
	numMedia := values.Get("NumMedia")
	if numMedia != "" && numMedia != "0" {
		// Twilio numbers media from MediaUrl0, MediaUrl1, etc.
		for i := 0; ; i++ {
			mediaURL := values.Get(fmt.Sprintf("MediaUrl%d", i))
			if mediaURL == "" {
				break
			}
			contentType := values.Get(fmt.Sprintf("MediaContentType%d", i))
			attType := mediaTypeToAttachmentType(contentType)
			cm.Attachments = append(cm.Attachments, channel.Attachment{
				Type:     attType,
				URL:      mediaURL,
				MimeType: contentType,
			})
		}
	}

	return cm, nil
}

// Send delivers a ChannelMessage as an SMS/MMS via the Twilio Messages API.
// It uses HTTP Basic Auth with the AccountSID and AuthToken.
func (a *SMSAdapter) Send(ctx context.Context, msg *channel.ChannelMessage) error {
	if msg == nil {
		return fmt.Errorf("sms: send: nil message")
	}

	form := url.Values{}
	form.Set("From", a.cfg.FromNumber)
	form.Set("To", msg.ChannelID)
	form.Set("Body", msg.Text)

	apiURL := fmt.Sprintf("%s/%s/Messages.json", twilioAPIBase, a.cfg.AccountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("sms: send: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(a.cfg.AccountSID, a.cfg.AuthToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms: send: HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sms: send: Twilio API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildFullURL reconstructs the full request URL including scheme and host.
func buildFullURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
		scheme = "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	var buf bytes.Buffer
	buf.WriteString(scheme)
	buf.WriteString("://")
	buf.WriteString(host)
	buf.WriteString(r.URL.RequestURI())
	return buf.String()
}

// mediaTypeToAttachmentType maps a MIME type to an Attachment.Type string.
func mediaTypeToAttachmentType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "file"
	}
}
