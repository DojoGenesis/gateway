package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/server/database"
	"github.com/gin-gonic/gin"
)

type mockDatabaseAdapter struct {
	users         map[string]*database.User
	apiKeys       map[string]map[string]*database.APIKey
	conversations map[string]*database.Conversation
	settings      map[string]*database.Settings
}

func newMockDatabaseAdapter() *mockDatabaseAdapter {
	return &mockDatabaseAdapter{
		users:         make(map[string]*database.User),
		apiKeys:       make(map[string]map[string]*database.APIKey),
		conversations: make(map[string]*database.Conversation),
		settings:      make(map[string]*database.Settings),
	}
}

func (m *mockDatabaseAdapter) GetUser(ctx context.Context, userID string) (*database.User, error) {
	if user, ok := m.users[userID]; ok {
		return user, nil
	}
	return nil, database.ErrUserNotFound
}

func (m *mockDatabaseAdapter) CreateUser(ctx context.Context, user *database.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockDatabaseAdapter) UpdateUser(ctx context.Context, user *database.User) error {
	if _, ok := m.users[user.ID]; !ok {
		return database.ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockDatabaseAdapter) StoreAPIKey(ctx context.Context, key *database.APIKey) error {
	if _, ok := m.apiKeys[key.UserID]; !ok {
		m.apiKeys[key.UserID] = make(map[string]*database.APIKey)
	}
	m.apiKeys[key.UserID][key.Provider] = key
	return nil
}

func (m *mockDatabaseAdapter) GetAPIKey(ctx context.Context, userID, provider string) (*database.APIKey, error) {
	if userKeys, ok := m.apiKeys[userID]; ok {
		if key, ok := userKeys[provider]; ok {
			return key, nil
		}
	}
	return nil, database.ErrAPIKeyNotFound
}

func (m *mockDatabaseAdapter) ListAPIKeys(ctx context.Context, userID string) ([]*database.APIKey, error) {
	if userKeys, ok := m.apiKeys[userID]; ok {
		keys := make([]*database.APIKey, 0, len(userKeys))
		for _, key := range userKeys {
			keys = append(keys, key)
		}
		return keys, nil
	}
	return []*database.APIKey{}, nil
}

func (m *mockDatabaseAdapter) DeleteAPIKey(ctx context.Context, userID, provider string) error {
	if userKeys, ok := m.apiKeys[userID]; ok {
		if _, ok := userKeys[provider]; ok {
			delete(userKeys, provider)
			return nil
		}
	}
	return database.ErrAPIKeyNotFound
}

func (m *mockDatabaseAdapter) UpdateAPIKeyLastUsed(ctx context.Context, userID, provider string) error {
	if userKeys, ok := m.apiKeys[userID]; ok {
		if key, ok := userKeys[provider]; ok {
			now := time.Now()
			key.LastUsedAt = &now
			return nil
		}
	}
	return database.ErrAPIKeyNotFound
}

func (m *mockDatabaseAdapter) CreateConversation(ctx context.Context, conv *database.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockDatabaseAdapter) GetConversation(ctx context.Context, id string) (*database.Conversation, error) {
	if conv, ok := m.conversations[id]; ok {
		return conv, nil
	}
	return nil, database.ErrConversationNotFound
}

func (m *mockDatabaseAdapter) ListConversations(ctx context.Context, userID string) ([]*database.Conversation, error) {
	convs := make([]*database.Conversation, 0)
	for _, conv := range m.conversations {
		if conv.UserID == userID {
			convs = append(convs, conv)
		}
	}
	return convs, nil
}

func (m *mockDatabaseAdapter) UpdateConversation(ctx context.Context, conv *database.Conversation) error {
	if _, ok := m.conversations[conv.ID]; !ok {
		return database.ErrConversationNotFound
	}
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockDatabaseAdapter) DeleteConversation(ctx context.Context, id string) error {
	if _, ok := m.conversations[id]; !ok {
		return database.ErrConversationNotFound
	}
	delete(m.conversations, id)
	return nil
}

func (m *mockDatabaseAdapter) GetSettings(ctx context.Context, userID string) (*database.Settings, error) {
	if settings, ok := m.settings[userID]; ok {
		return settings, nil
	}
	return nil, database.ErrSettingsNotFound
}

func (m *mockDatabaseAdapter) CreateSettings(ctx context.Context, settings *database.Settings) error {
	m.settings[settings.UserID] = settings
	return nil
}

func (m *mockDatabaseAdapter) UpdateSettings(ctx context.Context, settings *database.Settings) error {
	if _, ok := m.settings[settings.UserID]; !ok {
		return database.ErrSettingsNotFound
	}
	m.settings[settings.UserID] = settings
	return nil
}

func (m *mockDatabaseAdapter) Ping(ctx context.Context) error {
	return nil
}

func (m *mockDatabaseAdapter) Close() error {
	return nil
}

type mockSecureStorage struct {
	storage   map[string]map[string]string
	available bool
}

func newMockSecureStorage() *mockSecureStorage {
	return &mockSecureStorage{
		storage:   make(map[string]map[string]string),
		available: true,
	}
}

func (m *mockSecureStorage) Store(ctx context.Context, userID, provider, key string) error {
	if _, ok := m.storage[userID]; !ok {
		m.storage[userID] = make(map[string]string)
	}
	m.storage[userID][provider] = key
	return nil
}

func (m *mockSecureStorage) Retrieve(ctx context.Context, userID, provider string) (string, error) {
	if userKeys, ok := m.storage[userID]; ok {
		if key, ok := userKeys[provider]; ok {
			return key, nil
		}
	}
	return "", fmt.Errorf("key not found")
}

func (m *mockSecureStorage) Delete(ctx context.Context, userID, provider string) error {
	if userKeys, ok := m.storage[userID]; ok {
		delete(userKeys, provider)
	}
	return nil
}

func (m *mockSecureStorage) IsAvailable(ctx context.Context) bool {
	return m.available
}

func (m *mockSecureStorage) GetStorageType() string {
	return "mock"
}

func (m *mockSecureStorage) Close() error {
	return nil
}

func setupAPIKeyTestRouter() (*gin.Engine, *database.DatabaseManager, *mockSecureStorage) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})

	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.GET("/api/v1/keys", h.ListAPIKeys)
	router.GET("/api/v1/keys/:provider", h.GetAPIKey)
	router.PUT("/api/v1/keys/:provider", h.UpdateAPIKey)
	router.DELETE("/api/v1/keys/:provider", h.DeleteAPIKey)

	return router, dbMgr, mockStorage
}

func TestHandleCreateAPIKey(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	tests := []struct {
		name           string
		requestBody    CreateAPIKeyRequest
		expectedStatus int
		wantSuccess    bool
		wantError      string
	}{
		{
			name: "create valid API key",
			requestBody: CreateAPIKeyRequest{
				Provider: "openai",
				APIKey:   "sk-1234567890abcdef",
			},
			expectedStatus: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name: "create API key with name",
			requestBody: CreateAPIKeyRequest{
				Provider: "anthropic",
				APIKey:   "sk-ant-1234567890",
				KeyName:  stringPtr("Production Key"),
			},
			expectedStatus: http.StatusCreated,
			wantSuccess:    true,
		},
		{
			name: "missing provider",
			requestBody: CreateAPIKeyRequest{
				APIKey: "sk-1234567890",
			},
			expectedStatus: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "missing API key",
			requestBody: CreateAPIKeyRequest{
				Provider: "openai",
			},
			expectedStatus: http.StatusBadRequest,
			wantSuccess:    false,
		},
		{
			name: "empty provider",
			requestBody: CreateAPIKeyRequest{
				Provider: "   ",
				APIKey:   "sk-1234567890",
			},
			expectedStatus: http.StatusBadRequest,
			wantSuccess:    false,
			wantError:      "Provider is required",
		},
		{
			name: "empty API key",
			requestBody: CreateAPIKeyRequest{
				Provider: "openai",
				APIKey:   "   ",
			},
			expectedStatus: http.StatusBadRequest,
			wantSuccess:    false,
			wantError:      "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.Code, resp.Body.String())
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("Expected success: %v, got %v", tt.wantSuccess, success)
			}

			if tt.wantError != "" {
				errorMsg, _ := result["error"].(string)
				if errorMsg != tt.wantError {
					t.Errorf("Expected error: %s, got %s", tt.wantError, errorMsg)
				}
			}

			if tt.wantSuccess {
				data, ok := result["data"].(map[string]interface{})
				if !ok {
					t.Errorf("Expected data object in response")
					return
				}

				provider, _ := data["provider"].(string)
				if provider != tt.requestBody.Provider {
					t.Errorf("Expected provider: %s, got %s", tt.requestBody.Provider, provider)
				}

				keyPreview, ok := data["key_preview"].(string)
				if !ok {
					t.Errorf("Expected key_preview in response")
				} else if keyPreview == tt.requestBody.APIKey {
					t.Errorf("Key preview should not be the full API key")
				}
			}
		})
	}
}

func TestHandleCreateAPIKeyDuplicate(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	reqBody := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-1234567890abcdef",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("First request should succeed, got status %d", resp.Code)
	}

	body, _ = json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Errorf("Expected status %d for duplicate, got %d", http.StatusConflict, resp.Code)
	}
}

func TestHandleListAPIKeys(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-openai-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	createReq2 := CreateAPIKeyRequest{
		Provider: "anthropic",
		APIKey:   "sk-ant-key",
	}
	body, _ = json.Marshal(createReq2)
	req, _ = http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	req, _ = http.NewRequest("GET", "/api/v1/keys", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		t.Errorf("Expected success: true, got false")
	}

	count, _ := result["count"].(float64)
	if int(count) != 2 {
		t.Errorf("Expected count: 2, got %v", count)
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data array in response")
	}

	if len(data) != 2 {
		t.Errorf("Expected 2 keys in data array, got %d", len(data))
	}
}

func TestHandleGetAPIKey(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-openai-key-123456",
		KeyName:  stringPtr("Test Key"),
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	req, _ = http.NewRequest("GET", "/api/v1/keys/openai", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		t.Errorf("Expected success: true, got false")
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object in response")
	}

	provider, _ := data["provider"].(string)
	if provider != "openai" {
		t.Errorf("Expected provider: openai, got %s", provider)
	}

	keyName, _ := data["key_name"].(string)
	if keyName != "Test Key" {
		t.Errorf("Expected key_name: Test Key, got %s", keyName)
	}
}

func TestHandleGetAPIKeyNotFound(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/keys/nonexistent", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestHandleUpdateAPIKey(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-openai-key",
		KeyName:  stringPtr("Old Name"),
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	updateReq := UpdateAPIKeyRequest{
		KeyName: stringPtr("New Name"),
	}
	body, _ = json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", "/api/v1/keys/openai", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		t.Errorf("Expected success: true, got false")
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object in response")
	}

	keyName, _ := data["key_name"].(string)
	if keyName != "New Name" {
		t.Errorf("Expected key_name: New Name, got %s", keyName)
	}
}

func TestHandleUpdateAPIKeyNotFound(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	updateReq := UpdateAPIKeyRequest{
		KeyName: stringPtr("New Name"),
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/v1/keys/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestHandleDeleteAPIKey(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-openai-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	req, _ = http.NewRequest("DELETE", "/api/v1/keys/openai", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		t.Errorf("Expected success: true, got false")
	}

	req, _ = http.NewRequest("GET", "/api/v1/keys/openai", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected key to be deleted, got status %d", resp.Code)
	}
}

func TestHandleDeleteAPIKeyNotFound(t *testing.T) {
	router, _, _ := setupAPIKeyTestRouter()

	req, _ := http.NewRequest("DELETE", "/api/v1/keys/nonexistent", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestGetAPIKeyForProvider(t *testing.T) {
	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"
	apiKey := "sk-test-key-123456"

	key := &database.APIKey{
		ID:        "key-1",
		UserID:    userID,
		Provider:  provider,
		KeyHash:   hashAPIKey(apiKey),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_ = mockAdapter.StoreAPIKey(ctx, key)
	_ = mockStorage.Store(ctx, userID, provider, apiKey)

	retrievedKey, err := h.GetAPIKeyForProvider(ctx, userID, database.UserTypeGuest, provider)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrievedKey != apiKey {
		t.Errorf("Expected key: %s, got %s", apiKey, retrievedKey)
	}
}

func TestGetAPIKeyForProviderNotFound(t *testing.T) {
	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	ctx := context.Background()

	_, err := h.GetAPIKeyForProvider(ctx, "test-user", database.UserTypeGuest, "nonexistent")
	if err == nil {
		t.Errorf("Expected error for nonexistent provider, got nil")
	}
}

func TestGetAPIKeyForProviderInactive(t *testing.T) {
	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"
	apiKey := "sk-test-key-123456"

	key := &database.APIKey{
		ID:        "key-1",
		UserID:    userID,
		Provider:  provider,
		KeyHash:   hashAPIKey(apiKey),
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_ = mockAdapter.StoreAPIKey(ctx, key)
	_ = mockStorage.Store(ctx, userID, provider, apiKey)

	_, err := h.GetAPIKeyForProvider(ctx, userID, database.UserTypeGuest, provider)
	if err == nil {
		t.Errorf("Expected error for inactive key, got nil")
	}
}

func TestUserIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router1 := gin.New()
	router1.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router1.POST("/api/v1/keys", h.CreateAPIKey)
	router1.GET("/api/v1/keys", h.ListAPIKeys)

	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("user_id", "user-2")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router2.POST("/api/v1/keys", h.CreateAPIKey)
	router2.GET("/api/v1/keys", h.ListAPIKeys)

	createReq1 := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-user1-key",
	}
	body, _ := json.Marshal(createReq1)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router1.ServeHTTP(resp, req)

	createReq2 := CreateAPIKeyRequest{
		Provider: "anthropic",
		APIKey:   "sk-user2-key",
	}
	body, _ = json.Marshal(createReq2)
	req, _ = http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router2.ServeHTTP(resp, req)

	req, _ = http.NewRequest("GET", "/api/v1/keys", nil)
	resp = httptest.NewRecorder()
	router1.ServeHTTP(resp, req)

	var result1 map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result1)
	count1, _ := result1["count"].(float64)
	if int(count1) != 1 {
		t.Errorf("User 1 should have 1 key, got %v", count1)
	}

	req, _ = http.NewRequest("GET", "/api/v1/keys", nil)
	resp = httptest.NewRecorder()
	router2.ServeHTTP(resp, req)

	var result2 map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result2)
	count2, _ := result2["count"].(float64)
	if int(count2) != 1 {
		t.Errorf("User 2 should have 1 key, got %v", count2)
	}
}

func TestGetUserContextError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing user_id", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_type", database.UserTypeGuest)
			userID, userType, err := getUserContext(c)
			if err == nil {
				t.Errorf("Expected error when user_id is missing")
			}
			if userID != "" {
				t.Errorf("Expected empty userID, got %s", userID)
			}
			if userType != "" {
				t.Errorf("Expected empty userType, got %s", userType)
			}
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	})

	t.Run("missing user_type", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_id", "test-user")
			userID, userType, err := getUserContext(c)
			if err == nil {
				t.Errorf("Expected error when user_type is missing")
			}
			if userID != "" {
				t.Errorf("Expected empty userID, got %s", userID)
			}
			if userType != "" {
				t.Errorf("Expected empty userType, got %s", userType)
			}
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	})

	t.Run("invalid user_id type", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_id", 123)
			c.Set("user_type", database.UserTypeGuest)
			userID, userType, err := getUserContext(c)
			if err == nil {
				t.Errorf("Expected error when user_id is not a string")
			}
			if userID != "" {
				t.Errorf("Expected empty userID, got %s", userID)
			}
			if userType != "" {
				t.Errorf("Expected empty userType, got %s", userType)
			}
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	})

	t.Run("invalid user_type type", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_id", "test-user")
			c.Set("user_type", "invalid")
			userID, userType, err := getUserContext(c)
			if err == nil {
				t.Errorf("Expected error when user_type is not UserType")
			}
			if userID != "" {
				t.Errorf("Expected empty userID, got %s", userID)
			}
			if userType != "" {
				t.Errorf("Expected empty userType, got %s", userType)
			}
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	})
}

func TestGetKeyPreview(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "normal key",
			apiKey:   "sk-1234567890abcdef",
			expected: "sk-1...cdef",
		},
		{
			name:     "short key",
			apiKey:   "short",
			expected: "***",
		},
		{
			name:     "exact 8 chars",
			apiKey:   "12345678",
			expected: "***",
		},
		{
			name:     "9 chars",
			apiKey:   "123456789",
			expected: "1234...6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getKeyPreview(tt.apiKey)
			if result != tt.expected {
				t.Errorf("Expected preview: %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHandleListAPIKeysStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.GET("/api/v1/keys", h.ListAPIKeys)

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-test-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	delete(mockStorage.storage["test-user"], "openai")

	req, _ = http.NewRequest("GET", "/api/v1/keys", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d even with storage error, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	data, _ := result["data"].([]interface{})
	firstKey, _ := data[0].(map[string]interface{})
	keyPreview, _ := firstKey["key_preview"].(string)

	if keyPreview != "***" {
		t.Errorf("Expected fallback preview ***, got %s", keyPreview)
	}
}

func TestHandleDeleteAPIKeyStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.DELETE("/api/v1/keys/:provider", h.DeleteAPIKey)

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-test-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	req, _ = http.NewRequest("DELETE", "/api/v1/keys/openai", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}
}

func TestHandlersWithoutInitialization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAPIKeyHandler(nil, nil, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.GET("/api/v1/keys", h.ListAPIKeys)
	router.GET("/api/v1/keys/:provider", h.GetAPIKey)
	router.PUT("/api/v1/keys/:provider", h.UpdateAPIKey)
	router.DELETE("/api/v1/keys/:provider", h.DeleteAPIKey)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"create", "POST", "/api/v1/keys", CreateAPIKeyRequest{Provider: "openai", APIKey: "sk-test"}},
		{"list", "GET", "/api/v1/keys", nil},
		{"get", "GET", "/api/v1/keys/openai", nil},
		{"update", "PUT", "/api/v1/keys/openai", UpdateAPIKeyRequest{}},
		{"delete", "DELETE", "/api/v1/keys/openai", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusInternalServerError {
				t.Errorf("Expected status %d for uninitialized handler, got %d", http.StatusInternalServerError, resp.Code)
			}
		})
	}
}

func TestHandleCreateAPIKeyDatabaseErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router.POST("/api/v1/keys", h.CreateAPIKey)

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-test-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("First request should succeed, got status %d", resp.Code)
	}
}

func TestUnauthorizedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router := gin.New()
	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.GET("/api/v1/keys", h.ListAPIKeys)
	router.GET("/api/v1/keys/:provider", h.GetAPIKey)
	router.PUT("/api/v1/keys/:provider", h.UpdateAPIKey)
	router.DELETE("/api/v1/keys/:provider", h.DeleteAPIKey)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"create", "POST", "/api/v1/keys", CreateAPIKeyRequest{Provider: "openai", APIKey: "sk-test"}},
		{"list", "GET", "/api/v1/keys", nil},
		{"get", "GET", "/api/v1/keys/openai", nil},
		{"update", "PUT", "/api/v1/keys/openai", UpdateAPIKeyRequest{KeyName: stringPtr("test")}},
		{"delete", "DELETE", "/api/v1/keys/openai", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d for unauthorized request, got %d", http.StatusUnauthorized, resp.Code)
			}
		})
	}
}

func TestHandleGetAPIKeyStorageError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAdapter := newMockDatabaseAdapter()
	dbMgr := database.NewDatabaseManager(mockAdapter, nil, false)
	mockStorage := newMockSecureStorage()

	h := NewAPIKeyHandler(dbMgr, mockStorage, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("user_type", database.UserTypeGuest)
		c.Next()
	})
	router.POST("/api/v1/keys", h.CreateAPIKey)
	router.GET("/api/v1/keys/:provider", h.GetAPIKey)

	createReq := CreateAPIKeyRequest{
		Provider: "openai",
		APIKey:   "sk-test-key",
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	delete(mockStorage.storage["test-user"], "openai")

	req, _ = http.NewRequest("GET", "/api/v1/keys/openai", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d even with storage error, got %d", http.StatusOK, resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)
	data, _ := result["data"].(map[string]interface{})
	keyPreview, _ := data["key_preview"].(string)

	if keyPreview != "***" {
		t.Errorf("Expected fallback preview ***, got %s", keyPreview)
	}
}

func TestGetAPIKeyForProviderUninitalized(t *testing.T) {
	h := NewAPIKeyHandler(nil, nil, nil)

	ctx := context.Background()
	_, err := h.GetAPIKeyForProvider(ctx, "test-user", database.UserTypeGuest, "openai")
	if err == nil {
		t.Errorf("Expected error when handlers not initialized")
	}
}

func stringPtr(s string) *string {
	return &s
}
