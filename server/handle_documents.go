package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/server/database"
)

const maxUploadBytes = 10 << 20 // 10 MB

// ─── Request / Response types ────────────────────────────────────────────────

type searchDocumentsRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

type searchDocumentsResponse struct {
	Chunks   []*chunkWithDocument `json:"chunks"`
	Count    int                  `json:"count"`
	Query    string               `json:"query"`
}

type chunkWithDocument struct {
	*database.DocumentChunk
	Filename string `json:"filename"`
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// handleUploadDocument handles POST /v1/documents.
// Accepts a multipart form upload with field name "file".
// Processes synchronously: extract → chunk → index → ready.
func (s *Server) handleUploadDocument(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	// Parse multipart form; limit memory to 10 MB.
	if err := c.Request.ParseMultipartForm(maxUploadBytes); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Failed to parse multipart form")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Field 'file' is required")
		return
	}
	defer file.Close()

	if header.Size > maxUploadBytes {
		s.errorResponse(c, http.StatusRequestEntityTooLarge, "file_too_large",
			fmt.Sprintf("File exceeds maximum size of %d MB", maxUploadBytes>>20))
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to read file")
		return
	}
	if int64(len(content)) > maxUploadBytes {
		s.errorResponse(c, http.StatusRequestEntityTooLarge, "file_too_large",
			fmt.Sprintf("File exceeds maximum size of %d MB", maxUploadBytes>>20))
		return
	}

	// Determine content type from header, fall back to extension.
	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = inferContentType(header.Filename)
	}

	now := time.Now()
	doc := &database.Document{
		ID:          uuid.New().String(),
		UserID:      userID,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   int64(len(content)),
		ChunkCount:  0,
		Status:      "processing",
		CreatedAt:   now,
	}

	adapter := database.NewLocalAdapter(s.authDB)

	if err := adapter.CreateDocument(c.Request.Context(), doc); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to create document record")
		return
	}

	// Extract plain text from the uploaded file.
	text, err := ExtractText(content, contentType, header.Filename)
	if err != nil {
		// Mark document as error status and surface a useful message.
		_ = adapter.UpdateDocumentStatus(c.Request.Context(), doc.ID, "error", 0)
		s.errorResponse(c, http.StatusUnprocessableEntity, "extraction_failed", err.Error())
		return
	}

	// Chunk the text.
	textChunks := ChunkText(text, 2000, 200)

	// Build DocumentChunk records.
	dbChunks := make([]*database.DocumentChunk, len(textChunks))
	for i, ch := range textChunks {
		dbChunks[i] = &database.DocumentChunk{
			ID:         uuid.New().String(),
			DocumentID: doc.ID,
			ChunkIndex: i,
			Content:    ch,
			CreatedAt:  now,
		}
	}

	if err := adapter.CreateDocumentChunks(c.Request.Context(), dbChunks); err != nil {
		_ = adapter.UpdateDocumentStatus(c.Request.Context(), doc.ID, "error", 0)
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to index document chunks")
		return
	}

	// Mark document ready.
	if err := adapter.UpdateDocumentStatus(c.Request.Context(), doc.ID, "ready", len(dbChunks)); err != nil {
		// Non-fatal: document is indexed, status update failed.
		_ = err
	}
	doc.Status = "ready"
	doc.ChunkCount = len(dbChunks)

	c.JSON(http.StatusCreated, doc)
}

// handleListRAGDocuments handles GET /v1/documents.
func (s *Server) handleListRAGDocuments(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)
	docs, err := adapter.ListDocuments(c.Request.Context(), userID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to list documents")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": docs,
		"count":     len(docs),
	})
}

// handleGetRAGDocument handles GET /v1/documents/:id.
func (s *Server) handleGetRAGDocument(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	docID := c.Param("id")
	if docID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Document ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)
	doc, err := adapter.GetDocument(c.Request.Context(), docID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Document not found")
		return
	}

	if doc.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	c.JSON(http.StatusOK, doc)
}

// handleDeleteRAGDocument handles DELETE /v1/documents/:id.
func (s *Server) handleDeleteRAGDocument(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	docID := c.Param("id")
	if docID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Document ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership before deleting.
	doc, err := adapter.GetDocument(c.Request.Context(), docID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Document not found")
		return
	}
	if doc.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	if err := adapter.DeleteDocument(c.Request.Context(), docID); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to delete document")
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": docID})
}

// handleSearchRAGDocuments handles POST /v1/documents/search.
// Body: { "query": "string", "limit": 5 }
func (s *Server) handleSearchRAGDocuments(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req searchDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body: 'query' is required")
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "'query' must not be empty")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	chunks, err := adapter.SearchDocumentChunks(c.Request.Context(), userID, req.Query, limit)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Search failed")
		return
	}

	// Enrich chunks with filenames for convenience.
	filenameByDocID := make(map[string]string)
	enriched := make([]*chunkWithDocument, 0, len(chunks))
	for _, ch := range chunks {
		if _, seen := filenameByDocID[ch.DocumentID]; !seen {
			doc, err := adapter.GetDocument(c.Request.Context(), ch.DocumentID)
			if err == nil {
				filenameByDocID[ch.DocumentID] = doc.Filename
			} else {
				filenameByDocID[ch.DocumentID] = ch.DocumentID
			}
		}
		enriched = append(enriched, &chunkWithDocument{
			DocumentChunk: ch,
			Filename:      filenameByDocID[ch.DocumentID],
		})
	}

	c.JSON(http.StatusOK, searchDocumentsResponse{
		Chunks: enriched,
		Count:  len(enriched),
		Query:  req.Query,
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// inferContentType returns a MIME type inferred from the file extension.
func inferContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
