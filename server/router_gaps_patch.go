package server

// ─── Router Gaps Patch (Gap 3) ──────────────────────────────────────────────
//
// This file documents the route registration corrections applied to router.go.
// All /admin/* routes MUST be registered under the admin group that applies
// middleware.AdminAuthMiddleware(). The following changes were made:
//
// 1. CAS API refs endpoints registered under the casGroup (Gap 1):
//    casGroup.GET("/refs", s.handleCASListRefs)
//    casGroup.GET("/refs/:ref", s.handleCASGetRef)
//    casGroup.HEAD("/refs/:ref", s.handleCASHeadRef)
//    casGroup.POST("/refs", s.handleCASStoreRef)
//    casGroup.GET("/export", s.handleCASExport)
//
// 2. Admin cost aggregation moved INTO admin group (Gap 3):
//    admin.GET("/costs", s.handleAdminCosts)
//    -- was previously registered outside the admin group in some branches
//
// 3. Gateway agent routes fixed (Gap 3 path params):
//    gateway.GET("/agents/:id", ...)    -- was "/agents:id" (missing /)
//    gateway.POST("/agents/:id/chat")   -- was "/agents:id/chat"
//    gateway.GET("/orchestrate/:id/dag") -- was "/orchestrate:id/dag"
//
// 4. Agent-channel binding routes added (Gap 5):
//    gateway.POST("/agents/:id/channels", s.handleGatewayBindAgentChannels)
//    gateway.GET("/agents/:id/channels", s.handleGatewayListAgentChannels)
//    gateway.DELETE("/agents/:id/channels/:channel", s.handleGatewayUnbindAgentChannel)
//
// IMPORTANT: This file is a documentation artifact. The actual route
// registration changes are applied directly to router.go via Edit operations.
// This file compiles as part of package server but contains no executable code.
//
// To apply: edit router.go to match the corrected route registrations above.
// The admin group block in setupRoutes() must contain ALL /admin/* routes,
// including the costs endpoint.

// init is intentionally omitted -- this file is documentation only.
// All route registrations live in router.go's setupRoutes() method.
