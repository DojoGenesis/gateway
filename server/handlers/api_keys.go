package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/database"
	"github.com/DojoGenesis/gateway/server/secure_storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler handles API key-related HTTP requests.
type APIKeyHandler struct {
	db      *database.DatabaseManager
	storage secure_storage.SecureStorage
	plugins *provider.PluginManager
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(manager *database.DatabaseManager, storage secure_storage.SecureStorage, pm *provider.PluginManager) *APIKeyHandler {
	return &APIKeyHandler{
		db:      manager,
		storage: storage,
		plugins: pm,
	}
}

// mapAPIKeyProviderToPluginName maps the provider names used in the API key UI
// (openai, anthropic, deepseek, etc.) to the plugin names used in config.yaml.
func mapAPIKeyProviderToPluginName(apiKeyProvider string) string {
	switch strings.ToLower(apiKeyProvider) {
	case "deepseek":
		return "deepseek-api"
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "google":
		return "google"
	case "groq":
		return "groq"
	case "mistral":
		return "mistral"
	default:
		return ""
	}
}

type CreateAPIKeyRequest struct {
	Provider string  `json:"provider" binding:"required"`
	APIKey   string  `json:"api_key" binding:"required"`
	KeyName  *string `json:"key_name,omitempty"`
}

type UpdateAPIKeyRequest struct {
	KeyName *string `json:"key_name,omitempty"`
}

type APIKeyResponse struct {
	ID         string     `json:"id"`
	Provider   string     `json:"provider"`
	KeyName    *string    `json:"key_name,omitempty"`
	KeyPreview string     `json:"key_preview"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	IsActive   bool       `json:"is_active"`
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

func getKeyPreview(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

func apiKeyToResponse(key *database.APIKey, apiKeyPlain string) APIKeyResponse {
	return APIKeyResponse{
		ID:         key.ID,
		Provider:   key.Provider,
		KeyName:    key.KeyName,
		KeyPreview: getKeyPreview(apiKeyPlain),
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		LastUsedAt: key.LastUsedAt,
		IsActive:   key.IsActive,
	}
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	if h.db == nil || h.storage == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Provider) == "" {
		respondBadRequestWithSuccess(c, "Provider is required")
		return
	}

	if strings.TrimSpace(req.APIKey) == "" {
		respondBadRequestWithSuccess(c, "API key is required")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	// Ensure user exists in local_users (required for foreign key constraint)
	_, err = h.db.GetUser(ctx, userID, userType)
	if err != nil {
		now := time.Now()
		newUser := &database.User{
			ID:              userID,
			UserType:        userType,
			CreatedAt:       now,
			LastAccessedAt:  now,
			MigrationStatus: database.MigrationStatusNone,
		}
		if createErr := h.db.CreateUser(ctx, newUser); createErr != nil {
			slog.Error("failed to create user record", "error", createErr)
			respondInternalErrorWithSuccess(c, "Failed to create user record")
			return
		}
	}

	existingKey, err := h.db.GetAPIKey(ctx, userID, req.Provider, userType)
	if err == nil && existingKey != nil {
		respondErrorWithSuccess(c, http.StatusConflict, fmt.Sprintf("API key for provider '%s' already exists", req.Provider))
		return
	}

	if err := h.storage.Store(ctx, userID, req.Provider, req.APIKey); err != nil {
		slog.Error("failed to store API key", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to store API key")
		return
	}

	keyHash := hashAPIKey(req.APIKey)
	apiKey := &database.APIKey{
		ID:           uuid.New().String(),
		UserID:       userID,
		Provider:     req.Provider,
		KeyName:      req.KeyName,
		KeyHash:      keyHash,
		EncryptedKey: []byte(keyHash), // Placeholder; actual key stored in secure storage
		StorageType:  h.storage.GetStorageType(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	if err := h.db.StoreAPIKey(ctx, apiKey, userType); err != nil {
		h.storage.Delete(ctx, userID, req.Provider)
		slog.Error("failed to store API key metadata", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to store API key metadata")
		return
	}

	// Notify plugin system about new API key — restart the plugin with the key
	if h.plugins != nil {
		pluginName := mapAPIKeyProviderToPluginName(req.Provider)
		if pluginName != "" {
			apiKeyValue := req.APIKey
			go func() {
				configUpdate := map[string]interface{}{
					"api_key": apiKeyValue,
				}
				if err := h.plugins.UpdatePluginConfig(pluginName, configUpdate); err != nil {
					slog.Error("failed to update plugin with new API key", "plugin", pluginName, "error", err)
				} else {
					slog.Info("plugin restarted with new API key", "plugin", pluginName, "provider", req.Provider)
				}
			}()
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    apiKeyToResponse(apiKey, req.APIKey),
	})
}

func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	if h.db == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	keys, err := h.db.ListAPIKeys(ctx, userID, userType)
	if err != nil {
		slog.Error("failed to list API keys", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to list API keys")
		return
	}

	responses := make([]APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		apiKeyPlain, err := h.storage.Retrieve(ctx, userID, key.Provider)
		preview := "***"
		if err == nil {
			preview = getKeyPreview(apiKeyPlain)
		}

		responses = append(responses, APIKeyResponse{
			ID:         key.ID,
			Provider:   key.Provider,
			KeyName:    key.KeyName,
			KeyPreview: preview,
			CreatedAt:  key.CreatedAt,
			UpdatedAt:  key.UpdatedAt,
			LastUsedAt: key.LastUsedAt,
			IsActive:   key.IsActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responses,
		"count":   len(responses),
	})
}

func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	if h.db == nil || h.storage == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	provider := c.Param("provider")
	if strings.TrimSpace(provider) == "" {
		respondBadRequestWithSuccess(c, "Provider is required")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	key, err := h.db.GetAPIKey(ctx, userID, provider, userType)
	if err != nil {
		if err == database.ErrAPIKeyNotFound {
			respondNotFoundWithSuccess(c, fmt.Sprintf("API key for provider '%s' not found", provider))
			return
		}
		slog.Error("failed to get API key", "provider", provider, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to retrieve API key")
		return
	}

	apiKeyPlain, err := h.storage.Retrieve(ctx, userID, provider)
	preview := "***"
	if err == nil {
		preview = getKeyPreview(apiKeyPlain)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": APIKeyResponse{
			ID:         key.ID,
			Provider:   key.Provider,
			KeyName:    key.KeyName,
			KeyPreview: preview,
			CreatedAt:  key.CreatedAt,
			UpdatedAt:  key.UpdatedAt,
			LastUsedAt: key.LastUsedAt,
			IsActive:   key.IsActive,
		},
	})
}

func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	if h.db == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	provider := c.Param("provider")
	if strings.TrimSpace(provider) == "" {
		respondBadRequestWithSuccess(c, "Provider is required")
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	key, err := h.db.GetAPIKey(ctx, userID, provider, userType)
	if err != nil {
		if err == database.ErrAPIKeyNotFound {
			respondNotFoundWithSuccess(c, fmt.Sprintf("API key for provider '%s' not found", provider))
			return
		}
		slog.Error("failed to get API key", "provider", provider, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to retrieve API key")
		return
	}

	key.KeyName = req.KeyName
	key.UpdatedAt = time.Now()

	if err := h.db.StoreAPIKey(ctx, key, userType); err != nil {
		slog.Error("failed to update API key", "provider", provider, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to update API key")
		return
	}

	apiKeyPlain, err := h.storage.Retrieve(ctx, userID, provider)
	preview := "***"
	if err == nil {
		preview = getKeyPreview(apiKeyPlain)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": APIKeyResponse{
			ID:         key.ID,
			Provider:   key.Provider,
			KeyName:    key.KeyName,
			KeyPreview: preview,
			CreatedAt:  key.CreatedAt,
			UpdatedAt:  key.UpdatedAt,
			LastUsedAt: key.LastUsedAt,
			IsActive:   key.IsActive,
		},
	})
}

func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	if h.db == nil || h.storage == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	provider := c.Param("provider")
	if strings.TrimSpace(provider) == "" {
		respondBadRequestWithSuccess(c, "Provider is required")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	_, err = h.db.GetAPIKey(ctx, userID, provider, userType)
	if err != nil {
		if err == database.ErrAPIKeyNotFound {
			respondNotFoundWithSuccess(c, fmt.Sprintf("API key for provider '%s' not found", provider))
			return
		}
		slog.Error("failed to get API key", "provider", provider, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to retrieve API key")
		return
	}

	if err := h.db.DeleteAPIKey(ctx, userID, provider, userType); err != nil {
		slog.Error("failed to delete API key metadata", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to delete API key")
		return
	}

	if err := h.storage.Delete(ctx, userID, provider); err != nil {
		slog.Error("failed to delete secure storage", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to delete API key")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("API key for provider '%s' deleted successfully", provider),
	})
}

// GetAPIKeyForProvider retrieves an API key for a specific provider.
func (h *APIKeyHandler) GetAPIKeyForProvider(ctx context.Context, userID string, userType database.UserType, provider string) (string, error) {
	if h.db == nil || h.storage == nil {
		return "", fmt.Errorf("API key management not initialized")
	}

	key, err := h.db.GetAPIKey(ctx, userID, provider, userType)
	if err != nil {
		return "", err
	}

	if !key.IsActive {
		return "", fmt.Errorf("API key for provider '%s' is inactive", provider)
	}

	apiKey, err := h.storage.Retrieve(ctx, userID, provider)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve API key: %w", err)
	}

	_ = h.db.UpdateAPIKeyLastUsed(ctx, userID, provider, userType)

	return apiKey, nil
}

// TestAPIKeyResult holds the result of testing an API key against a provider.
type TestAPIKeyResult struct {
	Provider       string `json:"provider"`
	Status         string `json:"status"` // "valid", "invalid", or "error"
	Message        string `json:"message"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	TestedAt       string `json:"tested_at"`
}

// providerTestConfig defines how to test a specific provider's API key.
type providerTestConfig struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// getProviderTestConfig returns the test configuration for a given provider.
func getProviderTestConfig(provider string, apiKey string) (*providerTestConfig, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return &providerTestConfig{
			URL:     "https://api.openai.com/v1/models",
			Method:  http.MethodGet,
			Headers: map[string]string{"Authorization": "Bearer " + apiKey},
		}, nil
	case "anthropic":
		return &providerTestConfig{
			URL:    "https://api.anthropic.com/v1/messages",
			Method: http.MethodPost,
			Headers: map[string]string{
				"x-api-key":         apiKey,
				"anthropic-version": "2023-06-01",
				"Content-Type":      "application/json",
			},
			Body: `{"model":"claude-sonnet-4-20250514","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`,
		}, nil
	case "deepseek":
		return &providerTestConfig{
			URL:     "https://api.deepseek.com/v1/models",
			Method:  http.MethodGet,
			Headers: map[string]string{"Authorization": "Bearer " + apiKey},
		}, nil
	case "google":
		return &providerTestConfig{
			URL:    fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models?key=%s", apiKey),
			Method: http.MethodGet,
		}, nil
	case "groq":
		return &providerTestConfig{
			URL:     "https://api.groq.com/openai/v1/models",
			Method:  http.MethodGet,
			Headers: map[string]string{"Authorization": "Bearer " + apiKey},
		}, nil
	case "mistral":
		return &providerTestConfig{
			URL:     "https://api.mistral.ai/v1/models",
			Method:  http.MethodGet,
			Headers: map[string]string{"Authorization": "Bearer " + apiKey},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// TestAPIKey tests an API key by making a lightweight request to the provider.
func (h *APIKeyHandler) TestAPIKey(c *gin.Context) {
	if h.db == nil || h.storage == nil {
		respondInternalErrorWithSuccess(c, "API key management not initialized")
		return
	}

	provider := c.Param("provider")
	if strings.TrimSpace(provider) == "" {
		respondBadRequestWithSuccess(c, "Provider is required")
		return
	}

	userID, userType, err := getUserContext(c)
	if err != nil {
		respondErrorWithSuccess(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx := c.Request.Context()

	// Verify the key exists
	_, err = h.db.GetAPIKey(ctx, userID, provider, userType)
	if err != nil {
		if err == database.ErrAPIKeyNotFound {
			respondNotFoundWithSuccess(c, fmt.Sprintf("API key for provider '%s' not found", provider))
			return
		}
		slog.Error("failed to get API key", "provider", provider, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to retrieve API key")
		return
	}

	// Retrieve the actual key
	apiKey, err := h.storage.Retrieve(ctx, userID, provider)
	if err != nil {
		slog.Error("failed to retrieve API key from secure storage", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to retrieve API key")
		return
	}

	// Get test config for this provider
	testConfig, err := getProviderTestConfig(provider, apiKey)
	if err != nil {
		respondBadRequestWithSuccess(c, "Unsupported provider")
		return
	}

	// Make the test request
	httpClient := &http.Client{Timeout: 10 * time.Second}

	var reqBody io.Reader
	if testConfig.Body != "" {
		reqBody = bytes.NewBufferString(testConfig.Body)
	}

	req, err := http.NewRequestWithContext(ctx, testConfig.Method, testConfig.URL, reqBody)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": TestAPIKeyResult{
				Provider:       provider,
				Status:         "error",
				Message:        fmt.Sprintf("Failed to create request: %v", err),
				ResponseTimeMs: 0,
				TestedAt:       time.Now().UTC().Format(time.RFC3339),
			},
		})
		return
	}

	for k, v := range testConfig.Headers {
		req.Header.Set(k, v)
	}

	startTime := time.Now()
	resp, err := httpClient.Do(req)
	responseTimeMs := time.Since(startTime).Milliseconds()

	// Update last_used_at regardless of result
	_ = h.db.UpdateAPIKeyLastUsed(ctx, userID, provider, userType)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": TestAPIKeyResult{
				Provider:       provider,
				Status:         "error",
				Message:        fmt.Sprintf("Connection failed: %v", err),
				ResponseTimeMs: responseTimeMs,
				TestedAt:       time.Now().UTC().Format(time.RFC3339),
			},
		})
		return
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse
	io.Copy(io.Discard, resp.Body)

	// Determine result based on status code
	result := TestAPIKeyResult{
		Provider:       provider,
		ResponseTimeMs: responseTimeMs,
		TestedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		result.Status = "invalid"
		result.Message = "Invalid API key"
	case resp.StatusCode >= 200 && resp.StatusCode < 500:
		// 2xx-4xx (except 401/403) means the key is valid
		// 400 can happen with Anthropic when we send a minimal body
		result.Status = "valid"
		result.Message = "Connection successful"
	default:
		result.Status = "error"
		result.Message = fmt.Sprintf("Provider returned status %d", resp.StatusCode)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
