package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// NATSBus — production event bus adapter backed by an external NATS bus.
//
// The channel module is a separate Go module and cannot import runtime/event
// directly. Instead, NATSBus depends on two narrow interfaces (NATSPublisher,
// NATSSubscriber) that the wiring layer (cmd/dojo) satisfies by wrapping the
// real runtime/event.Bus.
//
// Subject convention: dojo.channel.{platform}.inbound
// ---------------------------------------------------------------------------

// NATSPublisher publishes events to the underlying NATS-based event bus.
// Implementations should adapt channel.Event to the bus's native event type.
// The subject follows the pattern "dojo.channel.{platform}.inbound".
type NATSPublisher interface {
	// PublishRaw publishes raw event bytes on the given subject.
	PublishRaw(ctx context.Context, subject string, data []byte) error
}

// NATSSubscriber subscribes to events from the underlying NATS-based event bus.
type NATSSubscriber interface {
	// SubscribeRaw registers a handler for events on the given subject pattern.
	// The handler receives the subject and raw event bytes.
	// Returns an unsubscribe function.
	SubscribeRaw(ctx context.Context, subjectPattern string, handler func(subject string, data []byte)) (func(), error)
}

// NATSBusOption configures a NATSBus.
type NATSBusOption func(*NATSBus)

// WithNATSSubscriber attaches a subscriber for inbound events.
func WithNATSSubscriber(sub NATSSubscriber) NATSBusOption {
	return func(b *NATSBus) {
		b.subscriber = sub
	}
}

// NATSBus implements EventPublisher using a NATS-backed event bus.
// It also supports Subscribe so ChannelBridge.BusHandler can register.
//
// Usage (production wiring in cmd/dojo):
//
//	eventBus, _ := event.NewBus(cfg)
//	adapter := NewNATSBusAdapter(eventBus)  // thin adapter in cmd/dojo
//	natsBus := channel.NewNATSBus(adapter, channel.WithNATSSubscriber(adapter))
//	gw := channel.NewWebhookGateway(natsBus, creds)
//	bridge := channel.NewChannelBridge(runner)
//	natsBus.Subscribe(bridge.BusHandler(ctx))
type NATSBus struct {
	publisher  NATSPublisher
	subscriber NATSSubscriber

	mu          sync.RWMutex
	subscribers []func(string, Event)
	unsubs      []func()
}

// NewNATSBus creates a NATSBus backed by the given publisher.
// Use WithNATSSubscriber to enable inbound event subscription.
func NewNATSBus(pub NATSPublisher, opts ...NATSBusOption) *NATSBus {
	b := &NATSBus{
		publisher: pub,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Publish serializes the event and publishes it to NATS on the subject
// derived from the event type. Implements EventPublisher.
func (b *NATSBus) Publish(subject string, evt Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("nats_bus: marshal event: %w", err)
	}

	if err := b.publisher.PublishRaw(context.Background(), subject, data); err != nil {
		return fmt.Errorf("nats_bus: publish: %w", err)
	}

	// Also deliver to local subscribers synchronously (same as InProcessBus
	// behavior) so the bridge receives events immediately.
	b.mu.RLock()
	subs := make([]func(string, Event), len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subs {
		sub(subject, evt)
	}

	return nil
}

// Subscribe registers fn to be called for every published event.
// This mirrors InProcessBus.Subscribe so BusHandler can be wired the same way.
func (b *NATSBus) Subscribe(fn func(subject string, evt Event)) {
	b.mu.Lock()
	b.subscribers = append(b.subscribers, fn)
	b.mu.Unlock()
}

// SubscribeNATS subscribes to inbound NATS events matching the given subject
// pattern (e.g. "dojo.channel.>") and delivers them to all local subscribers.
// This bridges real NATS messages back into the channel bridge.
// Call this once during startup if inbound events arrive via NATS (e.g. from
// other services publishing channel events).
func (b *NATSBus) SubscribeNATS(ctx context.Context, subjectPattern string) error {
	if b.subscriber == nil {
		return fmt.Errorf("nats_bus: no subscriber configured; use WithNATSSubscriber")
	}

	unsub, err := b.subscriber.SubscribeRaw(ctx, subjectPattern, func(subject string, data []byte) {
		var evt Event
		if err := json.Unmarshal(data, &evt); err != nil {
			slog.Warn("nats_bus: unmarshal inbound event", "error", err, "subject", subject)
			return
		}

		b.mu.RLock()
		subs := make([]func(string, Event), len(b.subscribers))
		copy(subs, b.subscribers)
		b.mu.RUnlock()

		for _, sub := range subs {
			sub(subject, evt)
		}
	})
	if err != nil {
		return fmt.Errorf("nats_bus: subscribe: %w", err)
	}

	b.mu.Lock()
	b.unsubs = append(b.unsubs, unsub)
	b.mu.Unlock()

	slog.Info("nats_bus: subscribed to NATS", "pattern", subjectPattern)
	return nil
}

// Close unsubscribes from all NATS subscriptions.
func (b *NATSBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, unsub := range b.unsubs {
		unsub()
	}
	b.unsubs = nil
}

// ChannelSubject returns the NATS subject for a given platform's inbound
// channel events: "dojo.channel.{platform}.inbound".
func ChannelSubject(platform string) string {
	return fmt.Sprintf("dojo.channel.%s.inbound", platform)
}

// ChannelSubjectWildcard returns the wildcard subject matching all channel
// events: "dojo.channel.>".
func ChannelSubjectWildcard() string {
	return "dojo.channel.>"
}

// NewChannelEvent creates a channel Event for a ChannelMessage, following
// the same pattern as ToCloudEvent in message.go.
func NewChannelEvent(msg *ChannelMessage) Event {
	data, _ := json.Marshal(msg)
	return Event{
		SpecVersion:     "1.0",
		Type:            fmt.Sprintf("dojo.channel.message.%s", msg.Platform),
		Source:          fmt.Sprintf("channel/%s", msg.Platform),
		ID:              uuid.New().String(),
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            data,
	}
}
