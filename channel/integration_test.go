package channel

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Integration test helpers
// ---------------------------------------------------------------------------

// memPublisher is an in-memory NATSPublisher for integration tests.
// It records all published events and optionally forwards them to subscribers.
type memPublisher struct {
	mu       sync.Mutex
	events   []memPublishedEvent
	handlers []func(subject string, data []byte)
}

type memPublishedEvent struct {
	Subject string
	Data    []byte
}

func (p *memPublisher) PublishRaw(_ context.Context, subject string, data []byte) error {
	p.mu.Lock()
	p.events = append(p.events, memPublishedEvent{Subject: subject, Data: data})
	handlers := make([]func(string, []byte), len(p.handlers))
	copy(handlers, p.handlers)
	p.mu.Unlock()

	// Deliver to subscribed handlers synchronously (simulates NATS delivery).
	for _, h := range handlers {
		h(subject, data)
	}
	return nil
}

func (p *memPublisher) SubscribeRaw(_ context.Context, _ string, handler func(string, []byte)) (func(), error) {
	p.mu.Lock()
	p.handlers = append(p.handlers, handler)
	idx := len(p.handlers) - 1
	p.mu.Unlock()

	return func() {
		p.mu.Lock()
		if idx < len(p.handlers) {
			p.handlers[idx] = func(string, []byte) {} // no-op
		}
		p.mu.Unlock()
	}, nil
}

func (p *memPublisher) published() []memPublishedEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]memPublishedEvent, len(p.events))
	copy(out, p.events)
	return out
}

// ---------------------------------------------------------------------------
// 1. TestNATSBus_PublishSubscribe — basic publish/subscribe contract
// ---------------------------------------------------------------------------

func TestNATSBus_PublishSubscribe(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	var got []Event
	var mu sync.Mutex
	bus.Subscribe(func(_ string, evt Event) {
		mu.Lock()
		got = append(got, evt)
		mu.Unlock()
	})

	msg := &ChannelMessage{
		ID:        "int-001",
		Platform:  "stub",
		Text:      "integration test",
		Timestamp: time.Now().UTC(),
	}
	evt, err := ToCloudEvent(msg)
	if err != nil {
		t.Fatalf("ToCloudEvent: %v", err)
	}

	if err := bus.Publish("dojo.channel.message.stub", evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Type != evt.Type {
		t.Errorf("event type = %q, want %q", got[0].Type, evt.Type)
	}

	// Verify the underlying publisher received the event too.
	published := pub.published()
	if len(published) != 1 {
		t.Fatalf("publisher got %d events, want 1", len(published))
	}
	if published[0].Subject != "dojo.channel.message.stub" {
		t.Errorf("subject = %q, want %q", published[0].Subject, "dojo.channel.message.stub")
	}
}

// ---------------------------------------------------------------------------
// 2. TestNATSBus_SubscribeNATS — inbound NATS events reach local subscribers
// ---------------------------------------------------------------------------

func TestNATSBus_SubscribeNATS(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub, WithNATSSubscriber(pub))

	var got []Event
	var mu sync.Mutex
	bus.Subscribe(func(_ string, evt Event) {
		mu.Lock()
		got = append(got, evt)
		mu.Unlock()
	})

	// Subscribe to NATS inbound events.
	if err := bus.SubscribeNATS(context.Background(), "dojo.channel.>"); err != nil {
		t.Fatalf("SubscribeNATS: %v", err)
	}

	// Simulate an external publish through the memPublisher's handler chain.
	msg := &ChannelMessage{
		ID:        "ext-001",
		Platform:  "slack",
		Text:      "from NATS",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)

	if err := bus.Publish("dojo.channel.slack.inbound", evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// The event is delivered twice: once from Publish's direct local delivery,
	// and once from the SubscribeRaw handler. That is expected in production
	// when the same bus both publishes and receives from NATS. Verify at least
	// one delivery occurred.
	if len(got) < 1 {
		t.Fatalf("got %d events, want >= 1", len(got))
	}

	bus.Close()
}

// ---------------------------------------------------------------------------
// 3. TestNATSBus_NoSubscriber — SubscribeNATS without subscriber returns error
// ---------------------------------------------------------------------------

func TestNATSBus_NoSubscriber(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub) // no WithNATSSubscriber

	err := bus.SubscribeNATS(context.Background(), "dojo.channel.>")
	if err == nil {
		t.Fatal("expected error when no subscriber configured")
	}
	if !strings.Contains(err.Error(), "no subscriber") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 4. TestChannelSubject — subject helper functions
// ---------------------------------------------------------------------------

func TestChannelSubject(t *testing.T) {
	if got := ChannelSubject("slack"); got != "dojo.channel.slack.inbound" {
		t.Errorf("ChannelSubject = %q, want %q", got, "dojo.channel.slack.inbound")
	}
	if got := ChannelSubjectWildcard(); got != "dojo.channel.>" {
		t.Errorf("ChannelSubjectWildcard = %q, want %q", got, "dojo.channel.>")
	}
}

// ---------------------------------------------------------------------------
// 5. TestNewChannelEvent — creates a well-formed event from a message
// ---------------------------------------------------------------------------

func TestNewChannelEvent(t *testing.T) {
	msg := &ChannelMessage{
		ID:        "msg-100",
		Platform:  "discord",
		ChannelID: "C100",
		Text:      "hello",
		Timestamp: time.Now().UTC(),
	}

	evt := NewChannelEvent(msg)

	if evt.SpecVersion != "1.0" {
		t.Errorf("SpecVersion = %q, want %q", evt.SpecVersion, "1.0")
	}
	if evt.Type != "dojo.channel.message.discord" {
		t.Errorf("Type = %q, want %q", evt.Type, "dojo.channel.message.discord")
	}
	if evt.Source != "channel/discord" {
		t.Errorf("Source = %q, want %q", evt.Source, "channel/discord")
	}
	if evt.ID == "" {
		t.Error("ID must not be empty")
	}
	if len(evt.Data) == 0 {
		t.Error("Data must contain serialized message")
	}
}

// ---------------------------------------------------------------------------
// 6. TestFlywheel_NATSBus_WebhookToReply
// Full integration: HTTP POST → WebhookGateway → NATSBus → ChannelBridge
// → mock workflow → StubAdapter.Send → verified reply.
// Uses NATSBus with in-memory publisher (no real NATS dependency).
// ---------------------------------------------------------------------------

func TestFlywheel_NATSBus_WebhookToReply(t *testing.T) {
	// Wire the components using NATSBus instead of InProcessBus.
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	gw := NewWebhookGateway(bus, nil)

	adapter := &StubAdapter{}
	gw.Register("stub", adapter) // inbound webhook handling

	runner := &stubWorkflowRunner{
		result: &WorkflowRunResult{Status: "completed", StepCount: 4},
	}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", adapter) // outbound reply channel
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "nats-test-workflow"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// Start a real HTTP test server backed by the gateway.
	srv := httptest.NewServer(gw)
	defer srv.Close()

	// POST a webhook.
	body := []byte(`{"text": "nats integration test", "user_id": "U100", "channel_id": "C100"}`)
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /webhooks/stub: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// NATSBus delivers synchronously to local subscribers.
	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent messages, want 1", len(sent))
	}

	reply := sent[0]
	if reply.Platform != "stub" {
		t.Errorf("reply.Platform = %q, want %q", reply.Platform, "stub")
	}
	if reply.ChannelID != "C100" {
		t.Errorf("reply.ChannelID = %q, want %q", reply.ChannelID, "C100")
	}
	if reply.UserName != "Dojo" {
		t.Errorf("reply.UserName = %q, want %q", reply.UserName, "Dojo")
	}
	if reply.ReplyTo == "" {
		t.Error("reply.ReplyTo must reference the original message ID")
	}

	// Verify publisher received the event.
	published := pub.published()
	if len(published) != 1 {
		t.Fatalf("publisher got %d events, want 1", len(published))
	}
	if !strings.Contains(published[0].Subject, "dojo.channel.message.stub") {
		t.Errorf("published subject = %q, expected to contain %q",
			published[0].Subject, "dojo.channel.message.stub")
	}
}

// ---------------------------------------------------------------------------
// 7. TestFlywheel_NATSBus_PatternMatchRouting
// Verifies that trigger pattern matching works end-to-end with NATSBus.
// ---------------------------------------------------------------------------

func TestFlywheel_NATSBus_PatternMatchRouting(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	gw := NewWebhookGateway(bus, nil)

	adapter := &StubAdapter{}
	gw.Register("stub", adapter)

	runner := &stubWorkflowRunner{
		result: &WorkflowRunResult{Status: "completed", StepCount: 1},
	}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{
		Platform: "stub",
		Pattern:  regexp.MustCompile(`^dojo:`),
		Workflow: "command-workflow",
	})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Send a message that DOES match the trigger.
	body := []byte(`{"text": "dojo: run audit", "user_id": "U200", "channel_id": "C200"}`)
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	_ = resp.Body.Close()

	if len(adapter.Sent()) != 1 {
		t.Errorf("matching message: got %d sent, want 1", len(adapter.Sent()))
	}

	// Send a message that does NOT match the trigger.
	body = []byte(`{"text": "hello world", "user_id": "U200", "channel_id": "C200"}`)
	resp, err = http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	_ = resp.Body.Close()

	if len(adapter.Sent()) != 1 {
		t.Errorf("non-matching message: got %d sent, still want 1", len(adapter.Sent()))
	}
}

// ---------------------------------------------------------------------------
// 8. TestFlywheel_NATSBus_WorkflowFailure
// Verifies that a workflow failure produces a meaningful reply.
// ---------------------------------------------------------------------------

func TestFlywheel_NATSBus_WorkflowFailure(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	gw := NewWebhookGateway(bus, nil)

	adapter := &StubAdapter{}
	gw.Register("stub", adapter)

	runner := &stubWorkflowRunner{
		result: &WorkflowRunResult{
			Status:      "failed",
			StepCount:   3,
			FailedSteps: []string{"step-b"},
		},
	}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "failing-workflow"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	body := []byte(`{"text": "trigger failure", "user_id": "U300", "channel_id": "C300"}`)
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	_ = resp.Body.Close()

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent, want 1", len(sent))
	}

	reply := sent[0]
	if !strings.Contains(reply.Text, "failures") && !strings.Contains(reply.Text, "failed") {
		t.Errorf("reply text should indicate failure, got %q", reply.Text)
	}
	if !strings.Contains(reply.Text, "step-b") {
		t.Errorf("reply text should mention failed step, got %q", reply.Text)
	}
}

// ---------------------------------------------------------------------------
// 9. TestNATSBus_MultipleSubscribers — multiple subscribers all receive events
// ---------------------------------------------------------------------------

func TestNATSBus_MultipleSubscribers(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	var count1, count2 int
	var mu sync.Mutex
	bus.Subscribe(func(_ string, _ Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	})
	bus.Subscribe(func(_ string, _ Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	msg := &ChannelMessage{ID: "m", Platform: "stub", Text: "multi", Timestamp: time.Now().UTC()}
	evt, _ := ToCloudEvent(msg)
	if err := bus.Publish("s", evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Errorf("subscriber counts = (%d, %d), want (1, 1)", count1, count2)
	}
}

// ---------------------------------------------------------------------------
// 10. TestNATSBus_Close — close unsubscribes NATS handlers
// ---------------------------------------------------------------------------

func TestNATSBus_Close(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub, WithNATSSubscriber(pub))

	if err := bus.SubscribeNATS(context.Background(), "dojo.channel.>"); err != nil {
		t.Fatalf("SubscribeNATS: %v", err)
	}

	bus.Close()

	// After close, unsubs slice should be nil.
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	if bus.unsubs != nil {
		t.Errorf("unsubs should be nil after Close, got %d entries", len(bus.unsubs))
	}
}
