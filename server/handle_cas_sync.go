package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// handleCASDelta returns content entries with sync_cursor > since.
// GET /api/cas/delta?since=0&limit=500
func (s *Server) handleCASDelta(c *gin.Context) {
	sinceStr := c.DefaultQuery("since", "0")
	limitStr := c.DefaultQuery("limit", "500")

	since, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_param", "since must be an integer")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_param", "limit must be 1-1000")
		return
	}

	// Type-assert to access Delta method on the concrete sqliteStore.
	type deltaQuerier interface {
		Delta(ctx context.Context, since int64, limit int) ([]cas.DeltaEntry, error)
	}

	dq, ok := s.workflowCAS.(deltaQuerier)
	if !ok {
		s.errorResponse(c, http.StatusNotImplemented, "not_supported", "Delta queries not supported on this store backend")
		return
	}

	entries, err := dq.Delta(c.Request.Context(), since, limit)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
		"since":   since,
	})
}
