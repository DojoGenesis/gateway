package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockModelProvider struct {
	mock.Mock
}

func (m *MockModelProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.ProviderInfo), args.Error(1)
}

func (m *MockModelProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]provider.ModelInfo), args.Error(1)
}

func (m *MockModelProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.CompletionResponse), args.Error(1)
}

func (m *MockModelProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *provider.CompletionChunk), args.Error(1)
}

func (m *MockModelProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.ToolCallResponse), args.Error(1)
}

func (m *MockModelProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

type MockPluginManager struct {
	providers map[string]provider.ModelProvider
}

func NewMockPluginManager() *MockPluginManager {
	return &MockPluginManager{
		providers: make(map[string]provider.ModelProvider),
	}
}

func (m *MockPluginManager) AddProvider(name string, provider provider.ModelProvider) {
	m.providers[name] = provider
}

func (m *MockPluginManager) GetProviders() map[string]provider.ModelProvider {
	return m.providers
}

func (m *MockPluginManager) GetProvider(name string) (provider.ModelProvider, error) {
	if provider, exists := m.providers[name]; exists {
		return provider, nil
	}
	return nil, errors.New("provider not found")
}

// CompressHistory satisfies memory.CompressionServiceInterface for garden manager tests.
func (m *MockPluginManager) CompressHistory(_ context.Context, sessionID string, memories []memory.Memory) (*memory.CompressedHistory, error) {
	return &memory.CompressedHistory{
		SessionID:         sessionID,
		CompressedContent: "compressed",
		CompressionRatio:  0.5,
	}, nil
}

// ExtractSeeds satisfies memory.CompressionServiceInterface for garden manager tests.
func (m *MockPluginManager) ExtractSeeds(_ context.Context, _ []memory.Memory) ([]*memory.MemorySeed, error) {
	return []*memory.MemorySeed{}, nil
}

func TestHandleListModels_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	mockProvider1 := new(MockModelProvider)
	mockProvider1.On("ListModels", mock.Anything).Return([]provider.ModelInfo{
		{
			ID:          "llama3.2",
			Name:        "Llama 3.2",
			ContextSize: 8192,
			Cost:        0,
		},
	}, nil)

	mockProvider2 := new(MockModelProvider)
	mockProvider2.On("ListModels", mock.Anything).Return([]provider.ModelInfo{
		{
			ID:          "deepseek-chat",
			Name:        "DeepSeek Chat",
			ContextSize: 16384,
			Cost:        0.0001,
		},
	}, nil)

	mockPM.AddProvider("ollama", mockProvider1)
	mockPM.AddProvider("deepseek-api", mockProvider2)

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/models", h.ListModels)

	req, _ := http.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Equal(t, 2, len(models))
	assert.Equal(t, float64(2), response["count"])

	mockProvider1.AssertExpectations(t)
	mockProvider2.AssertExpectations(t)
}

func TestHandleListModels_ProviderError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	mockProvider := new(MockModelProvider)
	mockProvider.On("ListModels", mock.Anything).Return(nil, errors.New("provider unavailable"))

	mockPM.AddProvider("failing-provider", mockProvider)

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/models", h.ListModels)

	req, _ := http.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "failed to list models", response["error"])
	assert.Equal(t, "provider: failing-provider", response["details"])

	mockProvider.AssertExpectations(t)
}

func TestHandleListModels_NoPluginManager(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewModelHandler(nil)

	router := gin.New()
	router.GET("/api/v1/models", h.ListModels)

	req, _ := http.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "plugin manager not initialized", response["error"])
}

func TestHandleListModels_EmptyProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/models", h.ListModels)

	req, _ := http.NewRequest("GET", "/api/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Equal(t, 0, len(models))
	assert.Equal(t, float64(0), response["count"])
}

func TestHandleListProviders_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	mockProvider1 := new(MockModelProvider)
	mockProvider1.On("GetInfo", mock.Anything).Return(&provider.ProviderInfo{
		Name:         "Ollama",
		Version:      "1.0.0",
		Description:  "Local Ollama LLM",
		Capabilities: []string{"completion", "streaming"},
	}, nil)

	mockProvider2 := new(MockModelProvider)
	mockProvider2.On("GetInfo", mock.Anything).Return(&provider.ProviderInfo{
		Name:         "DeepSeek API",
		Version:      "0.0.15",
		Description:  "Cloud-based LLM",
		Capabilities: []string{"completion", "streaming", "tools"},
	}, nil)

	mockPM.AddProvider("ollama", mockProvider1)
	mockPM.AddProvider("deepseek-api", mockProvider2)

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/providers", h.ListProviders)

	req, _ := http.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Equal(t, 2, len(providers))
	assert.Equal(t, float64(2), response["count"])

	for _, p := range providers {
		provider := p.(map[string]interface{})
		assert.Equal(t, "active", provider["status"])
		assert.NotNil(t, provider["info"])
	}

	mockProvider1.AssertExpectations(t)
	mockProvider2.AssertExpectations(t)
}

func TestHandleListProviders_ProviderError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	mockProvider := new(MockModelProvider)
	mockProvider.On("GetInfo", mock.Anything).Return(nil, errors.New("connection timeout"))

	mockPM.AddProvider("failing-provider", mockProvider)

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/providers", h.ListProviders)

	req, _ := http.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Equal(t, 1, len(providers))

	provider := providers[0].(map[string]interface{})
	assert.Equal(t, "failing-provider", provider["name"])
	assert.Equal(t, "error", provider["status"])
	assert.Equal(t, "connection timeout", provider["error"])

	mockProvider.AssertExpectations(t)
}

func TestHandleListProviders_NoPluginManager(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewModelHandler(nil)

	router := gin.New()
	router.GET("/api/v1/providers", h.ListProviders)

	req, _ := http.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "plugin manager not initialized", response["error"])
}

func TestHandleListProviders_EmptyProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/providers", h.ListProviders)

	req, _ := http.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Equal(t, 0, len(providers))
	assert.Equal(t, float64(0), response["count"])
}

func TestHandleListProviders_MixedStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPM := NewMockPluginManager()

	mockProvider1 := new(MockModelProvider)
	mockProvider1.On("GetInfo", mock.Anything).Return(&provider.ProviderInfo{
		Name:    "Working Provider",
		Version: "1.0.0",
	}, nil)

	mockProvider2 := new(MockModelProvider)
	mockProvider2.On("GetInfo", mock.Anything).Return(nil, errors.New("crashed"))

	mockPM.AddProvider("working", mockProvider1)
	mockPM.AddProvider("crashed", mockProvider2)

	h := NewModelHandler(mockPM)

	router := gin.New()
	router.GET("/api/v1/providers", h.ListProviders)

	req, _ := http.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Equal(t, 2, len(providers))

	statuses := make(map[string]string)
	for _, p := range providers {
		provider := p.(map[string]interface{})
		statuses[provider["name"].(string)] = provider["status"].(string)
	}

	assert.Equal(t, "active", statuses["working"])
	assert.Equal(t, "error", statuses["crashed"])

	mockProvider1.AssertExpectations(t)
	mockProvider2.AssertExpectations(t)
}
