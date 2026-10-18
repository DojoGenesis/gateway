package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// AccessObserver is told about each memory operation after it finishes, for
// audit. meta carries IDs, counts and hashes, never memory content or query
// text. Its signature matches audit.Recorder.MemoryAccess.
type AccessObserver func(ctx context.Context, op string, meta map[string]interface{}, err error, d time.Duration)

// SetAccessObserver installs o for this manager; nil removes it.
func (m *MemoryManager) SetAccessObserver(o AccessObserver) {
	if o == nil {
		m.observer.Store(nil)
		return
	}
	m.observer.Store(&o)
}

func (m *MemoryManager) observe(ctx context.Context, op string, meta map[string]interface{}, err error, start time.Time) {
	if o := m.observer.Load(); o != nil {
		(*o)(ctx, op, meta, err, time.Since(start))
	}
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// StoreMemory stores a new memory or updates an existing one (upsert).
func (m *MemoryManager) StoreMemory(ctx context.Context, memory *Memory) error {
	start := time.Now()
	err := m.storeMemory(ctx, memory)
	meta := map[string]interface{}{}
	if memory != nil {
		meta["memory_id"] = memory.ID
		meta["type"] = memory.Type
		meta["content_sha256"] = hashText(memory.Content)
	}
	m.observe(ctx, "store", meta, err, start)
	return err
}

// GetMemory retrieves a memory by ID.
func (m *MemoryManager) GetMemory(ctx context.Context, id string) (*Memory, error) {
	start := time.Now()
	mem, err := m.getMemory(ctx, id)
	m.observe(ctx, "get", map[string]interface{}{"memory_id": id}, err, start)
	return mem, err
}

// UpdateMemory updates an existing memory.
func (m *MemoryManager) UpdateMemory(ctx context.Context, memory *Memory) error {
	start := time.Now()
	err := m.updateMemory(ctx, memory)
	meta := map[string]interface{}{}
	if memory != nil {
		meta["memory_id"] = memory.ID
		meta["content_sha256"] = hashText(memory.Content)
	}
	m.observe(ctx, "update", meta, err, start)
	return err
}

// DeleteMemory deletes a memory by ID.
func (m *MemoryManager) DeleteMemory(ctx context.Context, id string) error {
	start := time.Now()
	err := m.deleteMemory(ctx, id)
	m.observe(ctx, "delete", map[string]interface{}{"memory_id": id}, err, start)
	return err
}

// SearchMemoriesPage is SearchMemories with limit/offset paging; see
// searchMemoriesPage.
func (m *MemoryManager) SearchMemoriesPage(ctx context.Context, query string, limit, offset int) ([]SearchResult, int, error) {
	start := time.Now()
	results, total, err := m.searchMemoriesPage(ctx, query, limit, offset)
	m.observe(ctx, "search", map[string]interface{}{
		"query_sha256": hashText(query),
		"returned":     len(results),
		"total":        total,
	}, err, start)
	return results, total, err
}

// ListMemories lists memories matching the given filter.
func (m *MemoryManager) ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error) {
	start := time.Now()
	mems, err := m.listMemories(ctx, filter)
	m.observe(ctx, "list", map[string]interface{}{
		"type":         filter.Type,
		"context_type": filter.ContextType,
		"returned":     len(mems),
	}, err, start)
	return mems, err
}

// SearchMemoriesSemantic searches memories using embedding cosine similarity.
func (m *MemoryManager) SearchMemoriesSemantic(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	start := time.Now()
	results, err := m.searchMemoriesSemantic(ctx, queryEmbedding, limit)
	m.observe(ctx, "search_semantic", map[string]interface{}{"returned": len(results)}, err, start)
	return results, err
}
