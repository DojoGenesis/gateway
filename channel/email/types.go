// Package email implements the SendGrid Inbound Parse webhook adapter for the
// Dojo Channel Bridge. It normalizes inbound email into ChannelMessage envelopes
// and delivers outbound messages via the SendGrid v3 mail/send API.
//
// This adapter uses stdlib only — no external email or SendGrid library is
// required (Phase 0 dependency isolation).
package email

// EmailConfig holds the runtime configuration for the email adapter.
type EmailConfig struct {
	// WebhookSecret is the shared secret used to authenticate inbound
	// SendGrid Inbound Parse requests via the X-Webhook-Secret header.
	WebhookSecret string

	// SendGridAPIKey is the Bearer token used when POSTing to the
	// SendGrid v3 mail/send API for outbound replies.
	SendGridAPIKey string

	// FromAddress is the sender email address used for outbound replies.
	FromAddress string

	// FromName is the human-readable display name for outbound replies.
	FromName string
}

// InboundEmail mirrors the multipart form fields that SendGrid Inbound Parse
// delivers to the registered webhook endpoint.
// See https://docs.sendgrid.com/for-developers/parsing-email/setting-up-the-inbound-parse-webhook
type InboundEmail struct {
	From        string            `json:"from"`
	To          string            `json:"to"`
	Subject     string            `json:"subject"`
	Text        string            `json:"text"`
	HTML        string            `json:"html"`
	Envelope    string            `json:"envelope"`
	Headers     string            `json:"headers"`
	Attachments map[string]string `json:"attachments,omitempty"`
}
