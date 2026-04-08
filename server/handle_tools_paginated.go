package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ─── Tool Pagination Fix (Gap 6) ────────────────────────────────────────────
//
// The existing handleListTools in handle_tools.go already has basic pagination
// but this file provides a corrected version with proper slice bounds checking
// and consistent total count in the response. The gateway tools endpoint
// (handleGatewayListTools) in handle_gateway.go lacks pagination entirely.
//
// This handler can be wired to GET /v1/tools as a drop-in replacement,
// or used for the gateway variant at GET /v1/gateway/tools.

// handleListToolsPaginated is the corrected tool listing with proper pagination.
// GET /v1/tools?limit=N&offset=N&category=X
//
// When limit=0 or omitted, all tools are returned (backward compatible).
// The response always includes "total" reflecting the unfiltered count.
func (s *Server) handleListToolsPaginated(c *gin.Context) {
	if s.toolRegistry == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable",
			"Tool registry not available")
		return
	}

	allTools, err := s.toolRegistry.List(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error",
			"Failed to list tools: "+err.Error())
		return
	}

	// Optional category filter
	category := c.Query("category")

	// Build response list with optional filtering
	toolList := make([]gin.H, 0, len(allTools))
	for _, tool := range allTools {
		ns := extractNamespace(tool.Name)

		// Apply category filter if provided
		if category != "" && ns != category {
			continue
		}

		toolList = append(toolList, gin.H{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
			"namespace":   ns,
		})
	}

	total := len(toolList)

	// Parse pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Clamp offset to valid range
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}

	// Apply pagination when limit > 0
	if limit > 0 {
		end := offset + limit
		if end > total {
			end = total
		}
		toolList = toolList[offset:end]
	} else {
		// No pagination -- return everything (limit=0 means unlimited)
		limit = total
		offset = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"tools":  toolList,
		"count":  len(toolList),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
