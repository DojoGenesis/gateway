package agent

import (
	"context"
	"strings"
	"testing"

	providerpkg "github.com/DojoGenesis/gateway/provider"
)

type mockCalculationProvider struct{}

func (m *mockCalculationProvider) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:        "mock-calculation",
		Version:     "1.0.0",
		Description: "Mock provider for calculation testing",
	}, nil
}

func (m *mockCalculationProvider) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	return []providerpkg.ModelInfo{{ID: "test-model", Name: "Test Model", Cost: 0}}, nil
}

func (m *mockCalculationProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	query := ""
	if len(req.Messages) > 0 {
		query = strings.ToLower(req.Messages[len(req.Messages)-1].Content)
	}

	content := ""

	if strings.Contains(query, "2+2") || strings.Contains(query, "2 + 2") {
		content = "4"
	} else if strings.Contains(query, "hello") || strings.Contains(query, "hi") {
		content = "Hello! How can I help you today?"
	} else if strings.Contains(query, "docker") {
		content = "Docker is a platform for developing, shipping, and running applications in containers."
	} else if strings.Contains(query, "python function") || strings.Contains(query, "code") {
		content = "def example():\n    pass"
	} else if strings.Contains(query, "15% of 200") {
		content = "30"
	} else {
		content = "Response for: " + query
	}

	return &providerpkg.CompletionResponse{
		ID:      "test-response",
		Content: content,
		Model:   req.Model,
		Usage: providerpkg.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		ToolCalls: []providerpkg.ToolCall{},
	}, nil
}

func (m *mockCalculationProvider) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk, 1)

	query := ""
	if len(req.Messages) > 0 {
		query = strings.ToLower(req.Messages[len(req.Messages)-1].Content)
	}

	content := ""
	if strings.Contains(query, "2+2") || strings.Contains(query, "2 + 2") {
		content = "4"
	} else if strings.Contains(query, "hello") {
		content = "Hello! How can I help you today?"
	} else {
		content = "Streamed response"
	}

	ch <- &providerpkg.CompletionChunk{
		ID:    "test-chunk",
		Delta: content,
		Done:  true,
	}
	close(ch)
	return ch, nil
}

func (m *mockCalculationProvider) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: map[string]interface{}{"success": true},
	}, nil
}

func (m *mockCalculationProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

type mockIntegrationPluginManager struct {
	provider *mockCalculationProvider
}

func (m *mockIntegrationPluginManager) GetProvider(name string) (providerpkg.ModelProvider, error) {
	return m.provider, nil
}

func (m *mockIntegrationPluginManager) GetProviders() map[string]providerpkg.ModelProvider {
	return map[string]providerpkg.ModelProvider{
		"mock-calculation": m.provider,
	}
}

func TestEndToEndRouting(t *testing.T) {
	ic := NewIntentClassifier()
	mockPM := &mockIntegrationPluginManager{provider: &mockCalculationProvider{}}
	agent := NewPrimaryAgent(mockPM)

	tests := []struct {
		name             string
		query            string
		expectedHandler  string
		expectedCategory IntentCategory
		expectedProvider string
		validateResponse func(t *testing.T, response string)
	}{
		{
			name:             "Calculation query: what is 2+2?",
			query:            "what is 2+2?",
			expectedHandler:  "llm-fast",
			expectedCategory: Calculation,
			expectedProvider: "llm-fast",
			validateResponse: func(t *testing.T, response string) {
				if !strings.Contains(response, "4") {
					t.Errorf("Expected response to contain '4', got: %s", response)
				}
			},
		},
		{
			name:             "Greeting: hello",
			query:            "hello",
			expectedHandler:  "template",
			expectedCategory: Greeting,
			expectedProvider: "",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for greeting")
				}
			},
		},
		{
			name:             "Complex code generation",
			query:            "write a Python function to sort a list using quicksort algorithm with detailed comments and input validation",
			expectedHandler:  "llm-reasoning",
			expectedCategory: CodeGeneration,
			expectedProvider: "llm-reasoning",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for code generation")
				}
			},
		},
		{
			name:             "Factual query: what is docker?",
			query:            "what is docker?",
			expectedHandler:  "template",
			expectedCategory: Factual,
			expectedProvider: "",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for factual query")
				}
			},
		},
		{
			name:             "Math calculation: 15% of 200",
			query:            "calculate 15% of 200",
			expectedHandler:  "llm-fast",
			expectedCategory: Calculation,
			expectedProvider: "llm-fast",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for calculation")
				}
			},
		},
		{
			name:             "Debugging query",
			query:            "why am i getting a null pointer error in my code",
			expectedHandler:  "llm-fast",
			expectedCategory: Debugging,
			expectedProvider: "llm-fast",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for debugging query")
				}
			},
		},
		{
			name:             "Planning query",
			query:            "design a scalable microservices architecture for an e-commerce platform with high availability and fault tolerance",
			expectedHandler:  "llm-reasoning",
			expectedCategory: Planning,
			expectedProvider: "llm-reasoning",
			validateResponse: func(t *testing.T, response string) {
				if response == "" {
					t.Error("Expected non-empty response for planning query")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler != tt.expectedHandler {
				t.Errorf("Handler mismatch: got %s, want %s", decision.Handler, tt.expectedHandler)
			}

			if decision.Category != tt.expectedCategory {
				t.Errorf("Category mismatch: got %s, want %s", decision.Category.String(), tt.expectedCategory.String())
			}

			if decision.Provider != tt.expectedProvider {
				t.Errorf("Provider mismatch: got %s, want %s", decision.Provider, tt.expectedProvider)
			}

			if decision.Confidence < 0.0 || decision.Confidence > 1.0 {
				t.Errorf("Confidence out of range [0,1]: got %f", decision.Confidence)
			}

			if len(decision.Reasoning) == 0 {
				t.Error("Expected non-empty reasoning chain")
			}

			if tt.expectedHandler == "llm-fast" || tt.expectedHandler == "llm-reasoning" {
				ctx := context.Background()

				response, err := agent.HandleQuery(ctx, tt.query, tt.expectedProvider, "", "test-user", "")
				if err != nil {
					t.Fatalf("HandleQuery failed: %v", err)
				}

				if response.Content == "" {
					t.Error("Expected non-empty response content")
				}

				if tt.validateResponse != nil {
					tt.validateResponse(t, response.Content)
				}
			}

			t.Logf("✓ Query: %q → Handler: %s, Category: %s, Confidence: %.2f",
				tt.query, decision.Handler, decision.Category.String(), decision.Confidence)
		})
	}
}

func TestRoutingPropagation(t *testing.T) {
	ic := NewIntentClassifier()
	mockPM := &mockIntegrationPluginManager{provider: &mockCalculationProvider{}}
	agent := NewPrimaryAgent(mockPM)

	tests := []struct {
		name        string
		query       string
		stream      bool
		expectError bool
	}{
		{
			name:        "Non-streaming calculation",
			query:       "what is 2+2?",
			stream:      false,
			expectError: false,
		},
		{
			name:        "Streaming calculation",
			query:       "what is 2+2?",
			stream:      true,
			expectError: false,
		},
		{
			name:        "Non-streaming complex query",
			query:       "write a Python function to calculate fibonacci",
			stream:      false,
			expectError: false,
		},
		{
			name:        "Streaming complex query",
			query:       "write a Python function to calculate fibonacci",
			stream:      true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler == "" {
				t.Error("Expected non-empty handler")
			}

			if decision.Handler == "template" {
				t.Skip("Template handler does not use LLM provider")
			}

			ctx := context.Background()

			if tt.stream {
				stream, err := agent.HandleStreamingQuery(ctx, tt.query, "", "", "test-user")
				if err != nil {
					if !tt.expectError {
						t.Fatalf("HandleStreamingQuery failed: %v", err)
					}
					return
				}

				chunks := []string{}
				for chunk := range stream {
					if chunk.Error != nil {
						if !tt.expectError {
							t.Fatalf("Stream error: %v", chunk.Error)
						}
						return
					}
					chunks = append(chunks, chunk.Delta)
				}

				if len(chunks) == 0 && !tt.expectError {
					t.Error("Expected at least one chunk in stream")
				}

				t.Logf("✓ Streaming response received %d chunks", len(chunks))
			} else {
				response, err := agent.HandleQuery(ctx, tt.query, "", "", "test-user", "")
				if err != nil {
					if !tt.expectError {
						t.Fatalf("HandleQuery failed: %v", err)
					}
					return
				}

				if response.Content == "" && !tt.expectError {
					t.Error("Expected non-empty response content")
				}

				if response.Provider == "" && !tt.expectError {
					t.Error("Expected provider to be set in response")
				}

				t.Logf("✓ Non-streaming response: %q (provider: %s)", response.Content, response.Provider)
			}
		})
	}
}

func TestBackwardCompatibilityIntegration(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name            string
		query           string
		expectedType    QueryType
		minConfidence   float64
		checkRouting    bool
		expectedHandler string
	}{
		{
			name:            "Simple greeting via Classify()",
			query:           "hello",
			expectedType:    Simple,
			minConfidence:   0.8,
			checkRouting:    true,
			expectedHandler: "template",
		},
		{
			name:            "Complex query via Classify()",
			query:           "Build a React todo app",
			expectedType:    Complex,
			minConfidence:   0.7,
			checkRouting:    true,
			expectedHandler: "llm-reasoning",
		},
		{
			name:            "Calculation via Classify() - edge case",
			query:           "what is 2+2?",
			expectedType:    Complex,
			minConfidence:   0.8,
			checkRouting:    true,
			expectedHandler: "llm-fast",
		},
		{
			name:            "Simple factual via Classify()",
			query:           "what is docker",
			expectedType:    Simple,
			minConfidence:   0.7,
			checkRouting:    true,
			expectedHandler: "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryType := ic.Classify(tt.query)
			if queryType != tt.expectedType {
				t.Errorf("Classify(%q) = %s, want %s", tt.query, queryType.String(), tt.expectedType.String())
			}

			result := ic.ClassifyWithConfidence(tt.query)
			if result.Type != tt.expectedType {
				t.Errorf("ClassifyWithConfidence(%q).Type = %s, want %s", tt.query, result.Type.String(), tt.expectedType.String())
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Confidence too low: got %.2f, want >= %.2f", result.Confidence, tt.minConfidence)
			}

			if result.Reasoning == "" {
				t.Error("Expected non-empty reasoning")
			}

			if tt.checkRouting {
				decision := ic.Route(tt.query)
				if decision.Handler != tt.expectedHandler {
					t.Errorf("Route(%q).Handler = %s, want %s", tt.query, decision.Handler, tt.expectedHandler)
				}

				expectedType := Simple
				if decision.Handler != "template" {
					expectedType = Complex
				}

				if queryType != expectedType {
					t.Errorf("Backward compatibility issue: Classify returned %s but Route handler is %s",
						queryType.String(), decision.Handler)
				}
			}

			t.Logf("✓ Backward compatibility verified: %q → Type: %s, Confidence: %.2f",
				tt.query, result.Type.String(), result.Confidence)
		})
	}
}

func TestMixedAPIUsage(t *testing.T) {
	ic := NewIntentClassifier()

	testQueries := []string{
		"hello",
		"what is 2+2?",
		"write a Python function",
		"what is docker?",
	}

	for _, query := range testQueries {
		oldResult := ic.Classify(query)

		newDecision := ic.Route(query)

		expectedOldType := Simple
		if newDecision.Handler != "template" {
			expectedOldType = Complex
		}

		if oldResult != expectedOldType {
			t.Errorf("API inconsistency for %q: Classify() = %s, but Route().Handler = %s (expected type: %s)",
				query, oldResult.String(), newDecision.Handler, expectedOldType.String())
		}
	}

	t.Log("✓ Mixed API usage verified - old and new APIs are consistent")
}

func TestRoutingDecisionCompleteness(t *testing.T) {
	ic := NewIntentClassifier()

	queries := []string{
		"hello",
		"what is 2+2?",
		"write a Python function to sort a list",
		"what is docker?",
		"debug this error",
		"design a system architecture",
	}

	for _, query := range queries {
		decision := ic.Route(query)

		if decision.Handler == "" {
			t.Errorf("Query %q: Handler is empty", query)
		}

		if decision.Category.String() == "Unknown" {
			t.Errorf("Query %q: Category is Unknown", query)
		}

		if decision.Confidence < 0.0 || decision.Confidence > 1.0 {
			t.Errorf("Query %q: Confidence out of range: %.2f", query, decision.Confidence)
		}

		if len(decision.Reasoning) == 0 {
			t.Errorf("Query %q: Reasoning chain is empty", query)
		}

		if decision.Handler == "llm-fast" || decision.Handler == "llm-reasoning" {
			if decision.Provider == "" {
				t.Errorf("Query %q: Provider is empty for LLM handler", query)
			}
		}

		if decision.Handler == "template" && decision.Fallback != "" && decision.Fallback != "llm-fast" {
			t.Errorf("Query %q: Invalid fallback %q for template handler", query, decision.Fallback)
		}

		t.Logf("✓ Query %q: Complete routing decision (Handler: %s, Category: %s, Provider: %s, Confidence: %.2f)",
			query, decision.Handler, decision.Category.String(), decision.Provider, decision.Confidence)
	}
}
