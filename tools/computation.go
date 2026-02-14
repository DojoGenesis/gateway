package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Knetic/govaluate"
)

func Calculate(ctx context.Context, params map[string]interface{}) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Evaluation error: %v", r),
			}
			err = nil
		}
	}()

	expression, ok := params["expression"].(string)
	if !ok || expression == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "expression parameter is required",
		}, nil
	}

	variables := GetMapParam(params, "variables", nil)

	expr, evalErr := govaluate.NewEvaluableExpression(expression)
	if evalErr != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Invalid expression: %v", evalErr),
		}, nil
	}

	var evalResult interface{}
	if variables != nil {
		evalResult, evalErr = expr.Evaluate(variables)
	} else {
		evalResult, evalErr = expr.Evaluate(nil)
	}

	if evalErr != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Evaluation error: %v", evalErr),
		}, nil
	}

	return map[string]interface{}{
		"success":    true,
		"expression": expression,
		"result":     evalResult,
		"variables":  variables,
	}, nil
}

type e2bExecutionRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type e2bExecutionResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error"`
}

func ExecutePython(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	code, ok := params["code"].(string)
	if !ok || code == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "code parameter is required",
		}, nil
	}

	timeout := GetDurationParam(params, "timeout", 30*time.Second)
	variables := GetMapParam(params, "variables", nil)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	apiKey := os.Getenv("E2B_API_KEY")
	if apiKey == "" {
		return executePythonLocal(ctx, code, variables)
	}

	reqBody := e2bExecutionRequest{
		Language: "python",
		Code:     code,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error marshaling request: %v", err),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.e2b.dev/v1/execute", bytes.NewBuffer(jsonData))
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error creating request: %v", err),
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Execution timeout after %v", timeout),
			}, nil
		}
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error executing request: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error reading response: %v", err),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("E2B API error (status %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	var execResp e2bExecutionResponse
	if err := json.Unmarshal(body, &execResp); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error parsing response: %v", err),
		}, nil
	}

	success := execResp.Error == "" && execResp.Stderr == ""

	return map[string]interface{}{
		"success":   success,
		"code":      code,
		"stdout":    execResp.Stdout,
		"stderr":    execResp.Stderr,
		"error":     execResp.Error,
		"variables": variables,
	}, nil
}

func executePythonLocal(ctx context.Context, code string, variables map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"success": false,
		"error":   "E2B_API_KEY environment variable not set. Python execution requires E2B API access or use the fallback mock implementation for testing.",
		"code":    code,
	}, nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "calculate",
		Description: "Evaluate mathematical expressions with support for variables",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "Mathematical expression to evaluate (e.g., '2 + 2', 'x * y + 10')",
				},
				"variables": map[string]interface{}{
					"type":        "object",
					"description": "Optional variables to use in the expression (e.g., {\"x\": 5, \"y\": 3})",
				},
			},
			"required": []string{"expression"},
		},
		Function: Calculate,
	})

	RegisterTool(&ToolDefinition{
		Name:        "execute_python",
		Description: "Execute Python code in a secure sandbox environment using E2B API",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"description": "Python code to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Execution timeout in seconds (default: 30)",
				},
				"variables": map[string]interface{}{
					"type":        "object",
					"description": "Optional variables to inject into execution context",
				},
			},
			"required": []string{"code"},
		},
		Function: ExecutePython,
	})
}
