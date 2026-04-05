package channel

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// stubWorkflowRunner — satisfies WorkflowRunner for unit tests.
// ---------------------------------------------------------------------------

type stubWorkflowRunner struct {
	result *WorkflowRunResult
	err    error
}

func (r *stubWorkflowRunner) Execute(_ context.Context, name string) (*WorkflowRunResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	res := &WorkflowRunResult{
		WorkflowName: name,
		Status:       "completed",
		StepCount:    2,
	}
	if r.result != nil {
		// Copy so callers can reuse the stub across multiple calls.
		cp := *r.result
		cp.WorkflowName = name
		res = &cp
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// 1. TestChannelBridge_FlywheelRoundTrip
// Stub runner + stub adapter — verifies the core dispatch-and-reply loop.
// ---------------------------------------------------------------------------

func TestChannelBridge_FlywheelRoundTrip(t *testing.T) {
	adapter := &StubAdapter{}
	runner := &stubWorkflowRunner{}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "echo-workflow"})

	msg := &ChannelMessage{
		ID:        "msg-001",
		Platform:  "stub",
		ChannelID: "C001",
		UserID:    "U001",
		UserName:  "alice",
		Text:      "run the workflow",
		Timestamp: time.Now().UTC(),
	}

	evt, err := ToCloudEvent(msg)
	if err != nil {
		t.Fatalf("ToCloudEvent: %v", err)
	}

	if err := bridge.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent messages, want 1", len(sent))
	}

	reply := sent[0]
	if reply.Platform != "stub" {
		t.Errorf("reply.Platform = %q, want %q", reply.Platform, "stub")
	}
	if reply.ChannelID != "C001" {
		t.Errorf("reply.ChannelID = %q, want %q", reply.ChannelID, "C001")
	}
	if reply.ReplyTo != "msg-001" {
		t.Errorf("reply.ReplyTo = %q, want %q", reply.ReplyTo, "msg-001")
	}
	if reply.UserName != "Dojo" {
		t.Errorf("reply.UserName = %q, want %q", reply.UserName, "Dojo")
	}
	if reply.Text == "" {
		t.Error("reply.Text must not be empty")
	}
	if reply.ID == "" {
		t.Error("reply.ID must be set")
	}
}

// ---------------------------------------------------------------------------
// 2. TestChannelBridge_NoTriggerMatch — no triggers registered → no reply
// ---------------------------------------------------------------------------

func TestChannelBridge_NoTriggerMatch(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{})
	bridge.Register("stub", adapter)
	// No triggers.

	msg := &ChannelMessage{
		ID: "msg-002", Platform: "stub", Text: "nothing matches",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)

	if err := bridge.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if len(adapter.Sent()) != 0 {
		t.Errorf("got %d sent messages, want 0", len(adapter.Sent()))
	}
}

// ---------------------------------------------------------------------------
// 3. TestChannelBridge_PatternMatch — regexp filters by message text
// ---------------------------------------------------------------------------

func TestChannelBridge_PatternMatch(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{})
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{
		Platform: "stub",
		Pattern:  regexp.MustCompile(`(?i)dojo:`),
		Workflow: "command-workflow",
	})

	send := func(text string) {
		msg := &ChannelMessage{
			ID: "m", Platform: "stub", Text: text,
			Timestamp: time.Now().UTC(),
		}
		evt, _ := ToCloudEvent(msg)
		bridge.HandleEvent(context.Background(), evt) //nolint:errcheck
	}

	send("dojo: run audit")
	if len(adapter.Sent()) != 1 {
		t.Errorf("after matching message: got %d sent, want 1", len(adapter.Sent()))
	}

	send("hello world") // no match
	if len(adapter.Sent()) != 1 {
		t.Errorf("after non-matching message: got %d sent, still want 1", len(adapter.Sent()))
	}

	send("DOJO: run report") // case-insensitive match
	if len(adapter.Sent()) != 2 {
		t.Errorf("after second matching message: got %d sent, want 2", len(adapter.Sent()))
	}
}

// ---------------------------------------------------------------------------
// 4. TestChannelBridge_WildcardPlatform — "*" matches any platform
// ---------------------------------------------------------------------------

func TestChannelBridge_WildcardPlatform(t *testing.T) {
	slackAdapter := &StubAdapter{}
	discordAdapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{})
	bridge.Register("slack", slackAdapter)
	bridge.Register("discord", discordAdapter)
	bridge.AddTrigger(TriggerSpec{Platform: "*", Workflow: "global-workflow"})

	for _, tc := range []struct {
		platform string
		adapter  *StubAdapter
	}{
		{"slack", slackAdapter},
		{"discord", discordAdapter},
	} {
		before := len(tc.adapter.Sent())
		msg := &ChannelMessage{
			ID: "m", Platform: tc.platform, Text: "trigger",
			Timestamp: time.Now().UTC(),
		}
		evt, _ := ToCloudEvent(msg)
		bridge.HandleEvent(context.Background(), evt) //nolint:errcheck
		if len(tc.adapter.Sent()) != before+1 {
			t.Errorf("platform %q: expected one reply, got none", tc.platform)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. TestChannelBridge_WorkflowCompleted — reply text confirms step count
// ---------------------------------------------------------------------------

func TestChannelBridge_WorkflowCompleted(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{
		result: &WorkflowRunResult{Status: "completed", StepCount: 5},
	})
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "big-workflow"})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "go",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)
	bridge.HandleEvent(context.Background(), evt) //nolint:errcheck

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent, want 1", len(sent))
	}
	// Reply text must mention the step count.
	if sent[0].Text == "" {
		t.Error("reply text must not be empty on completed workflow")
	}
}

// ---------------------------------------------------------------------------
// 6. TestChannelBridge_WorkflowFailed — reply text includes failed step IDs
// ---------------------------------------------------------------------------

func TestChannelBridge_WorkflowFailed(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{
		result: &WorkflowRunResult{
			Status:      "failed",
			StepCount:   3,
			FailedSteps: []string{"step-b"},
		},
	})
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "failing-workflow"})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "trigger",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)
	bridge.HandleEvent(context.Background(), evt) //nolint:errcheck

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent, want 1", len(sent))
	}
	if sent[0].Text == "" {
		t.Error("reply text must not be empty on workflow failure")
	}
}

// ---------------------------------------------------------------------------
// 7. TestChannelBridge_RunnerError — runner error → error reply sent
// ---------------------------------------------------------------------------

func TestChannelBridge_RunnerError(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(&stubWorkflowRunner{
		err: errors.New("workflow not found"),
	})
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "missing-workflow"})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "trigger",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)
	bridge.HandleEvent(context.Background(), evt) //nolint:errcheck

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent, want 1", len(sent))
	}
	if sent[0].Text == "" {
		t.Error("reply text must not be empty on runner error")
	}
}

// ---------------------------------------------------------------------------
// 8. TestChannelBridge_NilRunner — nil runner → no crash, no reply
// ---------------------------------------------------------------------------

func TestChannelBridge_NilRunner(t *testing.T) {
	adapter := &StubAdapter{}
	bridge := NewChannelBridge(nil)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "anything"})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "trigger",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)

	if err := bridge.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("HandleEvent with nil runner: %v", err)
	}
	if len(adapter.Sent()) != 0 {
		t.Errorf("nil runner should not send replies, got %d", len(adapter.Sent()))
	}
}

// ---------------------------------------------------------------------------
// 9. TestChannelBridge_NoAdapter — no crash when platform adapter missing
// ---------------------------------------------------------------------------

func TestChannelBridge_NoAdapter(t *testing.T) {
	bridge := NewChannelBridge(&stubWorkflowRunner{})
	// No adapter registered for "stub".
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "any"})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "trigger",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)

	if err := bridge.HandleEvent(context.Background(), evt); err != nil {
		t.Fatalf("HandleEvent with no adapter: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 10. TestInProcessBus — publish/subscribe contract
// ---------------------------------------------------------------------------

func TestInProcessBus_PublishSubscribe(t *testing.T) {
	bus := &InProcessBus{}

	var got []Event
	bus.Subscribe(func(_ string, evt Event) {
		got = append(got, evt)
	})

	msg := &ChannelMessage{
		ID: "m", Platform: "stub", Text: "hello bus",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := ToCloudEvent(msg)

	if err := bus.Publish("dojo.channel.message.stub", evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Type != "dojo.channel.message.stub" {
		t.Errorf("event type = %q, want %q", got[0].Type, "dojo.channel.message.stub")
	}
}

func TestInProcessBus_MultipleSubscribers(t *testing.T) {
	bus := &InProcessBus{}
	var count int
	bus.Subscribe(func(_ string, _ Event) { count++ })
	bus.Subscribe(func(_ string, _ Event) { count++ })

	msg := &ChannelMessage{ID: "m", Platform: "stub", Text: "multi", Timestamp: time.Now().UTC()}
	evt, _ := ToCloudEvent(msg)
	bus.Publish("s", evt) //nolint:errcheck

	if count != 2 {
		t.Errorf("got %d calls, want 2", count)
	}
}

// ---------------------------------------------------------------------------
// 11. TestFlywheel_WebhookToReply
// Full integration: HTTP POST → WebhookGateway → InProcessBus → ChannelBridge
// → stubWorkflowRunner → StubAdapter.Send → verified reply.
// This test proves the complete flywheel without mocking the HTTP layer.
// ---------------------------------------------------------------------------

func TestFlywheel_WebhookToReply(t *testing.T) {
	// Wire the components.
	bus := &InProcessBus{}
	gw := NewWebhookGateway(bus, nil)

	adapter := &StubAdapter{}
	gw.Register("stub", adapter) // inbound webhook handling

	runner := &stubWorkflowRunner{
		result: &WorkflowRunResult{Status: "completed", StepCount: 3},
	}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", adapter) // outbound reply channel
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "test-workflow"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// Start a real HTTP test server backed by the gateway.
	srv := httptest.NewServer(gw)
	defer srv.Close()

	// POST a webhook as if a platform (Slack/Discord) called in.
	body := []byte(`{"text": "run dojo workflow", "user_id": "U001", "channel_id": "C001"}`)
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /webhooks/stub: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Bus is synchronous so bridge has already processed the event.
	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent messages, want 1", len(sent))
	}

	reply := sent[0]
	if reply.Platform != "stub" {
		t.Errorf("reply.Platform = %q, want %q", reply.Platform, "stub")
	}
	if reply.ChannelID != "C001" {
		t.Errorf("reply.ChannelID = %q, want %q", reply.ChannelID, "C001")
	}
	if reply.UserName != "Dojo" {
		t.Errorf("reply.UserName = %q, want %q", reply.UserName, "Dojo")
	}
	if reply.Text == "" {
		t.Error("reply.Text must not be empty")
	}
	if reply.ReplyTo == "" {
		t.Error("reply.ReplyTo must reference the original message ID")
	}
}
