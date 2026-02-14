package provider

import "context"

type ModelProvider interface {
	GetInfo(ctx context.Context) (*ProviderInfo, error)
	ListModels(ctx context.Context) ([]ModelInfo, error)
	GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error)
	CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}
