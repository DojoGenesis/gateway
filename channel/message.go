package channel

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ChannelMessage is the normalized message envelope shared by all adapters.
// Core fields are normalized; the Metadata map preserves platform-specific
// richness without schema bloat (unified.to hybrid pattern, ADR-018).
type ChannelMessage struct {
	// ID is a unique message identifier.
	ID string `json:"id"`

	// Platform identifies the source (e.g. "slack", "discord", "telegram").
	Platform string `json:"platform"`

	// ChannelID is the platform-specific channel or conversation identifier.
	ChannelID string `json:"channel_id"`

	// UserID is the platform-specific user identifier.
	UserID string `json:"user_id"`

	// UserName is the human-readable display name.
	UserName string `json:"user_name"`

	// Text is the message body.
	Text string `json:"text"`

	// Attachments holds any files, images, or media.
	Attachments []Attachment `json:"attachments,omitempty"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// Metadata holds platform-specific fields (raw passthrough).
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ReplyTo references the message being replied to.
	ReplyTo string `json:"reply_to,omitempty"`

	// ThreadID groups messages in a threaded conversation.
	ThreadID string `json:"thread_id,omitempty"`
}

// Attachment represents a file, image, video, or audio attachment.
type Attachment struct {
	// Type is the attachment category: "image", "file", "video", "audio".
	Type string `json:"type"`

	// URL is the download or display URL.
	URL string `json:"url"`

	// Name is the human-readable file name.
	Name string `json:"name"`

	// Size is the file size in bytes.
	Size int64 `json:"size,omitempty"`

	// MimeType is the MIME content type (e.g. "image/png").
	MimeType string `json:"mime_type,omitempty"`
}

// Credentials holds platform authentication data. The Extra map allows
// platform-specific fields without schema changes.
type Credentials struct {
	// Platform identifies which adapter these credentials are for.
	Platform string `json:"platform"`

	// Token is the primary authentication token (e.g. bot token).
	Token string `json:"token"`

	// Secret is a secondary secret (e.g. signing secret, webhook secret).
	Secret string `json:"secret,omitempty"`

	// Extra holds additional platform-specific credential fields.
	Extra map[string]string `json:"extra,omitempty"`
}

// ---------------------------------------------------------------------------
// CloudEvent support — local types that mirror the pattern from
// runtime/event without importing it (Phase 0 dependency isolation).
// ---------------------------------------------------------------------------

// Event is a lightweight CloudEvents v1.0 representation used for bus
// publishing. This mirrors the structure in runtime/event without creating
// a module dependency. A future phase will unify these types.
type Event struct {
	// SpecVersion is always "1.0".
	SpecVersion string `json:"specversion"`

	// Type follows the pattern "dojo.channel.message.{platform}".
	Type string `json:"type"`

	// Source identifies the origin (e.g. "channel/{platform}").
	Source string `json:"source"`

	// ID is a unique event identifier.
	ID string `json:"id"`

	// Time is the event timestamp in RFC3339 format.
	Time time.Time `json:"time"`

	// DataContentType is the media type of Data.
	DataContentType string `json:"datacontenttype"`

	// Data is the serialized event payload.
	Data json.RawMessage `json:"data"`
}

// ToCloudEvent converts a ChannelMessage into a CloudEvent following the
// dojo.channel.message.{platform} convention (ADR-018 message flow).
func ToCloudEvent(msg *ChannelMessage) (Event, error) {
	if msg == nil {
		return Event{}, fmt.Errorf("channel: cannot convert nil message to CloudEvent")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return Event{}, fmt.Errorf("channel: failed to marshal message to CloudEvent data: %w", err)
	}

	return Event{
		SpecVersion:     "1.0",
		Type:            fmt.Sprintf("dojo.channel.message.%s", msg.Platform),
		Source:          fmt.Sprintf("channel/%s", msg.Platform),
		ID:              uuid.New().String(),
		Time:            msg.Timestamp,
		DataContentType: "application/json",
		Data:            data,
	}, nil
}
