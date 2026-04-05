// Package channel defines the Channel Bridge adapter interface for Era 3.
// Each platform (Slack, GitHub, Discord, Email) implements ChannelAdapter
// to normalize inbound/outbound messages and platform context.
package channel

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ChannelAdapter is the interface every platform must implement.
// Design principles (from starred repo analysis):
//   - "Simplify before injecting" (Figma-Context-MCP) — adapters process, not proxy
//   - Three MCP primitives (fastmcp) — messages (tools), context (resources), templates (prompts)
//   - Zero-setup convention (git-mcp) — channels discoverable at dojo.channel/{platform}
type ChannelAdapter interface {
	// Platform identity
	ID() string
	Version() string

	// Inbound: platform → Dojo
	ReceiveMessage(ctx context.Context, raw json.RawMessage) (*InboundMessage, error)
	ReceiveWebhook(ctx context.Context, r *http.Request) (*InboundMessage, error)

	// Outbound: Dojo → platform
	SendMessage(ctx context.Context, msg *OutboundMessage) error
	SendReaction(ctx context.Context, ref MessageRef, reaction string) error

	// Context: enrich messages with platform-specific context
	FetchContext(ctx context.Context, ref MessageRef) (*PlatformContext, error)

	// Lifecycle
	Connect(ctx context.Context, creds *ChannelCredentials) error
	Disconnect(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

// InboundMessage is the normalized form of any platform message.
type InboundMessage struct {
	ID          string            `json:"id"`
	ChannelID   string            `json:"channel_id"`
	AuthorID    string            `json:"author_id"`
	AuthorName  string            `json:"author_name"`
	Content     string            `json:"content"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	ThreadID    string            `json:"thread_id,omitempty"`
	Platform    string            `json:"platform"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// OutboundMessage is a message to send to a platform.
type OutboundMessage struct {
	ChannelID   string            `json:"channel_id"`
	Content     string            `json:"content"`
	ThreadID    string            `json:"thread_id,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// MessageRef is a reference to a specific message on a platform.
type MessageRef struct {
	Platform  string `json:"platform"`
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id,omitempty"`
}

// Attachment represents a file or media attachment.
type Attachment struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// PlatformContext is enriched context from the platform.
type PlatformContext struct {
	ChannelName string            `json:"channel_name"`
	ChannelType string            `json:"channel_type"` // "dm", "group", "public"
	Members     []string          `json:"members,omitempty"`
	Topic       string            `json:"topic,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ChannelCredentials holds authentication for a platform.
type ChannelCredentials struct {
	Platform string            `json:"platform"`
	TokenMap map[string]string `json:"tokens"` // e.g., {"bot_token": "xoxb-...", "app_token": "xapp-..."}
}
