package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

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
		if err := os.MkdirAll(dir, 0o750); err != nil {
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

	if errors.Is(err, sql.ErrNoRows) {
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

// SearchMemories performs a case-insensitive, token-based relevance search over
// memory content and returns up to limit results, best match first.
//
// This intentionally does NOT require the query to appear as one contiguous
// substring of the stored content -- it is instead a bag-of-tokens match (see
// rankedSearch), which is what lets a natural-language query like "IP strategy
// trade secret locked default" match content where those words appear scattered
// rather than in that exact order. An exact substring match of the whole query is
// still recognized and ranked first, so single-word/phrase lookups like "trade
// secret" keep behaving exactly as before.
//
// It does not do semantic/embedding matching: a query like "intellectual property"
// will not match content that only ever spells it "IP" -- there is no embedding
// infrastructure in this codebase, and adding one is out of scope for this fix.
func (m *MemoryManager) SearchMemories(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	results, _, err := m.SearchMemoriesPage(ctx, query, limit, 0)
	return results, err
}

// SearchMemoriesPage is SearchMemories with limit/offset paging, additionally
// returning the true number of matches *before* paging was applied. Exposing that
// total is what lets an HTTP caller tell "you got everything" apart from "you got
// a page of a larger result set" -- the same class of gap fixed for plain listing
// by CountMemories.
func (m *MemoryManager) SearchMemoriesPage(ctx context.Context, query string, limit, offset int) ([]SearchResult, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	trimmed := strings.TrimSpace(query)

	var all []SearchResult
	var err error
	if trimmed == "" {
		all, err = m.listAllAsResults(ctx, 0) // 0 => no limit; we page below.
	} else {
		all, err = m.rankedSearch(ctx, trimmed)
	}
	if err != nil {
		return nil, 0, err
	}

	total := len(all)
	if offset >= total {
		return []SearchResult{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// tokenExactPhraseBonus is added to a memory's score when the whole (trimmed,
// lowercased) query appears in its content as a contiguous substring. It is large
// enough to outrank any achievable sum of per-token weights, so an exact phrase
// match always sorts first -- preserving the historical contiguous-substring
// behavior as the top hit rather than replacing it.
const tokenExactPhraseBonus = 1000.0

// rankedSearch is the core of the token-based search. It returns every memory
// whose content contains at least one token of trimmedQuery (case-insensitive
// substring match per token), ranked best-first, with no limit/offset applied --
// callers do their own paging.
//
// Ranking:
//  1. An exact case-insensitive substring match of the whole query gets
//     tokenExactPhraseBonus added on top, so it always sorts first when present.
//  2. Otherwise, memories are scored by summed token weight: a token that appears
//     in few stored memories (rare/specific) is weighted higher than one that
//     appears in nearly all of them (common), an IDF-like weighting, plus a small
//     length bonus for longer tokens. A coverage term additionally rewards
//     matching more of the distinct query tokens, so "matches all 6 tokens"
//     outranks "matches 1 of 6" even when the matched token itself is common.
//  3. Ties fall back to updated_at DESC, matching the original ORDER BY.
func (m *MemoryManager) rankedSearch(ctx context.Context, trimmedQuery string) ([]SearchResult, error) {
	tokens := tokenizeQuery(trimmedQuery)
	if len(tokens) == 0 {
		return []SearchResult{}, nil
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at
		FROM memories
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var all []*Memory
	for rows.Next() {
		mem, err := m.scanMemory(rows)
		if err != nil {
			slog.Warn("failed to scan memory row", "error", err)
			continue
		}
		all = append(all, mem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	if len(all) == 0 {
		return []SearchResult{}, nil
	}

	lowerQuery := strings.ToLower(trimmedQuery)
	lowerContents := make([]string, len(all))
	docFreq := make(map[string]int, len(tokens))
	for i, mem := range all {
		lc := strings.ToLower(mem.Content)
		lowerContents[i] = lc
		for _, tok := range tokens {
			if strings.Contains(lc, tok) {
				docFreq[tok]++
			}
		}
	}

	type scored struct {
		mem   *Memory
		score float64
	}

	matches := make([]scored, 0, len(all))
	for i, mem := range all {
		lc := lowerContents[i]

		matchedTokens := 0
		var weightSum float64
		for _, tok := range tokens {
			if !strings.Contains(lc, tok) {
				continue
			}
			matchedTokens++
			// IDF-like weight: rarer tokens across the store count for more. A
			// small per-character bonus additionally favors longer, more
			// specific tokens over short ones with equal document frequency.
			idf := math.Log(float64(len(all))/float64(docFreq[tok]) + 1)
			weightSum += idf + float64(len(tok))*0.01
		}
		if matchedTokens == 0 {
			continue
		}

		score := weightSum + float64(matchedTokens)/float64(len(tokens))
		if strings.Contains(lc, lowerQuery) {
			score += tokenExactPhraseBonus
		}

		matches = append(matches, scored{mem: mem, score: score})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].mem.UpdatedAt.After(matches[j].mem.UpdatedAt)
	})

	var maxScore float64
	for _, sm := range matches {
		if sm.score > maxScore {
			maxScore = sm.score
		}
	}

	results := make([]SearchResult, 0, len(matches))
	for _, sm := range matches {
		snippet := sm.mem.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		similarity := 0.0
		if maxScore > 0 {
			similarity = sm.score / maxScore
		}
		results = append(results, SearchResult{
			Memory:     *sm.mem,
			Similarity: similarity,
			Snippet:    snippet,
			SearchMode: "text",
		})
	}

	return results, nil
}

// tokenizeQuery splits a query into lowercase word tokens (maximal runs of Unicode
// letters/digits), dropping punctuation and whitespace as delimiters and
// de-duplicating repeats so a repeated word isn't double-counted when scoring
// "how many distinct tokens matched".
func tokenizeQuery(query string) []string {
	var tokens []string
	seen := make(map[string]bool)
	var cur strings.Builder

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		tok := cur.String()
		cur.Reset()
		if !seen[tok] {
			seen[tok] = true
			tokens = append(tokens, tok)
		}
	}

	for _, r := range query {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()

	return tokens
}

// buildMemoryFilterWhere builds the WHERE clause (and its bound args) shared by
// ListMemories and CountMemories. Sharing this builder means the two queries
// cannot drift out of sync with each other -- a mismatch there is exactly how a
// "total" field silently stops meaning "total" (see CountMemories).
func buildMemoryFilterWhere(filter MemoryFilter) (string, []interface{}) {
	clause := " WHERE 1=1"
	var args []interface{}

	if filter.Type != "" {
		clause += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.ContextType != "" {
		clause += " AND context_type = ?"
		args = append(args, filter.ContextType)
	}

	if filter.CreatedAfter != nil {
		clause += " AND created_at > ?"
		args = append(args, *filter.CreatedAfter)
	}

	return clause, args
}

// ListMemories lists memories matching the given filter.
func (m *MemoryManager) ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error) {
	where, args := buildMemoryFilterWhere(filter)
	query := `SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at FROM memories` + where + ` ORDER BY updated_at DESC`

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += " LIMIT ?"
	args = append(args, limit)

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

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

// CountMemories returns the total number of memories matching filter, ignoring
// filter.Limit and filter.Offset. This is the "true total" that ListMemories's
// LIMIT/OFFSET otherwise hides from callers -- GET /v1/memory reported
// TotalCount as len(page) instead of this, so a caller reading the default
// (limit=20) page had no way to tell 20-of-20 apart from 20-of-34.
func (m *MemoryManager) CountMemories(ctx context.Context, filter MemoryFilter) (int, error) {
	where, args := buildMemoryFilterWhere(filter)
	query := `SELECT COUNT(*) FROM memories` + where

	var count int
	if err := m.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}
	return count, nil
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

// listAllAsResults returns every memory as a SearchResult, ordered by
// updated_at DESC. limit <= 0 means "no limit" (return everything) -- callers that
// want a bounded page (e.g. SearchMemories's empty-query case) must pass a
// positive limit explicitly.
func (m *MemoryManager) listAllAsResults(ctx context.Context, limit int) ([]SearchResult, error) {
	sqlQuery := `
		SELECT id, type, content, metadata, embedding, context_type, created_at, updated_at
		FROM memories
		ORDER BY updated_at DESC
	`
	var args []interface{}
	if limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(ctx, sqlQuery, args...)
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
