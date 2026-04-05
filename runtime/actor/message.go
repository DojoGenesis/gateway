package actor

import "time"

// MessageType categorizes actor messages.
type MessageType int

const (
	// MessageRequest is a request expecting a reply.
	MessageRequest MessageType = iota
	// MessageResponse is a reply to a request.
	MessageResponse
	// MessageEvent is a fire-and-forget event notification.
	MessageEvent
	// MessageControl is a system control message (shutdown, restart, health-check).
	MessageControl
)

// Message is the unit of communication between actors.
type Message struct {
	// ID is a unique message identifier.
	ID string

	// Type categorizes the message.
	Type MessageType

	// Payload is the message content as bytes.
	Payload []byte

	// ReplyTo is a channel for request/reply pattern. May be nil.
	ReplyTo chan Message

	// Timestamp is when the message was created.
	Timestamp time.Time

	// TraceID enables OTEL trace propagation.
	TraceID string
}
