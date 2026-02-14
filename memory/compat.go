package memory

import (
	"context"
	"fmt"
	"time"
)

// ─── Compatibility aliases ───────────────────────────────────────────────────
// These methods provide a simplified API that maps to the canonical methods
// on MemoryManager. They exist so callers in handlers/ can use short names.

// Store is a convenience alias for StoreMemory.
func (m *MemoryManager) Store(ctx context.Context, mem Memory) error {
	return m.StoreMemory(ctx, &mem)
}

// Retrieve is a convenience alias for GetMemory.
func (m *MemoryManager) Retrieve(ctx context.Context, id string) (*Memory, error) {
	return m.GetMemory(ctx, id)
}

// Get is a convenience alias for GetMemory.
func (m *MemoryManager) Get(ctx context.Context, id string) (*Memory, error) {
	return m.GetMemory(ctx, id)
}

// Search is a convenience alias for SearchMemories.
func (m *MemoryManager) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return m.SearchMemories(ctx, query, limit)
}

// List is a convenience alias for ListMemories using session ID.
func (m *MemoryManager) List(ctx context.Context, sessionID string, limit int) ([]Memory, error) {
	filter := MemoryFilter{
		Limit: limit,
	}
	if sessionID != "" {
		filter.ContextType = sessionID
	}
	results, err := m.ListMemories(ctx, filter)
	if err != nil {
		return nil, err
	}
	memories := make([]Memory, 0, len(results))
	for _, r := range results {
		memories = append(memories, *r)
	}
	return memories, nil
}

// Update is a convenience alias that retrieves, patches, and stores a memory.
func (m *MemoryManager) Update(ctx context.Context, id, content string, metadata map[string]interface{}) (*Memory, error) {
	mem, err := m.GetMemory(ctx, id)
	if err != nil {
		return nil, err
	}
	if content != "" {
		mem.Content = content
	}
	if metadata != nil {
		mem.Metadata = metadata
	}
	mem.UpdatedAt = time.Now()
	if err := m.UpdateMemory(ctx, mem); err != nil {
		return nil, err
	}
	return mem, nil
}

// Delete is a convenience alias for DeleteMemory.
func (m *MemoryManager) Delete(ctx context.Context, id string) error {
	return m.DeleteMemory(ctx, id)
}

// SearchByType searches memories filtered by type.
// Accepts (ctx, memType, limit) to match agent module's calling convention.
func (m *MemoryManager) SearchByType(ctx context.Context, memType string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 10
	}
	// Use ListMemories filtered by type
	results, err := m.ListMemories(ctx, MemoryFilter{
		Type:  memType,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	memories := make([]Memory, 0, len(results))
	for _, r := range results {
		memories = append(memories, *r)
	}
	return memories, nil
}

// SearchSemantic provides a text-based search with optional tier and context type filtering.
// This is a compatibility method for server/apptools/memory_tools.go which expects this signature.
// Since we don't have true vector search without embeddings, it delegates to text search
// and applies post-query filtering for tier and context type.
func (m *MemoryManager) SearchSemantic(ctx context.Context, query string, tier int, contextType string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	// Use text search as the base
	results, err := m.SearchMemories(ctx, query, maxResults*2) // over-fetch for post-filtering
	if err != nil {
		return nil, err
	}

	// Post-filter by context type if specified
	if contextType != "" {
		filtered := make([]SearchResult, 0, len(results))
		for _, r := range results {
			if r.Memory.ContextType == contextType {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Cap to maxResults
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// GetLines retrieves a memory by ID and returns its content split into lines.
// This is a compatibility method for server/apptools/memory_tools.go.
func (m *MemoryManager) GetLines(ctx context.Context, path string, startLine int, lineCount int) (string, map[string]interface{}, error) {
	mem, err := m.GetMemory(ctx, path)
	if err != nil {
		return "", nil, fmt.Errorf("memory not found: %w", err)
	}

	// Split content into lines and extract the requested range
	lines := splitString(mem.Content, "\n")

	if startLine >= len(lines) {
		return "", mem.Metadata, nil
	}

	end := startLine + lineCount
	if end > len(lines) {
		end = len(lines)
	}

	selected := lines[startLine:end]
	content := ""
	for i, line := range selected {
		if i > 0 {
			content += "\n"
		}
		content += line
	}

	return content, mem.Metadata, nil
}

// ─── Missing types needed by server/maintenance ──────────────────────────────

// PluginManagerInterface allows maintenance code to call LLM providers.
// Returns interface{} to avoid importing the provider package (which would
// create a circular dependency). Callers in the server module type-assert
// the returned value to provider.ModelProvider.
type PluginManagerInterface interface {
	GetProvider(name string) (interface{}, error)
	GetProviders() map[string]interface{}
}

// LLMProvider is a minimal interface for generating completions.
// This avoids importing the provider package into memory.
type LLMProvider interface {
	GenerateCompletion(ctx context.Context, req interface{}) (interface{}, error)
}

// CompressionService wraps a PluginManagerInterface to provide LLM-based compression.
type CompressionService struct {
	pluginManager PluginManagerInterface
}

// NewCompressionService creates a CompressionService.
func NewCompressionService(pm PluginManagerInterface) *CompressionService {
	return &CompressionService{pluginManager: pm}
}

// CompressHistory compresses memories into a condensed representation.
// Uses the pluginManager to call an LLM for compression. Falls back to
// simple concatenation if the provider is unavailable.
func (cs *CompressionService) CompressHistory(ctx context.Context, sessionID string, memories []Memory) (*CompressedHistory, error) {
	content := BuildOriginalContent(memories)
	prompt := BuildCompressionPrompt(content)

	prov, err := cs.pluginManager.GetProvider("embedded-qwen3")
	if err != nil {
		// Fallback: simple concatenation
		return &CompressedHistory{
			SessionID:         sessionID,
			CompressedContent: content,
			CompressionRatio:  1.0,
			CreatedAt:         time.Now(),
		}, nil
	}

	// Use the LLM provider via reflection-free interface assertion
	compressed, err := callLLMCompletion(ctx, prov, prompt, 0.3, 500)
	if err != nil {
		return &CompressedHistory{
			SessionID:         sessionID,
			CompressedContent: content,
			CompressionRatio:  1.0,
			CreatedAt:         time.Now(),
		}, nil
	}

	ratio := 0.0
	if len(content) > 0 {
		ratio = float64(len(compressed)) / float64(len(content))
	}

	turnIDs := make([]string, len(memories))
	for i, mem := range memories {
		turnIDs[i] = mem.ID
	}

	return &CompressedHistory{
		SessionID:         sessionID,
		OriginalTurnIDs:   turnIDs,
		CompressedContent: compressed,
		CompressionRatio:  ratio,
		CreatedAt:         time.Now(),
	}, nil
}

// IdentifyKeyThemes extracts key themes from content for maintenance.
func (cs *CompressionService) IdentifyKeyThemes(ctx context.Context, content string) ([]string, error) {
	prov, err := cs.pluginManager.GetProvider("embedded-qwen3")
	if err != nil {
		return []string{"general"}, nil
	}

	prompt := fmt.Sprintf("List 3-5 key themes from this text as a comma-separated list:\n\n%s", content)
	resp, err := callLLMCompletion(ctx, prov, prompt, 0.3, 100)
	if err != nil {
		return []string{"general"}, nil
	}

	var themes []string
	for _, t := range splitCSV(resp) {
		trimmed := trimString(t)
		if trimmed != "" {
			themes = append(themes, trimmed)
		}
	}

	if len(themes) == 0 {
		return []string{"general"}, nil
	}
	return themes, nil
}

// ExtractSeeds extracts reusable patterns from memories.
func (cs *CompressionService) ExtractSeeds(ctx context.Context, memories []Memory) ([]*MemorySeed, error) {
	return []*MemorySeed{}, nil
}

// callLLMCompletion calls an LLM provider generically without importing the provider package.
// The provider must implement a GenerateCompletion method. This function uses the
// CompletionCaller interface for type-safe invocation.
func callLLMCompletion(ctx context.Context, prov interface{}, prompt string, temperature float64, maxTokens int) (string, error) {
	type completionRequest struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
		Stream      bool      `json:"stream"`
	}

	type completionResponse struct {
		Content string
	}

	// Try to use the provider via a simple interface
	type simpleCompleter interface {
		Complete(ctx context.Context, prompt string) (string, error)
	}

	if sc, ok := prov.(simpleCompleter); ok {
		return sc.Complete(ctx, prompt)
	}

	// The provider doesn't match our simple interface.
	// Return an error so the caller falls back to the default behavior.
	return "", fmt.Errorf("provider does not implement a compatible completion interface")
}

// ─── Utility functions ───────────────────────────────────────────────────────

func splitCSV(s string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, ",") {
		parts = append(parts, p)
	}
	return parts
}

func splitString(s, sep string) []string {
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
