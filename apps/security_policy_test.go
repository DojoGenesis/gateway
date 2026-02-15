package apps

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityPolicy_BuildCSPHeader_Default(t *testing.T) {
	p := NewSecurityPolicy()

	csp := p.BuildCSPHeader(nil)

	required := []string{
		"default-src 'none'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self'",
	}

	for _, r := range required {
		if !strings.Contains(csp, r) {
			t.Errorf("CSP missing directive: %s\nGot: %s", r, csp)
		}
	}
}

func TestSecurityPolicy_BuildCSPHeader_WithOrigins(t *testing.T) {
	p := NewSecurityPolicy()
	p.AddAllowedOrigin("https://api.example.com")

	meta := &ResourceMeta{
		URI: "ui://test/app.html",
		CSP: []string{"https://api.example.com", "https://evil.com"},
	}

	csp := p.BuildCSPHeader(meta)

	if !strings.Contains(csp, "https://api.example.com") {
		t.Errorf("CSP should include allowed origin, got: %s", csp)
	}
	if strings.Contains(csp, "https://evil.com") {
		t.Errorf("CSP should NOT include non-allowed origin, got: %s", csp)
	}
}

func TestSecurityPolicy_ValidatePermissions_Safe(t *testing.T) {
	p := NewSecurityPolicy()

	safe := []string{"clipboard-read", "clipboard-write", "fullscreen"}
	if err := p.ValidatePermissions(safe); err != nil {
		t.Fatalf("expected safe permissions to pass, got: %v", err)
	}
}

func TestSecurityPolicy_ValidatePermissions_Dangerous(t *testing.T) {
	p := NewSecurityPolicy()

	dangerous := []string{"camera"}
	if err := p.ValidatePermissions(dangerous); err == nil {
		t.Fatal("expected camera permission to fail validation")
	}

	dangerous = []string{"microphone"}
	if err := p.ValidatePermissions(dangerous); err == nil {
		t.Fatal("expected microphone permission to fail validation")
	}

	dangerous = []string{"geolocation"}
	if err := p.ValidatePermissions(dangerous); err == nil {
		t.Fatal("expected geolocation permission to fail validation")
	}
}

func TestSecurityPolicy_ValidatePermissions_Empty(t *testing.T) {
	p := NewSecurityPolicy()

	if err := p.ValidatePermissions(nil); err != nil {
		t.Fatalf("nil permissions should pass, got: %v", err)
	}
	if err := p.ValidatePermissions([]string{}); err != nil {
		t.Fatalf("empty permissions should pass, got: %v", err)
	}
}

func TestSecurityPolicy_InjectHeaders(t *testing.T) {
	p := NewSecurityPolicy()

	meta := &ResourceMeta{URI: "ui://test/app.html"}
	w := httptest.NewRecorder()

	p.InjectSecurityHeaders(w, meta)

	headers := map[string]string{
		"Content-Security-Policy":   "default-src 'none'",
		"X-Frame-Options":           "SAMEORIGIN",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), camera=(), microphone=()",
	}

	for header, expectedContains := range headers {
		got := w.Header().Get(header)
		if got == "" {
			t.Errorf("missing header: %s", header)
			continue
		}
		if !strings.Contains(got, expectedContains) {
			t.Errorf("header %s = %q, want to contain %q", header, got, expectedContains)
		}
	}
}
