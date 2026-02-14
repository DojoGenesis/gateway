package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/google/uuid"
)

// GatewayMemoryStore adapts the existing memory.MemoryManager
// to implement the gateway.MemoryStore interface.
// This allows the memory manager to be used through the gateway's standard interface.
type GatewayMemoryStore struct {
	manager *MemoryManager
}

// NewGatewayMemoryStore creates a new gateway-compatible memory store.
func NewGatewayMemoryStore(manager *MemoryManager) *GatewayMemoryStore {
	if manager == nil {
		return nil
	}
	return &GatewayMemoryStore{
		manager: manager,
	}
}

// Store persists a memory entry to the backend.
func (s *GatewayMemoryStore) Store(ctx context.Context, entry *gateway.MemoryEntry) error {
	if entry == nil {
		return fmt.Errorf("memory entry cannot be nil")
	}

	// Convert gateway.MemoryEntry to memory.Memory
	memory := Memory{
		ID:        entry.ID,
		Type:      entry.EntryType,
		Content:   entry.Content,
		Metadata:  entry.Metadata,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}

	// Generate ID if not provided
	if memory.ID == "" {
		memory.ID = uuid.New().String()
		entry.ID = memory.ID
	}

	// Set timestamps if not provided
	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
		entry.CreatedAt = now
	}
	if memory.UpdatedAt.IsZero() {
		memory.UpdatedAt = now
		entry.UpdatedAt = now
	}

	// Store using memory manager
	return s.manager.Store(ctx, memory)
}

// Search queries memory entries using text or embedding similarity.
func (s *GatewayMemoryStore) Search(ctx context.Context, query *gateway.SearchQuery, limit int) ([]*gateway.MemoryEntry, error) {
	if query == nil {
		return nil, fmt.Errorf("search query cannot be nil")
	}

	// Search using memory manager
	searchResults, err := s.manager.Search(ctx, query.Text, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to gateway.MemoryEntry format
	entries := make([]*gateway.MemoryEntry, 0, len(searchResults))
	for _, result := range searchResults {
		// Filter by entry type if specified
		if query.EntryType != "" && result.Memory.Type != query.EntryType {
			continue
		}

		entry := &gateway.MemoryEntry{
			ID:        result.Memory.ID,
			EntryType: result.Memory.Type,
			Content:   result.Memory.Content,
			Metadata:  result.Memory.Metadata,
			CreatedAt: result.Memory.CreatedAt,
			UpdatedAt: result.Memory.UpdatedAt,
		}
		entries = append(entries, entry)
	}

	// Apply limit
	if len(entries) > limit && limit > 0 {
		entries = entries[:limit]
	}

	return entries, nil
}

// Get retrieves a specific memory entry by its unique ID.
func (s *GatewayMemoryStore) Get(ctx context.Context, id string) (*gateway.MemoryEntry, error) {
	if id == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
	}

	// Get from memory manager
	memory, err := s.manager.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}

	// Convert to gateway.MemoryEntry
	entry := &gateway.MemoryEntry{
		ID:        memory.ID,
		EntryType: memory.Type,
		Content:   memory.Content,
		Metadata:  memory.Metadata,
		CreatedAt: memory.CreatedAt,
		UpdatedAt: memory.UpdatedAt,
	}

	return entry, nil
}

// Delete removes a memory entry by its unique ID.
func (s *GatewayMemoryStore) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	// Delete using memory manager
	return s.manager.Delete(ctx, id)
}
