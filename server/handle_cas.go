package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/cas"
)

// ─── CAS API Types ──────────────────────────────────────────────────────────

// casTagRequest is the body for POST /api/cas/tags.
type casTagRequest struct {
	Name    string `json:"name" binding:"required"`
	Version string `json:"version" binding:"required"`
	Ref     string `json:"ref" binding:"required"`
}

// casPutRequest is the body for POST /api/cas/content.
type casPutRequest struct {
	Content []byte          `json:"content" binding:"required"`
	Meta    cas.ContentMeta `json:"meta"`
}

// ─── Handlers ───────────────────────────────────────────────────────────────

// handleCASListTags lists all tags matching an optional prefix.
// GET /api/cas/tags?prefix=X
func (s *Server) handleCASListTags(c *gin.Context) {
	prefix := c.DefaultQuery("prefix", "")

	entries, err := s.workflowCAS.List(c.Request.Context(), prefix)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to list tags: %v", err))
		return
	}

	// Build response items
	items := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		items = append(items, gin.H{
			"name":    e.Name,
			"version": e.Version,
			"ref":     string(e.Ref),
			"meta":    e.Meta,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tags":  items,
		"count": len(items),
	})
}

// handleCASResolveTag resolves a tag to its content.
// GET /api/cas/tags/:name/:version
func (s *Server) handleCASResolveTag(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")

	ref, err := s.workflowCAS.Resolve(c.Request.Context(), name, version)
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Tag not found: %s@%s", name, version))
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to resolve tag: %v", err))
		return
	}

	// Fetch the content for the resolved ref
	content, meta, err := s.workflowCAS.Get(c.Request.Context(), ref)
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Content not found for ref: %s", ref))
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to get content: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    name,
		"version": version,
		"ref":     string(ref),
		"content": string(content),
		"meta":    meta,
	})
}

// handleCASGetContent retrieves raw content by its SHA-256 reference.
// GET /api/cas/content/:ref
func (s *Server) handleCASGetContent(c *gin.Context) {
	refStr := c.Param("ref")

	content, meta, err := s.workflowCAS.Get(c.Request.Context(), cas.Ref(refStr))
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Content not found: %s", refStr))
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to get content: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ref":     refStr,
		"content": string(content),
		"meta":    meta,
		"size":    len(content),
	})
}

// handleCASPutContent stores new content in CAS.
// POST /api/cas/content
//
// Accepts either JSON body with {"content": "<base64 or utf-8>", "meta": {...}}
// or raw bytes in the request body with Content-Type: application/octet-stream.
func (s *Server) handleCASPutContent(c *gin.Context) {
	var content []byte
	var meta cas.ContentMeta

	contentType := c.ContentType()

	if contentType == "application/octet-stream" {
		// Read raw body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Failed to read body: %v", err))
			return
		}
		content = body
		// Parse meta from query params or headers if needed (minimal for raw upload)
		meta = cas.ContentMeta{
			CreatedBy: c.GetString("user_id"),
		}
	} else {
		// JSON body
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

// handleCASCreateTag assigns a human-readable tag to a content reference.
// POST /api/cas/tags
func (s *Server) handleCASCreateTag(c *gin.Context) {
	var req casTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	err := s.workflowCAS.Tag(c.Request.Context(), req.Name, req.Version, cas.Ref(req.Ref))
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to create tag: %v", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":    req.Name,
		"version": req.Version,
		"ref":     req.Ref,
	})
}

// handleCASDeleteTag removes a tag. Note: the CAS Store interface does not
// expose an Untag method. This handler performs the untag via a direct SQL
// delete on the underlying SQLite database.
//
// DELETE /api/cas/tags/:name/:version
//
// PREREQUISITE: A proper Untag(ctx, name, version) method should be added
// to the cas.Store interface. This implementation uses a workaround for now
// by attempting to resolve and verify the tag exists, then returning
// 501 Not Implemented since the Store interface lacks an Untag method.
func (s *Server) handleCASDeleteTag(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")

	// Verify the tag exists before attempting delete
	_, err := s.workflowCAS.Resolve(c.Request.Context(), name, version)
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Tag not found: %s@%s", name, version))
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Failed to resolve tag: %v", err))
		return
	}

	// The cas.Store interface does not currently expose an Untag method.
	// This requires adding Untag(ctx, name, version) to the Store interface.
	s.errorResponse(c, http.StatusNotImplemented, "not_implemented",
		"Tag deletion requires cas.Store.Untag() method — not yet implemented in the Store interface")
}

// handleCASGarbageCollect triggers garbage collection on unreferenced content.
// POST /api/cas/gc
func (s *Server) handleCASGarbageCollect(c *gin.Context) {
	result, err := s.workflowCAS.GC(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "cas_error", fmt.Sprintf("Garbage collection failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"removed":     result.Removed,
		"freed_bytes": result.Freed,
	})
}

// CAS ref-based endpoints (handleCASListRefs, handleCASGetRef, handleCASHeadRef,
// handleCASStoreRef, handleCASExport) are defined in handle_cas_extended.go.

