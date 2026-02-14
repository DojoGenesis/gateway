package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// MemoryManager provides CRUD operations for conversational memories.
type MemoryManager struct {
	db *sql.DB
}

// NewMemoryManager creates a new MemoryManager backed by SQLite at the given path.
func NewMemoryManager(dbPath string) (*MemoryManager, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// SQLite performance tuning
	pragmas := []string{
		"PRAGMA journal_mode=WAL",   // Better concurrent read/write performance
		"PRAGMA synchronous=NORMAL", // Safe with WAL, faster than FULL
		"PRAGMA cache_size=-64000",  // 64MB cache (negative = KiB)
		"PRAGMA foreign_keys=ON",    // Enforce referential integrity
		"PRAGMA busy_timeout=5000",  // Wait 5s on lock instead of failing immediately
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set %s: %w", pragma, err)
		}
	}

	// Connection pool tuning for SQLite (single-writer, many readers)
	db.SetMaxOpenConns(1)    // SQLite only supports one writer
	db.SetMaxIdleConns(1)    // Keep one idle connection warm
	db.SetConnMaxLifetime(0) // Don't expire connections

	mm := &MemoryManager{db: db}
	if err := mm.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return mm, nil
}

func (m *MemoryManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		embedding BLOB,
		context_type TEXT DEFAULT 'conversation',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
	CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
	CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at);
	CREATE INDEX IF NOT EXISTS idx_memories_context_type ON memories(context_type);
	`

	_, err := m.db.Exec(schema)
	return err
}

// StoreMemory stores a new memory or updates an existing one (upsert).
func (m *MemoryManager) StoreMemory(ctx context.Context, memory *Memory) error {
	if memory == nil {
		return fmt.Errorf("memory cannot be nil")
	}
	if memory.ID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}
	if memory.Type == "" {
		return fmt.Errorf("memory type cannot be empty")
	}

	if memory.ContextType == "" {
		memory.ContextType = "conversation"
	}

	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var embeddingBytes []byte
	if len(memory.Embedding) > 0 {
		embeddingBytes, _ = serializeEmbedding(memory.Embedding)
	}

	query := `
		INSERT INTO memories (id, type, content, metadata, embedding, context_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			content = excluded.content,
			metadata = excluded.metadata,
			embedding = excluded.embedding,
			context_type = excluded.context_type,
			updated_at = excluded.updated_at
	`

	_, err = m.db.ExecContext(ctx, query,
		memory.ID,
		memory.Type,
		memory.Content,
		string(metadataJSON),
		embeddingBytes,
		memory.ContextType,
		memory.CreatedAt,
		memory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store memory: %w", err)
	}

	return nil
}

// GetMemory retrieves a memory by ID.
func (m *MemoryManager) GetMemory(ctx context.Context, id string) (*Memory, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	query := `SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at FROM memories WHERE id = ?`

	var memory Memory
	var metadataJSON string
	var embeddingBytes []byte

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&memory.ID,
		&memory.Type,
		&memory.Content,
		&metadataJSON,
		&embeddingBytes,
		&memory.ContextType,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMemoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memory: %w", err)
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &memory.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	if len(embeddingBytes) > 0 {
		embedding, err := deserializeEmbedding(embeddingBytes)
		if err == nil {
			memory.Embedding = embedding
		}
	}

	return &memory, nil
}

// UpdateMemory updates an existing memory's content and metadata.
func (m *MemoryManager) UpdateMemory(ctx context.Context, memory *Memory) error {
	if memory == nil {
		return fmt.Errorf("memory cannot be nil")
	}
	if memory.ID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	memory.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var embeddingBytes []byte
	if len(memory.Embedding) > 0 {
		embeddingBytes, _ = serializeEmbedding(memory.Embedding)
	}

	query := `UPDATE memories SET type = ?, content = ?, metadata = ?, embedding = ?, context_type = ?, updated_at = ? WHERE id = ?`
	result, err := m.db.ExecContext(ctx, query,
		memory.Type,
		memory.Content,
		string(metadataJSON),
		embeddingBytes,
		memory.ContextType,
		memory.UpdatedAt,
		memory.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update memory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrMemoryNotFound
	}

	return nil
}

// DeleteMemory deletes a memory by ID.
func (m *MemoryManager) DeleteMemory(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	query := `DELETE FROM memories WHERE id = ?`
	result, err := m.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrMemoryNotFound
	}

	return nil
}

// SearchMemories searches memories by content (text-based).
func (m *MemoryManager) SearchMemories(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	if query == "" {
		return m.listAllAsResults(ctx, limit)
	}

	sqlQuery := `
		SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at
		FROM memories
		WHERE content LIKE ?
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := m.db.QueryContext(ctx, sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		memory, err := m.scanMemory(rows)
		if err != nil {
			slog.Warn("failed to scan memory row", "error", err)
			continue
		}

		snippet := memory.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results = append(results, SearchResult{
			Memory:     *memory,
			Similarity: 0.0,
			Snippet:    snippet,
			SearchMode: "text",
		})
	}

	return results, nil
}

// ListMemories lists memories matching the given filter.
func (m *MemoryManager) ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error) {
	query := `SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at FROM memories WHERE 1=1`
	var args []interface{}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.ContextType != "" {
		query += " AND context_type = ?"
		args = append(args, filter.ContextType)
	}

	if filter.CreatedAfter != nil {
		query += " AND created_at > ?"
		args = append(args, *filter.CreatedAfter)
	}

	query += " ORDER BY updated_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		memory, err := m.scanMemory(rows)
		if err != nil {
			slog.Warn("failed to scan memory row", "error", err)
			continue
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// SearchMemoriesSemantic searches memories using embedding cosine similarity.
func (m *MemoryManager) SearchMemoriesSemantic(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("query embedding cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	sqlQuery := `
		SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at
		FROM memories
		WHERE embedding IS NOT NULL
	`

	rows, err := m.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		memory, err := m.scanMemory(rows)
		if err != nil {
			continue
		}

		if len(memory.Embedding) > 0 {
			similarity := cosineSimilarity(queryEmbedding, memory.Embedding)

			snippet := memory.Content
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}

			results = append(results, SearchResult{
				Memory:     *memory,
				Similarity: similarity,
				Snippet:    snippet,
				SearchMode: "semantic",
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (m *MemoryManager) listAllAsResults(ctx context.Context, limit int) ([]SearchResult, error) {
	sqlQuery := `
		SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at
		FROM memories
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := m.db.QueryContext(ctx, sqlQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		memory, err := m.scanMemory(rows)
		if err != nil {
			continue
		}

		snippet := memory.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results = append(results, SearchResult{
			Memory:     *memory,
			Similarity: 0.0,
			Snippet:    snippet,
			SearchMode: "text",
		})
	}

	return results, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func (m *MemoryManager) scanMemory(row scannable) (*Memory, error) {
	var memory Memory
	var metadataJSON string
	var embeddingBytes []byte

	err := row.Scan(
		&memory.ID,
		&memory.Type,
		&memory.Content,
		&metadataJSON,
		&embeddingBytes,
		&memory.ContextType,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &memory.Metadata); err != nil {
			memory.Metadata = make(map[string]interface{})
		}
	}

	if len(embeddingBytes) > 0 {
		embedding, err := deserializeEmbedding(embeddingBytes)
		if err == nil {
			memory.Embedding = embedding
		}
	}

	return &memory, nil
}

// DB returns the underlying database connection.
func (m *MemoryManager) DB() *sql.DB {
	return m.db
}

// Close closes the database connection.
func (m *MemoryManager) Close() error {
	return m.db.Close()
}

// EstimateTokens provides a fast approximation of token count.
func EstimateTokens(text string) int {
	words := len(strings.Fields(text))
	chars := len(text)

	tokensFromWords := float64(words) * 1.3
	tokensFromChars := float64(chars) / 4.0

	estimate := (tokensFromWords + tokensFromChars) / 2.0
	return int(estimate)
}
