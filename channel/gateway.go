package channel

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// EventPublisher is a minimal interface for publishing CloudEvents to a bus.
// This mirrors the runtime/event.Bus Publish signature without importing it,
// keeping the channel module dependency-free in Phase 0.
type EventPublisher interface {
	Publish(subject string, evt Event) error
}

// WebhookGateway routes inbound HTTP webhook requests to per-platform
// WebhookAdapters. It follows the Chatwoot pattern (ADR-018): verify
// signature, return 200 immediately, normalize and publish to the event bus.
type WebhookGateway struct {
	mu       sync.RWMutex
	adapters map[string]WebhookAdapter
	bus      EventPublisher
	creds    CredentialStore
}

// NewWebhookGateway creates a gateway with the given event bus and credential
// store. Either or both may be nil for stub/testing scenarios.
func NewWebhookGateway(bus EventPublisher, creds CredentialStore) *WebhookGateway {
	return &WebhookGateway{
		adapters: make(map[string]WebhookAdapter),
		bus:      bus,
		creds:    creds,
	}
}

// Register adds a platform adapter to the gateway. The platform name is
// used as the URL path segment: /webhooks/{platform}.
func (gw *WebhookGateway) Register(platform string, adapter WebhookAdapter) {
	gw.mu.Lock()
	gw.adapters[platform] = adapter
	gw.mu.Unlock()
	slog.Info("channel: registered webhook adapter", "platform", platform)
}

// ServeHTTP implements http.Handler. It routes requests matching
// /webhooks/{platform} to the corresponding WebhookAdapter.
//
// Flow: verify signature -> read body -> normalize -> publish to bus -> 200 OK
func (gw *WebhookGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract platform from path: /webhooks/{platform}
	platform := extractPlatform(r.URL.Path)
	if platform == "" {
		http.Error(w, "channel: missing platform in path", http.StatusBadRequest)
		return
	}

	gw.mu.RLock()
	adapter, ok := gw.adapters[platform]
	gw.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("channel: unknown platform %q", platform), http.StatusNotFound)
		return
	}

	// Verify signature before processing.
	if err := adapter.VerifySignature(r); err != nil {
		slog.Warn("channel: signature verification failed",
			"platform", platform,
			"error", err,
		)
		http.Error(w, "channel: signature verification failed", http.StatusUnauthorized)
		return
	}

	// Read the body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("channel: failed to read request body",
			"platform", platform,
			"error", err,
		)
		http.Error(w, "channel: failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Normalize to ChannelMessage.
	msg, err := adapter.Normalize(body)
	if err != nil {
		slog.Error("channel: normalization failed",
			"platform", platform,
			"error", err,
		)
		http.Error(w, "channel: normalization failed", http.StatusBadRequest)
		return
	}

	// Fill in timestamp if the adapter didn't set it.
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}

	// Publish to event bus if available.
	if gw.bus != nil {
		evt, err := ToCloudEvent(msg)
		if err != nil {
			slog.Error("channel: failed to create CloudEvent",
				"platform", platform,
				"error", err,
			)
			// Still return 200 — the webhook was received.
		} else {
			subject := fmt.Sprintf("dojo.channel.message.%s", platform)
			if pubErr := gw.bus.Publish(subject, evt); pubErr != nil {
				slog.Error("channel: failed to publish event",
					"platform", platform,
					"subject", subject,
					"error", pubErr,
				)
			}
		}
	}

	// Return 200 immediately (Chatwoot pattern).
	w.WriteHeader(http.StatusOK)
}

// extractPlatform parses the platform name from a URL path like
// /webhooks/{platform} or /webhooks/{platform}/. Returns empty string
// if the path does not match the expected pattern.
func extractPlatform(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] != "webhooks" {
		return ""
	}
	return parts[1]
}
