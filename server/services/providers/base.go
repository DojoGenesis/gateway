package providers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

// APIKeyResolver dynamically resolves API keys at request time.
// This allows providers to pick up keys added via the Dev Mode UI
// without requiring a restart.
type APIKeyResolver func(ctx context.Context) string

// BaseProvider contains shared logic for all HTTP-based LLM provider adapters.
type BaseProvider struct {
	Name        string
	BaseURL     string
	APIKey      string
	KeyResolver APIKeyResolver
	Client      *http.Client
	EnvKeyName  string // e.g. "ANTHROPIC_API_KEY"
}

// ResolveAPIKey returns the current API key, checking:
// 1. Dynamic resolver (secure storage / Dev Mode UI)
// 2. Static key (from config)
// 3. Environment variable
func (b *BaseProvider) ResolveAPIKey(ctx context.Context) string {
	if b.KeyResolver != nil {
		if key := b.KeyResolver(ctx); key != "" {
			return key
		}
	}
	if b.APIKey != "" {
		return b.APIKey
	}
	return os.Getenv(b.EnvKeyName)
}

// HasAPIKey returns true if the provider has a usable API key.
func (b *BaseProvider) HasAPIKey(ctx context.Context) bool {
	return b.ResolveAPIKey(ctx) != ""
}

// SetKeyResolver sets the dynamic key resolver.
func (b *BaseProvider) SetKeyResolver(resolver APIKeyResolver) {
	b.KeyResolver = resolver
}

// SetAPIKey updates the static API key on the provider.
// This allows hot-updating a registered provider's key without restart.
func (b *BaseProvider) SetAPIKey(key string) {
	b.APIKey = key
}

// DoRequest executes an HTTP request with standard headers and error handling.
func (b *BaseProvider) DoRequest(ctx context.Context, method, path string, body io.Reader, extraHeaders map[string]string) (*http.Response, error) {
	url := b.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request to %s failed: %w", b.Name, err)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("%s API returned status %d: %s", b.Name, resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// StreamSSE reads an SSE stream and sends parsed data lines to a channel.
// Handles "data: [DONE]" termination and context cancellation.
func (b *BaseProvider) StreamSSE(ctx context.Context, resp *http.Response, ch chan<- string) {
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	gotData := false
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			// Log non-empty, non-SSE lines — these are often API error responses
			// that would otherwise be silently dropped.
			if line != "" && !strings.HasPrefix(line, ":") && !strings.HasPrefix(line, "event:") && !strings.HasPrefix(line, "id:") && !strings.HasPrefix(line, "retry:") {
				slog.Warn("non-SSE line in stream response",
					"provider", b.Name, "line", line)
			}
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}
		gotData = true
		ch <- data
	}
	if !gotData {
		slog.Warn("SSE stream closed with zero data lines",
			"provider", b.Name, "status", resp.StatusCode)
	}
}

// NewHTTPClient creates a standard HTTP client for provider use.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Minute, // Long timeout for streaming completions
	}
}

// ProviderRegistrationInfo describes a provider for registration purposes.
type ProviderRegistrationInfo struct {
	Name      string
	Available bool   // Whether this provider can serve requests right now
	Reason    string // Why it's unavailable (e.g. "no API key")
	Provider  provider.ModelProvider
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
