package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Adapter compliance
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Compliance(t *testing.T) {
	a := NewTelegramAdapter("test-token", "test-secret")

	validPayload := mustMarshal(t, Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 1,
			From:      &User{ID: 1, Username: "test"},
			Chat:      &Chat{ID: 1, Type: "private"},
			Date:      1712332800,
			Text:      "compliance test",
		},
	})

	invalidPayload := []byte(`{not valid json}`)

	signedReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		r.Header.Set(secretTokenHeader, "test-secret")
		return r
	}

	unsignedReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		r.Header.Set(secretTokenHeader, "wrong-secret")
		return r
	}

	channel.AdapterComplianceHelper(t, a, validPayload, invalidPayload, signedReq, unsignedReq,
		func(t *testing.T, _ channel.WebhookAdapter) {
			caps := a.Capabilities()
			if caps.SupportsThreads {
				t.Error("Telegram adapter should not support threads")
			}
		},
	)
}

// ---------------------------------------------------------------------------
// End-to-end smoke test
// ---------------------------------------------------------------------------

func TestTelegram_EndToEnd_SmokeTest(t *testing.T) {
	// Capture outbound Send calls via a mock HTTP server.
	var sentBody []byte
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer apiServer.Close()

	a := &TelegramAdapter{
		token:  "smoketoken",
		secret: "smoke-secret",
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{
				base:       http.DefaultTransport,
				serverAddr: apiServer.URL,
			},
		},
	}

	// Wire through the channel bridge.
	bus := &channel.InProcessBus{}
	gw := channel.NewWebhookGateway(bus, nil)
	gw.Register("telegram", a)

	runner := &mockWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 1},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("telegram", a)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "telegram", Workflow: "tg-smoke"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Build a webhook payload.
	payload := mustMarshal(t, Update{
		UpdateID: 500,
		Message: &Message{
			MessageID: 88,
			From:      &User{ID: 555, Username: "smoker"},
			Chat:      &Chat{ID: 9999, Type: "private"},
			Date:      1712332800,
			Text:      "telegram smoke trigger",
		},
	})

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/webhooks/telegram", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(secretTokenHeader, "smoke-secret")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /webhooks/telegram: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200", resp.StatusCode)
	}

	// Verify the adapter's Send was called.
	if len(sentBody) == 0 {
		t.Fatal("expected outbound sendMessage call, got none")
	}

	var sendReq sendMessageRequest
	if err := json.Unmarshal(sentBody, &sendReq); err != nil {
		t.Fatalf("unmarshal send body: %v", err)
	}
	if sendReq.ChatID != "9999" {
		t.Errorf("sendMessage chat_id = %q, want %q", sendReq.ChatID, "9999")
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — callback_query
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_CallbackQuery(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{
		UpdateID: 600,
		CallbackQuery: &CallbackQuery{
			ID:   "cb-001",
			From: &User{ID: 777, Username: "clicker", FirstName: "Click"},
			Message: &Message{
				MessageID: 99,
				Chat:      &Chat{ID: 888, Type: "group"},
				Date:      1712332800,
			},
			Data: "button_action_confirm",
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize callback_query: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "telegram")
	assertField(t, "ID", msg.ID, "cb-001")
	assertField(t, "Text", msg.Text, "button_action_confirm")
	assertField(t, "UserID", msg.UserID, "777")
	assertField(t, "UserName", msg.UserName, "clicker")
	assertField(t, "ChannelID", msg.ChannelID, "888")
	assertField(t, "ReplyTo", msg.ReplyTo, "99")

	// Verify metadata.
	if msg.Metadata["event_type"] != "callback_query" {
		t.Errorf("metadata event_type = %v, want callback_query", msg.Metadata["event_type"])
	}
}

func TestTelegramAdapter_Normalize_CallbackQuery_NoFrom(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{
		UpdateID: 601,
		CallbackQuery: &CallbackQuery{
			ID:   "cb-002",
			Data: "orphan_click",
		},
	})

	_, err := a.Normalize(raw)
	if err == nil {
		t.Fatal("expected error for callback_query without from, got nil")
	}
}

func TestTelegramAdapter_Normalize_CallbackQuery_NoMessage(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{
		UpdateID: 602,
		CallbackQuery: &CallbackQuery{
			ID:   "cb-003",
			From: &User{ID: 999, Username: "test"},
			Data: "no_msg_click",
		},
	})

	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize callback_query without message: %v", err)
	}
	// ChannelID should be empty when no source message.
	if msg.ChannelID != "" {
		t.Errorf("ChannelID = %q, want empty", msg.ChannelID)
	}
}

// ---------------------------------------------------------------------------
// Test: HandleWebhook with callback_query
// ---------------------------------------------------------------------------

func TestTelegramAdapter_HandleWebhook_CallbackQuery(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{
		UpdateID: 700,
		CallbackQuery: &CallbackQuery{
			ID:   "cb-hook",
			From: &User{ID: 111, Username: "hooker"},
			Message: &Message{
				MessageID: 50,
				Chat:      &Chat{ID: 222, Type: "private"},
				Date:      1712332800,
			},
			Data: "hook_action",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	a.HandleWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleWebhook callback_query status = %d, want 200", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — no message and no callback_query
// ---------------------------------------------------------------------------

func TestTelegramAdapter_Normalize_Empty(t *testing.T) {
	a := NewTelegramAdapter("test-token", "")

	raw := mustMarshal(t, Update{UpdateID: 800})
	_, err := a.Normalize(raw)
	if err == nil {
		t.Fatal("expected error for empty update, got nil")
	}
	if !strings.Contains(err.Error(), "no message or callback_query") {
		t.Errorf("error = %q, want to contain 'no message or callback_query'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
var _ channel.WebhookAdapter = (*TelegramAdapter)(nil)
