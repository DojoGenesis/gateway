package apptools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

var memoryManager *memory.MemoryManager

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

var searchRateLimiter = &rateLimiter{
	requests: make(map[string][]time.Time),
	limit:    10,
	window:   1 * time.Minute,
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.requests[key]
	valid := []time.Time{}
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true
}

func (rl *rateLimiter) remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.requests[key]
	count := 0
	for _, t := range timestamps {
		if t.After(cutoff) {
			count++
		}
	}

	return rl.limit - count
}

func InitializeMemoryTools(mm *memory.MemoryManager) {
	memoryManager = mm
}

func MemorySearch(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if memoryManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "memory manager not initialized",
		}, nil
	}

	sessionID := tools.GetStringParam(params, "session_id", "default")
	rateLimitKey := fmt.Sprintf("memory_search:%s", sessionID)

	if !searchRateLimiter.allow(rateLimitKey) {
		remaining := searchRateLimiter.remaining(rateLimitKey)
		return map[string]interface{}{
			"success":   false,
			"error":     "rate limit exceeded",
			"limit":     searchRateLimiter.limit,
			"window":    searchRateLimiter.window.String(),
			"remaining": remaining,
		}, nil
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "query parameter is required and must be a non-empty string",
		}, nil
	}

	tier := tools.GetIntParam(params, "tier", 0)
	if tier < 0 || tier > 3 {
		return map[string]interface{}{
			"success": false,
			"error":   "tier must be 0 (all), 1 (raw), 2 (curated), or 3 (archive)",
		}, nil
	}

	maxResults := tools.GetIntParam(params, "max_results", 10)
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 100 {
		maxResults = 100
	}

	contextType := tools.GetStringParam(params, "context_type", "")

	results, err := memoryManager.SearchSemantic(ctx, query, tier, contextType, maxResults)
	if err != nil {
		slog.Error("memory search failed", "error", err, "query", query)
		return map[string]interface{}{
			"success": false,
			"error":   "search failed",
		}, nil
	}

	searchResults := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		searchResults = append(searchResults, map[string]interface{}{
			"id":           result.Memory.ID,
			"type":         result.Memory.Type,
			"content":      result.Memory.Content,
			"snippet":      result.Snippet,
			"similarity":   result.Similarity,
			"context_type": result.Memory.ContextType,
			"created_at":   result.Memory.CreatedAt,
			"updated_at":   result.Memory.UpdatedAt,
			"metadata":     result.Memory.Metadata,
		})
	}

	return map[string]interface{}{
		"success":     true,
		"query":       query,
		"tier":        tier,
		"results":     searchResults,
		"count":       len(searchResults),
		"max_results": maxResults,
	}, nil
}

func MemoryGet(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if memoryManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "memory manager not initialized",
		}, nil
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "path parameter is required and must be a non-empty string",
		}, nil
	}

	startLine := tools.GetIntParam(params, "start_line", 0)
	if startLine < 0 {
		startLine = 0
	}

	lineCount := tools.GetIntParam(params, "line_count", 50)
	if lineCount <= 0 {
		lineCount = 50
	}
	if lineCount > 1000 {
		lineCount = 1000
	}

	content, metadata, err := memoryManager.GetLines(ctx, path, startLine, lineCount)
	if err != nil {
		slog.Error("failed to retrieve content", "error", err, "path", path)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to retrieve content",
		}, nil
	}

	return map[string]interface{}{
		"success":    true,
		"path":       path,
		"content":    content,
		"metadata":   metadata,
		"start_line": startLine,
		"line_count": lineCount,
	}, nil
}

func init() {
	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "memory_search",
		Description: "Search memory using semantic vector similarity. Returns the most relevant memories based on the query. Supports filtering by tier (1=raw notes, 2=curated wisdom, 3=archive) and context type (private/group/public). Use 'private' for single-user context, 'group' for shared team memories, 'public' for generally accessible information.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query to find semantically similar memories",
				},
				"tier": map[string]interface{}{
					"type":        "integer",
					"description": "Memory tier to search: 0 (all), 1 (raw daily notes), 2 (curated wisdom), 3 (compressed archive). Default: 0",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (1-100). Default: 10",
				},
				"context_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by context type: 'private' (single-user), 'group' (team/shared), or 'public' (general access). Default: all types. Privacy rule: non-private contexts only return Tier 1 (raw) data.",
				},
			},
			"required": []string{"query"},
		},
		Function: MemorySearch,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "memory_get",
		Description: "Retrieve specific lines from a memory by ID or file path. Supports both database-stored memories and file-based memories. Use this after memory_search to get detailed content.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Memory ID (for database memories) or file path (for file-based memories)",
				},
				"start_line": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (0-indexed). Default: 0",
				},
				"line_count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to retrieve (1-1000). Default: 50",
				},
			},
			"required": []string{"path"},
		},
		Function: MemoryGet,
	})
}
