package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// OpenAIModel represents a model in OpenAI format.
type OpenAIModel struct {
	ID         string            `json:"id"`
	Object     string            `json:"object"`
	Created    int64             `json:"created"`
	OwnedBy    string            `json:"owned_by"`
	Permission []OpenAIModelPerm `json:"permission"`
	Root       string            `json:"root"`
	Parent     interface{}       `json:"parent"`
}

// OpenAIModelPerm represents model permissions in OpenAI format.
type OpenAIModelPerm struct {
	ID                 string      `json:"id"`
	Object             string      `json:"object"`
	Created            int64       `json:"created"`
	AllowCreateEngine  bool        `json:"allow_create_engine"`
	AllowSampling      bool        `json:"allow_sampling"`
	AllowLogprobs      bool        `json:"allow_logprobs"`
	AllowSearchIndices bool        `json:"allow_search_indices"`
	AllowView          bool        `json:"allow_view"`
	AllowFineTuning    bool        `json:"allow_fine_tuning"`
	Organization       string      `json:"organization"`
	GroupID            interface{} `json:"group_id"`
	IsBlocking         bool        `json:"is_blocking"`
}

// OpenAIModelList represents the list models response in OpenAI format.
type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

// handleListModels handles GET /v1/models (OpenAI-compatible).
func (s *Server) handleListModels(c *gin.Context) {
	if s.pluginManager == nil {
		c.JSON(http.StatusOK, OpenAIModelList{
			Object: "list",
			Data:   []OpenAIModel{},
		})
		return
	}

	providers := s.pluginManager.GetProviders()
	var models []OpenAIModel

	for provName, prov := range providers {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		provModels, err := prov.ListModels(ctx)
		cancel()
		if err != nil {
			continue
		}

		for _, m := range provModels {
			now := time.Now().Unix()
			models = append(models, OpenAIModel{
				ID:      m.ID,
				Object:  "model",
				Created: now,
				OwnedBy: provName,
				Permission: []OpenAIModelPerm{
					{
						ID:                 "modelperm-" + m.ID,
						Object:             "model_permission",
						Created:            now,
						AllowCreateEngine:  false,
						AllowSampling:      true,
						AllowLogprobs:      true,
						AllowSearchIndices: false,
						AllowView:          true,
						AllowFineTuning:    false,
						Organization:       "*",
						GroupID:            nil,
						IsBlocking:         false,
					},
				},
				Root:   m.ID,
				Parent: nil,
			})
		}
	}

	if models == nil {
		models = []OpenAIModel{}
	}

	c.JSON(http.StatusOK, OpenAIModelList{
		Object: "list",
		Data:   models,
	})
}
