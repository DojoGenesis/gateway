package agent

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	_ "modernc.org/sqlite"
)

type MockProvider struct {
	models             []providerpkg.ModelInfo
	completionResponse *providerpkg.CompletionResponse
}

func (m *MockProvider) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:        "mock-provider",
		Version:     "1.0.0",
		Description: "Mock provider for testing",
	}, nil
}

func (m *MockProvider) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	if len(m.models) == 0 {
		return []providerpkg.ModelInfo{
			{
				ID:          "test-model",
				Name:        "Test Model",
				Provider:    "mock-provider",
				ContextSize: 4096,
				Cost:        0.0,
			},
		}, nil
	}
	return m.models, nil
}

func (m *MockProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	if m.completionResponse != nil {
		return m.completionResponse, nil
	}
	return &providerpkg.CompletionResponse{
		ID:      "test-response-id",
		Content: "Test response",
		Model:   "test-model",
		Usage: providerpkg.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}, nil
}

func (m *MockProvider) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk)
	close(ch)
	return ch, nil
}

func (m *MockProvider) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: "mock tool result",
		Error:  "",
	}, nil
}

func (m *MockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Return a mock embedding vector
	return []float32{0.1, 0.2, 0.3}, nil
}

func setupTraceTest(t *testing.T) (*PrimaryAgent, *trace.TraceLogger, chan events.StreamEvent) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	traceStorage, err := trace.NewTraceStorage(db)
	if err != nil {
		t.Fatalf("Failed to initialize trace storage: %v", err)
	}

	eventChan := make(chan events.StreamEvent, 100)
	traceLogger := trace.NewTraceLogger(traceStorage, eventChan)

	mockPM := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{
			"mock-plugin": &MockProvider{},
		},
	}

	agent := NewPrimaryAgent(mockPM)
	agent.SetTraceLogger(traceLogger)

	return agent, traceLogger, eventChan
}

func TestHandleQueryWithTools_WithTracing(t *testing.T) {
	agent, _, eventChan := setupTraceTest(t)

	req := QueryRequest{
		Query:    "What is 2 + 2?",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	eventsReceived := 0
	timeout := time.After(1 * time.Second)

drainLoop:
	for {
		select {
		case event := <-eventChan:
			eventsReceived++
			if event.Type == events.TraceSpanStart || event.Type == events.TraceSpanEnd {
				t.Logf("Received trace event: %s", event.Type)
			}
		case <-timeout:
			break drainLoop
		}
	}

	if eventsReceived == 0 {
		t.Error("Expected to receive trace events via channel")
	}
}

func TestHandleQueryWithTools_WithoutTracing(t *testing.T) {
	mockPM := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{
			"mock-plugin": &MockProvider{},
		},
	}

	agent := NewPrimaryAgent(mockPM)

	req := QueryRequest{
		Query:    "What is 2 + 2?",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed without tracer: %v", err)
	}
}

func TestTraceInstrumentation_IntentClassification(t *testing.T) {
	agent, traceLogger, eventChan := setupTraceTest(t)

	req := QueryRequest{
		Query:    "Build a calculator function",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	foundIntentSpan := false
	timeout := time.After(1 * time.Second)

drainLoop:
	for {
		select {
		case event := <-eventChan:
			if event.Type == events.TraceSpanStart {
				if name, ok := event.Data["name"].(string); ok && name == "intent_classification" {
					foundIntentSpan = true
					t.Logf("Found intent_classification span")
				}
			}
		case <-timeout:
			break drainLoop
		}
	}

	if !foundIntentSpan {
		t.Error("Expected to find intent_classification span")
	}

	_ = traceLogger
}

func TestTraceInstrumentation_ContextBuilding(t *testing.T) {
	agent, _, eventChan := setupTraceTest(t)

	req := QueryRequest{
		Query:     "What did we discuss?",
		UserID:    "test-user",
		UserTier:  "guest",
		UseMemory: true,
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	foundContextSpan := false
	timeout := time.After(1 * time.Second)

drainLoop:
	for {
		select {
		case event := <-eventChan:
			if event.Type == events.TraceSpanStart {
				if name, ok := event.Data["name"].(string); ok && name == "context_building" {
					foundContextSpan = true
					t.Logf("Found context_building span")
				}
			}
		case <-timeout:
			break drainLoop
		}
	}

	if !foundContextSpan {
		t.Error("Expected to find context_building span")
	}
}

func TestTraceInstrumentation_ModelInvocation(t *testing.T) {
	agent, _, eventChan := setupTraceTest(t)

	req := QueryRequest{
		Query:    "Hello world",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	foundModelSpan := false
	timeout := time.After(1 * time.Second)

drainLoop:
	for {
		select {
		case event := <-eventChan:
			if event.Type == events.TraceSpanStart {
				if name, ok := event.Data["name"].(string); ok && name == "model_invocation" {
					foundModelSpan = true
					t.Logf("Found model_invocation span")
				}
			}
		case <-timeout:
			break drainLoop
		}
	}

	if !foundModelSpan {
		t.Error("Expected to find model_invocation span")
	}
}

func BenchmarkHandleQueryWithTools_NoTrace(b *testing.B) {
	mockPM := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{
			"mock-plugin": &MockProvider{},
		},
	}

	agent := NewPrimaryAgent(mockPM)

	req := QueryRequest{
		Query:    "What is 2 + 2?",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.HandleQueryWithTools(ctx, req)
	}
}

func BenchmarkHandleQueryWithTools_WithTrace(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	traceStorage, err := trace.NewTraceStorage(db)
	if err != nil {
		b.Fatalf("Failed to initialize trace storage: %v", err)
	}

	eventChan := make(chan events.StreamEvent, 1000)
	traceLogger := trace.NewTraceLogger(traceStorage, eventChan)

	mockPM := &MockPluginManager{
		providers: map[string]providerpkg.ModelProvider{
			"mock-plugin": &MockProvider{},
		},
	}

	agent := NewPrimaryAgent(mockPM)
	agent.SetTraceLogger(traceLogger)

	go func() {
		for range eventChan {
		}
	}()

	req := QueryRequest{
		Query:    "What is 2 + 2?",
		UserID:   "test-user",
		UserTier: "guest",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.HandleQueryWithTools(ctx, req)
	}
}
