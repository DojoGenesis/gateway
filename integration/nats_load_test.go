package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Test 4: NATS bus under load
//
// Publish 100 synthetic messages to dojo.channel.message.slack in 10 seconds.
// Assert all 100 acknowledged by JetStream durable consumer.
// Assert zero goroutine leaks (before/after comparison).
// Assert no drops.
// ---------------------------------------------------------------------------

func TestNATSBus_Load_100Messages(t *testing.T) {
	const messageCount = 100

	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub, channel.WithNATSSubscriber(pub))

	// Track received messages.
	var receivedCount atomic.Int64
	var receivedMu sync.Mutex
	receivedIDs := make(map[string]bool, messageCount)

	bus.Subscribe(func(_ string, evt channel.Event) {
		receivedCount.Add(1)

		// Extract message ID from CloudEvent data.
		var msg channel.ChannelMessage
		if err := json.Unmarshal(evt.Data, &msg); err == nil {
			receivedMu.Lock()
			receivedIDs[msg.ID] = true
			receivedMu.Unlock()
		}
	})

	// Record goroutine count before the load test.
	runtime.GC()
	goroutinesBefore := runtime.NumGoroutine()

	// Publish 100 messages in rapid succession (well within 10 seconds).
	start := time.Now()

	for i := 0; i < messageCount; i++ {
		msg := &channel.ChannelMessage{
			ID:        fmt.Sprintf("load-msg-%03d", i),
			Platform:  "slack",
			ChannelID: "C_LOAD",
			UserID:    "U_LOAD",
			Text:      fmt.Sprintf("load test message %d", i),
			Timestamp: time.Now().UTC(),
		}

		evt, err := channel.ToCloudEvent(msg)
		if err != nil {
			t.Fatalf("ToCloudEvent[%d]: %v", i, err)
		}

		subject := "dojo.channel.message.slack"
		if err := bus.Publish(subject, evt); err != nil {
			t.Fatalf("Publish[%d]: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	t.Logf("published %d messages in %v", messageCount, elapsed)

	if elapsed > 10*time.Second {
		t.Errorf("publishing took %v, want < 10s", elapsed)
	}

	// --- Assert: all 100 acknowledged ---
	ackCount := pub.acknowledged()
	if ackCount != messageCount {
		t.Errorf("acknowledged = %d, want %d", ackCount, messageCount)
	}

	// --- Assert: all 100 received by subscriber ---
	received := receivedCount.Load()
	if received != messageCount {
		t.Errorf("received = %d, want %d", received, messageCount)
	}

	// --- Assert: zero drops (check unique IDs) ---
	receivedMu.Lock()
	uniqueReceived := len(receivedIDs)
	receivedMu.Unlock()

	if uniqueReceived != messageCount {
		t.Errorf("unique received = %d, want %d (drops detected)", uniqueReceived, messageCount)
	}

	// --- Assert: publisher received all 100 events ---
	published := pub.published()
	if len(published) != messageCount {
		t.Errorf("publisher got %d events, want %d", len(published), messageCount)
	}

	// Verify all published events are on the correct subject.
	for i, p := range published {
		if p.Subject != "dojo.channel.message.slack" {
			t.Errorf("event[%d] subject = %q, want %q", i, p.Subject, "dojo.channel.message.slack")
		}
	}

	// --- Cleanup ---
	bus.Close()

	// Allow goroutines to settle.
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	// --- Assert: zero goroutine leaks ---
	goroutinesAfter := runtime.NumGoroutine()

	// Allow a small margin for runtime goroutines.
	leakThreshold := goroutinesBefore + 5
	if goroutinesAfter > leakThreshold {
		t.Errorf("goroutine leak: before=%d, after=%d (threshold=%d)",
			goroutinesBefore, goroutinesAfter, leakThreshold)
	} else {
		t.Logf("goroutines: before=%d, after=%d (OK)", goroutinesBefore, goroutinesAfter)
	}
}

func TestNATSBus_Load_ConcurrentPublish(t *testing.T) {
	const (
		publisherCount = 10
		perPublisher   = 10
		totalMessages  = publisherCount * perPublisher
	)

	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)

	var receivedCount atomic.Int64
	bus.Subscribe(func(_ string, _ channel.Event) {
		receivedCount.Add(1)
	})

	// Record goroutine count before.
	runtime.GC()
	goroutinesBefore := runtime.NumGoroutine()

	// Launch concurrent publishers.
	var wg sync.WaitGroup
	for p := 0; p < publisherCount; p++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			for i := 0; i < perPublisher; i++ {
				msg := &channel.ChannelMessage{
					ID:        fmt.Sprintf("conc-%d-%d", publisherID, i),
					Platform:  "slack",
					ChannelID: "C_CONC",
					Text:      fmt.Sprintf("concurrent message %d-%d", publisherID, i),
					Timestamp: time.Now().UTC(),
				}
				evt, _ := channel.ToCloudEvent(msg)
				bus.Publish("dojo.channel.message.slack", evt) //nolint:errcheck
			}
		}(p)
	}

	wg.Wait()

	// Verify all messages received.
	received := receivedCount.Load()
	if received != int64(totalMessages) {
		t.Errorf("received = %d, want %d (concurrent drops detected)", received, totalMessages)
	}

	// Verify publisher received all events.
	published := pub.published()
	if len(published) != totalMessages {
		t.Errorf("publisher got %d, want %d", len(published), totalMessages)
	}

	// Check for goroutine leaks.
	bus.Close()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()
	if goroutinesAfter > goroutinesBefore+5 {
		t.Errorf("goroutine leak: before=%d, after=%d", goroutinesBefore, goroutinesAfter)
	}
}

func TestNATSBus_Load_WithBridge(t *testing.T) {
	const messageCount = 50

	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)

	adapter := &channel.StubAdapter{}
	runner := &stubWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 1},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "load-bridge-test"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// Publish messages through the full flywheel.
	for i := 0; i < messageCount; i++ {
		msg := &channel.ChannelMessage{
			ID:        fmt.Sprintf("bridge-load-%03d", i),
			Platform:  "stub",
			ChannelID: "C_LOAD_BRIDGE",
			UserID:    "U_LOAD",
			Text:      fmt.Sprintf("bridge load message %d", i),
			Timestamp: time.Now().UTC(),
		}
		evt, _ := channel.ToCloudEvent(msg)
		bus.Publish("dojo.channel.message.stub", evt) //nolint:errcheck
	}

	// Verify all messages produced replies.
	sent := adapter.Sent()
	if len(sent) != messageCount {
		t.Errorf("adapter sent %d replies, want %d", len(sent), messageCount)
	}

	// Verify the runner was called for each message.
	executed := runner.executedWorkflows()
	if len(executed) != messageCount {
		t.Errorf("runner executed %d workflows, want %d", len(executed), messageCount)
	}

	bus.Close()
}
