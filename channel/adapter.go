// Package channel implements the Channel Bridge — a multi-platform messaging
// adapter layer that normalizes inbound and outbound messages across
// communication platforms (Slack, Discord, Telegram, etc.) into a unified
// ChannelMessage envelope routed through CloudEvents.
//
// Two adapter subtypes share a common ChannelAdapter base:
//   - WebhookAdapter for HTTP webhook-based platforms
//   - ActorAdapter for persistent-connection platforms
//
// See ADR-018 for the full architectural rationale.
package channel

import (
	"context"
	"net/http"
)

// ChannelAdapter is the base interface that every platform adapter must
// implement. It covers name identification, message normalization,
// outbound sending, and capability discovery.
type ChannelAdapter interface {
	// Name returns the platform identifier (e.g. "slack", "discord").
	Name() string

	// Normalize converts raw platform-specific bytes into a ChannelMessage.
	Normalize(raw []byte) (*ChannelMessage, error)

	// Send delivers a ChannelMessage to the platform.
	Send(ctx context.Context, msg *ChannelMessage) error

	// Capabilities returns what this adapter supports.
	Capabilities() AdapterCapabilities
}

// WebhookAdapter extends ChannelAdapter for HTTP webhook-based platforms
// (Slack HTTP, Telegram, Email, SMS). The WebhookGateway routes inbound
// HTTP requests to the appropriate WebhookAdapter.
type WebhookAdapter interface {
	ChannelAdapter

	// HandleWebhook processes an inbound HTTP webhook request.
	HandleWebhook(w http.ResponseWriter, r *http.Request)

	// VerifySignature validates the request's cryptographic signature.
	VerifySignature(r *http.Request) error
}

// ActorAdapter extends ChannelAdapter for persistent-connection platforms
// (Discord Gateway WebSocket, WhatsApp Cloud API sessions). Each live
// session is managed as a supervised actor (ADR-014).
type ActorAdapter interface {
	ChannelAdapter

	// Connect establishes a persistent connection to the platform.
	Connect(ctx context.Context, creds Credentials) error

	// Disconnect gracefully closes the persistent connection.
	Disconnect(ctx context.Context) error

	// OnMessage registers a handler called for each inbound message.
	OnMessage(handler func(*ChannelMessage))
}

// AdapterCapabilities describes what features a platform adapter supports.
// Used by workflows to degrade gracefully on platforms that lack certain
// features (e.g. threads, reactions).
type AdapterCapabilities struct {
	// SupportsThreads indicates the platform has threaded conversations.
	SupportsThreads bool

	// SupportsReactions indicates the platform supports emoji reactions.
	SupportsReactions bool

	// SupportsAttachments indicates the platform can send/receive files.
	SupportsAttachments bool

	// SupportsEdits indicates the platform allows editing sent messages.
	SupportsEdits bool

	// MaxMessageLength is the maximum text length in bytes (0 = unlimited).
	MaxMessageLength int
}
