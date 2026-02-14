package provider

import "context"

// InProcessProvider wraps a ModelProvider for use without gRPC.
// This allows embedding provider implementations directly in the host process
// without the overhead of inter-process communication.
type InProcessProvider struct {
	impl ModelProvider
}

// NewInProcessProvider creates an InProcessProvider wrapping the given implementation.
func NewInProcessProvider(impl ModelProvider) *InProcessProvider {
	return &InProcessProvider{impl: impl}
}

func (p *InProcessProvider) GetInfo(ctx context.Context) (*ProviderInfo, error) {
	return p.impl.GetInfo(ctx)
}

func (p *InProcessProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return p.impl.ListModels(ctx)
}

func (p *InProcessProvider) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return p.impl.GenerateCompletion(ctx, req)
}

func (p *InProcessProvider) GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error) {
	return p.impl.GenerateCompletionStream(ctx, req)
}

func (p *InProcessProvider) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	return p.impl.CallTool(ctx, req)
}

func (p *InProcessProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return p.impl.GenerateEmbedding(ctx, text)
}
