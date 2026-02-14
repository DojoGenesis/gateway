package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/google/uuid"
)

type MockPluginForGarden struct{}

func (m *MockPluginForGarden) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:    "mock-plugin",
		Version: "1.0.0",
	}, nil
}

func (m *MockPluginForGarden) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	return []providerpkg.ModelInfo{
		{
			ID:   "mock-model",
			Name: "Mock Model",
			Cost: 0.0,
		},
	}, nil
}

func (m *MockPluginForGarden) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	return &providerpkg.CompletionResponse{
		ID:      "mock-completion",
		Content: "This is a mock response with context.",
		Model:   req.Model,
		Usage: providerpkg.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		ToolCalls: []providerpkg.ToolCall{},
	}, nil
}

func (m *MockPluginForGarden) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk)
	go func() {
		defer close(ch)
		ch <- &providerpkg.CompletionChunk{Delta: "Mock", Done: false}
		ch <- &providerpkg.CompletionChunk{Delta: " response", Done: false}
		ch <- &providerpkg.CompletionChunk{Delta: "", Done: true}
	}()
	return ch, nil
}

func (m *MockPluginForGarden) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: map[string]interface{}{"success": true},
		Error:  "",
	}, nil
}

func (m *MockPluginForGarden) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

type MockPluginManagerForGarden struct {
	provider *MockPluginForGarden
}

func (m *MockPluginManagerForGarden) GetProvider(name string) (providerpkg.ModelProvider, error) {
	return m.provider, nil
}

func (m *MockPluginManagerForGarden) GetProviders() map[string]providerpkg.ModelProvider {
	return map[string]providerpkg.ModelProvider{
		"mock-plugin": m.provider,
	}
}

func (m *MockPluginManagerForGarden) CompressHistory(ctx context.Context, sessionID string, memories []memory.Memory) (*memory.CompressedHistory, error) {
	return &memory.CompressedHistory{
		CompressedContent: "mock compressed history",
	}, nil
}

func (m *MockPluginManagerForGarden) ExtractSeeds(ctx context.Context, memories []memory.Memory) ([]*memory.MemorySeed, error) {
	return nil, nil
}

func setupGardenTest(t *testing.T) (*PrimaryAgent, *memory.MemoryManager, *memory.GardenManager, func()) {
	tmpDir, err := os.MkdirTemp("", "garden-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test_garden.db")

	memManager, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	mockPluginManager := &MockPluginManagerForGarden{
		provider: &MockPluginForGarden{},
	}

	gardenManager, err := memory.NewGardenManager(memManager, mockPluginManager)
	if err != nil {
		memManager.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create garden manager: %v", err)
	}

	contextBuilder := memory.NewContextBuilder(gardenManager)

	agent := NewPrimaryAgent(mockPluginManager)
	agent.SetMemoryManager(memManager)
	agent.SetGardenManager(gardenManager, contextBuilder)

	cleanup := func() {
		memManager.Close()
		os.RemoveAll(tmpDir)
	}

	return agent, memManager, gardenManager, cleanup
}

func TestGardenIntegration_BasicQuery(t *testing.T) {
	agent, _, _, cleanup := setupGardenTest(t)
	defer cleanup()

	ctx := context.Background()

	req := QueryRequest{
		Query:     "What is the capital of France?",
		UserID:    "test-user",
		UserTier:  "authenticated",
		UseMemory: true,
	}

	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Content == "" {
		t.Error("Expected non-empty response content")
	}
}

func TestGardenIntegration_ContextBuilding(t *testing.T) {
	agent, memManager, gardenManager, cleanup := setupGardenTest(t)
	defer cleanup()

	ctx := context.Background()

	sessionID := "test-session-1"

	for i := 0; i < 15; i++ {
		mem := memory.Memory{
			ID:      uuid.New().String(),
			Type:    "conversation",
			Content: "Test conversation turn",
			Metadata: map[string]interface{}{
				"session_id": sessionID,
				"turn":       i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := memManager.Store(ctx, mem); err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}
	}

	seed := &memory.Seed{
		ID:          uuid.New().String(),
		Name:        "Test Pattern",
		Description: "A test knowledge seed",
		Trigger:     "test, pattern",
		Content:     "This is test knowledge that should be recalled.",
		UsageCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := gardenManager.StoreSeed(ctx, seed); err != nil {
		t.Fatalf("Failed to store seed: %v", err)
	}

	req := QueryRequest{
		Query:     "Tell me about the test pattern",
		UserID:    sessionID,
		UserTier:  "authenticated",
		UseMemory: true,
	}

	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
}

func TestGardenIntegration_WithoutGarden(t *testing.T) {
	mockPluginManager := &MockPluginManagerForGarden{
		provider: &MockPluginForGarden{},
	}

	agent := NewPrimaryAgent(mockPluginManager)

	ctx := context.Background()

	req := QueryRequest{
		Query:     "What is the capital of France?",
		UserID:    "test-user",
		UserTier:  "authenticated",
		UseMemory: false,
	}

	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Content == "" {
		t.Error("Expected non-empty response content")
	}
}

func TestGardenIntegration_ContextCapacity(t *testing.T) {
	agent, memManager, _, cleanup := setupGardenTest(t)
	defer cleanup()

	ctx := context.Background()

	sessionID := "test-session-capacity"

	for i := 0; i < 50; i++ {
		mem := memory.Memory{
			ID:   uuid.New().String(),
			Type: "conversation",
			Content: "This is a long conversation turn with lots of content that should test the context capacity limits of the system. " +
				"We want to ensure that the context builder properly manages token limits and prunes content when necessary.",
			Metadata: map[string]interface{}{
				"session_id": sessionID,
				"turn":       i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := memManager.Store(ctx, mem); err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}
	}

	req := QueryRequest{
		Query:     "Summarize our conversation",
		UserID:    sessionID,
		UserTier:  "authenticated",
		UseMemory: true,
	}

	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
}

func TestGardenIntegration_NoMemory(t *testing.T) {
	agent, _, _, cleanup := setupGardenTest(t)
	defer cleanup()

	ctx := context.Background()

	req := QueryRequest{
		Query:     "What is 2+2?",
		UserID:    "test-user",
		UserTier:  "guest",
		UseMemory: false,
	}

	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
}
