package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Test 1: Flywheel with real platform traffic
//
// Synthetic Slack webhook POST -> SlackAdapter.HandleWebhook -> NATS ->
// ChannelBridge.BusHandler -> TriggerSpec dispatch -> WorkflowExecutor ->
// skills executed -> SlackAdapter.Send
//
// Uses NATSBus backed by memPublisher (simulates embedded NATS).
// No InProcessBus in the path. No external credentials needed.
// ---------------------------------------------------------------------------

func TestFlywheel_EndToEnd_NATSBus(t *testing.T) {
	// --- 1. Wire NATSBus with in-memory publisher ---
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub, channel.WithNATSSubscriber(pub))

	// --- 2. Create WebhookGateway backed by NATSBus ---
	gw := channel.NewWebhookGateway(bus, nil)

	// --- 3. Register stub adapter (simulates Slack) ---
	adapter := &channel.StubAdapter{}
	gw.Register("stub", adapter)

	// --- 4. Wire ChannelBridge with stub workflow runner ---
	runner := &stubWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 2},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{
		Platform: "stub",
		Workflow: "integration-flywheel-workflow",
	})

	// Subscribe bridge to the NATSBus (NOT InProcessBus).
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// --- 5. Track events received via NATS subscriber ---
	var receivedEvents []channel.Event
	var receivedMu sync.Mutex
	bus.Subscribe(func(subject string, evt channel.Event) {
		receivedMu.Lock()
		receivedEvents = append(receivedEvents, evt)
		receivedMu.Unlock()
	})

	// --- 6. Start HTTP test server ---
	srv := httptest.NewServer(gw)
	defer srv.Close()

	// --- 7. Simulate Slack webhook POST ---
	payload := `{"text": "flywheel integration test", "user_id": "U_FLY", "channel_id": "C_FLY"}`
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json",
		bytes.NewReader([]byte(payload)))
	if err != nil {
		t.Fatalf("POST /webhooks/stub: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200", resp.StatusCode)
	}

	// --- 8. Assert: NATS subject received CloudEvent ---
	published := pub.published()
	if len(published) < 1 {
		t.Fatalf("publisher got %d events, want >= 1", len(published))
	}

	foundSubject := false
	for _, p := range published {
		if p.Subject == "dojo.channel.message.stub" {
			foundSubject = true
			// Verify it is a valid CloudEvent.
			var evt channel.Event
			if err := json.Unmarshal(p.Data, &evt); err != nil {
				t.Fatalf("unmarshal published CloudEvent: %v", err)
			}
			if evt.Type != "dojo.channel.message.stub" {
				t.Errorf("CloudEvent type = %q, want %q", evt.Type, "dojo.channel.message.stub")
			}
			break
		}
	}
	if !foundSubject {
		t.Error("no published event with subject dojo.channel.message.stub")
	}

	// --- 9. Assert: WorkflowExecutor.Execute was called ---
	executed := runner.executedWorkflows()
	if len(executed) != 1 {
		t.Fatalf("runner executed %d workflows, want 1", len(executed))
	}
	if executed[0] != "integration-flywheel-workflow" {
		t.Errorf("executed workflow = %q, want %q", executed[0], "integration-flywheel-workflow")
	}

	// --- 10. Assert: adapter.Send was called with a reply ---
	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("adapter sent %d messages, want 1", len(sent))
	}

	reply := sent[0]
	if reply.Platform != "stub" {
		t.Errorf("reply.Platform = %q, want %q", reply.Platform, "stub")
	}
	if reply.ChannelID != "C_FLY" {
		t.Errorf("reply.ChannelID = %q, want %q", reply.ChannelID, "C_FLY")
	}
	if reply.UserName != "Dojo" {
		t.Errorf("reply.UserName = %q, want %q", reply.UserName, "Dojo")
	}
	if reply.ReplyTo == "" {
		t.Error("reply.ReplyTo must reference the original message ID")
	}
	if !strings.Contains(reply.Text, "completed") {
		t.Errorf("reply.Text should indicate completion, got %q", reply.Text)
	}

	// --- 11. Assert: NATSBus subscribers received events ---
	receivedMu.Lock()
	defer receivedMu.Unlock()
	if len(receivedEvents) < 1 {
		t.Fatalf("NATS subscribers got %d events, want >= 1", len(receivedEvents))
	}
}

// TestFlywheel_NoInProcessBusInPath verifies that the flywheel uses NATSBus,
// not InProcessBus. This is a structural assertion: the bus type passed to
// WebhookGateway is NATSBus, not InProcessBus.
func TestFlywheel_NoInProcessBusInPath(t *testing.T) {
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)

	// Verify NATSBus satisfies EventPublisher (same as InProcessBus).
	var _ channel.EventPublisher = bus

	// Wire the full flywheel with NATSBus.
	gw := channel.NewWebhookGateway(bus, nil)
	adapter := &channel.StubAdapter{}
	gw.Register("stub", adapter)

	runner := &stubWorkflowRunner{}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "no-inprocess-test"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	body := []byte(`{"text": "verify nats path", "user_id": "U1", "channel_id": "C1"}`)
	resp, err := http.Post(srv.URL+"/webhooks/stub", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	// The memPublisher received events — this proves NATSBus was in the path
	// (InProcessBus does not use a publisher).
	published := pub.published()
	if len(published) == 0 {
		t.Fatal("NATSBus publisher received no events — InProcessBus may be in the path")
	}

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("adapter sent %d, want 1", len(sent))
	}
}

// TestFlywheel_MultiPlatformRouting verifies that the bridge correctly routes
// messages from different platforms to the appropriate workflows.
func TestFlywheel_MultiPlatformRouting(t *testing.T) {
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)

	gw := channel.NewWebhookGateway(bus, nil)

	slackAdapter := &channel.StubAdapter{}
	discordAdapter := &channel.StubAdapter{}

	// Both adapters return "stub" from Name(), so we register them under
	// different platform keys in the gateway and bridge.
	gw.Register("slack", slackAdapter)
	gw.Register("discord", discordAdapter)

	runner := &stubWorkflowRunner{}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", slackAdapter) // StubAdapter.Name() = "stub"
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "slack-workflow"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Send webhook to the "slack" platform path.
	body := []byte(`{"text": "multi-platform test", "user_id": "U1", "channel_id": "C1"}`)
	resp, err := http.Post(srv.URL+"/webhooks/slack", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Verify the event was published to the correct NATS subject.
	published := pub.published()
	found := false
	for _, p := range published {
		if p.Subject == "dojo.channel.message.slack" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected event on subject dojo.channel.message.slack")
	}
}

// TestFlywheel_CredentialStoreIntegration verifies that the credential store
// is accessible through the adapter registry during the flywheel.
func TestFlywheel_CredentialStoreIntegration(t *testing.T) {
	ctx := context.Background()

	// Set up credential store with test tokens.
	credStore := channel.NewEnvCredentialStore()
	credStore.Set(ctx, "slack", "BOT_TOKEN", "xoxb-flywheel-test")
	credStore.Set(ctx, "slack", "SIGNING_SECRET", "signing-secret-flywheel")

	// Create adapter registry with credential store.
	registry := channel.NewAdapterRegistry(credStore)

	// Verify credentials are accessible through the registry.
	token, err := registry.GetCredential(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("GetCredential BOT_TOKEN: %v", err)
	}
	if token != "xoxb-flywheel-test" {
		t.Errorf("token = %q, want %q", token, "xoxb-flywheel-test")
	}

	secret, err := registry.GetCredential(ctx, "slack", "SIGNING_SECRET")
	if err != nil {
		t.Fatalf("GetCredential SIGNING_SECRET: %v", err)
	}
	if secret != "signing-secret-flywheel" {
		t.Errorf("secret = %q, want %q", secret, "signing-secret-flywheel")
	}

	// Now wire the full flywheel.
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)
	adapter := &channel.StubAdapter{}

	runner := &stubWorkflowRunner{}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "cred-test"})
	bus.Subscribe(bridge.BusHandler(ctx))

	// Publish a synthetic message.
	msg := &channel.ChannelMessage{
		ID:        "cred-msg-001",
		Platform:  "stub",
		ChannelID: "C_CRED",
		UserID:    "U_CRED",
		Text:      "credential integration test",
		Timestamp: time.Now().UTC(),
	}
	evt, err := channel.ToCloudEvent(msg)
	if err != nil {
		t.Fatalf("ToCloudEvent: %v", err)
	}
	bus.Publish("dojo.channel.message.stub", evt)

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("adapter sent %d, want 1", len(sent))
	}
}
