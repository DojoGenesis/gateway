// Package slack implements the Slack platform adapter for the Dojo Channel
// Bridge. It satisfies the channel.WebhookAdapter interface and supports both
// HTTP webhook mode and Socket Mode (for firewall-restricted operators).
//
// See ADR-018 for the full architectural rationale for the dual-mode design.
package slack

// SlackConfig holds all configuration needed to construct a SlackAdapter.
// Credentials are sourced from a channel.CredentialStore; this struct
// represents the resolved values for a single adapter instance.
type SlackConfig struct {
	// BotToken is the Slack bot OAuth token (xoxb-...).
	// Source: DOJO_SLACK_TOKEN environment variable or CredentialStore.
	BotToken string

	// SigningSecret is the Slack app signing secret used to verify webhook
	// payloads via HMAC-SHA256.
	// Source: DOJO_SLACK_SIGNINGSECRET environment variable or CredentialStore.
	SigningSecret string

	// Mode controls the transport layer. "http" (default) registers an HTTP
	// webhook handler. "socket" spawns a WebSocket goroutine using Socket Mode
	// (requires an App-Level Token). ADR-018 Q13: dual HTTP/Socket.
	Mode string

	// AppToken is the Slack app-level token (xapp-...) required for Socket
	// Mode. Ignored when Mode is "http".
	// Source: DOJO_SLACK_APPTOKEN environment variable or CredentialStore.
	AppToken string
}
