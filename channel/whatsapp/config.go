// Package whatsapp implements the WhatsApp Cloud API adapter for the Dojo
// Channel Bridge. It normalizes inbound webhook payloads into ChannelMessage
// envelopes and delivers outbound messages via the WhatsApp Cloud API.
//
// This adapter implements both channel.WebhookAdapter (for webhook verification
// and inbound message handling) and channel.ActorAdapter (for persistent session
// management). It uses the Cloud API (On-Premises sunset Oct 2025).
//
// Construction: use NewWhatsAppAdapter — do not create the struct directly.
package whatsapp

// WhatsAppConfig holds all configuration for the WhatsApp Cloud API adapter.
type WhatsAppConfig struct {
	// PhoneNumberID is the WhatsApp Business phone number ID registered
	// in the Meta Developer Portal.
	PhoneNumberID string

	// AccessToken is the permanent or temporary Meta access token used to
	// authenticate API calls.
	AccessToken string

	// VerifyToken is the token compared against hub.verify_token during
	// webhook verification (GET challenge).
	VerifyToken string

	// AppSecret is used for HMAC-SHA256 payload signature verification.
	// If empty, signature verification is skipped.
	AppSecret string

	// APIURL is the base URL for the WhatsApp Cloud API.
	// Defaults to "https://graph.facebook.com/v21.0" when empty.
	APIURL string
}

// defaultAPIURL is the WhatsApp Cloud API base URL (v21.0).
const defaultAPIURL = "https://graph.facebook.com/v21.0"

// apiURL returns the configured APIURL or the default.
func (c *WhatsAppConfig) apiURL() string {
	if c.APIURL != "" {
		return c.APIURL
	}
	return defaultAPIURL
}
