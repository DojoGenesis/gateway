package event_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/event"
)

func newTestBus(t *testing.T) event.Bus {
	t.Helper()
	cfg := event.DefaultConfig()
	cfg.WAL.Enabled = true
	bus, err := event.NewBus(cfg)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

func TestNewBus(t *testing.T) {
	bus, err := event.NewBus(event.DefaultConfig())
	if err != nil {
		t.Fatalf("NewBus returned error: %v", err)
	}
	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestPublishSubscribe(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var received event.Event
	var wg sync.WaitGroup
	wg.Add(1)

	_, err := bus.Subscribe(ctx, event.EventFilter{
		Types: []string{event.EventToolCompleted},
	}, func(_ context.Context, e event.Event) error {
		received = e
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	evt := event.Event{
		ID:   "evt-1",
		Type: event.EventToolCompleted,
		Data: []byte("hello"),
		Time: time.Now(),
	}
	if err := bus.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	wg.Wait()
	if received.ID != "evt-1" {
		t.Errorf("received event ID: got %q, want %q", received.ID, "evt-1")
	}
}

func TestFilterBySource(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var count atomic.Int32

	_, err := bus.Subscribe(ctx, event.EventFilter{
		Sources: []string{"agent/scout-1"},
	}, func(_ context.Context, _ event.Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	bus.Publish(ctx, event.Event{ID: "1", Type: "x", Source: "agent/scout-1"})
	bus.Publish(ctx, event.Event{ID: "2", Type: "x", Source: "agent/scout-2"})
	bus.Publish(ctx, event.Event{ID: "3", Type: "x", Source: "agent/scout-1"})

	if got := count.Load(); got != 2 {
		t.Errorf("filtered count: got %d, want 2", got)
	}
}

func TestFilterBySubject(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var count atomic.Int32

	_, err := bus.Subscribe(ctx, event.EventFilter{
		Subjects: []string{"workflow/abc"},
	}, func(_ context.Context, _ event.Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	bus.Publish(ctx, event.Event{ID: "1", Type: "dojo.test", Subject: "workflow/abc"})
	bus.Publish(ctx, event.Event{ID: "2", Type: "dojo.test", Subject: "workflow/xyz"})

	if got := count.Load(); got != 1 {
		t.Errorf("filtered count: got %d, want 1", got)
	}
}

func TestEmptyFilterMatchesAll(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var count atomic.Int32

	_, err := bus.Subscribe(ctx, event.EventFilter{}, func(_ context.Context, _ event.Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	bus.Publish(ctx, event.Event{ID: "1", Type: "dojo.a"})
	bus.Publish(ctx, event.Event{ID: "2", Type: "dojo.b"})
	bus.Publish(ctx, event.Event{ID: "3", Type: "dojo.c"})

	if got := count.Load(); got != 3 {
		t.Errorf("empty filter count: got %d, want 3", got)
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var count atomic.Int32

	sub, err := bus.Subscribe(ctx, event.EventFilter{}, func(_ context.Context, _ event.Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	bus.Publish(ctx, event.Event{ID: "1", Type: "dojo.test"})
	if got := count.Load(); got != 1 {
		t.Fatalf("before unsub: got %d, want 1", got)
	}

	sub.Unsubscribe()

	bus.Publish(ctx, event.Event{ID: "2", Type: "dojo.test"})
	if got := count.Load(); got != 1 {
		t.Errorf("after unsub: got %d, want 1", got)
	}
}

func TestSubscribeNilHandler(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	_, err := bus.Subscribe(ctx, event.EventFilter{}, nil)
	if err == nil {
		t.Error("Subscribe with nil handler: expected error")
	}
}

func TestReplay(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	start := time.Now().Add(-time.Second)

	bus.Publish(ctx, event.Event{ID: "1", Type: event.EventToolCompleted})
	bus.Publish(ctx, event.Event{ID: "2", Type: event.EventAgentSpawned})
	bus.Publish(ctx, event.Event{ID: "3", Type: event.EventToolCompleted})

	ch, err := bus.Replay(ctx, event.EventFilter{Types: []string{event.EventToolCompleted}}, start)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	var events []event.Event
	for e := range ch {
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("Replay: got %d events, want 2", len(events))
	}
}

func TestReplayDisabledWAL(t *testing.T) {
	cfg := event.DefaultConfig()
	cfg.WAL.Enabled = false
	bus, err := event.NewBus(cfg)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	defer bus.Close()

	_, err = bus.Replay(context.Background(), event.EventFilter{}, time.Time{})
	if err == nil {
		t.Error("Replay with WAL disabled: expected error")
	}
}

func TestRequestReply(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	// Subscriber that replies to requests.
	_, err := bus.Subscribe(ctx, event.EventFilter{
		Types: []string{"ping"},
	}, func(ctx context.Context, e event.Event) error {
		reply := event.Event{
			ID:      "reply-1",
			Type:    "ping.reply",
			Subject: e.ID,
			Data:    []byte("pong"),
		}
		return bus.Publish(ctx, reply)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	reply, err := bus.Request(ctx, event.Event{
		ID:   "req-1",
		Type: "ping",
		Data: []byte("ping"),
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if string(reply.Data) != "pong" {
		t.Errorf("Reply data: got %q, want %q", reply.Data, "pong")
	}
}

func TestRequestTimeout(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	// No subscriber to reply.
	_, err := bus.Request(ctx, event.Event{
		ID:   "req-timeout",
		Type: "nobody-listening",
	}, 50*time.Millisecond)
	if err == nil {
		t.Error("Request without responder: expected timeout error")
	}
}

func TestStats(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	bus.Subscribe(ctx, event.EventFilter{}, func(_ context.Context, _ event.Event) error {
		return nil
	})
	bus.Subscribe(ctx, event.EventFilter{}, func(_ context.Context, _ event.Event) error {
		return nil
	})

	bus.Publish(ctx, event.Event{ID: "1", Type: "dojo.test"})

	stats := bus.Stats()
	if stats.EventsPublished != 1 {
		t.Errorf("EventsPublished: got %d, want 1", stats.EventsPublished)
	}
	if stats.EventsDelivered != 2 {
		t.Errorf("EventsDelivered: got %d, want 2", stats.EventsDelivered)
	}
	if stats.ActiveSubscribers != 2 {
		t.Errorf("ActiveSubscribers: got %d, want 2", stats.ActiveSubscribers)
	}
}

func TestConcurrentPublish(t *testing.T) {
	bus := newTestBus(t)
	ctx := context.Background()

	var count atomic.Int64
	bus.Subscribe(ctx, event.EventFilter{}, func(_ context.Context, _ event.Event) error {
		count.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bus.Publish(ctx, event.Event{ID: "concurrent", Type: "test"})
		}(i)
	}
	wg.Wait()

	if got := count.Load(); got != 100 {
		t.Errorf("concurrent publish count: got %d, want 100", got)
	}
}

func TestPublishAfterClose(t *testing.T) {
	cfg := event.DefaultConfig()
	cfg.WAL.Enabled = true
	bus, err := event.NewBus(cfg)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	bus.Close()

	err = bus.Publish(context.Background(), event.Event{ID: "late"})
	if err == nil {
		t.Error("Publish after close: expected error")
	}
}
