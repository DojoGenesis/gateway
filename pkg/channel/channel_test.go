package channel

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"
)

// mockAdapter implements ChannelAdapter for testing.
type mockAdapter struct {
	id      string
	version string
}

func (m *mockAdapter) ID() string      { return m.id }
func (m *mockAdapter) Version() string  { return m.version }

func (m *mockAdapter) ReceiveMessage(_ context.Context, _ json.RawMessage) (*InboundMessage, error) {
	return &InboundMessage{ID: "msg-1", Platform: m.id}, nil
}

func (m *mockAdapter) ReceiveWebhook(_ context.Context, _ *http.Request) (*InboundMessage, error) {
	return &InboundMessage{ID: "webhook-1", Platform: m.id}, nil
}

func (m *mockAdapter) SendMessage(_ context.Context, _ *OutboundMessage) error {
	return nil
}

func (m *mockAdapter) SendReaction(_ context.Context, _ MessageRef, _ string) error {
	return nil
}

func (m *mockAdapter) FetchContext(_ context.Context, _ MessageRef) (*PlatformContext, error) {
	return &PlatformContext{ChannelName: "test-channel"}, nil
}

func (m *mockAdapter) Connect(_ context.Context, _ *ChannelCredentials) error {
	return nil
}

func (m *mockAdapter) Disconnect(_ context.Context) error {
	return nil
}

func (m *mockAdapter) HealthCheck(_ context.Context) error {
	return nil
}

// --- Registry tests ---

func TestNewAdapterRegistry(t *testing.T) {
	r := NewAdapterRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.List()) != 0 {
		t.Errorf("expected empty registry, got %d adapters", len(r.List()))
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewAdapterRegistry()
	adapter := &mockAdapter{id: "slack", version: "1.0"}

	err := r.Register(adapter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids := r.List()
	if len(ids) != 1 || ids[0] != "slack" {
		t.Errorf("expected [slack], got %v", ids)
	}
}

func TestRegistry_RegisterNil(t *testing.T) {
	r := NewAdapterRegistry()
	err := r.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil adapter")
	}
}

func TestRegistry_RegisterEmptyID(t *testing.T) {
	r := NewAdapterRegistry()
	adapter := &mockAdapter{id: "", version: "1.0"}
	err := r.Register(adapter)
	if err == nil {
		t.Fatal("expected error for empty adapter ID")
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewAdapterRegistry()
	adapter1 := &mockAdapter{id: "slack", version: "1.0"}
	adapter2 := &mockAdapter{id: "slack", version: "2.0"}

	if err := r.Register(adapter1); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	err := r.Register(adapter2)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewAdapterRegistry()
	adapter := &mockAdapter{id: "discord", version: "1.0"}
	_ = r.Register(adapter)

	got, err := r.Get("discord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID() != "discord" {
		t.Errorf("expected discord adapter, got %s", got.ID())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewAdapterRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing adapter")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewAdapterRegistry()
	adapters := []*mockAdapter{
		{id: "slack", version: "1.0"},
		{id: "discord", version: "1.0"},
		{id: "email", version: "1.0"},
	}
	for _, a := range adapters {
		_ = r.Register(a)
	}

	ids := r.List()
	if len(ids) != 3 {
		t.Fatalf("expected 3 adapters, got %d", len(ids))
	}

	sort.Strings(ids)
	expected := []string{"discord", "email", "slack"}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, id)
		}
	}
}

// --- Mock adapter behavior tests ---

func TestMockAdapter_ReceiveMessage(t *testing.T) {
	adapter := &mockAdapter{id: "slack", version: "1.0"}
	msg, err := adapter.ReceiveMessage(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Platform != "slack" {
		t.Errorf("expected platform slack, got %s", msg.Platform)
	}
}

func TestMockAdapter_FetchContext(t *testing.T) {
	adapter := &mockAdapter{id: "discord", version: "1.0"}
	ctx, err := adapter.FetchContext(context.Background(), MessageRef{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ChannelName != "test-channel" {
		t.Errorf("expected test-channel, got %s", ctx.ChannelName)
	}
}

func TestMockAdapter_Lifecycle(t *testing.T) {
	adapter := &mockAdapter{id: "email", version: "2.0"}

	// Connect
	if err := adapter.Connect(context.Background(), &ChannelCredentials{
		Platform: "email",
		TokenMap: map[string]string{"api_key": "test"},
	}); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// HealthCheck
	if err := adapter.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}

	// Disconnect
	if err := adapter.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
}

// --- Struct tests ---

func TestInboundMessage_JSON(t *testing.T) {
	msg := InboundMessage{
		ID:         "msg-1",
		ChannelID:  "ch-1",
		AuthorID:   "user-1",
		AuthorName: "Alice",
		Content:    "hello",
		Platform:   "slack",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded InboundMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != msg.ID || decoded.Platform != msg.Platform || decoded.Content != msg.Content {
		t.Errorf("round-trip mismatch: got %+v", decoded)
	}
}

func TestOutboundMessage_JSON(t *testing.T) {
	msg := OutboundMessage{
		ChannelID: "ch-1",
		Content:   "response",
		ThreadID:  "thread-1",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded OutboundMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ChannelID != msg.ChannelID || decoded.ThreadID != msg.ThreadID {
		t.Errorf("round-trip mismatch: got %+v", decoded)
	}
}

func TestChannelCredentials_JSON(t *testing.T) {
	creds := ChannelCredentials{
		Platform: "slack",
		TokenMap: map[string]string{"bot_token": "xoxb-123"},
	}
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ChannelCredentials
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.TokenMap["bot_token"] != "xoxb-123" {
		t.Errorf("expected bot_token xoxb-123, got %s", decoded.TokenMap["bot_token"])
	}
}
