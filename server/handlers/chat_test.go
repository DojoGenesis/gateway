package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPluginManager struct {
	providers map[string]provider.ModelProvider
	err       error
}

func (m *mockPluginManager) GetProvider(name string) (provider.ModelProvider, error) {
	if m.err != nil {
		return nil, m.err
	}
	if provider, ok := m.providers[name]; ok {
		return provider, nil
	}
	if len(m.providers) > 0 {
		for _, p := range m.providers {
			return p, nil
		}
	}
	return nil, errors.New("no provider found")
}

func (m *mockPluginManager) GetProviders() map[string]provider.ModelProvider {
	return m.providers
}

func (m *mockPluginManager) DiscoverPlugins() error {
	return nil
}

func (m *mockPluginManager) LoadPlugin(name string) error {
	return nil
}

func (m *mockPluginManager) Shutdown() error {
	return nil
}

// Ensure mockPluginManager implements both interfaces
var _ agent.PluginManagerInterface = (*mockPluginManager)(nil)

type mockModelProvider struct {
	response       *provider.CompletionResponse
	streamResponse chan *provider.CompletionChunk
	err            error
}

func (m *mockModelProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "Mock Provider",
		Version:     "1.0.0",
		Description: "Test provider",
	}, nil
}

func (m *mockModelProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:          "mock-model",
			Name:        "Mock Model",
			Provider:    "mock",
			ContextSize: 8192,
			Cost:        0.0,
		},
	}, nil
}

func (m *mockModelProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockModelProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.streamResponse, nil
}

func (m *mockModelProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockModelProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([]float32, 768)
	for i := range result {
		result[i] = 0.1
	}
	return result, nil
}

func setupChatTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHandleChat_ValidationErrors(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "This is a test response",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON",
			body:           `{"message": invalid}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request format",
		},
		{
			name:           "missing message",
			body:           `{"stream": false}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "message is required",
		},
		{
			name:           "empty message",
			body:           `{"message": "", "stream": false}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "message is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestHandleChat_SimpleQuery_NonStreaming(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "This is a test response",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	tests := []struct {
		name            string
		message         string
		expectedContent string
		usesLLM         bool
	}{
		{
			name:            "greeting - hello",
			message:         "hello",
			expectedContent: "Hello! I'm Dojo Genesis",
		},
		{
			name:            "greeting - hi",
			message:         "hi",
			expectedContent: "Hi!",
		},
		{
			name:            "help request",
			message:         "help",
			expectedContent: "I'm Dojo Genesis, an AI coding assistant",
		},
		{
			name:            "capabilities",
			message:         "what can you do",
			expectedContent: "This is a test response",
			usesLLM:         true,
		},
		{
			name:            "farewell",
			message:         "thanks",
			expectedContent: "You're welcome!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := ChatRequest{
				Message:   tt.message,
				Stream:    false,
				SessionID: "test-session",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response ChatResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "complete", response.Type)
			assert.Contains(t, response.Content, tt.expectedContent)
			if tt.usesLLM {
				assert.NotNil(t, response.Usage)
			} else {
				assert.Nil(t, response.Usage)
			}
		})
	}
}

func TestHandleChat_SimpleQuery_Streaming(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "This is a test response",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "hello",
		Stream:    true,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "event:")
	assert.Contains(t, responseBody, "chunk")
	assert.Contains(t, responseBody, "complete")
	assert.Contains(t, responseBody, "Hello! I'm Dojo Genesis")
}

func TestHandleChat_ComplexQuery_NonStreaming(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Here's a React todo app implementation...",
			Usage: provider.Usage{
				InputTokens:  100,
				OutputTokens: 500,
				TotalTokens:  600,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "Build a React todo app",
		Stream:    false,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ChatResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "complete", response.Type)
	assert.Equal(t, "Here's a React todo app implementation...", response.Content)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 100, response.Usage.InputTokens)
	assert.Equal(t, 500, response.Usage.OutputTokens)
}

func TestHandleChat_ComplexQuery_Streaming(t *testing.T) {
	t.Skip("Streaming tests require a real HTTP connection; httptest.ResponseRecorder doesn't support CloseNotifier")

	// Test is skipped - streaming tests require real HTTP connection
	// This is a placeholder for when streaming tests are enabled
}

func TestHandleChat_ErrorHandling(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		err: errors.New("provider error"),
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "Build a React app",
		Stream:    false,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "failed to generate response")
}

func TestHandleChat_WithModel(t *testing.T) {
	ic := agent.NewIntentClassifier()

	customProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Response from specific model",
			Usage: provider.Usage{
				InputTokens:  50,
				OutputTokens: 100,
				TotalTokens:  150,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": customProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "Build something complex",
		Model:     "mock-model",
		Stream:    false,
		SessionID: "test-session",
		UserID:    "user-123",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var response ChatResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "complete", response.Type)
	assert.Equal(t, "Response from specific model", response.Content)
	assert.NotNil(t, response.Usage)
}

func TestGetTemplateResponse(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectedKeyword string
	}{
		{
			name:            "hello with punctuation",
			message:         "Hello!",
			expectedKeyword: "Hello! I'm Dojo Genesis",
		},
		{
			name:            "mixed case",
			message:         "HeLLo",
			expectedKeyword: "Hello! I'm Dojo Genesis",
		},
		{
			name:            "with spaces",
			message:         "  hello  ",
			expectedKeyword: "Hello! I'm Dojo Genesis",
		},
		{
			name:            "help request",
			message:         "Help?",
			expectedKeyword: "I'm Dojo Genesis, an AI coding assistant",
		},
		{
			name:            "unknown query",
			message:         "some random text",
			expectedKeyword: "I'm here to help!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := getTemplateResponse(tt.message)
			assert.Contains(t, response, tt.expectedKeyword)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "HeLLo",
			expected: "hello",
		},
		{
			name:     "with punctuation",
			input:    "Hello!",
			expected: "hello",
		},
		{
			name:     "multiple punctuation",
			input:    "!!!Hello???",
			expected: "hello",
		},
		{
			name:     "with spaces",
			input:    "  hello  ",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleChat_NotInitialized(t *testing.T) {
	h := NewChatHandler(nil, nil, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "hello",
		Stream:    false,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "chat handler not initialized", response["error"])
}

type mockCancellableProvider struct {
	mockModelProvider
}

func (m *mockCancellableProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, errors.New("should have been cancelled")
	}
}

func TestHandleChat_ContextCancellation(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockCancellableProvider{}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "Build a complex application",
		Stream:    false,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cancel()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "failed to generate response")
}

func TestHandleChat_RoutingDecisionHeaders(t *testing.T) {
	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Here's the answer: 4",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	tests := []struct {
		name             string
		message          string
		expectedCategory string
		expectedHandler  string
	}{
		{
			name:             "greeting routes to template",
			message:          "hello",
			expectedCategory: "Greeting",
			expectedHandler:  "template",
		},
		{
			name:             "calculation routes to llm-fast",
			message:          "what is 2+2?",
			expectedCategory: "Calculation",
			expectedHandler:  "llm-fast",
		},
		{
			name:             "code generation routes to llm handler",
			message:          "write a function",
			expectedCategory: "CodeGeneration",
			expectedHandler:  "llm-fast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupChatTestRouter()
			router.POST("/chat", h.Chat)

			reqBody := ChatRequest{
				Message:   tt.message,
				Stream:    false,
				SessionID: "test-session",
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			assert.Equal(t, tt.expectedCategory, w.Header().Get("X-Intent-Category"), "Category header should match")
			assert.Equal(t, tt.expectedHandler, w.Header().Get("X-Intent-Handler"), "Handler header should match")

			confidence := w.Header().Get("X-Intent-Confidence")
			assert.NotEmpty(t, confidence, "Confidence header should be set")
		})
	}
}
