package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCalculateIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("invoke through registry", func(t *testing.T) {
		result, err := InvokeTool(ctx, "calculate", map[string]interface{}{
			"expression": "42 + 8",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 50.0 {
			t.Errorf("expected result=50.0, got %v", result["result"])
		}
	})

	t.Run("invoke with timeout", func(t *testing.T) {
		result, err := InvokeToolWithTimeout(ctx, "calculate", map[string]interface{}{
			"expression": "100 * 100",
		}, 5*time.Second)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		if result["result"].(float64) != 10000.0 {
			t.Errorf("expected result=10000.0, got %v", result["result"])
		}
	})

	t.Run("complex math operations", func(t *testing.T) {
		tests := []struct {
			name       string
			expression string
			variables  map[string]interface{}
			expected   float64
		}{
			{
				name:       "quadratic formula",
				expression: "(-b + (b**2 - 4*a*c)**0.5) / (2*a)",
				variables: map[string]interface{}{
					"a": 1.0,
					"b": -5.0,
					"c": 6.0,
				},
				expected: 3.0,
			},
			{
				name:       "compound interest",
				expression: "p * (1 + r/n)**(n*t)",
				variables: map[string]interface{}{
					"p": 1000.0,
					"r": 0.05,
					"n": 12.0,
					"t": 10.0,
				},
				expected: 1647.0095,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := Calculate(ctx, map[string]interface{}{
					"expression": tt.expression,
					"variables":  tt.variables,
				})

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !result["success"].(bool) {
					t.Fatalf("expected success=true, got: %v", result)
				}

				actualResult := result["result"].(float64)
				if actualResult < tt.expected-0.01 || actualResult > tt.expected+0.01 {
					t.Errorf("expected result≈%.4f, got %.4f", tt.expected, actualResult)
				}
			})
		}
	})
}

func TestExecutePythonIntegration(t *testing.T) {
	apiKey := os.Getenv("E2B_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping E2B integration tests: E2B_API_KEY not set")
	}

	ctx := context.Background()

	t.Run("simple print statement", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{
			"code": "print('Hello from E2B!')",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		stdout := result["stdout"].(string)
		if !strings.Contains(stdout, "Hello from E2B") {
			t.Errorf("expected stdout to contain 'Hello from E2B', got: %s", stdout)
		}
	})

	t.Run("basic calculation", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{
			"code": "result = 2 + 2\nprint(result)",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		stdout := result["stdout"].(string)
		if !strings.Contains(stdout, "4") {
			t.Errorf("expected stdout to contain '4', got: %s", stdout)
		}
	})

	t.Run("import and use library", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{
			"code": "import json\ndata = {'key': 'value'}\nprint(json.dumps(data))",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}

		stdout := result["stdout"].(string)
		if !strings.Contains(stdout, "key") || !strings.Contains(stdout, "value") {
			t.Errorf("expected stdout to contain JSON, got: %s", stdout)
		}
	})

	t.Run("syntax error", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{
			"code": "print('missing quote)",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for syntax error")
		}

		stderr := result["stderr"].(string)
		if stderr == "" && result["error"].(string) == "" {
			t.Errorf("expected error output for syntax error")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		result, err := ExecutePython(ctx, map[string]interface{}{
			"code":    "import time\ntime.sleep(10)",
			"timeout": 2,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["success"].(bool) {
			t.Errorf("expected success=false for timeout")
		}

		if !strings.Contains(result["error"].(string), "timeout") {
			t.Errorf("expected timeout error, got: %s", result["error"])
		}
	})

	t.Run("invoke through registry", func(t *testing.T) {
		result, err := InvokeTool(ctx, "execute_python", map[string]interface{}{
			"code": "print('Registry invocation works!')",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", result)
		}
	})
}

func TestComputationToolsEndToEnd(t *testing.T) {
	ctx := context.Background()

	t.Run("calculate and verify with python", func(t *testing.T) {
		calcResult, err := Calculate(ctx, map[string]interface{}{
			"expression": "123 * 456",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !calcResult["success"].(bool) {
			t.Fatalf("expected success=true, got: %v", calcResult)
		}

		expectedResult := calcResult["result"].(float64)

		if os.Getenv("E2B_API_KEY") == "" {
			t.Skip("Skipping Python verification: E2B_API_KEY not set")
		}

		pythonResult, err := ExecutePython(ctx, map[string]interface{}{
			"code": "print(123 * 456)",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !pythonResult["success"].(bool) {
			t.Fatalf("expected success=true for python, got: %v", pythonResult)
		}

		stdout := strings.TrimSpace(pythonResult["stdout"].(string))
		if !strings.Contains(stdout, "56088") {
			t.Errorf("expected Python output '56088', got: %s (calc result: %v)", stdout, expectedResult)
		}
	})
}
