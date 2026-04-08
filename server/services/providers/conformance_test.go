package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/provider"
)

// mockOAIServer returns a mock server that responds with OpenAI-compatible responses.
func mockOAIServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for streaming
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if stream, ok := reqBody["stream"].(bool); ok && stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte(`data: {"id":"test-1","choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n"))
			w.Write([]byte(`data: {"id":"test-1","choices":[{"delta":{"content":""},"finish_reason":"stop"}]}` + "\n\n"))
			w.Write([]byte("data: [DONE]\n\n"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "test-1",
			"model": "test-model",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello, world!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		})
	}))
}

// mockAnthropicServer returns a mock server for Anthropic API.
func mockAnthropicServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if stream, ok := reqBody["stream"].(bool); ok && stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}` + "\n\n"))
			w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "msg-test-1",
			"type":  "message",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Hello, world!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		})
	}))
}

// mockGeminiServer returns a mock server for Google Gemini API.
func mockGeminiServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "" && len(r.URL.Path) > 1 {
			// Check if streaming endpoint
			if r.URL.Query().Get("alt") == "sse" {
				w.Header().Set("Content-Type", "text/event-stream")
				resp := map[string]interface{}{
					"candidates": []map[string]interface{}{
						{
							"content": map[string]interface{}{
								"parts": []map[string]interface{}{{"text": "Hello"}},
								"role":  "model",
							},
							"finishReason": "STOP",
						},
					},
				}
				data, _ := json.Marshal(resp)
				w.Write([]byte("data: " + string(data) + "\n\n"))
				return
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{{"text": "Hello, world!"}},
						"role":  "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
				"totalTokenCount":      15,
			},
		})
	}))
}

// mockOllamaServer returns a mock server for Ollama API.
func mockOllamaServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []map[string]interface{}{
					{"name": "llama3.2", "size": 4000000000},
				},
			})
		case "/api/chat":
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			if stream, ok := reqBody["stream"].(bool); ok && stream {
				// NDJSON streaming
				chunk1, _ := json.Marshal(map[string]interface{}{
					"model":   "llama3.2",
					"message": map[string]interface{}{"role": "assistant", "content": "Hello"},
					"done":    false,
				})
				chunk2, _ := json.Marshal(map[string]interface{}{
					"model":   "llama3.2",
					"message": map[string]interface{}{"role": "assistant", "content": ""},
					"done":    true,
				})
				w.Write(chunk1)
				w.Write([]byte("\n"))
				w.Write(chunk2)
				w.Write([]byte("\n"))
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"model":   "llama3.2",
				"message": map[string]interface{}{"role": "assistant", "content": "Hello, world!"},
				"done":    true,
			})
		case "/api/embeddings":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"embedding": []float32{0.1, 0.2, 0.3},
			})
		}
	}))
}

// TestConformance_GetInfo verifies GetInfo returns valid data for all providers.
func TestConformance_GetInfo(t *testing.T) {
	oaiServer := mockOAIServer(t)
	defer oaiServer.Close()
	anthropicServer := mockAnthropicServer(t)
	defer anthropicServer.Close()
	geminiServer := mockGeminiServer(t)
	defer geminiServer.Close()
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	providers := createTestProviders(oaiServer.URL, anthropicServer.URL, geminiServer.URL, ollamaServer.URL)

	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			info, err := p.GetInfo(context.Background())
			if err != nil {
				t.Fatalf("GetInfo failed: %v", err)
			}
			if info.Name == "" {
				t.Error("GetInfo returned empty name")
			}
			if info.Version == "" {
				t.Error("GetInfo returned empty version")
			}
			if len(info.Capabilities) == 0 {
				t.Error("GetInfo returned no capabilities")
			}
		})
	}
}

// TestConformance_ListModels verifies ListModels returns at least one model.
func TestConformance_ListModels(t *testing.T) {
	oaiServer := mockOAIServer(t)
	defer oaiServer.Close()
	anthropicServer := mockAnthropicServer(t)
	defer anthropicServer.Close()
	geminiServer := mockGeminiServer(t)
	defer geminiServer.Close()
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	providers := createTestProviders(oaiServer.URL, anthropicServer.URL, geminiServer.URL, ollamaServer.URL)

	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			models, err := p.ListModels(context.Background())
			if err != nil {
				t.Fatalf("ListModels failed: %v", err)
			}
			if len(models) == 0 {
				t.Error("ListModels returned no models")
			}
		})
	}
}

// TestConformance_GenerateCompletion verifies completion returns non-empty content.
func TestConformance_GenerateCompletion(t *testing.T) {
	oaiServer := mockOAIServer(t)
	defer oaiServer.Close()
	anthropicServer := mockAnthropicServer(t)
	defer anthropicServer.Close()
	geminiServer := mockGeminiServer(t)
	defer geminiServer.Close()
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	providers := createTestProviders(oaiServer.URL, anthropicServer.URL, geminiServer.URL, ollamaServer.URL)

	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			resp, err := p.GenerateCompletion(context.Background(), &provider.CompletionRequest{
				Messages: []provider.Message{{Role: "user", Content: "Hello"}},
			})
			if err != nil {
				t.Fatalf("GenerateCompletion failed: %v", err)
			}
			if resp.Content == "" {
				t.Error("GenerateCompletion returned empty content")
			}
		})
	}
}

// TestConformance_GenerateCompletionStream verifies streaming returns chunks.
func TestConformance_GenerateCompletionStream(t *testing.T) {
	oaiServer := mockOAIServer(t)
	defer oaiServer.Close()
	anthropicServer := mockAnthropicServer(t)
	defer anthropicServer.Close()
	geminiServer := mockGeminiServer(t)
	defer geminiServer.Close()
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	providers := createTestProviders(oaiServer.URL, anthropicServer.URL, geminiServer.URL, ollamaServer.URL)

	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			ch, err := p.GenerateCompletionStream(context.Background(), &provider.CompletionRequest{
				Messages: []provider.Message{{Role: "user", Content: "Hello"}},
			})
			if err != nil {
				t.Fatalf("GenerateCompletionStream failed: %v", err)
			}

			gotChunk := false
			gotDone := false
			for chunk := range ch {
				if chunk.Delta != "" {
					gotChunk = true
				}
				if chunk.Done {
					gotDone = true
				}
			}
			if !gotChunk {
				t.Error("expected at least one content chunk")
			}
			if !gotDone {
				t.Error("expected done=true chunk")
			}
		})
	}
}

// TestConformance_CallTool verifies CallTool returns an appropriate response.
func TestConformance_CallTool(t *testing.T) {
	oaiServer := mockOAIServer(t)
	defer oaiServer.Close()
	anthropicServer := mockAnthropicServer(t)
	defer anthropicServer.Close()
	geminiServer := mockGeminiServer(t)
	defer geminiServer.Close()
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	providers := createTestProviders(oaiServer.URL, anthropicServer.URL, geminiServer.URL, ollamaServer.URL)

	for name, p := range providers {
		t.Run(name, func(t *testing.T) {
			resp, err := p.CallTool(context.Background(), &provider.ToolCallRequest{
				ToolCall: provider.ToolCall{Name: "test_tool"},
			})
			if err != nil {
				t.Fatalf("CallTool failed: %v", err)
			}
			if resp.Error == "" {
				t.Error("expected CallTool to return an error message (unsupported)")
			}
		})
	}
}

// TestConformance_GenerateCompletionWithTools verifies tool calling paths.
func TestConformance_GenerateCompletionWithTools(t *testing.T) {
	// OAI mock that returns tool_calls
	oaiToolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "test-1", "model": "test-model",
			"choices": []map[string]interface{}{{
				"message": map[string]interface{}{
					"role": "assistant", "content": "",
					"tool_calls": []map[string]interface{}{{
						"id": "call_1", "type": "function",
						"function": map[string]interface{}{
							"name":      "get_weather",
							"arguments": `{"location":"Paris"}`,
						},
					}},
				},
				"finish_reason": "tool_calls",
			}},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	defer oaiToolServer.Close()

	// Anthropic mock that returns tool_use
	anthropicToolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "msg-1", "type": "message", "model": "claude-sonnet-4-20250514",
			"content": []map[string]interface{}{
				{"type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": map[string]interface{}{"location": "Paris"}},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]interface{}{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer anthropicToolServer.Close()

	// Google mock that returns functionCall
	geminiToolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{{
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"functionCall": map[string]interface{}{"name": "get_weather", "args": map[string]interface{}{"location": "Paris"}}},
					},
					"role": "model",
				},
				"finishReason": "STOP",
			}},
			"usageMetadata": map[string]interface{}{"promptTokenCount": 10, "candidatesTokenCount": 5, "totalTokenCount": 15},
		})
	}))
	defer geminiToolServer.Close()

	tools := []provider.Tool{{
		Name:        "get_weather",
		Description: "Get weather for a location",
		Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{"location": map[string]interface{}{"type": "string"}}},
	}}

	req := &provider.CompletionRequest{
		Messages: []provider.Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "What is the weather in Paris?"},
		},
		Tools:       tools,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	t.Run("openai_tools", func(t *testing.T) {
		p := NewOpenAIProvider("test-key")
		p.BaseURL = oaiToolServer.URL + "/v1"
		resp, err := p.GenerateCompletion(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.ToolCalls) == 0 {
			t.Error("expected tool calls in response")
		}
		if resp.ToolCalls[0].Name != "get_weather" {
			t.Errorf("expected tool name get_weather, got %s", resp.ToolCalls[0].Name)
		}
	})

	t.Run("anthropic_tools", func(t *testing.T) {
		p := NewAnthropicProvider("test-key")
		p.BaseURL = anthropicToolServer.URL
		resp, err := p.GenerateCompletion(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.ToolCalls) == 0 {
			t.Error("expected tool calls in response")
		}
		if resp.ToolCalls[0].Name != "get_weather" {
			t.Errorf("expected tool name get_weather, got %s", resp.ToolCalls[0].Name)
		}
	})

	t.Run("google_tools", func(t *testing.T) {
		p := NewGoogleProvider("test-key")
		p.BaseURL = geminiToolServer.URL
		resp, err := p.GenerateCompletion(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.ToolCalls) == 0 {
			t.Error("expected tool calls in response")
		}
		if resp.ToolCalls[0].Name != "get_weather" {
			t.Errorf("expected tool name get_weather, got %s", resp.ToolCalls[0].Name)
		}
	})
}

// TestConformance_GenerateEmbedding verifies embedding for Ollama.
func TestConformance_GenerateEmbedding(t *testing.T) {
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()

	p := NewOllamaProvider()
	p.BaseURL = ollamaServer.URL

	embedding, err := p.GenerateEmbedding(context.Background(), "test text")
	if err != nil {
		t.Fatalf("GenerateEmbedding failed: %v", err)
	}
	if len(embedding) == 0 {
		t.Error("expected non-empty embedding")
	}
}

// TestConformance_EmbeddingUnsupported verifies non-embedding providers return errors.
func TestConformance_EmbeddingUnsupported(t *testing.T) {
	p := NewAnthropicProvider("test-key")
	_, err := p.GenerateEmbedding(context.Background(), "test")
	if err == nil {
		t.Error("expected error from Anthropic GenerateEmbedding")
	}

	g := NewGoogleProvider("test-key")
	_, err = g.GenerateEmbedding(context.Background(), "test")
	if err == nil {
		t.Error("expected error from Google GenerateEmbedding")
	}

	o := NewOpenAIProvider("test-key")
	_, err = o.GenerateEmbedding(context.Background(), "test")
	if err == nil {
		t.Error("expected error from OpenAI GenerateEmbedding")
	}
}

// TestOllamaIsAvailable verifies the Ollama availability check.
func TestOllamaIsAvailable(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		server := mockOllamaServer(t)
		defer server.Close()

		p := NewOllamaProvider()
		p.BaseURL = server.URL

		if !p.IsAvailable(context.Background()) {
			t.Error("expected Ollama to be available")
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		p := NewOllamaProvider()
		p.BaseURL = "http://localhost:1" // unreachable

		if p.IsAvailable(context.Background()) {
			t.Error("expected Ollama to be unavailable")
		}
	})
}

// createTestProviders creates all 8 providers configured to use mock servers.
func createTestProviders(oaiURL, anthropicURL, geminiURL, ollamaURL string) map[string]provider.ModelProvider {
	result := map[string]provider.ModelProvider{}

	// OpenAI-compatible providers all point to the same mock
	openai := NewOpenAIProvider("test-key")
	openai.BaseURL = oaiURL + "/v1"
	result["openai"] = openai

	groq := NewGroqProvider("test-key")
	groq.BaseURL = oaiURL + "/v1"
	result["groq"] = groq

	mistral := NewMistralProvider("test-key")
	mistral.BaseURL = oaiURL + "/v1"
	result["mistral"] = mistral

	kimi := NewKimiProvider("test-key")
	kimi.BaseURL = oaiURL + "/v1"
	result["kimi"] = kimi

	deepseek := NewDeepSeekProvider("test-key")
	deepseek.BaseURL = oaiURL + "/v1"
	result["deepseek-api"] = deepseek

	// Custom providers
	anthropic := NewAnthropicProvider("test-key")
	anthropic.BaseURL = anthropicURL
	result["anthropic"] = anthropic

	google := NewGoogleProvider("test-key")
	google.BaseURL = geminiURL
	result["google"] = google

	ollama := NewOllamaProvider()
	ollama.BaseURL = ollamaURL
	result["ollama"] = ollama

	return result
}
