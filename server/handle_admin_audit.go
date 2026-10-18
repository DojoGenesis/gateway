package server

import (
	"net/http"
	"strconv"

	"github.com/DojoGenesis/gateway/observation/audit"
	"github.com/gin-gonic/gin"
)

const maxAuditEntries = 1000

// handleAdminAuditVerify walks the audit log's hash chain and reports any
// entry that was edited, removed or reordered.
// GET /admin/audit/verify
func (s *Server) handleAdminAuditVerify(c *gin.Context) {
	if s.auditRecorder == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Audit log is not enabled")
		return
	}
	rep, err := audit.Verify(c.Request.Context(), s.auditRecorder.Log())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "audit_verify_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, rep)
}

// handleAdminAuditEntries lists audit entries, newest first.
// GET /admin/audit/entries?agent_id=&action=&tool=&limit=&offset=
func (s *Server) handleAdminAuditEntries(c *gin.Context) {
	if s.auditRecorder == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Audit log is not enabled")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 || limit > maxAuditEntries {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "limit must be between 1 and 1000")
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "offset must be 0 or more")
		return
	}

	filter := audit.AuditFilter{
		AgentID: c.Query("agent_id"),
		Tool:    c.Query("tool"),
		Limit:   limit,
		Offset:  offset,
	}
	if a := c.Query("action"); a != "" {
		filter.Actions = []audit.Action{audit.Action(a)}
	}

	entries, err := s.auditRecorder.Log().Query(c.Request.Context(), filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "audit_query_failed", err.Error())
		return
	}
	if entries == nil {
		entries = []audit.AuditEntry{}
	}
	headSeq, headHash := s.auditRecorder.Head()
	c.JSON(http.StatusOK, gin.H{
		"entries":   entries,
		"count":     len(entries),
		"head_seq":  headSeq,
		"head_hash": headHash,
	})
}
