package server

// Registration (add to setupRoutes in router.go under the admin group):
//   admin.GET("/routing/mode", s.handleAdminRoutingMode)
//   admin.POST("/routing/mode", s.handleAdminSetRoutingMode)
//   admin.GET("/routing/stats", s.handleAdminRoutingStats)
//
// Server struct field (add to the Server struct in server.go):
//   // semanticRouter provides hot-switchable routing mode control.
//   // Nil when the semantic routing feature is not enabled.
//   semanticRouter *agent.SemanticRouter

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/server/agent"
)

// availableRoutingModes is the fixed set of valid mode strings exposed by the API.
var availableRoutingModes = []string{"cascade", "llm", "embedding"}

// routingModeFromString maps a user-supplied string to the RoutingMode constant.
// Returns (mode, true) on success or (0, false) when the string is unrecognised.
func routingModeFromString(s string) (agent.RoutingMode, bool) {
	switch s {
	case "cascade":
		return agent.RoutingModeCascade, true
	case "llm":
		return agent.RoutingModeLLM, true
	case "embedding":
		return agent.RoutingModeEmbedding, true
	default:
		return 0, false
	}
}

// routeDefinitionSummary is a JSON-serialisable subset of agent.RouteDefinition
// suitable for admin API responses (omits embedding centroid and utterances).
type routeDefinitionSummary struct {
	Name      string  `json:"name"`
	Handler   string  `json:"handler"`
	Threshold float64 `json:"threshold"`
}

// buildRouteSummaries converts raw RouteDefinitions to the wire-format summary
// (omitting centroids and utterances).
func buildRouteSummaries(routes []agent.RouteDefinition) []routeDefinitionSummary {
	summaries := make([]routeDefinitionSummary, 0, len(routes))
	for _, r := range routes {
		summaries = append(summaries, routeDefinitionSummary{
			Name:      r.Name,
			Handler:   r.Handler,
			Threshold: r.Threshold,
		})
	}
	return summaries
}

// handleAdminRoutingMode returns the current routing mode and available modes.
//
// GET /admin/routing/mode
//
// Response 200:
//
//	{
//	  "current_mode": "cascade",
//	  "available_modes": ["cascade", "llm", "embedding"],
//	  "routes": [
//	    {"name": "trivial", "handler": "template", "threshold": 0.65}
//	  ]
//	}
//
// Response 503 when the semantic router has not been initialised.
func (s *Server) handleAdminRoutingMode(c *gin.Context) {
	if s.semanticRouter == nil {
		slog.Warn("handleAdminRoutingMode: semanticRouter not initialised")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "semantic router not initialised",
		})
		return
	}

	currentMode := s.semanticRouter.GetMode()
	summaries := buildRouteSummaries(s.semanticRouter.GetRoutes())

	slog.Info("admin: routing mode queried", "mode", currentMode.String(), "routes", len(summaries))

	c.JSON(http.StatusOK, gin.H{
		"current_mode":    currentMode.String(),
		"available_modes": availableRoutingModes,
		"routes":          summaries,
	})
}

// handleAdminSetRoutingMode hot-switches the active routing mode.
//
// POST /admin/routing/mode
//
// Request body:
//
//	{"mode": "llm"}   // "cascade" | "llm" | "embedding"
//
// Response 200 — new state (same shape as GET /admin/routing/mode).
// Response 400 — unknown mode string or missing body field.
// Response 503 — semantic router not initialised.
func (s *Server) handleAdminSetRoutingMode(c *gin.Context) {
	if s.semanticRouter == nil {
		slog.Warn("handleAdminSetRoutingMode: semanticRouter not initialised")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "semantic router not initialised",
		})
		return
	}

	var req struct {
		Mode string `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("handleAdminSetRoutingMode: invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request body must contain a 'mode' field",
		})
		return
	}

	newMode, ok := routingModeFromString(req.Mode)
	if !ok {
		slog.Warn("handleAdminSetRoutingMode: unknown routing mode", "requested", req.Mode)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "unknown routing mode: " + req.Mode,
			"available_modes": availableRoutingModes,
		})
		return
	}

	previousMode := s.semanticRouter.GetMode()
	s.semanticRouter.SetMode(newMode)

	slog.Info("admin: routing mode updated",
		"previous", previousMode.String(),
		"new", newMode.String(),
	)

	summaries := buildRouteSummaries(s.semanticRouter.GetRoutes())

	c.JSON(http.StatusOK, gin.H{
		"current_mode":    newMode.String(),
		"previous_mode":   previousMode.String(),
		"available_modes": availableRoutingModes,
		"routes":          summaries,
	})
}

// handleAdminRoutingStats returns routing statistics and a route inventory.
//
// GET /admin/routing/stats
//
// The SemanticRouter does not yet expose per-route counters; this endpoint
// returns the current mode and route count as a stable placeholder. When the
// SemanticRouter gains a Stats() method, replace the placeholder body below.
//
// Response 200:
//
//	{
//	  "current_mode": "cascade",
//	  "route_count": 4,
//	  "stats": null
//	}
//
// Response 503 — semantic router not initialised.
func (s *Server) handleAdminRoutingStats(c *gin.Context) {
	if s.semanticRouter == nil {
		slog.Warn("handleAdminRoutingStats: semanticRouter not initialised")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "semantic router not initialised",
		})
		return
	}

	currentMode := s.semanticRouter.GetMode()
	routes := s.semanticRouter.GetRoutes()

	slog.Info("admin: routing stats queried", "mode", currentMode.String(), "route_count", len(routes))

	// stats field is intentionally null until SemanticRouter exposes counters.
	c.JSON(http.StatusOK, gin.H{
		"current_mode": currentMode.String(),
		"route_count":  len(routes),
		"stats":        nil,
	})
}
