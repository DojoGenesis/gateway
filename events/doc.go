// Package events defines the Server-Sent Event (SSE) types emitted by the
// Agentic Gateway during agent interactions, tool invocations, artifact
// operations, and orchestration execution.
//
// The authoritative event catalog lives in catalog.go, which enumerates all
// 21 event types with documented JSON payload schemas. Constructor functions
// (New*Event) enforce the documented payload shape at compile time.
//
// Wire format follows RFC 8895:
//
//	event: <EventType>
//	data: {"type": "...", "data": {...}, "timestamp": "..."}
//
// Use AllEventTypes() to discover available events and IsValidEventType() to
// validate event type strings.
package events
