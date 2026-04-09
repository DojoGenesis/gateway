package integration

import (
	"context"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Test 5: Credential store rotation
//
// Create EnvCredentialStore with a test token.
// SlackAdapter reads token via CredentialStore interface.
// Update the store value (simulate rotation).
// Assert new value is returned without adapter restart.
// Uses InfisicalCredentialStore mock for the cache invalidation path.
// ---------------------------------------------------------------------------

func TestCredentialRotation_EnvStore(t *testing.T) {
	ctx := context.Background()

	// --- 1. Create EnvCredentialStore with initial token ---
	store := channel.NewEnvCredentialStore()
	if err := store.Set(ctx, "slack", "BOT_TOKEN", "xoxb-initial-token"); err != nil {
		t.Fatalf("set initial token: %v", err)
	}

	// --- 2. Verify initial token is readable ---
	token, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get initial token: %v", err)
	}
	if token != "xoxb-initial-token" {
		t.Errorf("initial token = %q, want %q", token, "xoxb-initial-token")
	}

	// --- 3. Wire adapter registry with credential store ---
	registry := channel.NewAdapterRegistry(store)

	// Verify registry-level access.
	regToken, err := registry.GetCredential(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("registry GetCredential: %v", err)
	}
	if regToken != "xoxb-initial-token" {
		t.Errorf("registry token = %q, want %q", regToken, "xoxb-initial-token")
	}

	// --- 4. Simulate token rotation ---
	if err := store.Set(ctx, "slack", "BOT_TOKEN", "xoxb-rotated-token"); err != nil {
		t.Fatalf("set rotated token: %v", err)
	}

	// --- 5. Verify new value without adapter restart ---
	rotatedToken, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get rotated token: %v", err)
	}
	if rotatedToken != "xoxb-rotated-token" {
		t.Errorf("rotated token = %q, want %q", rotatedToken, "xoxb-rotated-token")
	}

	// Registry should also return the rotated value.
	regRotated, err := registry.GetCredential(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("registry GetCredential after rotation: %v", err)
	}
	if regRotated != "xoxb-rotated-token" {
		t.Errorf("registry rotated = %q, want %q", regRotated, "xoxb-rotated-token")
	}
}

func TestCredentialRotation_InfisicalStore_CacheInvalidation(t *testing.T) {
	ctx := context.Background()

	// --- 1. Create mock Infisical client with initial secret ---
	mockClient := newMockInfisicalClient()
	mockClient.setSecret("SLACK_BOT_TOKEN", "xoxb-infisical-v1")

	store := channel.NewInfisicalCredentialStore(mockClient, channel.InfisicalConfig{
		Environment: "test",
		SecretPath:  "/channel",
	})

	// --- 2. Verify initial value ---
	token, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get initial infisical token: %v", err)
	}
	if token != "xoxb-infisical-v1" {
		t.Errorf("initial token = %q, want %q", token, "xoxb-infisical-v1")
	}

	// --- 3. Simulate remote rotation (change the mock secret) ---
	mockClient.setSecret("SLACK_BOT_TOKEN", "xoxb-infisical-v2")

	// --- 4. Cache still returns old value ---
	cachedToken, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get cached token: %v", err)
	}
	if cachedToken != "xoxb-infisical-v1" {
		t.Errorf("cached token = %q, want old value %q", cachedToken, "xoxb-infisical-v1")
	}

	// --- 5. Invalidate cache ---
	store.InvalidateCache()

	// --- 6. Now Get returns the rotated value ---
	refreshedToken, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get refreshed token: %v", err)
	}
	if refreshedToken != "xoxb-infisical-v2" {
		t.Errorf("refreshed token = %q, want %q", refreshedToken, "xoxb-infisical-v2")
	}
}

func TestCredentialRotation_InfisicalStore_SetOverridesRemote(t *testing.T) {
	ctx := context.Background()

	mockClient := newMockInfisicalClient()
	mockClient.setSecret("SLACK_BOT_TOKEN", "xoxb-remote")

	store := channel.NewInfisicalCredentialStore(mockClient, channel.InfisicalConfig{})

	// Set a local override (simulates hot-reload).
	store.Set(ctx, "slack", "BOT_TOKEN", "xoxb-local-override")

	// Local override takes precedence.
	token, err := store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get overridden token: %v", err)
	}
	if token != "xoxb-local-override" {
		t.Errorf("token = %q, want %q", token, "xoxb-local-override")
	}

	// Invalidate cache — local overrides are also cached, so they get cleared.
	store.InvalidateCache()

	// After invalidation, remote value should be returned.
	token, err = store.Get(ctx, "slack", "BOT_TOKEN")
	if err != nil {
		t.Fatalf("get after invalidate: %v", err)
	}
	if token != "xoxb-remote" {
		t.Errorf("after invalidate: token = %q, want remote %q", token, "xoxb-remote")
	}
}

func TestCredentialRotation_MultipleKeys(t *testing.T) {
	ctx := context.Background()

	store := channel.NewEnvCredentialStore()
	store.Set(ctx, "slack", "BOT_TOKEN", "xoxb-bot-v1")
	store.Set(ctx, "slack", "SIGNING_SECRET", "signing-v1")
	store.Set(ctx, "discord", "TOKEN", "discord-v1")

	// Rotate Slack bot token only.
	store.Set(ctx, "slack", "BOT_TOKEN", "xoxb-bot-v2")

	// Verify Slack bot token rotated.
	botToken, _ := store.Get(ctx, "slack", "BOT_TOKEN")
	if botToken != "xoxb-bot-v2" {
		t.Errorf("bot token = %q, want %q", botToken, "xoxb-bot-v2")
	}

	// Verify signing secret unchanged.
	signingSecret, _ := store.Get(ctx, "slack", "SIGNING_SECRET")
	if signingSecret != "signing-v1" {
		t.Errorf("signing secret = %q, want %q (should not have changed)", signingSecret, "signing-v1")
	}

	// Verify Discord token unchanged.
	discordToken, _ := store.Get(ctx, "discord", "TOKEN")
	if discordToken != "discord-v1" {
		t.Errorf("discord token = %q, want %q (should not have changed)", discordToken, "discord-v1")
	}
}

func TestCredentialRotation_FullFlywheel(t *testing.T) {
	ctx := context.Background()

	// Create credential store with initial token.
	credStore := channel.NewEnvCredentialStore()
	credStore.Set(ctx, "slack", "BOT_TOKEN", "xoxb-flywheel-v1")

	// Wire the flywheel.
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)
	adapter := &channel.StubAdapter{}

	runner := &stubWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 1},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "cred-rotation-workflow"})
	bus.Subscribe(bridge.BusHandler(ctx))

	// Send first message.
	msg1 := &channel.ChannelMessage{
		ID: "cred-msg-1", Platform: "stub", ChannelID: "C1",
		Text: "before rotation", Timestamp: time.Now().UTC(),
	}
	evt1, _ := channel.ToCloudEvent(msg1)
	bus.Publish("dojo.channel.message.stub", evt1)

	if len(adapter.Sent()) != 1 {
		t.Fatalf("expected 1 reply before rotation, got %d", len(adapter.Sent()))
	}

	// Rotate credential.
	credStore.Set(ctx, "slack", "BOT_TOKEN", "xoxb-flywheel-v2")

	// Verify rotation picked up.
	newToken, _ := credStore.Get(ctx, "slack", "BOT_TOKEN")
	if newToken != "xoxb-flywheel-v2" {
		t.Fatalf("rotated token = %q, want %q", newToken, "xoxb-flywheel-v2")
	}

	// Send second message — the flywheel should still work after rotation.
	msg2 := &channel.ChannelMessage{
		ID: "cred-msg-2", Platform: "stub", ChannelID: "C1",
		Text: "after rotation", Timestamp: time.Now().UTC(),
	}
	evt2, _ := channel.ToCloudEvent(msg2)
	bus.Publish("dojo.channel.message.stub", evt2)

	if len(adapter.Sent()) != 2 {
		t.Fatalf("expected 2 replies total, got %d", len(adapter.Sent()))
	}

	bus.Close()
}
