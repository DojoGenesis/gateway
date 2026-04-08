package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/gin-gonic/gin"
)

var (
	modelListTimeout    = getHandlerEnvDuration("MODEL_LIST_TIMEOUT", 5*time.Second)
	providerInfoTimeout = getHandlerEnvDuration("PROVIDER_INFO_TIMEOUT", 2*time.Second)
)

func getHandlerEnvDuration(envKey string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultVal
}

type PluginManagerInterface interface {
	GetProviders() map[string]provider.ModelProvider
	GetProvider(name string) (provider.ModelProvider, error)
}

// ModelHandler handles model and provider HTTP requests.
type ModelHandler struct {
	pluginManager PluginManagerInterface
}

// NewModelHandler creates a new ModelHandler.
func NewModelHandler(pm PluginManagerInterface) *ModelHandler {
	return &ModelHandler{pluginManager: pm}
}

type ProviderStatus struct {
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Info   *provider.ProviderInfo `json:"info,omitempty"`
	Error  string                 `json:"error,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

func (h *ModelHandler) ListModels(c *gin.Context) {
	if h.pluginManager == nil {
		respondInternalError(c, "plugin manager not initialized")
		return
	}

	providers := h.pluginManager.GetProviders()
	allModels := []provider.ModelInfo{}

	for name, provider := range providers {
		ctx, cancel := context.WithTimeout(c.Request.Context(), modelListTimeout)
		defer cancel()

		models, err := provider.ListModels(ctx)
		if err != nil {
			slog.Error("failed to list models", "error", err, "provider", name)
			respondError(c, http.StatusInternalServerError, "failed to list models",
				fmt.Sprintf("provider: %s", name))
			return
		}

		for i := range models {
			models[i].Provider = name
		}

		allModels = append(allModels, models...)
	}

	c.JSON(http.StatusOK, gin.H{
		"models": allModels,
		"count":  len(allModels),
	})
}

func (h *ModelHandler) ListProviders(c *gin.Context) {
	if h.pluginManager == nil {
		respondInternalError(c, "plugin manager not initialized")
		return
	}

	providers := h.pluginManager.GetProviders()
	providerStatuses := []ProviderStatus{}

	for name, provider := range providers {
		ctx, cancel := context.WithTimeout(c.Request.Context(), providerInfoTimeout)
		defer cancel()

		info, err := provider.GetInfo(ctx)
		if err != nil {
			slog.Warn("failed to get provider info", "error", err, "provider", name)
			providerStatuses = append(providerStatuses, ProviderStatus{
				Name:   name,
				Status: "error",
				Error:  err.Error(),
			})
			continue
		}

		providerStatuses = append(providerStatuses, ProviderStatus{
			Name:   name,
			Status: "active",
			Info:   info,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providerStatuses,
		"count":     len(providerStatuses),
	})
}
