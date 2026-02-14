package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/streaming"
	"github.com/stretchr/testify/assert"
)

func TestStreamingIntegration(t *testing.T) {
	mockProvider := &TestMockProvider{
		name:        "test-provider",
		models:      []provider.ModelInfo{{ID: "test-model", Name: "Test Model", ContextSize: 8192, Cost: 0.0}},
		streamDelay: 10 * time.Millisecond,
	}

	mockManager := NewMockPluginManager()
	mockManager.AddProvider("test-provider", mockProvider)

	pa := agent.NewPrimaryAgentWithConfig(mockManager, "test-provider", "test-provider", "test-provider")
	streamingAgent := streaming.NewStreamingAgentWithEvents(pa)

	tests := []struct {
		name           string
		message        string
		expectEvents   []streaming.EventType
		validateEvents func(*testing.T, []streaming.EventType)
	}{
		{
			name:    "Complex query with all streaming events",
			message: "Explain how to build a RESTful API with authentication and authorization mechanisms in Go, including best practices",
			expectEvents: []streaming.EventType{
				streaming.IntentClassified,
				streaming.ProviderSelected,
				streaming.MemoryRetrieved,
				streaming.Thinking,
				streaming.ResponseChunk,
				streaming.Complete,
			},
			validateEvents: func(t *testing.T, eventTypes []streaming.EventType) {
				assert.GreaterOrEqual(t, len(eventTypes), 5, "Should have at least 5 events")

				foundIntentClassified := false
				foundProviderSelected := false
				foundThinking := false
				foundResponseChunk := false
				foundComplete := false

				for _, et := range eventTypes {
					switch et {
					case streaming.IntentClassified:
						foundIntentClassified = true
					case streaming.ProviderSelected:
						foundProviderSelected = true
					case streaming.Thinking:
						foundThinking = true
					case streaming.ResponseChunk:
						foundResponseChunk = true
					case streaming.Complete:
						foundComplete = true
					}
				}

				assert.True(t, foundIntentClassified, "Should have intent_classified event")
				assert.True(t, foundProviderSelected, "Should have provider_selected event")
				assert.True(t, foundThinking, "Should have thinking event")
				assert.True(t, foundResponseChunk, "Should have chunk event")
				assert.True(t, foundComplete, "Should have complete event")
			},
		},
		{
			name:    "Build query triggers appropriate events",
			message: "Build a comprehensive search function for finding information about Go programming language features and best practices",
			expectEvents: []streaming.EventType{
				streaming.IntentClassified,
				streaming.ProviderSelected,
				streaming.Thinking,
				streaming.ResponseChunk,
				streaming.Complete,
			},
			validateEvents: func(t *testing.T, eventTypes []streaming.EventType) {
				assert.GreaterOrEqual(t, len(eventTypes), 4, "Should have at least 4 events")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			queryReq := agent.QueryRequest{
				Query:        tt.message,
				ProviderName: "test-provider",
				ModelID:      "test-model",
				UserID:       "test-user",
				UserTier:     "authenticated",
				UseMemory:    false,
				Temperature:  agent.DefaultTemperature,
				MaxTokens:    agent.DefaultMaxTokens,
			}

			eventChan, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
			assert.NoError(t, err)
			assert.NotNil(t, eventChan)

			var eventTypes []streaming.EventType
			for event := range eventChan {
				eventTypes = append(eventTypes, event.Type)
			}

			if tt.validateEvents != nil {
				tt.validateEvents(t, eventTypes)
			}
		})
	}
}

func TestStreamingWithToolCalls(t *testing.T) {
	mockProvider := &TestMockProvider{
		name:        "test-provider",
		models:      []provider.ModelInfo{{ID: "test-model", Name: "Test Model", ContextSize: 8192, Cost: 0.0}},
		streamDelay: 10 * time.Millisecond,
		withTools:   true,
	}

	mockManager := NewMockPluginManager()
	mockManager.AddProvider("test-provider", mockProvider)

	pa := agent.NewPrimaryAgentWithConfig(mockManager, "test-provider", "test-provider", "test-provider")
	streamingAgent := streaming.NewStreamingAgentWithEvents(pa)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queryReq := agent.QueryRequest{
		Query:        "calculate 2 + 2",
		ProviderName: "test-provider",
		ModelID:      "test-model",
		UserID:       "test-user",
		UserTier:     "authenticated",
		UseMemory:    false,
		Temperature:  agent.DefaultTemperature,
		MaxTokens:    agent.DefaultMaxTokens,
	}

	eventChan, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
	assert.NoError(t, err)
	assert.NotNil(t, eventChan)

	var eventTypes []streaming.EventType
	for event := range eventChan {
		eventTypes = append(eventTypes, event.Type)
	}

	// Should contain tool invocation events when tools are enabled
	foundToolInvoked := false
	foundToolCompleted := false
	for _, et := range eventTypes {
		if et == streaming.ToolInvoked {
			foundToolInvoked = true
		}
		if et == streaming.ToolCompleted {
			foundToolCompleted = true
		}
	}

	assert.True(t, foundToolInvoked, "Should contain tool_invoked event")
	assert.True(t, foundToolCompleted, "Should contain tool_completed event")
}

func TestStreamingEventOrder(t *testing.T) {
	mockProvider := &TestMockProvider{
		name:        "test-provider",
		models:      []provider.ModelInfo{{ID: "test-model", Name: "Test Model", ContextSize: 8192, Cost: 0.0}},
		streamDelay: 5 * time.Millisecond,
	}

	mockManager := NewMockPluginManager()
	mockManager.AddProvider("test-provider", mockProvider)

	pa := agent.NewPrimaryAgentWithConfig(mockManager, "test-provider", "test-provider", "test-provider")
	streamingAgent := streaming.NewStreamingAgentWithEvents(pa)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queryReq := agent.QueryRequest{
		Query:        "Create a detailed system design for a microservices architecture with API gateway and service mesh",
		ProviderName: "test-provider",
		ModelID:      "test-model",
		UserID:       "test-user",
		UserTier:     "authenticated",
		UseMemory:    false,
		Temperature:  agent.DefaultTemperature,
		MaxTokens:    agent.DefaultMaxTokens,
	}

	eventChan, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
	assert.NoError(t, err)

	var eventTypes []streaming.EventType
	for event := range eventChan {
		eventTypes = append(eventTypes, event.Type)
	}

	// Verify event order
	assert.GreaterOrEqual(t, len(eventTypes), 3, "Should have at least 3 events")

	// First event should be intent_classified
	assert.Equal(t, streaming.IntentClassified, eventTypes[0], "First event should be intent_classified")

	// Last event should be complete
	assert.Equal(t, streaming.Complete, eventTypes[len(eventTypes)-1], "Last event should be complete")
}

func TestStreamingErrorHandling(t *testing.T) {
	mockProvider := &TestMockProvider{
		name:        "test-provider",
		models:      []provider.ModelInfo{{ID: "test-model", Name: "Test Model", ContextSize: 8192, Cost: 0.0}},
		shouldError: true,
	}

	mockManager := NewMockPluginManager()
	mockManager.AddProvider("test-provider", mockProvider)

	pa := agent.NewPrimaryAgentWithConfig(mockManager, "test-provider", "test-provider", "test-provider")
	streamingAgent := streaming.NewStreamingAgentWithEvents(pa)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queryReq := agent.QueryRequest{
		Query:        "This will cause an error",
		ProviderName: "test-provider",
		ModelID:      "test-model",
		UserID:       "test-user",
		UserTier:     "authenticated",
		UseMemory:    false,
		Temperature:  agent.DefaultTemperature,
		MaxTokens:    agent.DefaultMaxTokens,
	}

	eventChan, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
	assert.NoError(t, err)

	var eventTypes []streaming.EventType
	for event := range eventChan {
		eventTypes = append(eventTypes, event.Type)
	}

	// Should contain error event
	foundError := false
	for _, et := range eventTypes {
		if et == streaming.Error {
			foundError = true
			break
		}
	}

	assert.True(t, foundError, "Should contain error event")
}

func TestStreamingCancellation(t *testing.T) {
	mockProvider := &TestMockProvider{
		name:        "test-provider",
		models:      []provider.ModelInfo{{ID: "test-model", Name: "Test Model", ContextSize: 8192, Cost: 0.0}},
		streamDelay: 100 * time.Millisecond,
	}

	mockManager := NewMockPluginManager()
	mockManager.AddProvider("test-provider", mockProvider)

	pa := agent.NewPrimaryAgentWithConfig(mockManager, "test-provider", "test-provider", "test-provider")
	streamingAgent := streaming.NewStreamingAgentWithEvents(pa)

	ctx, cancel := context.WithCancel(context.Background())

	queryReq := agent.QueryRequest{
		Query:        "This will be cancelled",
		ProviderName: "test-provider",
		ModelID:      "test-model",
		UserID:       "test-user",
		UserTier:     "authenticated",
		UseMemory:    false,
		Temperature:  agent.DefaultTemperature,
		MaxTokens:    agent.DefaultMaxTokens,
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	eventChan, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
	assert.NoError(t, err)

	// Should complete without panic
	assert.NotPanics(t, func() {
		for range eventChan {
			// Consume events until cancelled
		}
	})
}

func TestGetUserTier(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		expected string
	}{
		{
			name:     "Empty user ID returns guest",
			userID:   "",
			expected: "guest",
		},
		{
			name:     "Non-empty user ID returns authenticated",
			userID:   "user123",
			expected: "authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUserTier(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMockProvider for testing with streaming support
type TestMockProvider struct {
	name        string
	models      []provider.ModelInfo
	streamDelay time.Duration
	shouldError bool
	withTools   bool
}

func (m *TestMockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:    m.name,
		Version: "1.0.0",
	}, nil
}

func (m *TestMockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return m.models, nil
}

func (m *TestMockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	response := "This is a test response"
	var toolCalls []provider.ToolCall

	if m.withTools {
		toolCalls = []provider.ToolCall{
			{
				ID:   "call_1",
				Name: "calculate",
				Arguments: map[string]interface{}{
					"expression": "2 + 2",
				},
			},
		}
	}

	return &provider.CompletionResponse{
		ID:      "test-id",
		Content: response,
		Model:   req.Model,
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		ToolCalls: toolCalls,
	}, nil
}

func (m *TestMockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	if m.shouldError {
		return nil, assert.AnError
	}

	ch := make(chan *provider.CompletionChunk)

	go func() {
		defer close(ch)

		response := "This is a streaming test response"
		words := strings.Split(response, " ")

		for _, word := range words {
			select {
			case <-ctx.Done():
				return
			case ch <- &provider.CompletionChunk{Delta: word + " ", Done: false}:
				time.Sleep(m.streamDelay)
			}
		}

		ch <- &provider.CompletionChunk{Done: true}
	}()

	return ch, nil
}

func (m *TestMockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: "4",
		Error:  "",
	}, nil
}

func (m *TestMockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	result := make([]float32, 768)
	for i := range result {
		result[i] = 0.1
	}
	return result, nil
}

func (m *TestMockProvider) Shutdown() error {
	return nil
}
