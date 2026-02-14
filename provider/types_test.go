package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderInfoFields(t *testing.T) {
	info := ProviderInfo{
		Name:         "openai",
		Version:      "1.0.0",
		Description:  "OpenAI provider",
		Capabilities: []string{"text-completion", "streaming", "embeddings"},
	}
	assert.Equal(t, "openai", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "OpenAI provider", info.Description)
	assert.Len(t, info.Capabilities, 3)
}

func TestModelInfoFields(t *testing.T) {
	model := ModelInfo{
		ID:          "gpt-4-turbo",
		Name:        "GPT-4 Turbo",
		Provider:    "openai",
		ContextSize: 128000,
		Cost:        0.00003,
	}
	assert.Equal(t, "gpt-4-turbo", model.ID)
	assert.Equal(t, 128000, model.ContextSize)
	assert.InDelta(t, 0.00003, model.Cost, 1e-10)
}

func TestCompletionRequestFields(t *testing.T) {
	req := CompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []Tool{
			{Name: "search", Description: "Search the web"},
		},
		Stream: true,
	}
	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.InDelta(t, 0.7, req.Temperature, 1e-10)
	assert.True(t, req.Stream)
	assert.Len(t, req.Tools, 1)
}

func TestCompletionResponseFields(t *testing.T) {
	resp := CompletionResponse{
		ID:      "chatcmpl-123",
		Model:   "gpt-4",
		Content: "Hello!",
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
		ToolCalls: []ToolCall{
			{ID: "tc-1", Name: "search", Arguments: map[string]interface{}{"query": "test"}},
		},
	}
	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "search", resp.ToolCalls[0].Name)
}

func TestCompletionChunkFields(t *testing.T) {
	chunk := CompletionChunk{
		ID:    "chunk-1",
		Delta: "Hello",
		Done:  false,
	}
	assert.Equal(t, "chunk-1", chunk.ID)
	assert.Equal(t, "Hello", chunk.Delta)
	assert.False(t, chunk.Done)

	finalChunk := CompletionChunk{ID: "chunk-2", Delta: "", Done: true}
	assert.True(t, finalChunk.Done)
}

func TestMessageFields(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "I'll search for that.",
		ToolCalls: []ToolCall{
			{ID: "tc-1", Name: "search"},
		},
	}
	assert.Equal(t, "assistant", msg.Role)
	assert.Len(t, msg.ToolCalls, 1)

	toolMsg := Message{
		Role:       "tool",
		Content:    `{"results": []}`,
		ToolCallID: "tc-1",
	}
	assert.Equal(t, "tool", toolMsg.Role)
	assert.Equal(t, "tc-1", toolMsg.ToolCallID)
}

func TestToolCallRequestResponse(t *testing.T) {
	req := ToolCallRequest{
		ToolCall: ToolCall{
			ID:        "tc-1",
			Name:      "web_search",
			Arguments: map[string]interface{}{"query": "golang"},
		},
		Context: map[string]interface{}{"project_id": "proj-123"},
	}
	assert.Equal(t, "web_search", req.ToolCall.Name)
	assert.Equal(t, "proj-123", req.Context["project_id"])

	resp := ToolCallResponse{
		Result: map[string]interface{}{"results": []interface{}{}},
		Error:  "",
	}
	assert.Empty(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestUsageCalculation(t *testing.T) {
	usage := Usage{
		InputTokens:  50,
		OutputTokens: 200,
		TotalTokens:  250,
	}
	assert.Equal(t, usage.InputTokens+usage.OutputTokens, usage.TotalTokens)
}
