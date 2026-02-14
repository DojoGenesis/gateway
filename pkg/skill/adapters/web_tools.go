package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebToolAdapter wraps gateway web tools for skill invocation.
// It provides web_search and web_fetch capabilities for Tier 2 skills.
type WebToolAdapter struct {
	braveAPIKey  string
	fallbackMode bool      // Use gateway-native search if Brave API unavailable
	httpClient   *http.Client
	maxResults   int       // Maximum search results to return
	timeout      time.Duration
}

// WebToolAdapterConfig configures the web tool adapter
type WebToolAdapterConfig struct {
	BraveAPIKey string
	MaxResults  int
	Timeout     time.Duration
}

// NewWebToolAdapter creates a new web tool adapter
func NewWebToolAdapter(config *WebToolAdapterConfig) *WebToolAdapter {
	if config == nil {
		config = &WebToolAdapterConfig{
			MaxResults: 10,
			Timeout:    10 * time.Second,
		}
	}

	return &WebToolAdapter{
		braveAPIKey:  config.BraveAPIKey,
		fallbackMode: config.BraveAPIKey == "",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		maxResults: config.MaxResults,
		timeout:    config.Timeout,
	}
}

// Search executes a web search via Brave API or fallback
func (w *WebToolAdapter) Search(ctx context.Context, query string) (map[string]interface{}, error) {
	if w.fallbackMode {
		// Fallback: use gateway-native search (placeholder for Phase 4a)
		return w.fallbackSearch(ctx, query)
	}

	// Use Brave Search API
	return w.braveSearch(ctx, query)
}

// Fetch retrieves and parses a URL
func (w *WebToolAdapter) Fetch(ctx context.Context, url string, mode string) (map[string]interface{}, error) {
	// Validate URL
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "AgenticGateway/0.3.0 (Skill Executor)")

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse based on mode
	switch mode {
	case "markdown":
		// Convert HTML to markdown (placeholder for Phase 4a)
		return map[string]interface{}{
			"content":      string(body),
			"url":          url,
			"status_code":  resp.StatusCode,
			"content_type": resp.Header.Get("Content-Type"),
			"mode":         mode,
		}, nil
	case "json":
		// Return as-is for JSON content
		return map[string]interface{}{
			"content":      string(body),
			"url":          url,
			"status_code":  resp.StatusCode,
			"content_type": resp.Header.Get("Content-Type"),
			"mode":         mode,
		}, nil
	case "raw":
		// Return raw content
		return map[string]interface{}{
			"content":      string(body),
			"url":          url,
			"status_code":  resp.StatusCode,
			"content_type": resp.Header.Get("Content-Type"),
			"mode":         mode,
		}, nil
	default:
		return nil, fmt.Errorf("invalid mode: %s (valid: markdown, json, raw)", mode)
	}
}

// braveSearch executes a search using Brave Search API
func (w *WebToolAdapter) braveSearch(ctx context.Context, query string) (map[string]interface{}, error) {
	// Build Brave Search API request
	url := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d", query, w.maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	// Set Brave API key header
	req.Header.Set("X-Subscription-Token", w.braveAPIKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Brave API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Brave API error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Brave API response: %w", err)
	}

	// Return raw JSON for now (Phase 4a)
	// Phase 4b will add structured parsing
	return map[string]interface{}{
		"query":   query,
		"results": string(body),
		"source":  "brave",
		"count":   w.maxResults,
	}, nil
}

// fallbackSearch provides a simple fallback search mechanism
// Phase 4a: returns placeholder results
// Phase 4b: will integrate with gateway-native search
func (w *WebToolAdapter) fallbackSearch(ctx context.Context, query string) (map[string]interface{}, error) {
	// Phase 4a: Return placeholder indicating fallback mode
	return map[string]interface{}{
		"query":   query,
		"results": fmt.Sprintf("Fallback search for: %s (Brave API key not configured)", query),
		"source":  "fallback",
		"count":   0,
		"warning": "Brave API key not configured. Set BRAVE_API_KEY environment variable for full search functionality.",
	}, nil
}

// IsAvailable returns true if the adapter is configured and ready
func (w *WebToolAdapter) IsAvailable() bool {
	return w.httpClient != nil
}

// IsFallbackMode returns true if the adapter is in fallback mode (no Brave API key)
func (w *WebToolAdapter) IsFallbackMode() bool {
	return w.fallbackMode
}
