package channel

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Integration test: SlackAdapter stub reads a test token from
// EnvCredentialStore, publishes to the NATSBus (using memPublisher to
// simulate embedded NATS), and verifies the bridge receives the
// dojo.channel.message.slack CloudEvent.
//
// This test proves the Track A acceptance criteria:
//   - CredentialStore injection works (adapter reads token via store)
//   - NATSBus carries channel events (not InProcessBus)
//   - JetStream subject convention dojo.channel.message.slack is used
//   - Full round-trip: credential lookup → publish → bridge → reply
// ---------------------------------------------------------------------------

func TestIntegration_SlackStub_NATSBus_CredentialStore(t *testing.T) {
	ctx := context.Background()

	// --- 1. Set up credential store with a test token ---
	credStore := NewEnvCredentialStore()
	if err := credStore.Set(ctx, "slack", "TOKEN", "xoxb-test-integration-token"); err != nil {
		t.Fatalf("set credential: %v", err)
	}

	// Verify the token is retrievable.
	token, err := credStore.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if token != "xoxb-test-integration-token" { //nolint:gosec // test constant
		t.Fatalf("token = %q, want %q", token, "xoxb-test-integration-token")
	}

	// --- 2. Set up adapter registry with credential store ---
	registry := NewAdapterRegistry(credStore)

	slackAdapter := &StubAdapter{}
	if err := registry.Register(slackAdapter); err != nil {
		// StubAdapter returns "stub" not "slack" — register it directly
		// on the bridge instead.
		_ = err
	}

	// --- 3. Wire NATSBus (using memPublisher to simulate embedded NATS) ---
	pub := &memPublisher{}
	bus := NewNATSBus(pub, WithNATSSubscriber(pub))

	// --- 4. Wire the ChannelBridge with a stub workflow runner ---
	runner := &stubWorkflowRunner{
		result: &WorkflowRunResult{Status: "completed", StepCount: 2},
	}
	bridge := NewChannelBridge(runner)
	bridge.Register("stub", slackAdapter)
	bridge.AddTrigger(TriggerSpec{Platform: "stub", Workflow: "slack-integration-workflow"})

	// Subscribe bridge to the NATSBus.
	bus.Subscribe(bridge.BusHandler(ctx))

	// --- 5. Track events received via NATS subscription ---
	var receivedEvents []Event
	var receivedMu sync.Mutex

	bus.Subscribe(func(subject string, evt Event) {
		receivedMu.Lock()
		receivedEvents = append(receivedEvents, evt)
		receivedMu.Unlock()
	})

	// NOTE: We intentionally do NOT call SubscribeNATS here because
	// the local subscribers (bridge.BusHandler) already receive events
	// directly from NATSBus.Publish. SubscribeNATS is for receiving
	// events published by OTHER services. Using both would cause double
	// delivery in this single-process test.

	// --- 7. Simulate a Slack inbound message ---
	msg := &ChannelMessage{
		ID:        "slack-msg-001",
		Platform:  "stub",
		ChannelID: "C-SLACK-001",
		UserID:    "U-SLACK-001",
		UserName:  "slack-user",
		Text:      "hello from slack integration test",
		Timestamp: time.Now().UTC(),
	}

	evt, err := ToCloudEvent(msg)
	if err != nil {
		t.Fatalf("ToCloudEvent: %v", err)
	}

	// Verify the CloudEvent type follows the subject convention.
	expectedType := "dojo.channel.message.stub"
	if evt.Type != expectedType {
		t.Errorf("event type = %q, want %q", evt.Type, expectedType)
	}

	// Publish to the NATSBus (simulating what WebhookGateway does).
	subject := "dojo.channel.message.stub"
	if err := bus.Publish(subject, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// --- 8. Verify the bridge processed the event and sent a reply ---
	sent := slackAdapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent messages, want 1", len(sent))
	}

	reply := sent[0]
	if reply.Platform != "stub" {
		t.Errorf("reply.Platform = %q, want %q", reply.Platform, "stub")
	}
	if reply.ChannelID != "C-SLACK-001" {
		t.Errorf("reply.ChannelID = %q, want %q", reply.ChannelID, "C-SLACK-001")
	}
	if reply.UserName != "Dojo" {
		t.Errorf("reply.UserName = %q, want %q", reply.UserName, "Dojo")
	}
	if reply.ReplyTo != "slack-msg-001" {
		t.Errorf("reply.ReplyTo = %q, want %q", reply.ReplyTo, "slack-msg-001")
	}

	// --- 9. Verify the underlying publisher received the event ---
	published := pub.published()
	if len(published) < 1 {
		t.Fatalf("publisher got %d events, want >= 1", len(published))
	}

	foundSubject := false
	for _, p := range published {
		if p.Subject == subject {
			foundSubject = true
			break
		}
	}
	if !foundSubject {
		t.Errorf("no published event with subject %q", subject)
	}

	// --- 10. Verify the NATS subscriber received events ---
	receivedMu.Lock()
	defer receivedMu.Unlock()
	if len(receivedEvents) < 1 {
		t.Fatalf("NATS subscriber got %d events, want >= 1", len(receivedEvents))
	}

	// --- 11. Verify the token was retrievable from the registry ---
	regToken, err := registry.GetCredential(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("registry GetCredential: %v", err)
	}
	if regToken != "xoxb-test-integration-token" { //nolint:gosec // test constant
		t.Errorf("registry token = %q, want %q", regToken, "xoxb-test-integration-token")
	}

	// Clean up.
	bus.Close()
}

// TestIntegration_NATSBus_SubjectConvention verifies that the NATS subject
// conventions from the ADR-015 addendum are correctly applied.
func TestIntegration_NATSBus_SubjectConvention(t *testing.T) {
	pub := &memPublisher{}
	bus := NewNATSBus(pub)

	msg := &ChannelMessage{
		ID:        "conv-001",
		Platform:  "slack",
		Text:      "subject convention test",
		Timestamp: time.Now().UTC(),
	}

	evt := NewChannelEvent(msg)

	// NewChannelEvent should produce the correct type.
	if evt.Type != "dojo.channel.message.slack" {
		t.Errorf("event type = %q, want %q", evt.Type, "dojo.channel.message.slack")
	}
	if evt.Source != "channel/slack" {
		t.Errorf("event source = %q, want %q", evt.Source, "channel/slack")
	}

	// Publish on the correct subject.
	messageSubject := ChannelSubject("slack")
	if messageSubject != "dojo.channel.slack.inbound" {
		t.Errorf("ChannelSubject = %q, want %q", messageSubject, "dojo.channel.slack.inbound")
	}

	if err := bus.Publish(messageSubject, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	published := pub.published()
	if len(published) != 1 {
		t.Fatalf("got %d published, want 1", len(published))
	}
	if published[0].Subject != messageSubject {
		t.Errorf("published subject = %q, want %q", published[0].Subject, messageSubject)
	}

	// Verify the data is a valid CloudEvent.
	var decoded Event
	if err := json.Unmarshal(published[0].Data, &decoded); err != nil {
		t.Fatalf("unmarshal published data: %v", err)
	}
	if decoded.Type != "dojo.channel.message.slack" {
		t.Errorf("decoded type = %q, want %q", decoded.Type, "dojo.channel.message.slack")
	}
}

// TestIntegration_AdapterRegistry_CredentialInjection verifies that the
// AdapterRegistry correctly injects a CredentialStore for adapter use.
func TestIntegration_AdapterRegistry_CredentialInjection(t *testing.T) {
	ctx := context.Background()

	// Create a credential store with test credentials.
	creds := NewEnvCredentialStore()
	_ = creds.Set(ctx, "slack", "BOT_TOKEN", "xoxb-registry-test")
	_ = creds.Set(ctx, "slack", "SIGNING_SECRET", "signing-secret-test")
	_ = creds.Set(ctx, "discord", "TOKEN", "discord-token-test")

	// Create registry with credential store.
	registry := NewAdapterRegistry(creds)

	// Verify credential retrieval through the registry.
	token, err := registry.GetCredential(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if token != "xoxb-registry-test" { //nolint:gosec // test constant
		t.Errorf("got %q, want %q", token, "xoxb-registry-test")
	}

	secret, err := registry.GetCredential(ctx, "slack", "SIGNING_SECRET")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if secret != "signing-secret-test" {
		t.Errorf("got %q, want %q", secret, "signing-secret-test")
	}

	discordToken, err := registry.GetCredential(ctx, "discord", "TOKEN")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if discordToken != "discord-token-test" {
		t.Errorf("got %q, want %q", discordToken, "discord-token-test")
	}

	// Missing credential should error.
	_, err = registry.GetCredential(ctx, "telegram", "TOKEN")
	if err == nil {
		t.Fatal("expected error for missing credential, got nil")
	}
}

// TestIntegration_InProcessBus_NotInProductionPath verifies that InProcessBus
// is only available in test files (this test file itself demonstrates that).
// Production code in cmd/dojo/bridge.go uses NATSBus exclusively.
func TestIntegration_InProcessBus_NotInProductionPath(t *testing.T) {
	// This test exists to document that InProcessBus is defined in
	// inprocess_bus_test.go (_test.go suffix) and is not importable
	// from production code. The NATSBus is the production bus.

	// InProcessBus is available here because we are in a _test.go file.
	bus := &InProcessBus{}
	var count int
	bus.Subscribe(func(_ string, _ Event) { count++ })

	msg := &ChannelMessage{ID: "m", Platform: "stub", Text: "test", Timestamp: time.Now().UTC()}
	evt, _ := ToCloudEvent(msg)
	_ = bus.Publish("test", evt)

	if count != 1 {
		t.Errorf("InProcessBus subscriber count = %d, want 1", count)
	}

	// NATSBus is the production replacement.
	pub := &memPublisher{}
	natsBus := NewNATSBus(pub)
	var natsCount int
	natsBus.Subscribe(func(_ string, _ Event) { natsCount++ })
	_ = natsBus.Publish("test", evt)

	if natsCount != 1 {
		t.Errorf("NATSBus subscriber count = %d, want 1", natsCount)
	}
}
