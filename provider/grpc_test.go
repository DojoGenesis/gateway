package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertModelInfoSlice(t *testing.T) {
	models := []ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai", ContextSize: 128000, Cost: 0.03},
		{ID: "gpt-3.5", Name: "GPT-3.5", Provider: "openai", ContextSize: 16385, Cost: 0.0005},
	}

	protoModels := convertModelInfoSlice(models)
	assert.Len(t, protoModels, 2)
	assert.Equal(t, "gpt-4", protoModels[0].Id)
	assert.Equal(t, int32(128000), protoModels[0].ContextSize)
	assert.InDelta(t, 0.03, protoModels[0].Cost, 1e-10)
}

func TestConvertFromProtoModelInfoSlice(t *testing.T) {
	protoModels := convertModelInfoSlice([]ModelInfo{
		{ID: "model-1", Name: "Model 1", Provider: "test", ContextSize: 4096, Cost: 0.01},
	})

	models := convertFromProtoModelInfoSlice(protoModels)
	assert.Len(t, models, 1)
	assert.Equal(t, "model-1", models[0].ID)
	assert.Equal(t, 4096, models[0].ContextSize)
}

func TestConvertCompletionRequestRoundTrip(t *testing.T) {
	req := &CompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []Tool{
			{
				Name:        "search",
				Description: "Search the web",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
		Stream: true,
	}

	protoReq := convertToProtoCompletionRequest(req)
	roundTripped := convertFromProtoCompletionRequest(protoReq)

	assert.Equal(t, req.Model, roundTripped.Model)
	assert.Len(t, roundTripped.Messages, 2)
	assert.Equal(t, "system", roundTripped.Messages[0].Role)
	assert.Equal(t, "Hello", roundTripped.Messages[1].Content)
	assert.InDelta(t, req.Temperature, roundTripped.Temperature, 1e-10)
	assert.Equal(t, req.MaxTokens, roundTripped.MaxTokens)
	assert.Equal(t, req.Stream, roundTripped.Stream)
	assert.Len(t, roundTripped.Tools, 1)
	assert.Equal(t, "search", roundTripped.Tools[0].Name)
}

func TestConvertCompletionResponseRoundTrip(t *testing.T) {
	resp := &CompletionResponse{
		ID:      "chatcmpl-abc",
		Model:   "gpt-4",
		Content: "Hello there!",
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
		ToolCalls: []ToolCall{
			{
				ID:        "tc-1",
				Name:      "search",
				Arguments: map[string]interface{}{"query": "golang"},
			},
		},
	}

	protoResp := convertToProtoCompletionResponse(resp)
	roundTripped := convertFromProtoCompletionResponse(protoResp)

	assert.Equal(t, resp.ID, roundTripped.ID)
	assert.Equal(t, resp.Model, roundTripped.Model)
	assert.Equal(t, resp.Content, roundTripped.Content)
	assert.Equal(t, resp.Usage.InputTokens, roundTripped.Usage.InputTokens)
	assert.Equal(t, resp.Usage.OutputTokens, roundTripped.Usage.OutputTokens)
	assert.Equal(t, resp.Usage.TotalTokens, roundTripped.Usage.TotalTokens)
	assert.Len(t, roundTripped.ToolCalls, 1)
	assert.Equal(t, "tc-1", roundTripped.ToolCalls[0].ID)
	assert.Equal(t, "search", roundTripped.ToolCalls[0].Name)
}

func TestConvertEmptySlices(t *testing.T) {
	// Empty model info
	protoModels := convertModelInfoSlice([]ModelInfo{})
	assert.Len(t, protoModels, 0)

	models := convertFromProtoModelInfoSlice(nil)
	assert.Len(t, models, 0)

	// Empty messages and tools
	req := &CompletionRequest{Model: "test"}
	protoReq := convertToProtoCompletionRequest(req)
	roundTripped := convertFromProtoCompletionRequest(protoReq)
	assert.Len(t, roundTripped.Messages, 0)
	assert.Len(t, roundTripped.Tools, 0)
}

func TestToolCallArgumentsSerialization(t *testing.T) {
	args := map[string]interface{}{
		"query": "golang best practices",
		"limit": float64(10),
	}

	argsBytes, err := json.Marshal(args)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(argsBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "golang best practices", parsed["query"])
	assert.Equal(t, float64(10), parsed["limit"])
}

func TestGRPCPluginType(t *testing.T) {
	plugin := &ModelProviderGRPCPlugin{
		Impl: &mockProvider{
			info: &ProviderInfo{Name: "test", Version: "1.0.0"},
		},
	}
	assert.NotNil(t, plugin.Impl)
}
