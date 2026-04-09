package discord

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Adapter compliance
// ---------------------------------------------------------------------------

func TestDiscordAdapter_Compliance(t *testing.T) {
	a, priv := newTestAdapter(t)

	validPayload := []byte(`{
		"id": "111222333",
		"channel_id": "CHAN-001",
		"content": "compliance test",
		"timestamp": "2026-04-05T12:00:00Z",
		"author": {"id": "USER-42", "username": "alice"}
	}`)

	invalidPayload := []byte(`{not valid json}`)

	signedReq := func() *http.Request {
		body := []byte(`{"type":1}`)
		return signedRequest(t, body, priv)
	}

	unsignedReq := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{"type":1}`)))
	}

	channel.AdapterComplianceHelper(t, a, validPayload, invalidPayload, signedReq, unsignedReq,
		func(t *testing.T, _ channel.WebhookAdapter) {
			caps := a.Capabilities()
			if !caps.SupportsThreads {
				t.Error("Discord adapter should support threads")
			}
		},
	)
}

// ---------------------------------------------------------------------------
// End-to-end smoke test
// ---------------------------------------------------------------------------

func TestDiscord_EndToEnd_SmokeTest(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	a, err := New(DiscordConfig{
		PublicKey: hex.EncodeToString(pub),
		AppID:     "test-app",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Use in-process bus for the smoke test.
	bus := &channel.InProcessBus{}
	gw := channel.NewWebhookGateway(bus, nil)
	gw.Register("discord", a)

	runner := &mockWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 1},
	}
	bridge := channel.NewChannelBridge(runner)

	// Since the discord adapter requires a session for Send, use a stub
	// adapter for the reply channel.
	replyAdapter := &channel.StubAdapter{}
	bridge.Register("discord", replyAdapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "discord", Workflow: "discord-smoke"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Build a Discord MESSAGE_CREATE payload.
	msgPayload, _ := json.Marshal(map[string]interface{}{
		"id":         "MSG001",
		"channel_id": "CHAN001",
		"content":    "discord smoke test",
		"timestamp":  "2026-04-05T12:00:00Z",
		"author": map[string]string{
			"id":       "USER001",
			"username": "tester",
		},
	})

	// Sign the payload.
	ts := fmt.Sprintf("%d", time.Now().Unix())
	msg := append([]byte(ts), msgPayload...)
	sig := ed25519.Sign(priv, msg)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/webhooks/discord", bytes.NewReader(msgPayload))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	req.Header.Set("X-Signature-Timestamp", ts)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /webhooks/discord: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200", resp.StatusCode)
	}

	// The reply was sent via the stub adapter.
	sent := replyAdapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(sent))
	}
	if sent[0].Platform != "discord" {
		t.Errorf("reply.Platform = %q, want %q", sent[0].Platform, "discord")
	}
}

// ---------------------------------------------------------------------------
// Test: NormalizeInteraction — APPLICATION_COMMAND
// ---------------------------------------------------------------------------

func TestDiscordAdapter_NormalizeInteraction_ApplicationCommand(t *testing.T) {
	a, _ := newTestAdapter(t)

	raw := []byte(`{
		"id": "INT001",
		"type": 2,
		"channel_id": "CHAN-CMD",
		"guild_id": "GUILD-001",
		"member": {
			"user": {
				"id": "USER-99",
				"username": "commander"
			}
		},
		"data": {
			"name": "review",
			"options": [
				{"name": "target", "value": "PR-42"}
			]
		}
	}`)

	msg, err := a.NormalizeInteraction(raw)
	if err != nil {
		t.Fatalf("NormalizeInteraction: %v", err)
	}
	if msg.Platform != "discord" {
		t.Errorf("Platform = %q, want %q", msg.Platform, "discord")
	}
	if msg.ChannelID != "CHAN-CMD" {
		t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "CHAN-CMD")
	}
	if msg.UserID != "USER-99" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "USER-99")
	}
	if msg.Text != "/review target=PR-42" {
		t.Errorf("Text = %q, want %q", msg.Text, "/review target=PR-42")
	}
	if msg.Metadata["guild_id"] != "GUILD-001" {
		t.Errorf("guild_id metadata = %v", msg.Metadata["guild_id"])
	}
	if msg.Metadata["command_name"] != "review" {
		t.Errorf("command_name metadata = %v", msg.Metadata["command_name"])
	}
}

// ---------------------------------------------------------------------------
// Test: HandleWebhook — APPLICATION_COMMAND (type 2) returns deferred ACK
// ---------------------------------------------------------------------------

func TestDiscordAdapter_HandleWebhook_ApplicationCommand(t *testing.T) {
	a, priv := newTestAdapter(t)

	cmdBody, _ := json.Marshal(map[string]interface{}{
		"id":         "INT002",
		"type":       2,
		"channel_id": "C001",
		"data":       map[string]string{"name": "audit"},
	})
	req := signedRequest(t, cmdBody, priv)
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Type 5 = DEFERRED_CHANNEL_MESSAGE_WITH_SOURCE
	if resp["type"] != 5 {
		t.Errorf("response type = %d, want 5", resp["type"])
	}
}

// ---------------------------------------------------------------------------
// Test: Discord Opcode 6 resume state survives simulated actor restart
// ---------------------------------------------------------------------------

func TestDiscordAdapter_ResumeState_SurvivesRestart(t *testing.T) {
	store := newMemResumeStore()
	cfg := DiscordConfig{
		PublicKey: "0000000000000000000000000000000000000000000000000000000000000000",
		AppID:     "test-app",
		GuildID:   "guild-abc",
	}

	// Create adapter #1, save resume state.
	a1, err := NewWithResumeStore(cfg, store)
	if err != nil {
		t.Fatalf("NewWithResumeStore: %v", err)
	}

	state := ResumeState{
		ResumeGatewayURL: "wss://gateway.discord.gg/?v=10&encoding=json",
		SessionID:        "session-xyz-123",
		Seq:              42,
	}
	if err := a1.SaveResumeState(state); err != nil {
		t.Fatalf("SaveResumeState: %v", err)
	}

	// Simulate restart: create adapter #2 with the same store.
	a2, err := NewWithResumeStore(cfg, store)
	if err != nil {
		t.Fatalf("NewWithResumeStore #2: %v", err)
	}

	loaded, err := a2.LoadResumeState()
	if err != nil {
		t.Fatalf("LoadResumeState: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected resume state after restart, got nil")
	}
	if loaded.SessionID != "session-xyz-123" {
		t.Errorf("SessionID = %q, want %q", loaded.SessionID, "session-xyz-123")
	}
	if loaded.ResumeGatewayURL != "wss://gateway.discord.gg/?v=10&encoding=json" {
		t.Errorf("ResumeGatewayURL = %q", loaded.ResumeGatewayURL)
	}
	if loaded.Seq != 42 {
		t.Errorf("Seq = %d, want 42", loaded.Seq)
	}
}

func TestDiscordAdapter_ResumeState_NoStore(t *testing.T) {
	a, _ := newTestAdapter(t) // no resume store

	err := a.SaveResumeState(ResumeState{SessionID: "x"})
	if err == nil {
		t.Fatal("expected error when no resume store, got nil")
	}

	_, err = a.LoadResumeState()
	if err == nil {
		t.Fatal("expected error when no resume store, got nil")
	}
}

func TestDiscordAdapter_ResumeState_EmptyGuildID(t *testing.T) {
	store := newMemResumeStore()
	cfg := DiscordConfig{
		PublicKey: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	a, err := NewWithResumeStore(cfg, store)
	if err != nil {
		t.Fatalf("NewWithResumeStore: %v", err)
	}

	state := ResumeState{SessionID: "default-session"}
	if err := a.SaveResumeState(state); err != nil {
		t.Fatalf("SaveResumeState: %v", err)
	}

	// Key should be "discord.resume.default".
	if _, ok := store.data["discord.resume.default"]; !ok {
		t.Error("expected data stored at key discord.resume.default")
	}
}

// ---------------------------------------------------------------------------
// Test: DiscordConfig.ResumeStateKey
// ---------------------------------------------------------------------------

func TestDiscordConfig_ResumeStateKey(t *testing.T) {
	tests := []struct {
		guildID string
		want    string
	}{
		{"guild-123", "discord.resume.guild-123"},
		{"", "discord.resume.default"},
	}
	for _, tc := range tests {
		cfg := DiscordConfig{GuildID: tc.guildID}
		got := cfg.ResumeStateKey()
		if got != tc.want {
			t.Errorf("ResumeStateKey(%q) = %q, want %q", tc.guildID, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// memResumeStore is an in-memory ResumeStore for testing.
type memResumeStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemResumeStore() *memResumeStore {
	return &memResumeStore{data: make(map[string][]byte)}
}

func (s *memResumeStore) Put(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	s.data[key] = cp
	return nil
}

func (s *memResumeStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

// mockWorkflowRunner satisfies channel.WorkflowRunner for smoke tests.
type mockWorkflowRunner struct {
	result *channel.WorkflowRunResult
}

func (r *mockWorkflowRunner) Execute(_ context.Context, name string) (*channel.WorkflowRunResult, error) {
	res := &channel.WorkflowRunResult{
		WorkflowName: name,
		Status:       "completed",
		StepCount:    1,
	}
	if r.result != nil {
		cp := *r.result
		cp.WorkflowName = name
		res = &cp
	}
	return res, nil
}

// Compile-time interface check.
var _ channel.WebhookAdapter = (*DiscordAdapter)(nil)
