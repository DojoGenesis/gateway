package server

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// casBatchEntry is a single blob in a PUT /cas/batch request.
type casBatchEntry struct {
	Hash        string `json:"hash"`
	ContentType string `json:"content_type"`
	DataBase64  string `json:"data_base64"`
}

// handleCASBatch accepts a batch of blobs and stores them in the CAS.
// PUT /api/cas/batch
//
// Request body: {"entries": [{"hash": "...", "content_type": "skill", "data_base64": "..."}]}
// Each entry is idempotent: if the hash already exists, meta is updated.
func (s *Server) handleCASBatch(c *gin.Context) {
	var req struct {
		Entries []casBatchEntry `json:"entries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if len(req.Entries) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "entries must not be empty")
		return
	}

	if len(req.Entries) > 500 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "max 500 entries per batch")
		return
	}

	var stored []string
	var errors []gin.H

	for _, entry := range req.Entries {
		data, err := base64.StdEncoding.DecodeString(entry.DataBase64)
		if err != nil {
			errors = append(errors, gin.H{
				"hash":  entry.Hash,
				"error": "invalid base64: " + err.Error(),
			})
			continue
		}

		meta := cas.ContentMeta{
			Type:      cas.ContentType(entry.ContentType),
			CreatedAt: time.Now().UTC(),
			CreatedBy: "d1-sync",
			Size:      int64(len(data)),
		}

		ref, err := s.workflowCAS.Put(c.Request.Context(), data, meta)
		if err != nil {
			errors = append(errors, gin.H{
				"hash":  entry.Hash,
				"error": err.Error(),
			})
			continue
		}

		stored = append(stored, string(ref))
	}

	status := http.StatusOK
	if len(errors) > 0 && len(stored) == 0 {
		status = http.StatusInternalServerError
	} else if len(errors) > 0 {
		status = http.StatusPartialContent // 206 — some succeeded
	}

	c.JSON(status, gin.H{
		"stored":  stored,
		"errors":  errors,
		"total":   len(req.Entries),
		"success": len(stored),
		"failed":  len(errors),
	})
}

// handleCASSyncStatus returns the sync health status.
// GET /api/cas/status
func (s *Server) handleCASSyncStatus(c *gin.Context) {
	// Check if D1 syncer is available
	if s.d1Syncer == nil {
		c.JSON(http.StatusOK, gin.H{
			"sync_enabled": false,
			"message":      "D1 sync not configured",
		})
		return
	}

	status := s.d1Syncer.Status()

	c.JSON(http.StatusOK, gin.H{
		"sync_enabled": true,
		"last_cursor":  status.LastCursor,
		"last_sync_at": status.LastSyncAt,
		"lag_seconds":  status.LagSeconds,
		"healthy":      status.Healthy,
	})
}
