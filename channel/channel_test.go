package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// 1. TestStubAdapterNormalize — raw JSON -> ChannelMessage
// ---------------------------------------------------------------------------

func TestStubAdapterNormalize(t *testing.T) {
	adapter := &StubAdapter{}

	raw := []byte(`{"text": "hello world", "user_id": "U123", "user_name": "alice", "channel_id": "C456"}`)
	msg, err := adapter.Normalize(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Text != "hello world" {
		t.Errorf("got text %q, want %q", msg.Text, "hello world")
	}
	if msg.UserID != "U123" {
		t.Errorf("got user_id %q, want %q", msg.UserID, "U123")
	}
	if msg.UserName != "alice" {
		t.Errorf("got user_name %q, want %q", msg.UserName, "alice")
	}
	if msg.ChannelID != "C456" {
		t.Errorf("got channel_id %q, want %q", msg.ChannelID, "C456")
	}
	if msg.Platform != "stub" {
		t.Errorf("got platform %q, want %q", msg.Platform, "stub")
	}
	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
}

func TestStubAdapterNormalize_MissingText(t *testing.T) {
	adapter := &StubAdapter{}

	raw := []byte(`{"user_id": "U123"}`)
	_, err := adapter.Normalize(raw)
	if err == nil {
		t.Fatal("expected error for missing text, got nil")
	}
}

func TestStubAdapterNormalize_InvalidJSON(t *testing.T) {
	adapter := &StubAdapter{}

	_, err := adapter.Normalize([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// 2. TestStubAdapterSend — sends message, inspects internal slice
// ---------------------------------------------------------------------------

func TestStubAdapterSend(t *testing.T) {
	adapter := &StubAdapter{}
	ctx := context.Background()

	msg1 := &ChannelMessage{ID: "1", Text: "first", Platform: "stub"}
	msg2 := &ChannelMessage{ID: "2", Text: "second", Platform: "stub"}

	if err := adapter.Send(ctx, msg1); err != nil {
		t.Fatalf("send msg1: %v", err)
	}
	if err := adapter.Send(ctx, msg2); err != nil {
		t.Fatalf("send msg2: %v", err)
	}

	sent := adapter.Sent()
	if len(sent) != 2 {
		t.Fatalf("got %d sent messages, want 2", len(sent))
	}
	if sent[0].Text != "first" {
		t.Errorf("sent[0].Text = %q, want %q", sent[0].Text, "first")
	}
	if sent[1].Text != "second" {
		t.Errorf("sent[1].Text = %q, want %q", sent[1].Text, "second")
	}
}

// ---------------------------------------------------------------------------
// 3. TestWebhookGateway_Routes — register stub, send HTTP request, verify
// ---------------------------------------------------------------------------

func TestWebhookGateway_Routes(t *testing.T) {
	adapter := &StubAdapter{}
	gw := NewWebhookGateway(nil, nil)
	gw.Register("stub", adapter)

	body := []byte(`{"text": "webhook test", "user_id": "U789"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stub", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWebhookGateway_Routes_WithBus(t *testing.T) {
	adapter := &StubAdapter{}
	bus := &mockPublisher{}
	gw := NewWebhookGateway(bus, nil)
	gw.Register("stub", adapter)

	body := []byte(`{"text": "bus test"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stub", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if len(bus.events) != 1 {
		t.Fatalf("got %d published events, want 1", len(bus.events))
	}
	if bus.subjects[0] != "dojo.channel.message.stub" {
		t.Errorf("got subject %q, want %q", bus.subjects[0], "dojo.channel.message.stub")
	}
}

// ---------------------------------------------------------------------------
// 4. TestWebhookGateway_UnknownPlatform — returns 404
// ---------------------------------------------------------------------------

func TestWebhookGateway_UnknownPlatform(t *testing.T) {
	gw := NewWebhookGateway(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/unknown", nil)
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWebhookGateway_MissingPlatform(t *testing.T) {
	gw := NewWebhookGateway(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/", nil)
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	// /webhooks/ with no platform segment should be a bad request.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// 5. TestEnvCredentialStore_GetSet — set env var, get via store
// ---------------------------------------------------------------------------

func TestEnvCredentialStore_GetSet(t *testing.T) {
	store := NewEnvCredentialStore()
	ctx := context.Background()

	// Set via the store's overlay.
	if err := store.Set(ctx, "slack", "TOKEN", "xoxb-test-token"); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := store.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "xoxb-test-token" {
		t.Errorf("got %q, want %q", val, "xoxb-test-token")
	}
}

func TestEnvCredentialStore_GetFromEnv(t *testing.T) {
	store := NewEnvCredentialStore()
	ctx := context.Background()

	key := "DOJO_TESTPLATFORM_SECRET"
	os.Setenv(key, "env-secret-value")
	defer os.Unsetenv(key)

	val, err := store.Get(ctx, "testplatform", "SECRET")
	if err != nil {
		t.Fatalf("get from env: %v", err)
	}
	if val != "env-secret-value" {
		t.Errorf("got %q, want %q", val, "env-secret-value")
	}
}

func TestEnvCredentialStore_GetMissing(t *testing.T) {
	store := NewEnvCredentialStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent", "KEY")
	if err == nil {
		t.Fatal("expected error for missing credential, got nil")
	}
}

func TestEnvCredentialStore_List(t *testing.T) {
	store := NewEnvCredentialStore()
	ctx := context.Background()

	// Set two keys for the same platform.
	_ = store.Set(ctx, "discord", "TOKEN", "tok")
	_ = store.Set(ctx, "discord", "SECRET", "sec")

	keys, err := store.List(ctx, "discord")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) < 2 {
		t.Fatalf("got %d keys, want at least 2", len(keys))
	}

	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	if !found["TOKEN"] {
		t.Error("missing TOKEN in list")
	}
	if !found["SECRET"] {
		t.Error("missing SECRET in list")
	}
}

// ---------------------------------------------------------------------------
// 6. TestTokenBucketLimiter_Allow — allows up to burst, then blocks
// ---------------------------------------------------------------------------

func TestTokenBucketLimiter_Allow(t *testing.T) {
	limiter := NewTokenBucketLimiter(1.0, 3) // 1 token/sec, burst 3
	ctx := context.Background()

	// Should allow up to burst (3 tokens).
	for i := 0; i < 3; i++ {
		ok, err := limiter.Allow(ctx, "test")
		if err != nil {
			t.Fatalf("allow[%d]: %v", i, err)
		}
		if !ok {
			t.Fatalf("allow[%d]: expected true, got false", i)
		}
	}

	// Fourth call should be denied (bucket empty).
	ok, err := limiter.Allow(ctx, "test")
	if err != nil {
		t.Fatalf("allow[3]: %v", err)
	}
	if ok {
		t.Error("expected deny after burst exhausted, got allow")
	}
}

func TestTokenBucketLimiter_IndependentKeys(t *testing.T) {
	limiter := NewTokenBucketLimiter(1.0, 1)
	ctx := context.Background()

	ok1, _ := limiter.Allow(ctx, "key-a")
	ok2, _ := limiter.Allow(ctx, "key-b")
	if !ok1 || !ok2 {
		t.Error("independent keys should each have their own bucket")
	}

	// Both should now be exhausted.
	ok1, _ = limiter.Allow(ctx, "key-a")
	ok2, _ = limiter.Allow(ctx, "key-b")
	if ok1 || ok2 {
		t.Error("both keys should be exhausted after one token each")
	}
}

// ---------------------------------------------------------------------------
// 7. TestTokenBucketLimiter_Wait — blocks until refill
// ---------------------------------------------------------------------------

func TestTokenBucketLimiter_Wait(t *testing.T) {
	limiter := NewTokenBucketLimiter(100.0, 1) // 100 tokens/sec for fast test
	ctx := context.Background()

	// Exhaust the single token.
	ok, _ := limiter.Allow(ctx, "wait-test")
	if !ok {
		t.Fatal("first allow should succeed")
	}

	// Wait should succeed once a token refills (at 100/sec, ~10ms).
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx, "wait-test")
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
}

func TestTokenBucketLimiter_Wait_ContextCancel(t *testing.T) {
	limiter := NewTokenBucketLimiter(0.01, 1) // Very slow: 1 token per 100 seconds
	ctx := context.Background()

	// Exhaust the token.
	limiter.Allow(ctx, "cancel-test")

	// Cancel immediately.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx, "cancel-test")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// ---------------------------------------------------------------------------
// 8. TestNoOpLimiter — always allows
// ---------------------------------------------------------------------------

func TestNoOpLimiter(t *testing.T) {
	limiter := NoOpLimiter{}
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		ok, err := limiter.Allow(ctx, "any-key")
		if err != nil {
			t.Fatalf("noop allow: %v", err)
		}
		if !ok {
			t.Fatal("noop limiter should always allow")
		}
	}

	if err := limiter.Wait(ctx, "any-key"); err != nil {
		t.Fatalf("noop wait: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 9. TestChannelMessageToCloudEvent — verify event type, source, data
// ---------------------------------------------------------------------------

func TestChannelMessageToCloudEvent(t *testing.T) {
	msg := &ChannelMessage{
		ID:        "msg-001",
		Platform:  "slack",
		ChannelID: "C12345",
		UserID:    "U67890",
		UserName:  "bob",
		Text:      "cloud event test",
		Timestamp: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
	}

	evt, err := ToCloudEvent(msg)
	if err != nil {
		t.Fatalf("ToCloudEvent: %v", err)
	}

	if evt.SpecVersion != "1.0" {
		t.Errorf("specversion = %q, want %q", evt.SpecVersion, "1.0")
	}
	if evt.Type != "dojo.channel.message.slack" {
		t.Errorf("type = %q, want %q", evt.Type, "dojo.channel.message.slack")
	}
	if evt.Source != "channel/slack" {
		t.Errorf("source = %q, want %q", evt.Source, "channel/slack")
	}
	if evt.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if evt.DataContentType != "application/json" {
		t.Errorf("datacontenttype = %q, want %q", evt.DataContentType, "application/json")
	}

	// Verify the data contains the original message.
	var decoded ChannelMessage
	if err := json.Unmarshal(evt.Data, &decoded); err != nil {
		t.Fatalf("unmarshal event data: %v", err)
	}
	if decoded.Text != "cloud event test" {
		t.Errorf("data text = %q, want %q", decoded.Text, "cloud event test")
	}
	if decoded.Platform != "slack" {
		t.Errorf("data platform = %q, want %q", decoded.Platform, "slack")
	}
}

func TestChannelMessageToCloudEvent_NilMessage(t *testing.T) {
	_, err := ToCloudEvent(nil)
	if err == nil {
		t.Fatal("expected error for nil message, got nil")
	}
}

// ---------------------------------------------------------------------------
// 10. TestAdapterCapabilities — verify struct fields
// ---------------------------------------------------------------------------

func TestAdapterCapabilities(t *testing.T) {
	caps := AdapterCapabilities{
		SupportsThreads:     true,
		SupportsReactions:   true,
		SupportsAttachments: true,
		SupportsEdits:       false,
		MaxMessageLength:    4000,
	}

	if !caps.SupportsThreads {
		t.Error("expected SupportsThreads = true")
	}
	if !caps.SupportsReactions {
		t.Error("expected SupportsReactions = true")
	}
	if !caps.SupportsAttachments {
		t.Error("expected SupportsAttachments = true")
	}
	if caps.SupportsEdits {
		t.Error("expected SupportsEdits = false")
	}
	if caps.MaxMessageLength != 4000 {
		t.Errorf("MaxMessageLength = %d, want 4000", caps.MaxMessageLength)
	}
}

func TestStubAdapterCapabilities(t *testing.T) {
	adapter := &StubAdapter{}
	caps := adapter.Capabilities()

	if caps.SupportsThreads {
		t.Error("stub should not support threads")
	}
	if caps.MaxMessageLength != 4096 {
		t.Errorf("stub MaxMessageLength = %d, want 4096", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 11. TestNewSlackLimiter — Slack-specific rate limiter defaults
// ---------------------------------------------------------------------------

func TestNewSlackLimiter(t *testing.T) {
	limiter := NewSlackLimiter()
	ctx := context.Background()

	// Slack allows 1 msg/sec/channel with burst 1.
	ok, err := limiter.Allow(ctx, "channel-1")
	if err != nil || !ok {
		t.Fatal("first request should be allowed")
	}
	ok, _ = limiter.Allow(ctx, "channel-1")
	if ok {
		t.Error("second request in same tick should be denied (burst 1)")
	}
}

// ---------------------------------------------------------------------------
// 12. TestNewDiscordLimiter — Discord-specific rate limiter defaults
// ---------------------------------------------------------------------------

func TestNewDiscordLimiter(t *testing.T) {
	limiter := NewDiscordLimiter()
	ctx := context.Background()

	// Discord allows burst 5.
	for i := 0; i < 5; i++ {
		ok, err := limiter.Allow(ctx, "guild-1")
		if err != nil || !ok {
			t.Fatalf("request %d should be allowed (burst 5)", i+1)
		}
	}
	ok, _ := limiter.Allow(ctx, "guild-1")
	if ok {
		t.Error("6th request should be denied (burst 5)")
	}
}

// ---------------------------------------------------------------------------
// 13. TestNewTelegramLimiter — Telegram dual-mode rate limiter
// ---------------------------------------------------------------------------

func TestNewTelegramLimiter(t *testing.T) {
	limiter := NewTelegramLimiter()
	ctx := context.Background()

	// DM: 1 msg/sec burst 1
	ok, err := limiter.Allow(ctx, "dm:12345")
	if err != nil || !ok {
		t.Fatal("DM first request should be allowed")
	}
	ok, _ = limiter.Allow(ctx, "dm:12345")
	if ok {
		t.Error("DM second request should be denied (burst 1)")
	}

	// Group: 30 msg/sec burst 30
	for i := 0; i < 30; i++ {
		ok, err := limiter.Allow(ctx, "group:67890")
		if err != nil || !ok {
			t.Fatalf("group request %d should be allowed (burst 30)", i+1)
		}
	}
	ok, _ = limiter.Allow(ctx, "group:67890")
	if ok {
		t.Error("group 31st request should be denied (burst 30)")
	}
}

// ---------------------------------------------------------------------------
// 14. TestTelegramDualLimiter_Wait — verifies Wait uses correct limiter
// ---------------------------------------------------------------------------

func TestTelegramDualLimiter_Wait(t *testing.T) {
	limiter := NewTelegramLimiter()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// DM wait should succeed (first call, token available).
	if err := limiter.Wait(ctx, "dm:999"); err != nil {
		t.Fatalf("DM wait: %v", err)
	}

	// Group wait should succeed (first call).
	if err := limiter.Wait(ctx, "group:999"); err != nil {
		t.Fatalf("group wait: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mockPublisher captures published events for test verification.
type mockPublisher struct {
	mu       sync.Mutex
	events   []Event
	subjects []string
}

func (m *mockPublisher) Publish(subject string, evt Event) error {
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.subjects = append(m.subjects, subject)
	m.mu.Unlock()
	return nil
}
