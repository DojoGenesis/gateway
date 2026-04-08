package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	providerpkg "github.com/DojoGenesis/gateway/provider"
)

type MockPluginManager struct {
	providers map[string]providerpkg.ModelProvider
}

func (m *MockPluginManager) GetProvider(name string) (providerpkg.ModelProvider, error) {
	if provider, ok := m.providers[name]; ok {
		return provider, nil
	}
	return nil, fmt.Errorf("provider not found: %s", name)
}

func (m *MockPluginManager) GetProviders() map[string]providerpkg.ModelProvider {
	return m.providers
}

type MockModelProvider struct {
	models             []providerpkg.ModelInfo
	completionResponse *providerpkg.CompletionResponse
	completionError    error
}

func (m *MockModelProvider) Name() string {
	return "mock-provider"
}

func (m *MockModelProvider) Version() string {
	return "1.0.0"
}

func (m *MockModelProvider) Info() providerpkg.ProviderInfo {
	return providerpkg.ProviderInfo{
		Name:        "mock-provider",
		Version:     "1.0.0",
		Description: "Mock provider for testing",
	}
}

func (m *MockModelProvider) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	return m.models, nil
}

func (m *MockModelProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	if m.completionError != nil {
		return nil, m.completionError
	}
	return m.completionResponse, nil
}

func (m *MockModelProvider) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk)
	close(ch)
	return ch, nil
}

func (m *MockModelProvider) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:        "mock-provider",
		Version:     "1.0.0",
		Description: "Mock provider for testing",
	}, nil
}

func (m *MockModelProvider) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: "mock tool result",
		Error:  "",
	}, nil
}

func (m *MockModelProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func setupMemoryTest(t *testing.T) (*PrimaryAgent, *memory.MemoryManager, string, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	mm, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	mockProvider := &MockModelProvider{
		models: []providerpkg.ModelInfo{
			{
				ID:          "test-model",
				Name:        "Test Model",
				Provider:    "mock-provider",
				ContextSize: 4096,
				Cost:        0.0,
			},
		},
		completionResponse: &providerpkg.CompletionResponse{
			ID:      "test-completion-id",
			Model:   "test-model",
			Content: "This is a test response",
			Usage: providerpkg.Usage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
			ToolCalls: []providerpkg.ToolCall{},
		},
	}

	pm := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{
			"mock-provider": mockProvider,
		},
	}

	agent := NewPrimaryAgentWithConfig(pm, "mock-provider", "mock-provider", "mock-provider")
	agent.SetMemoryManager(mm)

	cleanup := func() {
		mm.Close()
		os.RemoveAll(tempDir)
	}

	return agent, mm, dbPath, cleanup
}

func TestPrimaryAgent_SetMemoryManager(t *testing.T) {
	pm := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{},
	}
	agent := NewPrimaryAgent(pm)

	if agent.memoryManager != nil {
		t.Error("Expected memoryManager to be nil initially")
	}

	mm, err := memory.NewMemoryManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}
	defer mm.Close()

	agent.SetMemoryManager(mm)

	if agent.memoryManager == nil {
		t.Error("Expected memoryManager to be set")
	}
}

func TestPrimaryAgent_BuildMessagesWithContext_NoMemory(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	systemPrompt := "You are a helpful assistant"
	query := "What is 2+2?"
	userID := "test-user-1"

	messages, err := agent.buildMessagesWithContext(ctx, systemPrompt, query, userID, false)
	if err != nil {
		t.Fatalf("Failed to build messages: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages (system + user), got %d", len(messages))
	}

	if messages[0].Role != "system" || messages[0].Content != systemPrompt {
		t.Errorf("Expected system message, got role=%s content=%s", messages[0].Role, messages[0].Content)
	}

	if messages[1].Role != "user" || messages[1].Content != query {
		t.Errorf("Expected user message, got role=%s content=%s", messages[1].Role, messages[1].Content)
	}

	_ = mm
}

func TestPrimaryAgent_BuildMessagesWithContext_WithMemory(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-2"

	convMem1 := ConversationMemory{
		UserMessage:      "What is Go?",
		AssistantMessage: "Go is a programming language.",
		Timestamp:        time.Now().Add(-2 * time.Hour),
		Model:            "test-model",
		Provider:         "mock-provider",
	}
	convJSON1, _ := json.Marshal(convMem1)
	mem1 := memory.Memory{
		ID:        "mem-1",
		Type:      "conversation:test-user-2",
		Content:   string(convJSON1),
		Metadata:  map[string]interface{}{"user_id": userID},
		CreatedAt: convMem1.Timestamp,
		UpdatedAt: convMem1.Timestamp,
	}
	if err := mm.Store(ctx, mem1); err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	convMem2 := ConversationMemory{
		UserMessage:      "Tell me more",
		AssistantMessage: "Go was created at Google.",
		Timestamp:        time.Now().Add(-1 * time.Hour),
		Model:            "test-model",
		Provider:         "mock-provider",
	}
	convJSON2, _ := json.Marshal(convMem2)
	mem2 := memory.Memory{
		ID:        "mem-2",
		Type:      "conversation:test-user-2",
		Content:   string(convJSON2),
		Metadata:  map[string]interface{}{"user_id": userID},
		CreatedAt: convMem2.Timestamp,
		UpdatedAt: convMem2.Timestamp,
	}
	if err := mm.Store(ctx, mem2); err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	systemPrompt := "You are a helpful assistant"
	query := "What year was it created?"

	messages, err := agent.buildMessagesWithContext(ctx, systemPrompt, query, userID, true)
	if err != nil {
		t.Fatalf("Failed to build messages: %v", err)
	}

	if len(messages) < 5 {
		t.Errorf("Expected at least 5 messages (system + 2 conversation turns + new user), got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("Expected first message to be system, got %s", messages[0].Role)
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" || lastMsg.Content != query {
		t.Errorf("Expected last message to be new user query, got role=%s content=%s", lastMsg.Role, lastMsg.Content)
	}
}

func TestPrimaryAgent_BuildMessagesWithContext_NoMemoryManager(t *testing.T) {
	pm := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{},
	}
	agent := NewPrimaryAgent(pm)

	ctx := context.Background()
	systemPrompt := "You are a helpful assistant"
	query := "Test query"
	userID := "test-user-3"

	messages, err := agent.buildMessagesWithContext(ctx, systemPrompt, query, userID, true)
	if err != nil {
		t.Fatalf("Failed to build messages: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages when memory manager is nil, got %d", len(messages))
	}
}

func TestPrimaryAgent_BuildMessagesWithContext_EmptyUserID(t *testing.T) {
	agent, _, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	systemPrompt := "You are a helpful assistant"
	query := "Test query"

	messages, err := agent.buildMessagesWithContext(ctx, systemPrompt, query, "", true)
	if err != nil {
		t.Fatalf("Failed to build messages: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages when userID is empty, got %d", len(messages))
	}
}

func TestPrimaryAgent_StoreConversation(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-4"
	query := "What is Kubernetes?"
	response := &Response{
		ID:        "resp-1",
		Content:   "Kubernetes is a container orchestration platform.",
		Model:     "test-model",
		Provider:  "mock-provider",
		Usage:     providerpkg.Usage{InputTokens: 10, OutputTokens: 15, TotalTokens: 25},
		ToolCalls: []providerpkg.ToolCall{},
		Timestamp: time.Now(),
	}

	err := agent.storeConversation(ctx, userID, query, response)
	if err != nil {
		t.Fatalf("Failed to store conversation: %v", err)
	}

	memories, err := mm.SearchByType(ctx, "conversation:test-user-4", 10)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(memories))
	}

	var convMem ConversationMemory
	if err := json.Unmarshal([]byte(memories[0].Content), &convMem); err != nil {
		t.Fatalf("Failed to unmarshal conversation memory: %v", err)
	}

	if convMem.UserMessage != query {
		t.Errorf("Expected user message %s, got %s", query, convMem.UserMessage)
	}

	if convMem.AssistantMessage != response.Content {
		t.Errorf("Expected assistant message %s, got %s", response.Content, convMem.AssistantMessage)
	}

	if convMem.Model != response.Model {
		t.Errorf("Expected model %s, got %s", response.Model, convMem.Model)
	}
}

func TestPrimaryAgent_StoreConversation_NoMemoryManager(t *testing.T) {
	pm := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{},
	}
	agent := NewPrimaryAgent(pm)

	ctx := context.Background()
	userID := "test-user-5"
	query := "Test query"
	response := &Response{
		ID:        "resp-2",
		Content:   "Test response",
		Model:     "test-model",
		Provider:  "mock-provider",
		Usage:     providerpkg.Usage{},
		ToolCalls: []providerpkg.ToolCall{},
		Timestamp: time.Now(),
	}

	err := agent.storeConversation(ctx, userID, query, response)
	if err != nil {
		t.Errorf("Expected no error when memory manager is nil, got %v", err)
	}
}

func TestPrimaryAgent_StoreConversation_EmptyUserID(t *testing.T) {
	agent, _, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	query := "Test query"
	response := &Response{
		ID:        "resp-3",
		Content:   "Test response",
		Model:     "test-model",
		Provider:  "mock-provider",
		Usage:     providerpkg.Usage{},
		ToolCalls: []providerpkg.ToolCall{},
		Timestamp: time.Now(),
	}

	err := agent.storeConversation(ctx, "", query, response)
	if err != nil {
		t.Errorf("Expected no error when userID is empty, got %v", err)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_WithMemory(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-6"

	convMem1 := ConversationMemory{
		UserMessage:      "What is Docker?",
		AssistantMessage: "Docker is a containerization platform.",
		Timestamp:        time.Now().Add(-1 * time.Hour),
		Model:            "test-model",
		Provider:         "mock-provider",
	}
	convJSON1, _ := json.Marshal(convMem1)
	mem1 := memory.Memory{
		ID:        "mem-3",
		Type:      "conversation:test-user-6",
		Content:   string(convJSON1),
		Metadata:  map[string]interface{}{"user_id": userID},
		CreatedAt: convMem1.Timestamp,
		UpdatedAt: convMem1.Timestamp,
	}
	if err := mm.Store(ctx, mem1); err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	req := QueryRequest{
		Query:     "Tell me more about it",
		UserID:    userID,
		UserTier:  "authenticated",
		UseMemory: true,
	}

	response, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("Failed to handle query: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Content == "" {
		t.Error("Expected non-empty response content")
	}

	memories, err := mm.SearchByType(ctx, "conversation:test-user-6", 10)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}

	if len(memories) != 2 {
		t.Errorf("Expected 2 memories (1 existing + 1 new), got %d", len(memories))
	}
}

func TestPrimaryAgent_HandleQueryWithTools_WithoutMemory(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-7"

	req := QueryRequest{
		Query:     "What is Python?",
		UserID:    userID,
		UserTier:  "guest",
		UseMemory: false,
	}

	response, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("Failed to handle query: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	memories, err := mm.SearchByType(ctx, "conversation:test-user-7", 10)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}

	if len(memories) != 0 {
		t.Errorf("Expected 0 memories when UseMemory=false, got %d", len(memories))
	}
}

func TestPrimaryAgent_MultipleConversationTurns(t *testing.T) {
	agent, mm, _, cleanup := setupMemoryTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-8"

	queries := []string{
		"What is Rust?",
		"What are its main features?",
		"How does it compare to C++?",
	}

	for i, query := range queries {
		req := QueryRequest{
			Query:     query,
			UserID:    userID,
			UserTier:  "authenticated",
			UseMemory: true,
		}

		response, err := agent.HandleQueryWithTools(ctx, req)
		if err != nil {
			t.Fatalf("Failed to handle query %d: %v", i+1, err)
		}

		if response == nil {
			t.Fatalf("Expected response for query %d, got nil", i+1)
		}
	}

	memories, err := mm.SearchByType(ctx, "conversation:test-user-8", 10)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}

	if len(memories) != 3 {
		t.Errorf("Expected 3 memories, got %d", len(memories))
	}

	for i := range memories {
		var convMem ConversationMemory
		if err := json.Unmarshal([]byte(memories[i].Content), &convMem); err != nil {
			t.Errorf("Failed to unmarshal memory %d: %v", i, err)
		}

		if convMem.UserMessage == "" || convMem.AssistantMessage == "" {
			t.Errorf("Memory %d has empty messages", i)
		}
	}
}

func TestPrimaryAgent_GenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) != 32 {
		t.Errorf("Expected ID length 32 (16 bytes hex), got %d", len(id1))
	}
}
