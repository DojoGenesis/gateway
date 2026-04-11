package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// ─── /api/cas/refs/* endpoints (Gap 1) ──────────────────────────────────────
//
// These handlers extend the existing CAS API with ref-centric access patterns.
// The existing handle_cas.go provides tag/content endpoints; this file adds
// the refs-based equivalents that the frontend spec expects.

// handleCASListRefs lists all tags matching an optional prefix, returning refs.
// GET /api/cas/refs?prefix=X
func (s *Server) handleCASListRefs(c *gin.Context) {
	prefix := c.DefaultQuery("prefix", "")

	entries, err := s.workflowCAS.List(c.Request.Context(), prefix)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to list refs: %v", err))
		return
	}

	refs := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, gin.H{
			"ref":     string(e.Ref),
			"name":    e.Name,
			"version": e.Version,
			"meta":    e.Meta,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"refs":  refs,
		"count": len(refs),
	})
}

// handleCASGetRef retrieves content by its SHA-256 reference.
// GET /api/cas/refs/:ref
func (s *Server) handleCASGetRef(c *gin.Context) {
	refStr := c.Param("ref")

	content, meta, err := s.workflowCAS.Get(c.Request.Context(), cas.Ref(refStr))
	if err != nil {
		if isNotFound(err) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Ref not found: %s", refStr))
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to get ref: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ref":     refStr,
		"content": string(content),
		"meta":    meta,
		"size":    len(content),
	})
}

// handleCASHeadRef checks if a ref exists without returning content.
// HEAD /api/cas/refs/:ref
func (s *Server) handleCASHeadRef(c *gin.Context) {
	refStr := c.Param("ref")

	exists, err := s.workflowCAS.Has(c.Request.Context(), cas.Ref(refStr))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}

// handleCASStoreRef stores new content and returns the computed ref.
// POST /api/cas/refs
//
// Accepts either JSON body with {"content": "<base64 or utf-8>", "meta": {...}}
// or raw bytes in the request body with Content-Type: application/octet-stream.
func (s *Server) handleCASStoreRef(c *gin.Context) {
	var content []byte
	var meta cas.ContentMeta

	contentType := c.ContentType()

	if contentType == "application/octet-stream" {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Failed to read body: %v", err))
			return
		}
		content = body
		meta = cas.ContentMeta{
			CreatedBy: c.GetString("user_id"),
		}
	} else {
		var req casPutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
			return
		}
		content = req.Content
		meta = req.Meta
		if meta.CreatedBy == "" {
			meta.CreatedBy = c.GetString("user_id")
		}
	}

	if len(content) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Content must not be empty")
		return
	}

	ref, err := s.workflowCAS.Put(c.Request.Context(), content, meta)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to store content: %v", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ref":  string(ref),
		"size": len(content),
	})
}

// handleCASExport exports refs as a tar archive.
// GET /api/cas/export?refs=sha1,sha2,...
func (s *Server) handleCASExport(c *gin.Context) {
	refsParam := c.Query("refs")
	if refsParam == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Query parameter 'refs' is required (comma-separated SHA-256 hashes)")
		return
	}

	refStrings := strings.Split(refsParam, ",")
	refs := make([]cas.Ref, 0, len(refStrings))
	for _, r := range refStrings {
		trimmed := strings.TrimSpace(r)
		if trimmed == "" {
			continue
		}
		refs = append(refs, cas.Ref(trimmed))
	}

	if len(refs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "At least one ref is required")
		return
	}

	// Verify all refs exist before streaming
	for _, ref := range refs {
		exists, err := s.workflowCAS.Has(c.Request.Context(), ref)
		if err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to check ref %s: %v", ref, err))
			return
		}
		if !exists {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Ref not found: %s", ref))
			return
		}
	}

	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", "attachment; filename=cas-export.tar")

	if err := s.workflowCAS.Export(c.Request.Context(), refs, c.Writer); err != nil {
		// Headers already sent -- log the error; can't send JSON error
		c.Status(http.StatusInternalServerError)
		return
	}
}

// handleCASImport imports content from a tar archive.
// POST /api/cas/import
//
// Accepts a tar archive in the request body (Content-Type: application/x-tar).
// Returns the list of refs that were imported.
func (s *Server) handleCASImport(c *gin.Context) {
	refs, err := s.workflowCAS.Import(c.Request.Context(), c.Request.Body)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "import_error", fmt.Sprintf("Failed to import archive: %v", err))
		return
	}

	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = append(result, string(ref))
	}

	c.JSON(http.StatusOK, gin.H{
		"refs":  result,
		"count": len(result),
	})
}

// isNotFound checks if an error is a CAS not-found error.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
