package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModelProviderInterfaceCompliance ensures the mockProvider satisfies ModelProvider.
func TestModelProviderInterfaceCompliance(t *testing.T) {
	var _ ModelProvider = &mockProvider{}
}

// TestModelProviderInterfaceMethodCount uses reflection to verify 6 methods.
func TestModelProviderInterfaceMethodCount(t *testing.T) {
	// Verify via mockProvider that all 6 methods exist and are callable
	mock := &mockProvider{
		info: &ProviderInfo{
			Name:    "test",
			Version: "1.0.0",
		},
	}
	ctx := context.Background()

	info, err := mock.GetInfo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test", info.Name)

	_, err = mock.ListModels(ctx)
	assert.NoError(t, err)

	resp, err := mock.GenerateCompletion(ctx, &CompletionRequest{Model: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	ch, err := mock.GenerateCompletionStream(ctx, &CompletionRequest{Model: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, ch)
	for range ch {
	}

	toolResp, err := mock.CallTool(ctx, &ToolCallRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, toolResp)

	embedding, err := mock.GenerateEmbedding(ctx, "test text")
	assert.NoError(t, err)
	assert.Len(t, embedding, 768)
}

// TestInProcessProviderCompliance ensures InProcessProvider satisfies ModelProvider.
func TestInProcessProviderCompliance(t *testing.T) {
	var _ ModelProvider = &InProcessProvider{}
}

// TestInProcessProviderDelegation ensures all calls delegate to the underlying impl.
func TestInProcessProviderDelegation(t *testing.T) {
	mock := &mockProvider{
		info: &ProviderInfo{
			Name:         "wrapped",
			Version:      "2.0.0",
			Description:  "Wrapped provider",
			Capabilities: []string{"completion"},
		},
	}
	p := NewInProcessProvider(mock)
	ctx := context.Background()

	info, err := p.GetInfo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "wrapped", info.Name)
	assert.Equal(t, "2.0.0", info.Version)

	_, err = p.ListModels(ctx)
	assert.NoError(t, err)

	resp, err := p.GenerateCompletion(ctx, &CompletionRequest{Model: "test"})
	assert.NoError(t, err)
	assert.Equal(t, "This is a mock response", resp.Content)

	ch, err := p.GenerateCompletionStream(ctx, &CompletionRequest{Model: "test"})
	assert.NoError(t, err)
	for range ch {
	}

	toolResp, err := p.CallTool(ctx, &ToolCallRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "tool result", toolResp.Result)

	embedding, err := p.GenerateEmbedding(ctx, "hello")
	assert.NoError(t, err)
	assert.Len(t, embedding, 768)
}
