package handlers

import (
	"net/http"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/gin-gonic/gin"
)

// MemoryScope resolves whose memories a request may touch (DGS-163).
//
// The default is the authenticated subject only, for every role, admins
// included. ?all_owners=true widens the scope to every owner (legacy rows
// too) and is honoured only for an admin token; service tokens are never
// admin (middleware.IsAdminRequest). Cross-user access is therefore always
// something the caller asked for, never what it gets by default.
//
// On refusal it returns the HTTP status and a message, and the zero scope.
func MemoryScope(c *gin.Context) (memory.OwnerScope, int, string) {
	subject := c.GetString("user_id")
	if subject == "" {
		return memory.OwnerScope{}, http.StatusUnauthorized, "Authentication required"
	}
	if c.Query("all_owners") == "true" {
		if !middleware.IsAdminRequest(c) {
			return memory.OwnerScope{}, http.StatusForbidden, "all_owners requires an admin token"
		}
		return memory.AllOwners(), 0, ""
	}
	return memory.ForOwner(subject), 0, ""
}
