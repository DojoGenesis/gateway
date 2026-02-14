package tools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCalculate(t *testing.T) {
	ctx := context.Background()

	t.Run("simple addition", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "2 + 2",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 4.0 {
			t.Errorf("expected result=4.0, got %v", result["result"])
		}
	})

	t.Run("complex expression", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "(10 + 5) * 2 - 8 / 4",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 28.0 {
			t.Errorf("expected result=28.0, got %v", result["result"])
		}
	})

	t.Run("with variables", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "x * y + 10",
			"variables": map[string]interface{}{
				"x": 5.0,
				"y": 3.0,
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 25.0 {
			t.Errorf("expected result=25.0, got %v", result["result"])
		}
	})

	t.Run("power operation", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "2 ** 8",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 256.0 {
			t.Errorf("expected result=256.0, got %v", result["result"])
		}
	})

	t.Run("modulo operation", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "17 % 5",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 2.0 {
			t.Errorf("expected result=2.0, got %v", result["result"])
		}
	})

	t.Run("comparison operations", func(t *testing.T) {
		tests := []struct {
			name       string
			expression string
			expected   bool
		}{
			{"greater than true", "10 > 5", true},
			{"greater than false", "5 > 10", false},
			{"less than true", "5 < 10", true},
			{"less than false", "10 < 5", false},
			{"equal true", "5 == 5", true},
			{"equal false", "5 == 10", false},
			{"not equal true", "5 != 10", true},
			{"not equal false", "5 != 5", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := Calculate(ctx, map[string]interface{}{
					"expression": tt.expression,
				})

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !result["success"].(bool) {
					t.Fatalf("expected success=true, got: %v", result)
				}

				if result["result"].(bool) != tt.expected {
					t.Errorf("expected result=%v, got %v", tt.expected, result["result"])
				}
			})
		}
	})

	t.Run("logical operations", func(t *testing.T) {
		tests := []struct {
			name       string
			expression string
			expected   bool
		}{
			{"AND true", "true && true", true},
			{"AND false", "true && false", false},
			{"OR true", "true || false", true},
			{"OR false", "false || false", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := Calculate(ctx, map[string]interface{}{
					"expression": tt.expression,
				})

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !result["success"].(bool) {
					t.Fatalf("expected success=true, got: %v", result)
				}

				if result["result"].(bool) != tt.expected {
					t.Errorf("expected result=%v, got %v", tt.expected, result["result"])
				}
			})
		}
	})

	t.Run("missing expression", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing expression")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})

	t.Run("invalid expression", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "2 + + 2",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for invalid expression")
		}

		if !strings.Contains(result["error"].(string), "Invalid expression") {
			t.Errorf("expected 'Invalid expression' error, got: %s", result["error"])
		}
	})

	t.Run("undefined variable", func(t *testing.T) {
		result, err := Calculate(ctx, map[string]interface{}{
			"expression": "x + 5",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for undefined variable")
		}

		if !strings.Contains(result["error"].(string), "Evaluation error") {
			t.Errorf("expected 'Evaluation error', got: %s", result["error"])
		}
	})
}

func TestExecutePython(t *testing.T) {
	ctx := context.Background()

	t.Run("missing code", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for missing code")
		}

		if !strings.Contains(result["error"].(string), "required") {
			t.Errorf("expected 'required' error, got: %s", result["error"])
		}
	})

	t.Run("no E2B API key", func(t *testing.T) {
		os.Unsetenv("E2B_API_KEY")

		result, err := ExecutePython(ctx, map[string]interface{}{
			"code": "print('Hello, World!')",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false when E2B_API_KEY is not set")
		}

		if !strings.Contains(result["error"].(string), "E2B_API_KEY") {
			t.Errorf("expected 'E2B_API_KEY' error, got: %s", result["error"])
		}
	})
}

func TestCalculateToolRegistration(t *testing.T) {
	tool, err := GetTool("calculate")
	if err != nil {
		t.Fatalf("calculate tool not registered: %v", err)
	}

	if tool.Name != "calculate" {
		t.Errorf("expected tool name 'calculate', got %s", tool.Name)
	}

	if tool.Function == nil {
		t.Errorf("expected non-nil function")
	}

	if tool.Parameters == nil {
		t.Errorf("expected non-nil parameters")
	}
}

func TestExecutePythonToolRegistration(t *testing.T) {
	tool, err := GetTool("execute_python")
	if err != nil {
		t.Fatalf("execute_python tool not registered: %v", err)
	}

	if tool.Name != "execute_python" {
		t.Errorf("expected tool name 'execute_python', got %s", tool.Name)
	}

	if tool.Function == nil {
		t.Errorf("expected non-nil function")
	}

	if tool.Parameters == nil {
		t.Errorf("expected non-nil parameters")
	}
}
