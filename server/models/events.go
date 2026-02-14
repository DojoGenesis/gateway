package models

import (
	"fmt"
	"time"
)

// BroadcastRequest is sent from Python backend to Go gateway
type BroadcastRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	Event    string `json:"event" binding:"required"`
	Data     string `json:"data" binding:"required"` // JSON string from Python
}

// Client represents an active SSE connection
type Client struct {
	ID      string
	Channel chan string
	Created time.Time
}

// SSEEvent formats data for Server-Sent Events wire protocol
type SSEEvent struct {
	Event string `json:"event,omitempty"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry int    `json:"retry,omitempty"` // Milliseconds
}

// FormatSSE converts SSEEvent to SSE wire format
// Returns a string formatted according to the SSE specification:
// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
func (e *SSEEvent) FormatSSE() string {
	msg := ""

	if e.Event != "" {
		msg += fmt.Sprintf("event: %s\n", e.Event)
	}

	if e.Data != "" {
		msg += fmt.Sprintf("data: %s\n", e.Data)
	}

	if e.ID != "" {
		msg += fmt.Sprintf("id: %s\n", e.ID)
	}

	if e.Retry > 0 {
		msg += fmt.Sprintf("retry: %d\n", e.Retry)
	}

	// SSE messages must end with double newline
	return msg + "\n"
}
