package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// RegisterDIPTools registers the DIP tool suite on the MCP server.
// Each tool proxies to the DIP REST API at dipBaseURL (e.g. "http://localhost:8081").
// This exposes DIP's design intelligence capabilities to any MCP-connected agent.
func RegisterDIPTools(server Server, dipBaseURL string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. dojo.dip.query — Execute a natural-language or structured query against the DIP graph.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.query",
		Description: "Query the DIP semantic design graph using natural language or structured filters. Returns matching nodes, edges, and measurements.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language query (e.g. 'components with low color consistency')",
				},
				"node_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by node type: philosophy, substance, reference, rule, component",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 50)",
				},
			},
			"required": []string{"query"},
		},
		Handler: func(ctx context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			body, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("marshal query args: %w", err)
			}
			return dipPost(ctx, client, dipBaseURL+"/api/v1/query", body)
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.query: %w", err)
	}

	// 2. dojo.dip.component — Get a specific node (component, philosophy, etc.) by ID.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.component",
		Description: "Retrieve a specific node from the DIP design graph by its UUID. Returns the full node with type-specific extensions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "The UUID of the node to retrieve",
				},
			},
			"required": []string{"id"},
		},
		Handler: func(ctx context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			id, _ := args["id"].(string)
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}
			return dipGet(ctx, client, fmt.Sprintf("%s/api/v1/nodes/%s", dipBaseURL, url.PathEscape(id)))
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.component: %w", err)
	}

	// 3. dojo.dip.score — Record a measurement (score) for a component against a philosophy lens.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.score",
		Description: "Record a design measurement: score a component against a philosophy lens. Supports manual, LLM, or hybrid measurement methods.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"component_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the component to score",
				},
				"philosophy_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the philosophy to measure against",
				},
				"lens_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the lens to use for measurement",
				},
				"score": map[string]interface{}{
					"type":        "number",
					"description": "Score from 0.0 to 1.0",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "Measurement method: manual, llm, hybrid, painted",
				},
				"reasoning": map[string]interface{}{
					"type":        "string",
					"description": "Explanation of the score",
				},
				"deviation_state": map[string]interface{}{
					"type":        "string",
					"description": "Deviation classification: conforming, intentional, exploratory",
				},
			},
			"required": []string{"component_id", "philosophy_id", "lens_id", "score", "method"},
		},
		Handler: func(ctx context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			body, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("marshal score args: %w", err)
			}
			return dipPost(ctx, client, dipBaseURL+"/api/v1/measurements", body)
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.score: %w", err)
	}

	// 4. dojo.dip.lenses — List all available lenses.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.lenses",
		Description: "List all DIP lenses (evaluation criteria sets) available for measuring design alignment.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
			return dipGet(ctx, client, dipBaseURL+"/api/v1/lenses")
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.lenses: %w", err)
	}

	// 5. dojo.dip.profile — Get the measurement profile for a component.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.profile",
		Description: "Get the measurement profile for a component: all scores across all lenses and philosophies.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"component_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the component",
				},
			},
			"required": []string{"component_id"},
		},
		Handler: func(ctx context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			componentID, _ := args["component_id"].(string)
			if componentID == "" {
				return nil, fmt.Errorf("component_id is required")
			}
			return dipGet(ctx, client, fmt.Sprintf("%s/api/v1/components/%s/profile", dipBaseURL, url.PathEscape(componentID)))
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.profile: %w", err)
	}

	// 6. dojo.dip.deviations — Get deviation patterns for a component.
	if err := server.RegisterTool(ToolRegistration{
		Name:        "dojo.dip.deviations",
		Description: "Detect deviation patterns for a component: where it diverges from design philosophy, whether intentionally or not.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"component_id": map[string]interface{}{
					"type":        "string",
					"description": "UUID of the component",
				},
			},
			"required": []string{"component_id"},
		},
		Handler: func(ctx context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			componentID, _ := args["component_id"].(string)
			if componentID == "" {
				return nil, fmt.Errorf("component_id is required")
			}
			return dipGet(ctx, client, fmt.Sprintf("%s/api/v1/components/%s/deviations", dipBaseURL, url.PathEscape(componentID)))
		},
	}); err != nil {
		return fmt.Errorf("register dojo.dip.deviations: %w", err)
	}

	return nil
}

// dipGet performs an HTTP GET to the DIP API and returns the parsed JSON response.
func dipGet(ctx context.Context, client *http.Client, url string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", "mcp-server")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB max
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		errBody := body
		if len(errBody) > 512 {
			errBody = errBody[:512]
		}
		return nil, fmt.Errorf("DIP API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}

// uuidPattern validates that a string looks like a UUID.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// dipPost performs an HTTP POST to the DIP API and returns the parsed JSON response.
func dipPost(ctx context.Context, client *http.Client, url string, jsonBody []byte) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", "mcp-server")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB max
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		errBody := body
		if len(errBody) > 512 {
			errBody = errBody[:512]
		}
		return nil, fmt.Errorf("DIP API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return result, nil
}
