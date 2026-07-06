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
// Flow: read body -> handshake check -> verify signature -> normalize -> publish -> 200 OK
//
// The body is read once and restored on the request so that VerifySignature
// and HandleWebhook can both re-read it. Handshake requests (e.g. Slack
// url_verification) are detected before signature verification and delegated
// directly to HandleWebhook, because Slack does not sign those requests.
func (gw *WebhookGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract platform from path: /webhooks/{platform}
	platform := extractPlatform(r.URL.Path)
	if platform == "" {
		http.Error(w, "channel: missing platform in path", http.StatusBadRequest)
		return
	}
	// Sanitize platform for safe logging: strip CR/LF to prevent log injection.
	safePlatform := strings.NewReplacer("\r", "", "\n", "").Replace(platform)

	gw.mu.RLock()
	adapter, ok := gw.adapters[platform]
	gw.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("channel: unknown platform %q", platform), http.StatusNotFound)
		return
	}

	// Read the body once and restore it so both VerifySignature and
	// HandleWebhook / Normalize can re-read it.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "channel: failed to read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Detect handshake payloads (e.g. Slack url_verification). These are not
	// signed by the platform, so they must be handled before VerifySignature.
	if isHandshakePayload(body) {
		adapter.HandleWebhook(w, r)
		return
	}

	// Verify signature before processing.
	if err := adapter.VerifySignature(r); err != nil {
		slog.Warn("channel: signature verification failed", //nolint:gosec // G706 -- platform sanitized (CR/LF stripped) and validated against registered adapter map
			"platform", safePlatform,
			"error", err,
		)
		http.Error(w, "channel: signature verification failed", http.StatusUnauthorized)
		return
	}

	// Restore body again after VerifySignature consumed it.
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Normalize to ChannelMessage.
	msg, err := adapter.Normalize(body)
	if err != nil {
		slog.Error("channel: normalization failed", //nolint:gosec // G706 -- platform sanitized (CR/LF stripped) and validated against registered adapter map
			"platform", safePlatform,
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
			slog.Error("channel: failed to create CloudEvent", //nolint:gosec // G706 -- platform sanitized (CR/LF stripped) and validated against registered adapter map
				"platform", safePlatform,
				"error", err,
			)
			// Still return 200 — the webhook was received.
		} else {
			subject := fmt.Sprintf("dojo.channel.message.%s", safePlatform)
			if pubErr := gw.bus.Publish(subject, evt); pubErr != nil {
				slog.Error("channel: failed to publish event", //nolint:gosec // G706 -- platform sanitized (CR/LF stripped) and validated against registered adapter map
					"platform", safePlatform,
					"subject", subject,
					"error", pubErr,
				)
			}
		}
	}

	// Return 200 immediately (Chatwoot pattern).
	w.WriteHeader(http.StatusOK)
}

// isHandshakePayload returns true when the body is a platform handshake
// request that must be answered before signature verification.
// Currently detects Slack url_verification challenges.
func isHandshakePayload(body []byte) bool {
	return strings.Contains(string(body), `"url_verification"`)
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
