package actor_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/actor"
)

// testHandler is a simple AgentHandler for testing.
type testHandler struct {
	onMessage func(ctx context.Context, self *actor.AgentRef, msg actor.Message) error
	started   atomic.Bool
	stopped   atomic.Bool
}

func (h *testHandler) HandleMessage(ctx context.Context, self *actor.AgentRef, msg actor.Message) error {
	if h.onMessage != nil {
		return h.onMessage(ctx, self, msg)
	}
	// Reply if requested.
	if msg.ReplyTo != nil {
		msg.ReplyTo <- actor.Message{
			ID:      "reply-" + msg.ID,
			Type:    actor.MessageResponse,
			Payload: []byte("ok"),
		}
	}
	return nil
}

func (h *testHandler) OnStart(_ context.Context, _ *actor.AgentRef) error {
	h.started.Store(true)
	return nil
}

func (h *testHandler) OnStop(_ context.Context, _ *actor.AgentRef) error {
	h.stopped.Store(true)
	return nil
}

func newTestSupervisor(t *testing.T) actor.Supervisor {
	t.Helper()
	sup, err := actor.NewSupervisor(actor.DefaultConfig())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	t.Cleanup(func() { sup.Close() })
	return sup
}

func TestNewSupervisor(t *testing.T) {
	sup, err := actor.NewSupervisor(actor.DefaultConfig())
	if err != nil {
		t.Fatalf("NewSupervisor returned error: %v", err)
	}
	if sup == nil {
		t.Fatal("NewSupervisor returned nil")
	}
	if err := sup.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestSpawnAndList(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{}
	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "agent-1",
		Handler: handler,
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if ref == nil {
		t.Fatal("Spawn returned nil ref")
	}
	if ref.ID != "agent-1" {
		t.Errorf("ref.ID: got %q, want %q", ref.ID, "agent-1")
	}

	// Give OnStart time to run.
	time.Sleep(50 * time.Millisecond)

	if !handler.started.Load() {
		t.Error("OnStart was not called")
	}

	agents, err := sup.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("List: got %d agents, want 1", len(agents))
	}
}

func TestSpawnDuplicateID(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{}
	_, err := sup.Spawn(ctx, actor.SpawnOptions{ID: "dup", Handler: handler})
	if err != nil {
		t.Fatalf("First spawn: %v", err)
	}

	_, err = sup.Spawn(ctx, actor.SpawnOptions{ID: "dup", Handler: handler})
	if err == nil {
		t.Error("Duplicate spawn: expected error")
	}
}

func TestSpawnMissingHandler(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	_, err := sup.Spawn(ctx, actor.SpawnOptions{ID: "no-handler"})
	if err == nil {
		t.Error("Spawn without handler: expected error")
	}
}

func TestSendMessage(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var received atomic.Int32
	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			received.Add(1)
			return nil
		},
	}

	ref, _ := sup.Spawn(ctx, actor.SpawnOptions{ID: "msg-agent", Handler: handler})
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 5; i++ {
		err := sup.Send(ctx, ref, actor.Message{ID: "m", Payload: []byte("hi")})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)
	if got := received.Load(); got != 5 {
		t.Errorf("messages received: got %d, want 5", got)
	}
}

func TestRequestReply(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{} // default handler replies
	ref, _ := sup.Spawn(ctx, actor.SpawnOptions{ID: "reply-agent", Handler: handler})
	time.Sleep(50 * time.Millisecond)

	reply, err := sup.Request(ctx, ref, actor.Message{ID: "req-1"}, 2*time.Second)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if reply.Type != actor.MessageResponse {
		t.Errorf("reply type: got %d, want %d", reply.Type, actor.MessageResponse)
	}
}

func TestShutdown(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{}
	ref, _ := sup.Spawn(ctx, actor.SpawnOptions{ID: "shutdown-agent", Handler: handler})
	time.Sleep(50 * time.Millisecond)

	if err := sup.Shutdown(ctx, ref); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	agents, _ := sup.List(ctx)
	if len(agents) != 0 {
		t.Errorf("List after shutdown: got %d agents, want 0", len(agents))
	}
}

func TestSendToNonexistent(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	err := sup.Send(ctx, &actor.AgentRef{ID: "ghost"}, actor.Message{})
	if err == nil {
		t.Error("Send to nonexistent agent: expected error")
	}
}

func TestStats(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{}
	sup.Spawn(ctx, actor.SpawnOptions{ID: "stats-1", Handler: handler})
	sup.Spawn(ctx, actor.SpawnOptions{ID: "stats-2", Handler: handler})

	stats := sup.Stats()
	if stats.ActiveAgents != 2 {
		t.Errorf("ActiveAgents: got %d, want 2", stats.ActiveAgents)
	}
	if stats.TotalSpawned != 2 {
		t.Errorf("TotalSpawned: got %d, want 2", stats.TotalSpawned)
	}
}

func TestSetStrategy(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{}
	ref, _ := sup.Spawn(ctx, actor.SpawnOptions{ID: "strat-agent", Handler: handler})
	time.Sleep(50 * time.Millisecond)

	newStrat := actor.SupervisionStrategy{
		MaxRestarts:   10,
		RestartWindow: 30 * time.Second,
		Backoff:       actor.BackoffConfig{Initial: 50 * time.Millisecond, Max: 5 * time.Second, Multiplier: 1.5},
		OnExhausted:   actor.PolicyStop,
	}
	if err := sup.SetStrategy(ref, newStrat); err != nil {
		t.Fatalf("SetStrategy: %v", err)
	}
}

func TestConcurrentSpawn(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			handler := &testHandler{}
			_, err := sup.Spawn(ctx, actor.SpawnOptions{
				ID:      fmt.Sprintf("concurrent-%d", i),
				Handler: handler,
			})
			if err != nil {
				t.Errorf("concurrent Spawn %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	agents, _ := sup.List(ctx)
	if len(agents) != 20 {
		t.Errorf("agents after concurrent spawn: got %d, want 20", len(agents))
	}
}

// --- Failure injection tests ---

func TestPanicRecovery(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var callCount atomic.Int32
	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			n := callCount.Add(1)
			if n == 1 {
				panic("test panic")
			}
			return nil
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "panic-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   3,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 10 * time.Millisecond, Max: 100 * time.Millisecond, Multiplier: 1.5},
			OnExhausted:   actor.PolicyStop,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // wait for OnStart

	// First message triggers panic → restart.
	sup.Send(ctx, ref, actor.Message{ID: "msg-1", Payload: []byte("boom")})
	time.Sleep(200 * time.Millisecond) // wait for restart + backoff

	// Second message should succeed (agent restarted).
	sup.Send(ctx, ref, actor.Message{ID: "msg-2", Payload: []byte("ok")})
	time.Sleep(100 * time.Millisecond)

	if got := callCount.Load(); got < 2 {
		t.Errorf("call count: got %d, want >= 2 (panic + success)", got)
	}

	stats := sup.Stats()
	if stats.TotalRestarted < 1 {
		t.Errorf("TotalRestarted: got %d, want >= 1", stats.TotalRestarted)
	}
}

func TestErrorTriggersRestart(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var callCount atomic.Int32
	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			n := callCount.Add(1)
			if n == 1 {
				return fmt.Errorf("deliberate error")
			}
			return nil
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "error-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   3,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 10 * time.Millisecond, Max: 100 * time.Millisecond, Multiplier: 1.5},
			OnExhausted:   actor.PolicyStop,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// First message returns error → restart.
	sup.Send(ctx, ref, actor.Message{ID: "msg-1"})
	time.Sleep(200 * time.Millisecond)

	// Second message should succeed.
	sup.Send(ctx, ref, actor.Message{ID: "msg-2"})
	time.Sleep(100 * time.Millisecond)

	if got := callCount.Load(); got < 2 {
		t.Errorf("call count: got %d, want >= 2", got)
	}
}

func TestMaxRestartsExhausted_Stop(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			panic("always panic")
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "stop-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   2,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 10 * time.Millisecond, Max: 50 * time.Millisecond, Multiplier: 1.5},
			OnExhausted:   actor.PolicyStop,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Send messages to trigger panics. Each panic triggers a restart until exhausted.
	for i := 0; i < 5; i++ {
		sup.Send(ctx, ref, actor.Message{ID: fmt.Sprintf("m-%d", i)})
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// Agent should be stopped (OnStop called).
	if !handler.stopped.Load() {
		t.Error("expected OnStop to be called after restart exhaustion")
	}
}

func TestMaxRestartsExhausted_LogAndContinue(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var callCount atomic.Int32
	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			n := callCount.Add(1)
			if n <= 5 {
				panic("keep panicking")
			}
			return nil
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "continue-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   2,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 10 * time.Millisecond, Max: 50 * time.Millisecond, Multiplier: 1.5},
			OnExhausted:   actor.PolicyLogAndContinue,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Send messages to trigger panics past the MaxRestarts threshold.
	for i := 0; i < 8; i++ {
		sup.Send(ctx, ref, actor.Message{ID: fmt.Sprintf("m-%d", i)})
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// With LogAndContinue, the agent should still be in the list (counter resets).
	agents, _ := sup.List(ctx)
	found := false
	for _, a := range agents {
		if a.ID == "continue-agent" {
			found = true
		}
	}
	if !found {
		t.Error("agent should still be listed with PolicyLogAndContinue")
	}
}

func TestBackoffTiming(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	var timestamps []time.Time
	var mu sync.Mutex

	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			mu.Lock()
			timestamps = append(timestamps, time.Now())
			mu.Unlock()
			panic("timed panic")
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "backoff-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   5,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 50 * time.Millisecond, Max: 1 * time.Second, Multiplier: 2.0},
			OnExhausted:   actor.PolicyStop,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Send multiple messages to trigger panics. After each panic+restart the next
	// message will be handled.
	for i := 0; i < 4; i++ {
		sup.Send(ctx, ref, actor.Message{ID: fmt.Sprintf("m-%d", i)})
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(timestamps) < 3 {
		t.Fatalf("expected at least 3 panic timestamps, got %d", len(timestamps))
	}

	// Verify gaps are increasing (backoff).
	for i := 2; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		prevGap := timestamps[i-1].Sub(timestamps[i-2])
		// Allow some slack for scheduling jitter.
		if gap < prevGap-100*time.Millisecond {
			t.Errorf("backoff gap[%d]=%v should be >= gap[%d]=%v (increasing)", i, gap, i-1, prevGap)
		}
	}
}

func TestRequestTimeoutNoLeak(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	// Handler that never replies — simulates crash before reply.
	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, msg actor.Message) error {
			// Intentionally don't send on ReplyTo.
			return nil
		},
	}

	ref, _ := sup.Spawn(ctx, actor.SpawnOptions{ID: "leak-agent", Handler: handler})
	time.Sleep(50 * time.Millisecond)

	// Request should time out, not block forever.
	_, err := sup.Request(ctx, ref, actor.Message{ID: "req-leak"}, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error from Request with no reply")
	}
}

func TestMaxRestartsExhausted_Escalate(t *testing.T) {
	sup := newTestSupervisor(t)
	ctx := context.Background()

	handler := &testHandler{
		onMessage: func(_ context.Context, _ *actor.AgentRef, _ actor.Message) error {
			panic("escalate panic")
		},
	}

	ref, err := sup.Spawn(ctx, actor.SpawnOptions{
		ID:      "escalate-agent",
		Handler: handler,
		Strategy: actor.SupervisionStrategy{
			MaxRestarts:   2,
			RestartWindow: 10 * time.Second,
			Backoff:       actor.BackoffConfig{Initial: 10 * time.Millisecond, Max: 50 * time.Millisecond, Multiplier: 1.5},
			OnExhausted:   actor.PolicyEscalate,
		},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 5; i++ {
		sup.Send(ctx, ref, actor.Message{ID: fmt.Sprintf("m-%d", i)})
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// Agent should be stopped via escalation (OnStop called).
	if !handler.stopped.Load() {
		t.Error("expected OnStop to be called after escalation")
	}

	// Verify TotalFailed was NOT double-counted.
	stats := sup.Stats()
	if stats.TotalFailed < 1 {
		t.Errorf("TotalFailed: got %d, want >= 1", stats.TotalFailed)
	}
}
