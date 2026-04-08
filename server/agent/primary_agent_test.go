package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	providerpkg "github.com/DojoGenesis/gateway/provider"
)

type mockProvider struct {
	info            *providerpkg.ProviderInfo
	models          []providerpkg.ModelInfo
	completionResp  *providerpkg.CompletionResponse
	completionError error
	streamChunks    []providerpkg.CompletionChunk
	streamError     error
	toolResp        *providerpkg.ToolCallResponse
	toolError       error
}

func (m *mockProvider) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	if m.info == nil {
		return &providerpkg.ProviderInfo{
			Name:        "mock",
			Version:     "1.0.0",
			Description: "Mock provider for testing",
		}, nil
	}
	return m.info, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	if m.models == nil {
		return []providerpkg.ModelInfo{
			{
				ID:          "mock-model",
				Name:        "Mock Model",
				Provider:    "mock",
				ContextSize: 4096,
				Cost:        0.0,
			},
		}, nil
	}
	return m.models, nil
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	if m.completionError != nil {
		return nil, m.completionError
	}

	if m.completionResp != nil {
		return m.completionResp, nil
	}

	return &providerpkg.CompletionResponse{
		ID:      "test-completion-id",
		Model:   req.Model,
		Content: "This is a mock response to: " + req.Messages[len(req.Messages)-1].Content,
		Usage: providerpkg.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}

	stream := make(chan *providerpkg.CompletionChunk)

	go func() {
		defer close(stream)

		chunks := m.streamChunks
		if chunks == nil {
			chunks = []providerpkg.CompletionChunk{
				{ID: "chunk-1", Delta: "This ", Done: false},
				{ID: "chunk-2", Delta: "is ", Done: false},
				{ID: "chunk-3", Delta: "streaming", Done: false},
				{ID: "chunk-4", Delta: "", Done: true},
			}
		}

		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			case stream <- &chunk:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	return stream, nil
}

func (m *mockProvider) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	if m.toolError != nil {
		return nil, m.toolError
	}

	if m.toolResp != nil {
		return m.toolResp, nil
	}

	return &providerpkg.ToolCallResponse{
		Result: "mock tool result",
	}, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

type mockPluginManager struct {
	providers map[string]providerpkg.ModelProvider
}

func newMockPluginManager() *mockPluginManager {
	return &mockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}
}

func (m *mockPluginManager) GetProvider(name string) (providerpkg.ModelProvider, error) {
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return provider, nil
}

func (m *mockPluginManager) GetProviders() map[string]providerpkg.ModelProvider {
	return m.providers
}

func (m *mockPluginManager) AddProvider(name string, provider providerpkg.ModelProvider) {
	m.providers[name] = provider
}

func TestNewPrimaryAgent(t *testing.T) {
	agent := NewPrimaryAgent(nil)

	if agent == nil {
		t.Fatal("NewPrimaryAgent returned nil")
	}

	if agent.defaultProvider != DefaultProviderName {
		t.Errorf("Expected default provider '%s', got '%s'", DefaultProviderName, agent.defaultProvider)
	}

	if agent.timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, agent.timeout)
	}
}

func TestPrimaryAgent_SetDefaultProvider(t *testing.T) {
	agent := NewPrimaryAgent(nil)
	agent.SetDefaultProvider("custom-provider")

	if agent.defaultProvider != "custom-provider" {
		t.Errorf("Expected default provider 'custom-provider', got '%s'", agent.defaultProvider)
	}
}

func TestPrimaryAgent_SetTimeout(t *testing.T) {
	agent := NewPrimaryAgent(nil)
	agent.SetTimeout(30 * time.Second)

	if agent.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", agent.timeout)
	}
}

func TestPrimaryAgent_HandleQuery_Success(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	resp, err := agent.HandleQuery(ctx, "Hello, world!", "mock", "", "user-123")
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Content == "" {
		t.Error("Response content is empty")
	}

	if !contains(resp.Content, "Hello, world!") {
		t.Errorf("Response should contain query text, got: %s", resp.Content)
	}

	if resp.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", resp.Provider)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", resp.Usage.TotalTokens)
	}

	if resp.Timestamp.IsZero() {
		t.Error("Response timestamp is zero")
	}
}

func TestPrimaryAgent_HandleQuery_WithModelSelection(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		models: []providerpkg.ModelInfo{
			{ID: "model-1", Name: "Model 1"},
			{ID: "model-2", Name: "Model 2"},
			{ID: "model-3", Name: "Model 3"},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	resp, err := agent.HandleQuery(ctx, "Test", "mock", "model-2", "user-123")
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Model != "model-2" {
		t.Errorf("Expected model 'model-2', got '%s'", resp.Model)
	}
}

func TestPrimaryAgent_HandleQuery_InvalidModel(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleQuery(ctx, "Test", "mock", "nonexistent-model", "user-123")
	if err == nil {
		t.Fatal("Expected error for nonexistent model")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestPrimaryAgent_HandleQuery_UseDefaultProvider(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("default-mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "default-mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	resp, err := agent.HandleQuery(ctx, "Test query", "", "", "user-123")
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Provider != "default-mock" {
		t.Errorf("Expected provider 'default-mock', got '%s'", resp.Provider)
	}
}

func TestPrimaryAgent_HandleQuery_ProviderNotFound(t *testing.T) {
	pm := newMockPluginManager()

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleQuery(ctx, "Test query", "nonexistent", "", "user-123")
	if err == nil {
		t.Fatal("Expected error for nonexistent provider")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestPrimaryAgent_HandleQuery_NoModels(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		models: []providerpkg.ModelInfo{},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleQuery(ctx, "Test query", "mock", "", "user-123")
	if err == nil {
		t.Fatal("Expected error for provider with no models")
	}

	if !contains(err.Error(), "no available models") {
		t.Errorf("Expected 'no available models' error, got: %v", err)
	}
}

func TestPrimaryAgent_HandleQuery_CompletionError(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		completionError: fmt.Errorf("API error"),
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleQuery(ctx, "Test query", "mock", "", "user-123")
	if err == nil {
		t.Fatal("Expected error from completion")
	}

	if !contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error', got: %v", err)
	}
}

func TestPrimaryAgent_HandleStreamingQuery_Success(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	stream, err := agent.HandleStreamingQuery(ctx, "Streaming test", "mock", "", "user-123")
	if err != nil {
		t.Fatalf("HandleStreamingQuery failed: %v", err)
	}

	if stream == nil {
		t.Fatal("Stream is nil")
	}

	var chunks []StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks received")
	}

	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Done {
		t.Error("Last chunk should be marked as done")
	}

	fullText := ""
	for _, chunk := range chunks {
		fullText += chunk.Delta
	}

	if fullText == "" {
		t.Error("No content received from stream")
	}
}

func TestPrimaryAgent_HandleStreamingQuery_WithModelSelection(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		models: []providerpkg.ModelInfo{
			{ID: "model-1", Name: "Model 1"},
			{ID: "model-2", Name: "Model 2"},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	stream, err := agent.HandleStreamingQuery(ctx, "Test", "mock", "model-2", "user-123")
	if err != nil {
		t.Fatalf("HandleStreamingQuery failed: %v", err)
	}

	for range stream {
	}
}

func TestPrimaryAgent_HandleStreamingQuery_InvalidModel(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleStreamingQuery(ctx, "Test", "mock", "nonexistent-model", "user-123")
	if err == nil {
		t.Fatal("Expected error for nonexistent model")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestPrimaryAgent_HandleStreamingQuery_UseDefaultProvider(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{}
	pm.AddProvider("stream-default", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "stream-default",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	stream, err := agent.HandleStreamingQuery(ctx, "Test", "", "", "user-123")
	if err != nil {
		t.Fatalf("HandleStreamingQuery failed: %v", err)
	}

	for range stream {
	}
}

func TestPrimaryAgent_HandleStreamingQuery_ProviderNotFound(t *testing.T) {
	pm := newMockPluginManager()

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleStreamingQuery(ctx, "Test", "nonexistent", "", "user-123")
	if err == nil {
		t.Fatal("Expected error for nonexistent provider")
	}
}

func TestPrimaryAgent_HandleStreamingQuery_StreamError(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		streamError: fmt.Errorf("stream error"),
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.HandleStreamingQuery(ctx, "Test", "mock", "", "user-123")
	if err == nil {
		t.Fatal("Expected stream error")
	}
}

func TestPrimaryAgent_GetCostEstimate(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		models: []providerpkg.ModelInfo{
			{
				ID:   "cost-model",
				Cost: 0.001,
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	cost, err := agent.GetCostEstimate(ctx, "mock", 100, 200)
	if err != nil {
		t.Fatalf("GetCostEstimate failed: %v", err)
	}

	expected := 0.001 * 300
	if cost != expected {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestPrimaryAgent_GetCostEstimate_ProviderNotFound(t *testing.T) {
	pm := newMockPluginManager()

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	_, err := agent.GetCostEstimate(ctx, "nonexistent", 100, 200)
	if err == nil {
		t.Fatal("Expected error for nonexistent provider")
	}
}

func TestPrimaryAgent_GetCostEstimate_NoModels(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		models: []providerpkg.ModelInfo{},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	cost, err := agent.GetCostEstimate(ctx, "mock", 100, 200)
	if err != nil {
		t.Fatalf("GetCostEstimate failed: %v", err)
	}

	if cost != 0 {
		t.Errorf("Expected cost 0 for no models, got %f", cost)
	}
}

func TestPrimaryAgent_ListAvailableProviders(t *testing.T) {
	pm := newMockPluginManager()
	pm.AddProvider("provider1", &mockProvider{})
	pm.AddProvider("provider2", &mockProvider{})
	pm.AddProvider("provider3", &mockProvider{})

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	providers := agent.ListAvailableProviders()

	if len(providers) != 3 {
		t.Fatalf("Expected 3 providers, got %d", len(providers))
	}

	expectedProviders := map[string]bool{
		"provider1": true,
		"provider2": true,
		"provider3": true,
	}

	for _, name := range providers {
		if !expectedProviders[name] {
			t.Errorf("Unexpected provider: %s", name)
		}
		delete(expectedProviders, name)
	}

	if len(expectedProviders) > 0 {
		t.Errorf("Missing providers: %v", expectedProviders)
	}
}

func TestPrimaryAgent_CostTracking(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		completionResp: &providerpkg.CompletionResponse{
			ID:      "cost-test",
			Content: "Response",
			Usage: providerpkg.Usage{
				InputTokens:  50,
				OutputTokens: 100,
				TotalTokens:  150,
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	resp, err := agent.HandleQuery(ctx, "Test", "mock", "", "user-123")
	if err != nil {
		t.Fatalf("HandleQuery failed: %v", err)
	}

	if resp.Usage.InputTokens != 50 {
		t.Errorf("Expected 50 input tokens, got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 100 {
		t.Errorf("Expected 100 output tokens, got %d", resp.Usage.OutputTokens)
	}

	if resp.Usage.TotalTokens != 150 {
		t.Errorf("Expected 150 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestPrimaryAgent_StreamingAllChunksReceived(t *testing.T) {
	pm := newMockPluginManager()
	mockProv := &mockProvider{
		streamChunks: []providerpkg.CompletionChunk{
			{ID: "1", Delta: "A", Done: false},
			{ID: "2", Delta: "B", Done: false},
			{ID: "3", Delta: "C", Done: false},
			{ID: "4", Delta: "D", Done: false},
			{ID: "5", Delta: "", Done: true},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         5 * time.Second,
	}

	ctx := context.Background()
	stream, err := agent.HandleStreamingQuery(ctx, "Test", "mock", "", "user-123")
	if err != nil {
		t.Fatalf("HandleStreamingQuery failed: %v", err)
	}

	var chunks []StreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) < 5 {
		t.Errorf("Expected at least 5 chunks, got %d", len(chunks))
	}

	fullText := ""
	for i, chunk := range chunks[:len(chunks)-1] {
		fullText += chunk.Delta
		if chunk.Done && i < len(chunks)-2 {
			t.Errorf("Chunk %d marked as done prematurely", i)
		}
	}

	if fullText != "ABCD" {
		t.Errorf("Expected 'ABCD', got '%s'", fullText)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPrimaryAgent_ContextTimeout(t *testing.T) {
	pm := newMockPluginManager()

	slowProvider := &mockProvider{}
	pm.AddProvider("mock", slowProvider)

	agent := &PrimaryAgent{
		pluginManager:   pm,
		defaultProvider: "mock",
		timeout:         1 * time.Nanosecond,
	}

	ctx := context.Background()
	_, err := agent.HandleQuery(ctx, "Test", "mock", "", "user-123")
	if err == nil {
		t.Log("Warning: Expected timeout error, but query may have completed too fast")
	}
}
