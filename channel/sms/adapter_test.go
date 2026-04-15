package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Twilio mandates HMAC-SHA1
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// 1. TestSMSAdapter_Name
// ---------------------------------------------------------------------------

func TestSMSAdapter_Name(t *testing.T) {
	a := NewSMSAdapter(SMSConfig{AccountSID: "AC123", AuthToken: "token", FromNumber: "+15550000000"})
	if got := a.Name(); got != "sms" {
		t.Errorf("Name() = %q, want %q", got, "sms")
	}
}

// ---------------------------------------------------------------------------
// 2. TestSMSAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestSMSAdapter_Capabilities(t *testing.T) {
	a := NewSMSAdapter(SMSConfig{})
	caps := a.Capabilities()

	if caps.SupportsThreads {
		t.Error("SupportsThreads should be false for SMS")
	}
	if caps.SupportsReactions {
		t.Error("SupportsReactions should be false for SMS")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true for SMS (MMS)")
	}
	if caps.SupportsEdits {
		t.Error("SupportsEdits should be false for SMS")
	}
	if caps.MaxMessageLength != 1600 {
		t.Errorf("MaxMessageLength = %d, want 1600", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestSMSAdapter_Normalize_TextMessage
// ---------------------------------------------------------------------------

func TestSMSAdapter_Normalize_TextMessage(t *testing.T) {
	a := NewSMSAdapter(SMSConfig{})

	payload := url.Values{}
	payload.Set("MessageSid", "SM123")
	payload.Set("From", "+15551112222")
	payload.Set("To", "+15553334444")
	payload.Set("Body", "hello sms")
	payload.Set("NumMedia", "0")

	msg, err := a.Normalize([]byte(payload.Encode()))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "sms")
	assertField(t, "ID", msg.ID, "SM123")
	assertField(t, "UserID", msg.UserID, "+15551112222")
	assertField(t, "ChannelID", msg.ChannelID, "+15553334444")
	assertField(t, "Text", msg.Text, "hello sms")

	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(msg.Attachments))
	}
}

// ---------------------------------------------------------------------------
// 4. TestSMSAdapter_Normalize_WithMedia
// ---------------------------------------------------------------------------

func TestSMSAdapter_Normalize_WithMedia(t *testing.T) {
	a := NewSMSAdapter(SMSConfig{})

	payload := url.Values{}
	payload.Set("MessageSid", "SM456")
	payload.Set("From", "+15551112222")
	payload.Set("To", "+15553334444")
	payload.Set("Body", "")
	payload.Set("NumMedia", "1")
	payload.Set("MediaUrl0", "https://api.twilio.com/2010-04-01/Accounts/AC123/Messages/MM1/Media/ME1")
	payload.Set("MediaContentType0", "image/jpeg")

	msg, err := a.Normalize([]byte(payload.Encode()))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if len(msg.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(msg.Attachments))
	}
	att := msg.Attachments[0]
	assertField(t, "Attachment.Type", att.Type, "image")
	assertField(t, "Attachment.MimeType", att.MimeType, "image/jpeg")
	if att.URL == "" {
		t.Error("Attachment.URL should not be empty")
	}
}

// ---------------------------------------------------------------------------
// 5. TestSMSAdapter_VerifySignature_Valid
// ---------------------------------------------------------------------------

func TestSMSAdapter_VerifySignature_Valid(t *testing.T) {
	authToken := "test-auth-token"
	a := NewSMSAdapter(SMSConfig{AuthToken: authToken})

	// Build form values matching what Twilio sends.
	form := url.Values{}
	form.Set("From", "+15551112222")
	form.Set("To", "+15553334444")
	form.Set("Body", "test")

	// Compute expected signature: HMAC-SHA1(authToken, fullURL + sorted params).
	targetURL := "https://example.com/webhook/sms"
	sig := computeTwilioSignature(authToken, targetURL, form)

	req := httptest.NewRequest(http.MethodPost, "/webhook/sms",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(sigHeader, sig)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "example.com"

	if err := a.VerifySignature(req); err != nil {
		t.Errorf("expected valid signature to pass, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 6. TestSMSAdapter_VerifySignature_Invalid
// ---------------------------------------------------------------------------

func TestSMSAdapter_VerifySignature_Invalid(t *testing.T) {
	a := NewSMSAdapter(SMSConfig{AuthToken: "correct-token"})

	t.Run("wrong_signature", func(t *testing.T) {
		form := url.Values{}
		form.Set("From", "+15551112222")
		form.Set("To", "+15553334444")

		req := httptest.NewRequest(http.MethodPost, "/webhook/sms",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set(sigHeader, "bad-signature")
		req.Host = "example.com"

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for wrong signature, got nil")
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/sms", nil)

		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing header, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 7. TestSMSAdapter_HandleWebhook
// ---------------------------------------------------------------------------

func TestSMSAdapter_HandleWebhook(t *testing.T) {
	// Use an adapter with no auth token so signature check is skipped.
	a := NewSMSAdapter(SMSConfig{})

	form := url.Values{}
	form.Set("MessageSid", "SM789")
	form.Set("From", "+15551112222")
	form.Set("To", "+15553334444")
	form.Set("Body", "webhook test")
	form.Set("NumMedia", "0")

	req := httptest.NewRequest(http.MethodPost, "/webhook/sms",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook status = %d, want 200", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 8. TestSMSAdapter_Send
// ---------------------------------------------------------------------------

func TestSMSAdapter_Send(t *testing.T) {
	var capturedBody string
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM001","status":"queued"}`))
	}))
	defer srv.Close()

	cfg := SMSConfig{
		AccountSID: "AC123456",
		AuthToken:  "mytoken",
		FromNumber: "+15550001111",
	}
	a := NewSMSAdapterWithClient(cfg, &http.Client{
		Transport: &urlRewriteTransport{
			base:       http.DefaultTransport,
			serverAddr: srv.URL,
		},
	})

	msg := &channel.ChannelMessage{
		Platform:  "sms",
		ChannelID: "+15552223333",
		Text:      "hello from send test",
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if capturedAuth == "" {
		t.Error("Authorization header should be set for Twilio Basic Auth")
	}

	parsed, err := url.ParseQuery(capturedBody)
	if err != nil {
		t.Fatalf("parse captured body: %v", err)
	}
	if parsed.Get("To") != "+15552223333" {
		t.Errorf("To = %q, want %q", parsed.Get("To"), "+15552223333")
	}
	if parsed.Get("From") != "+15550001111" {
		t.Errorf("From = %q, want %q", parsed.Get("From"), "+15550001111")
	}
	if parsed.Get("Body") != "hello from send test" {
		t.Errorf("Body = %q, want %q", parsed.Get("Body"), "hello from send test")
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// computeTwilioSignature replicates Twilio's HMAC-SHA1 signature algorithm.
func computeTwilioSignature(authToken, fullURL string, params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(fullURL)
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(params.Get(k))
	}

	mac := hmac.New(sha1.New, []byte(authToken)) //nolint:gosec
	mac.Write([]byte(sb.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func assertField(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// urlRewriteTransport redirects all requests to the given mock server.
type urlRewriteTransport struct {
	base       http.RoundTripper
	serverAddr string
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(t.serverAddr, "http://")
	return t.base.RoundTrip(cloned)
}
