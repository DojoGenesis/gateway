package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
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

var (
	pluginManager PluginManagerInterface
)

func InitializeModelHandlers(pm PluginManagerInterface) {
	pluginManager = pm
}

type ProviderStatus struct {
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Info   *provider.ProviderInfo   `json:"info,omitempty"`
	Error  string                 `json:"error,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

func HandleListModels(c *gin.Context) {
	if pluginManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "plugin manager not initialized",
		})
		return
	}

	providers := pluginManager.GetProviders()
	allModels := []provider.ModelInfo{}

	for name, provider := range providers {
		ctx, cancel := context.WithTimeout(c.Request.Context(), modelListTimeout)
		defer cancel()

		models, err := provider.ListModels(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":    "failed to list models",
				"provider": name,
				"details":  err.Error(),
			})
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

func HandleListProviders(c *gin.Context) {
	if pluginManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "plugin manager not initialized",
		})
		return
	}

	providers := pluginManager.GetProviders()
	providerStatuses := []ProviderStatus{}

	for name, provider := range providers {
		ctx, cancel := context.WithTimeout(c.Request.Context(), providerInfoTimeout)
		defer cancel()

		info, err := provider.GetInfo(ctx)
		if err != nil {
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
