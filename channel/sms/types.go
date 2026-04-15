// Package sms implements the Twilio SMS/MMS adapter for the Dojo Channel Bridge.
// It normalizes Twilio webhook payloads into ChannelMessage envelopes and
// delivers outbound messages via the Twilio Messages API.
//
// This adapter uses net/http + encoding/json directly against the Twilio
// REST API — no external Twilio library is required (Phase 0 dependency
// isolation, ADR-018).
package sms

// SMSConfig holds the credentials and configuration for the Twilio SMS adapter.
type SMSConfig struct {
	// AccountSID is the Twilio Account SID (e.g. "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx").
	AccountSID string

	// AuthToken is the Twilio Auth Token used for API authentication and
	// HMAC-SHA1 signature verification of inbound webhooks.
	AuthToken string

	// FromNumber is the Twilio phone number used for outbound replies
	// (e.g. "+15551234567").
	FromNumber string
}

