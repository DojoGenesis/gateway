package main

import (
	"strings"
	"testing"
)

// TestParseTunnelURL verifies URL extraction from various cloudflared log lines.
func TestParseTunnelURL(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantURL  string
	}{
		{
			name:    "standard quick-tunnel line",
			line:    "INF Your quick tunnel has been created! Visit it at: https://abc-def-ghi.trycloudflare.com",
			wantURL: "https://abc-def-ghi.trycloudflare.com",
		},
		{
			name:    "URL with trailing slash is trimmed",
			line:    "INF Visit https://foo-bar-baz.trycloudflare.com/ for details",
			wantURL: "https://foo-bar-baz.trycloudflare.com",
		},
		{
			name:    "URL-only line",
			line:    "https://xyz-123-abc.trycloudflare.com",
			wantURL: "https://xyz-123-abc.trycloudflare.com",
		},
		{
			name:    "no URL returns empty string",
			line:    "INF Starting tunnel credentials file=/home/user/.cloudflared/cert.pem",
			wantURL: "",
		},
		{
			name:    "non-trycloudflare https URL is ignored",
			line:    "INF Registered tunnel connection https://example.com",
			wantURL: "",
		},
		{
			name:    "empty line returns empty string",
			line:    "",
			wantURL: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTunnelURL(tc.line)
			if got != tc.wantURL {
				t.Errorf("parseTunnelURL(%q) = %q, want %q", tc.line, got, tc.wantURL)
			}
		})
	}
}

// TestGenerateWebhookURLs verifies that all four platform webhook paths are produced.
func TestGenerateWebhookURLs(t *testing.T) {
	tunnelURL := "https://abc-def-ghi.trycloudflare.com"
	webhooks := generateWebhookURLs(tunnelURL)

	expected := map[string]string{
		"Slack":    tunnelURL + "/webhooks/slack",
		"Discord":  tunnelURL + "/webhooks/discord",
		"Telegram": tunnelURL + "/webhooks/telegram",
		"Email":    tunnelURL + "/webhooks/email",
	}

	for platform, want := range expected {
		got, ok := webhooks[platform]
		if !ok {
			t.Errorf("generateWebhookURLs: missing key %q", platform)
			continue
		}
		if got != want {
			t.Errorf("generateWebhookURLs[%q] = %q, want %q", platform, got, want)
		}
	}

	// No unexpected keys.
	if len(webhooks) != len(expected) {
		t.Errorf("generateWebhookURLs returned %d entries, want %d", len(webhooks), len(expected))
	}
}

// TestGenerateWebhookURLs_TrailingSlash ensures a trailing slash on the tunnel URL
// does not produce double slashes in webhook paths.
func TestGenerateWebhookURLs_TrailingSlash(t *testing.T) {
	tunnelURL := "https://abc-def-ghi.trycloudflare.com/"
	webhooks := generateWebhookURLs(tunnelURL)

	for platform, got := range webhooks {
		if strings.Contains(got, "//webhooks") {
			t.Errorf("generateWebhookURLs[%q] has double slash: %q", platform, got)
		}
	}
}

// TestTunnelCommand_NoCloudflared verifies that when cloudflared is absent the
// error message contains install instructions — without requiring the binary.
func TestTunnelCommand_NoCloudflared(t *testing.T) {
	err := buildCloudflaredNotFoundError()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	msg := err.Error()

	if !strings.Contains(msg, "cloudflared not found") {
		t.Errorf("error message missing 'cloudflared not found': %q", msg)
	}

	if !strings.Contains(msg, "Install it") {
		t.Errorf("error message missing 'Install it': %q", msg)
	}

	// On any platform the message should reference cloudflared somehow.
	if !strings.Contains(msg, "cloudflared") {
		t.Errorf("error message missing install hint reference to cloudflared: %q", msg)
	}
}

// TestRunTunnelCommand_InvalidPort checks argument validation for non-numeric ports.
func TestRunTunnelCommand_InvalidPort(t *testing.T) {
	err := runTunnelCommand([]string{"notaport"})
	if err == nil {
		t.Fatal("expected error for invalid port, got nil")
	}
	if !strings.Contains(err.Error(), "invalid port") {
		t.Errorf("expected 'invalid port' in error, got: %q", err.Error())
	}
}
