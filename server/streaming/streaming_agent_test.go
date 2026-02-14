package streaming

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

type mockPluginManager struct {
	provider provider.ModelProvider
}

func (m *mockPluginManager) GetProvider(name string) (provider.ModelProvider, error) {
	return m.provider, nil
}

func (m *mockPluginManager) GetProviders() map[string]provider.ModelProvider {
	return map[string]provider.ModelProvider{
		"mock-plugin": m.provider,
	}
}

type mockModelProvider struct {
	completionResponse *provider.CompletionResponse
	completionError    error
	models             []provider.ModelInfo
}

func (m *mockModelProvider) Name() string {
	return "mock-plugin"
}

func (m *mockModelProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	if m.models != nil {
		return m.models, nil
	}
	return []provider.ModelInfo{
		{ID: "mock-model", Name: "Mock Model", Cost: 0.0},
	}, nil
}

func (m *mockModelProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if m.completionError != nil {
		return nil, m.completionError
	}
	if m.completionResponse != nil {
		return m.completionResponse, nil
	}
	return &provider.CompletionResponse{
		ID:      "test-id",
		Content: "This is a test response",
		Model:   req.Model,
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		ToolCalls: []provider.ToolCall{},
	}, nil
}

func (m *mockModelProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk)
	close(ch)
	return ch, nil
}

func (m *mockModelProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:    "mock-plugin",
		Version: "1.0.0",
	}, nil
}

func (m *mockModelProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: map[string]interface{}{"success": true},
	}, nil
}

func (m *mockModelProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

func createTestAgent() *agent.PrimaryAgent {
	pm := &mockPluginManager{
		provider: &mockModelProvider{},
	}
	return agent.NewPrimaryAgent(pm)
}

func TestNewStreamingAgent(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgent(pa)

	if sa == nil {
		t.Fatal("NewStreamingAgent returned nil")
	}
	if sa.agent != pa {
		t.Error("StreamingAgent.agent not set correctly")
	}
}

func TestStreamingAgent_NilAgent(t *testing.T) {
	sa := &StreamingAgent{agent: nil}

	_, err := sa.HandleQueryStreaming(context.Background(), agent.QueryRequest{
		Query: "test",
	})

	if err == nil {
		t.Error("Expected error for nil agent, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Expected 'nil' in error, got: %v", err)
	}
}

func TestStreamingAgent_BasicQuery(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgent(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreaming(ctx, agent.QueryRequest{
		Query:    "What is 2+2?",
		UserTier: "guest",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreaming failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)
	if len(events) == 0 {
		t.Fatal("No events received")
	}

	hasChunk := false
	hasComplete := false

	for _, event := range events {
		switch event.Type {
		case ResponseChunk:
			hasChunk = true
		case Complete:
			hasComplete = true
			if event.Data["usage"] == nil {
				t.Error("Complete event missing usage data")
			}
		case Error:
			t.Errorf("Unexpected error event: %v", event.Data)
		}
	}

	if !hasChunk {
		t.Error("No response chunk events received")
	}
	if !hasComplete {
		t.Error("No complete event received")
	}
}

func TestStreamingAgent_ContextCancellation(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgent(pa)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	eventChan, err := sa.HandleQueryStreaming(ctx, agent.QueryRequest{
		Query: "test",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreaming failed: %v", err)
	}

	events := collectEvents(eventChan, 2*time.Second)

	for _, event := range events {
		if event.Type == Error {
			return
		}
	}
}

func TestStreamingAgent_EmptyQuery(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgent(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreaming(ctx, agent.QueryRequest{
		Query: "",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreaming failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)
	if len(events) == 0 {
		t.Fatal("No events received")
	}
}

func TestStreamingAgent_LongResponse(t *testing.T) {
	longResponse := strings.Repeat("word ", 100)

	pm := &mockPluginManager{
		provider: &mockModelProvider{
			completionResponse: &provider.CompletionResponse{
				ID:      "test-id",
				Content: longResponse,
				Model:   "mock-model",
				Usage: provider.Usage{
					InputTokens:  10,
					OutputTokens: 200,
					TotalTokens:  210,
				},
				ToolCalls: []provider.ToolCall{},
			},
		},
	}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgent(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreaming(ctx, agent.QueryRequest{
		Query: "Generate a long response",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreaming failed: %v", err)
	}

	events := collectEvents(eventChan, 10*time.Second)

	chunkCount := 0
	for _, event := range events {
		if event.Type == ResponseChunk {
			chunkCount++
		}
	}

	if chunkCount < 2 {
		t.Errorf("Expected multiple chunks for long response, got %d", chunkCount)
	}
}

func TestNewStreamingAgentWithEvents(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	if sa == nil {
		t.Fatal("NewStreamingAgentWithEvents returned nil")
	}
	if sa.agent != pa {
		t.Error("StreamingAgentWithEvents.agent not set correctly")
	}
}

func TestStreamingAgentWithEvents_NilAgent(t *testing.T) {
	sa := &StreamingAgentWithEvents{agent: nil}

	_, err := sa.HandleQueryStreamingWithEvents(context.Background(), agent.QueryRequest{
		Query: "test",
	})

	if err == nil {
		t.Error("Expected error for nil agent, got nil")
	}
}

func TestStreamingAgentWithEvents_AllEventTypes(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:    "What is AI?",
		UserTier: "guest",
		UserID:   "test-user",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)
	if len(events) == 0 {
		t.Fatal("No events received")
	}

	eventTypes := make(map[EventType]bool)
	for _, event := range events {
		eventTypes[event.Type] = true
	}

	expectedTypes := []EventType{
		IntentClassified,
		ProviderSelected,
		Thinking,
		ResponseChunk,
		Complete,
	}

	for _, expectedType := range expectedTypes {
		if !eventTypes[expectedType] {
			t.Errorf("Missing expected event type: %s", expectedType)
		}
	}
}

func TestStreamingAgentWithEvents_IntentClassification(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:    "How does this work?",
		UserTier: "guest",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	var intentEvent *StreamEvent
	for _, event := range events {
		if event.Type == IntentClassified {
			intentEvent = &event
			break
		}
	}

	if intentEvent == nil {
		t.Fatal("No intent classified event received")
	}

	intent, ok := intentEvent.Data["intent"].(string)
	if !ok {
		t.Error("Intent data missing or wrong type")
	}
	if intent == "" {
		t.Error("Intent is empty")
	}

	confidence, ok := intentEvent.Data["confidence"].(float64)
	if !ok {
		t.Error("Confidence data missing or wrong type")
	}
	if confidence < 0 || confidence > 1 {
		t.Errorf("Confidence out of range: %f", confidence)
	}
}

func TestStreamingAgentWithEvents_ProviderSelection(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:    "test",
		UserTier: "guest",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	var providerEvent *StreamEvent
	for _, event := range events {
		if event.Type == ProviderSelected {
			providerEvent = &event
			break
		}
	}

	if providerEvent == nil {
		t.Fatal("No provider selected event received")
	}

	provider, ok := providerEvent.Data["provider"].(string)
	if !ok || provider == "" {
		t.Error("Provider data missing or invalid")
	}

	model, ok := providerEvent.Data["model"].(string)
	if !ok || model == "" {
		t.Error("Model data missing or invalid")
	}
}

func TestStreamingAgentWithEvents_MemoryRetrieval(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:     "test",
		UserTier:  "guest",
		UserID:    "test-user",
		UseMemory: true,
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	var memoryEvent *StreamEvent
	for _, event := range events {
		if event.Type == MemoryRetrieved {
			memoryEvent = &event
			break
		}
	}

	if memoryEvent == nil {
		t.Fatal("No memory retrieved event received when UseMemory=true")
	}
}

func TestStreamingAgentWithEvents_ThinkingEvent(t *testing.T) {
	pa := createTestAgent()
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query: "test",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	var thinkingEvent *StreamEvent
	for _, event := range events {
		if event.Type == Thinking {
			thinkingEvent = &event
			break
		}
	}

	if thinkingEvent == nil {
		t.Fatal("No thinking event received")
	}

	message, ok := thinkingEvent.Data["message"].(string)
	if !ok || message == "" {
		t.Error("Thinking message missing or invalid")
	}
}

func TestStreamingAgentWithEvents_WithToolCalls(t *testing.T) {
	pm := &mockPluginManager{
		provider: &mockModelProvider{
			completionResponse: &provider.CompletionResponse{
				ID:      "test-id",
				Content: "Used tools to answer",
				Model:   "mock-model",
				Usage: provider.Usage{
					InputTokens:  10,
					OutputTokens: 20,
					TotalTokens:  30,
				},
				ToolCalls: []provider.ToolCall{
					{
						ID:   "tool-1",
						Name: "calculate",
						Arguments: map[string]interface{}{
							"expression": "2+2",
						},
					},
				},
			},
		},
	}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query: "What is 2+2?",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	var toolInvokedEvent *StreamEvent
	var toolCompletedEvent *StreamEvent

	for _, event := range events {
		if event.Type == ToolInvoked {
			toolInvokedEvent = &event
		}
		if event.Type == ToolCompleted {
			toolCompletedEvent = &event
		}
	}

	if toolInvokedEvent == nil {
		t.Fatal("No tool invoked event received")
	}

	if toolCompletedEvent == nil {
		t.Fatal("No tool completed event received")
	}

	toolName, ok := toolInvokedEvent.Data["tool"].(string)
	if !ok || toolName == "" {
		t.Error("Tool name missing or invalid in tool invoked event")
	}
}

func TestSplitIntoChunks_EmptyString(t *testing.T) {
	chunks := splitIntoChunks("", 50)
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty string, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_SingleWord(t *testing.T) {
	chunks := splitIntoChunks("hello", 50)
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != "hello" {
		t.Errorf("Expected 'hello', got '%s'", chunks[0])
	}
}

func TestSplitIntoChunks_MultipleWords(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog"
	chunks := splitIntoChunks(text, 20)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for text longer than chunk size, got %d", len(chunks))
	}

	for _, chunk := range chunks {
		if len(chunk) > 25 {
			t.Errorf("Chunk exceeds max size: '%s' (len=%d)", chunk, len(chunk))
		}
	}

	reconstructed := strings.Join(chunks, " ")
	if reconstructed != text {
		t.Error("Reconstructed text doesn't match original")
	}
}

func TestSplitIntoChunks_ZeroChunkSize(t *testing.T) {
	text := "hello world"
	chunks := splitIntoChunks(text, 0)

	if len(chunks) == 0 {
		t.Error("Expected chunks with default size when chunkSize=0")
	}
}

func TestSplitIntoChunks_LongWords(t *testing.T) {
	text := "supercalifragilisticexpialidocious is a long word"
	chunks := splitIntoChunks(text, 10)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func collectEvents(eventChan <-chan StreamEvent, timeout time.Duration) []StreamEvent {
	var events []StreamEvent
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return events
			}
			events = append(events, event)
		case <-timer.C:
			return events
		}
	}
}
