package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleAdminProviders returns the status of all loaded providers.
// GET /admin/providers
func (s *Server) handleAdminProviders(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	providerMap := s.pluginManager.GetProviders()

	type providerStatus struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Description  string   `json:"description"`
		Capabilities []string `json:"capabilities"`
		Models       int      `json:"models"`
		Healthy      bool     `json:"healthy"`
	}

	var providerStatuses []providerStatus

	for name, p := range providerMap {
		ps := providerStatus{
			Name:    name,
			Healthy: true,
		}

		info, err := p.GetInfo(ctx)
		if err != nil {
			ps.Healthy = false
		} else {
			ps.Version = info.Version
			ps.Description = info.Description
			ps.Capabilities = info.Capabilities
		}

		models, err := p.ListModels(ctx)
		if err != nil {
			ps.Models = 0
		} else {
			ps.Models = len(models)
		}

		providerStatuses = append(providerStatuses, ps)
	}

	// Build response with routing info per spec
	response := gin.H{
		"providers": providerStatuses,
		"total":     len(providerStatuses),
	}
	if s.userRouter != nil {
		response["routing"] = s.userRouter.GetRoutingInfo()
	}

	c.JSON(http.StatusOK, response)
}
