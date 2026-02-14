package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchHandler handles conversation search HTTP requests.
type SearchHandler struct {
	db *sql.DB
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(db *sql.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

type SearchConversationsRequest struct {
	Query      string `form:"q" binding:"required"`
	MaxResults int    `form:"max_results"`
}

type ConversationSearchResult struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Snippet   string                 `json:"snippet"`
	Type      string                 `json:"type"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func (h *SearchHandler) SearchConversations(c *gin.Context) {
	if h.db == nil {
		respondInternalErrorWithSuccess(c, "search handler not initialized")
		return
	}

	var req SearchConversationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequestWithSuccess(c, "Query parameter 'q' is required")
		return
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		respondBadRequestWithSuccess(c, "Search query cannot be empty")
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 100 {
		maxResults = 100
	}

	// Case-insensitive search across all message content
	sqlQuery := `
		SELECT id, type, content, metadata, created_at, updated_at
		FROM memories
		WHERE content LIKE '%' || ? || '%' COLLATE NOCASE
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := h.db.QueryContext(c.Request.Context(), sqlQuery, query, maxResults)
	if err != nil {
		slog.Error("failed to search conversations", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to search conversations")
		return
	}
	defer rows.Close()

	results := []ConversationSearchResult{}
	for rows.Next() {
		var result ConversationSearchResult
		var metadataJSON string

		err := rows.Scan(
			&result.ID,
			&result.Type,
			&result.Content,
			&metadataJSON,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &result.Metadata); err != nil {
				result.Metadata = make(map[string]interface{})
			}
		} else {
			result.Metadata = make(map[string]interface{})
		}

		// Create a snippet: find the match position and extract surrounding text
		result.Snippet = createSnippet(result.Content, query, 100)

		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(results),
		"query":   query,
		"results": results,
	})
}

// createSnippet extracts a snippet around the first match of query in content.
func createSnippet(content, query string, maxLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		// No match found, return start of content
		if len(content) > maxLen {
			return content[:maxLen] + "..."
		}
		return content
	}

	// Calculate snippet boundaries around the match
	start := idx - maxLen/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}
