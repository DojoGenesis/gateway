package slack

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
	slackgo "github.com/slack-go/slack"
)

// ---------------------------------------------------------------------------
// Adapter compliance
// ---------------------------------------------------------------------------

func TestSlackAdapter_Compliance(t *testing.T) {
	adapter := newTestAdapter()

	validPayload := []byte(`{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U12345",
			"text": "compliance test",
			"channel": "C98765",
			"ts": "1609459200.000001"
		}
	}`)

	invalidPayload := []byte(`{"type": "event_callback", "event": null}`)

	signedReq := func() *http.Request {
		body := []byte(`{"type":"event_callback","event":{"text":"hi"}}`)
		return buildSignedRequest(t, body)
	}

	unsignedReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/webhooks/slack", strings.NewReader(`{}`))
		r.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
		r.Header.Set("X-Slack-Signature", "v0=badbadbadbadbadbad")
		return r
	}

	channel.AdapterComplianceHelper(t, adapter, validPayload, invalidPayload, signedReq, unsignedReq,
		func(t *testing.T, _ channel.WebhookAdapter) {
			// Verify Slack-specific: adapter supports threads.
			caps := adapter.Capabilities()
			if !caps.SupportsThreads {
				t.Error("Slack adapter should support threads")
			}
		},
	)
}

// ---------------------------------------------------------------------------
// End-to-end smoke test: synthetic webhook POST -> normalize to ChannelMessage
// -> publish to in-process bus -> mock WorkflowRunner.Execute -> adapter.Send
// ---------------------------------------------------------------------------

func TestSlack_EndToEnd_SmokeTest(t *testing.T) {
	// Build the Slack adapter with a mock sender.
	mock := &mockSlackSender{}
	cfg := SlackConfig{ //nolint:gosec // G101 -- test fixture, not a real credential
		BotToken:      "xoxb-smoke-test",
		SigningSecret: testSigningSecret,
		Mode:          "http",
	}
	adapter := NewWithSender(cfg, mock)

	// Wire through the channel bridge.
	bus := &channel.InProcessBus{}
	gw := channel.NewWebhookGateway(bus, nil)
	gw.Register("slack", adapter)

	runner := &mockWorkflowRunner{
		result: &channel.WorkflowRunResult{Status: "completed", StepCount: 2},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("slack", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "slack", Workflow: "smoke-test"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// Build a signed webhook payload.
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"user": "U_SMOKE",
			"text": "smoke test trigger",
			"channel": "C_SMOKE",
			"ts": "1609459200.000001"
		}
	}`
	body := []byte(payload)

	// Start HTTP server.
	srv := httptest.NewServer(gw)
	defer srv.Close()

	// Build a properly signed request (signature headers are required by
	// the WebhookGateway -> adapter.VerifySignature flow).
	signedReq := buildSignedRequest(t, body)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhooks/slack", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header = signedReq.Header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /webhooks/slack: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200", resp.StatusCode)
	}

	// Verify the adapter's Send was called via mock.
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 PostMessage call, got %d", len(mock.calls))
	}
	if mock.calls[0].channelID != "C_SMOKE" {
		t.Errorf("PostMessage channelID = %q, want %q", mock.calls[0].channelID, "C_SMOKE")
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — app_mention event
// ---------------------------------------------------------------------------

func TestSlackAdapter_Normalize_AppMention(t *testing.T) {
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "app_mention",
			"user": "U_MENTION",
			"text": "<@B123> run audit",
			"channel": "C_MENTION",
			"ts": "1609459200.000001"
		}
	}`

	a := newTestAdapter()
	msg, err := a.Normalize([]byte(payload))
	if err != nil {
		t.Fatalf("Normalize app_mention: %v", err)
	}
	if msg.UserID != "U_MENTION" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "U_MENTION")
	}
	if msg.Text != "<@B123> run audit" {
		t.Errorf("Text = %q", msg.Text)
	}
	// Verify metadata captures the event type.
	if et, ok := msg.Metadata["slack_event_type"]; !ok || et != "app_mention" {
		t.Errorf("slack_event_type = %v, want %q", et, "app_mention")
	}
}

// ---------------------------------------------------------------------------
// Test: Normalize — message_changed event
// ---------------------------------------------------------------------------

func TestSlackAdapter_Normalize_MessageChanged(t *testing.T) {
	payload := `{
		"type": "event_callback",
		"event": {
			"type": "message",
			"subtype": "message_changed",
			"channel": "C_EDITED",
			"ts": "1609459300.000000",
			"message": {
				"user": "U_EDITOR",
				"text": "edited message text",
				"ts": "1609459200.000001",
				"thread_ts": "1609459100.000000"
			}
		}
	}`

	a := newTestAdapter()
	msg, err := a.Normalize([]byte(payload))
	if err != nil {
		t.Fatalf("Normalize message_changed: %v", err)
	}
	if msg.UserID != "U_EDITOR" {
		t.Errorf("UserID = %q, want %q", msg.UserID, "U_EDITOR")
	}
	if msg.Text != "edited message text" {
		t.Errorf("Text = %q, want %q", msg.Text, "edited message text")
	}
	if msg.ChannelID != "C_EDITED" {
		t.Errorf("ChannelID = %q, want %q", msg.ChannelID, "C_EDITED")
	}
	if msg.ThreadID != "1609459100.000000" {
		t.Errorf("ThreadID = %q, want %q", msg.ThreadID, "1609459100.000000")
	}
	if et, ok := msg.Metadata["slack_event_type"]; !ok || et != "message_changed" {
		t.Errorf("slack_event_type = %v, want %q", et, "message_changed")
	}
}

// ---------------------------------------------------------------------------
// Test: Socket Mode (guarded by TEST_SLACK_APP_TOKEN)
// ---------------------------------------------------------------------------

func TestSlackAdapter_SocketMode(t *testing.T) {
	appToken := os.Getenv("TEST_SLACK_APP_TOKEN")
	if appToken == "" {
		t.Skip("TEST_SLACK_APP_TOKEN not set — skipping Socket Mode test")
	}

	cfg := SlackConfig{
		BotToken:      os.Getenv("TEST_SLACK_BOT_TOKEN"),
		SigningSecret: os.Getenv("TEST_SLACK_SIGNING_SECRET"),
		Mode:          "socket",
		AppToken:      appToken,
	}

	if !cfg.IsSocketMode() {
		t.Fatal("IsSocketMode() should return true for mode=socket")
	}

	// Verify the adapter can be constructed in socket mode without panicking.
	a := New(cfg)
	if a.Name() != "slack" {
		t.Errorf("Name() = %q, want %q", a.Name(), "slack")
	}
}

// ---------------------------------------------------------------------------
// Test: SlackConfig.IsSocketMode
// ---------------------------------------------------------------------------

func TestSlackConfig_IsSocketMode(t *testing.T) {
	if (SlackConfig{Mode: "http"}).IsSocketMode() {
		t.Error("http mode should not be socket mode")
	}
	if !(SlackConfig{Mode: "socket"}).IsSocketMode() {
		t.Error("socket mode should be socket mode")
	}
	if (SlackConfig{}).IsSocketMode() {
		t.Error("empty mode should not be socket mode")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mockWorkflowRunner satisfies channel.WorkflowRunner for smoke tests.
type mockWorkflowRunner struct {
	result *channel.WorkflowRunResult
	err    error
}

func (r *mockWorkflowRunner) Execute(_ context.Context, name string) (*channel.WorkflowRunResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	res := &channel.WorkflowRunResult{
		WorkflowName: name,
		Status:       "completed",
		StepCount:    2,
	}
	if r.result != nil {
		cp := *r.result
		cp.WorkflowName = name
		res = &cp
	}
	return res, nil
}

// Compile-time interface compliance (ensures SlackAdapter still satisfies
// channel.WebhookAdapter after all changes).
var _ channel.WebhookAdapter = (*SlackAdapter)(nil)

// Compile-time interface check for slackSender.
var _ slackSender = (*slackgo.Client)(nil)
