package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUserAgent       = "DojoGenesis/0.0.16"
	defaultTimeoutFallback = 30 * time.Second
	maxRetries             = 3
	retryDelay             = 1 * time.Second
)

var defaultTimeout = getWebTimeout()

func getWebTimeout() time.Duration {
	if val := os.Getenv("WEB_OPERATION_TIMEOUT"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultTimeoutFallback
}

type httpClient struct {
	client *http.Client
}

func newHTTPClient(timeout time.Duration) *httpClient {
	return &httpClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *httpClient) doRequestWithRetry(ctx context.Context, req *http.Request, retries int) (*http.Response, error) {
	var lastErr error

	for i := 0; i <= retries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay * time.Duration(i)):
			}
		}

		resp, err := c.client.Do(req)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return resp, nil
			}
			if resp.StatusCode >= 500 && resp.StatusCode < 600 && i < retries {
				resp.Body.Close()
				lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
				continue
			}
			return resp, nil
		}

		lastErr = err
		if !isRetriableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset")
}

func WebSearch(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "query parameter is required",
		}, nil
	}

	maxResults := GetIntParam(params, "max_results", 10)
	searchEngine := GetStringParam(params, "search_engine", "serpapi")

	if searchEngine != "serpapi" {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported search engine: %s", searchEngine),
		}, nil
	}

	apiKey := os.Getenv("SERPAPI_KEY")
	if apiKey == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "SERPAPI_KEY environment variable not set",
		}, nil
	}

	url := fmt.Sprintf("https://serpapi.com/search?q=%s&num=%d&api_key=%s",
		strings.ReplaceAll(query, " ", "+"),
		maxResults,
		apiKey)

	timeout := GetDurationParam(params, "timeout", defaultTimeout)
	client := newHTTPClient(timeout)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.doRequestWithRetry(ctx, req, maxRetries)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body)),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to parse response: %v", err),
		}, nil
	}

	organicResults, _ := result["organic_results"].([]interface{})
	results := make([]map[string]interface{}, 0, len(organicResults))

	for _, item := range organicResults {
		if r, ok := item.(map[string]interface{}); ok {
			results = append(results, map[string]interface{}{
				"title":   r["title"],
				"link":    r["link"],
				"snippet": r["snippet"],
			})
		}
	}

	return map[string]interface{}{
		"success": true,
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}

func FetchURL(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	headers := GetMapParam(params, "headers", map[string]interface{}{})
	timeout := GetDurationParam(params, "timeout", defaultTimeout)
	followRedirects := GetBoolParam(params, "follow_redirects", true)

	client := newHTTPClient(timeout)
	if !followRedirects {
		client.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	for key, val := range headers {
		if strVal, ok := val.(string); ok {
			req.Header.Set(key, strVal)
		}
	}

	resp, err := client.doRequestWithRetry(ctx, req, maxRetries)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	return map[string]interface{}{
		"success":     true,
		"url":         url,
		"status_code": resp.StatusCode,
		"headers":     respHeaders,
		"content":     string(body),
		"size_bytes":  len(body),
	}, nil
}

func APICall(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "url parameter is required",
		}, nil
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return map[string]interface{}{
			"success": false,
			"error":   "url must start with http:// or https://",
		}, nil
	}

	method := strings.ToUpper(GetStringParam(params, "method", "GET"))
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}

	if !validMethods[method] {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported HTTP method: %s", method),
		}, nil
	}

	headers := GetMapParam(params, "headers", map[string]interface{}{})
	body := GetMapParam(params, "body", nil)
	timeout := GetDurationParam(params, "timeout", defaultTimeout)

	var bodyReader io.Reader
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("failed to marshal body: %v", err),
			}, nil
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	client := newHTTPClient(timeout)

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, val := range headers {
		if strVal, ok := val.(string); ok {
			req.Header.Set(key, strVal)
		}
	}

	resp, err := client.doRequestWithRetry(ctx, req, maxRetries)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	var jsonResponse interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		json.Unmarshal(respBody, &jsonResponse)
	}

	result := map[string]interface{}{
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
		"url":         url,
		"method":      method,
		"status_code": resp.StatusCode,
		"headers":     respHeaders,
		"size_bytes":  len(respBody),
	}

	if jsonResponse != nil {
		result["json"] = jsonResponse
	} else {
		result["content"] = string(respBody)
	}

	return result, nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "web_search",
		Description: "Search the web using a search engine (default: SerpAPI)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query string",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 10)",
					"default":     10,
				},
				"search_engine": map[string]interface{}{
					"type":        "string",
					"description": "Search engine to use (default: serpapi)",
					"default":     "serpapi",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
			},
			"required": []string{"query"},
		},
		Function: WebSearch,
	})

	RegisterTool(&ToolDefinition{
		Name:        "fetch_url",
		Description: "Fetch content from a URL using HTTP GET",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to fetch (must start with http:// or https://)",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "HTTP headers to include in the request",
					"default":     map[string]interface{}{},
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
				"follow_redirects": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to follow HTTP redirects (default: true)",
					"default":     true,
				},
			},
			"required": []string{"url"},
		},
		Function: FetchURL,
	})

	RegisterTool(&ToolDefinition{
		Name:        "api_call",
		Description: "Make a generic REST API call with full control over method, headers, and body",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "API endpoint URL (must start with http:// or https://)",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method (GET, POST, PUT, PATCH, DELETE)",
					"default":     "GET",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "HTTP headers to include in the request",
					"default":     map[string]interface{}{},
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "Request body (JSON object for POST/PUT/PATCH)",
					"default":     nil,
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30)",
					"default":     30,
				},
			},
			"required": []string{"url"},
		},
		Function: APICall,
	})
}
