package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
)

const (
	MaxSnippetLength = 200
	DefaultPageLimit = 50
)

// FileTracker manages file-based memory tracking for the hierarchical memory system.
type FileTracker struct {
	db *sql.DB
}

// NewFileTracker creates a new FileTracker instance.
func NewFileTracker(db *sql.DB) *FileTracker {
	return &FileTracker{
		db: db,
	}
}

// TrackFile stores or updates a file in the memory tracking system.
// Accepts an optional pre-computed embedding.
func (ft *FileTracker) TrackFile(ctx context.Context, path string, tier int, content string, embedding []float32) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if tier < 1 || tier > 3 {
		return fmt.Errorf("tier must be 1 (raw), 2 (curated), or 3 (archive)")
	}

	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	var embeddingBytes []byte
	if len(embedding) > 0 {
		var err error
		embeddingBytes, err = serializeEmbedding(embedding)
		if err != nil {
			return fmt.Errorf("failed to serialize embedding: %w", err)
		}
	}

	themesJSON, _ := json.Marshal([]string{})

	var existingID string
	err := ft.db.QueryRowContext(ctx, "SELECT id FROM memory_files WHERE file_path = ?", path).Scan(&existingID)

	now := time.Now()
	if err == sql.ErrNoRows {
		id := uuid.New().String()
		query := `
			INSERT INTO memory_files (id, file_path, tier, content, embedding, themes, created_at, updated_at, archived_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)
		`
		_, err = ft.db.ExecContext(ctx, query, id, path, tier, content, embeddingBytes, string(themesJSON), now, now)
		if err != nil {
			return fmt.Errorf("failed to insert memory file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	} else {
		query := `
			UPDATE memory_files
			SET tier = ?, content = ?, embedding = ?, themes = ?, updated_at = ?
			WHERE file_path = ?
		`
		_, err = ft.db.ExecContext(ctx, query, tier, content, embeddingBytes, string(themesJSON), now, path)
		if err != nil {
			return fmt.Errorf("failed to update memory file: %w", err)
		}
	}

	return nil
}

// GetFile retrieves a single file by its path.
func (ft *FileTracker) GetFile(ctx context.Context, path string) (*MemoryFile, error) {
	if path == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	query := `
		SELECT id, file_path, tier, content, embedding, themes, created_at, updated_at, archived_at
		FROM memory_files
		WHERE file_path = ?
	`

	var file MemoryFile
	var embeddingBytes []byte
	var themesJSON string
	var archivedAt sql.NullTime

	err := ft.db.QueryRowContext(ctx, query, path).Scan(
		&file.ID,
		&file.FilePath,
		&file.Tier,
		&file.Content,
		&embeddingBytes,
		&themesJSON,
		&file.CreatedAt,
		&file.UpdatedAt,
		&archivedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	if len(embeddingBytes) > 0 {
		embedding, err := deserializeEmbedding(embeddingBytes)
		if err == nil {
			file.Embedding = embedding
		}
	}

	if err := json.Unmarshal([]byte(themesJSON), &file.Themes); err != nil {
		file.Themes = []string{}
	}

	if archivedAt.Valid {
		file.ArchivedAt = &archivedAt.Time
	}

	return &file, nil
}

// ListFiles retrieves files for a specific tier with pagination support.
func (ft *FileTracker) ListFiles(ctx context.Context, tier int, limit int, offset int, includeArchived bool) ([]MemoryFile, error) {
	if tier < 0 || tier > 3 {
		return nil, fmt.Errorf("tier must be 0 (all), 1 (raw), 2 (curated), or 3 (archive)")
	}

	if limit <= 0 {
		limit = DefaultPageLimit
	}

	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, file_path, tier, content, embedding, themes, created_at, updated_at, archived_at
		FROM memory_files
		WHERE 1=1
	`
	var args []interface{}

	if tier > 0 {
		query += " AND tier = ?"
		args = append(args, tier)
	}

	if !includeArchived {
		query += " AND archived_at IS NULL"
	}

	query += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := ft.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []MemoryFile
	for rows.Next() {
		var file MemoryFile
		var embeddingBytes []byte
		var themesJSON string
		var archivedAt sql.NullTime

		err := rows.Scan(
			&file.ID,
			&file.FilePath,
			&file.Tier,
			&file.Content,
			&embeddingBytes,
			&themesJSON,
			&file.CreatedAt,
			&file.UpdatedAt,
			&archivedAt,
		)

		if err != nil {
			slog.Warn("failed to scan file row", "error", err)
			continue
		}

		if len(embeddingBytes) > 0 {
			embedding, err := deserializeEmbedding(embeddingBytes)
			if err == nil {
				file.Embedding = embedding
			}
		}

		if err := json.Unmarshal([]byte(themesJSON), &file.Themes); err != nil {
			file.Themes = []string{}
		}

		if archivedAt.Valid {
			file.ArchivedAt = &archivedAt.Time
		}

		files = append(files, file)
	}

	return files, nil
}

// ArchiveFile marks a file as archived.
func (ft *FileTracker) ArchiveFile(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	now := time.Now()
	query := `UPDATE memory_files SET archived_at = ? WHERE file_path = ?`

	result, err := ft.db.ExecContext(ctx, query, now, path)
	if err != nil {
		return fmt.Errorf("failed to archive file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("file not found: %s", path)
	}

	return nil
}

// SearchFiles performs semantic search across files using provided query embedding.
// Falls back to text search if no embedding is provided.
func (ft *FileTracker) SearchFiles(ctx context.Context, query string, tier int, queryEmbedding []float32) ([]FileSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if len(queryEmbedding) == 0 {
		return ft.searchFilesFallback(ctx, query, tier)
	}

	sqlQuery := `
		SELECT id, file_path, tier, content, embedding, themes, created_at, updated_at, archived_at
		FROM memory_files
		WHERE embedding IS NOT NULL AND archived_at IS NULL
	`

	var args []interface{}
	if tier > 0 {
		sqlQuery += " AND tier = ?"
		args = append(args, tier)
	}

	rows, err := ft.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}
	defer rows.Close()

	var results []FileSearchResult
	for rows.Next() {
		file, err := ft.scanFileRow(rows)
		if err != nil {
			continue
		}

		if len(file.Embedding) > 0 {
			similarity := cosineSimilarity(queryEmbedding, file.Embedding)

			snippet := file.Content
			if len(snippet) > MaxSnippetLength {
				snippet = snippet[:MaxSnippetLength] + "..."
			}

			results = append(results, FileSearchResult{
				File:       *file,
				Similarity: similarity,
				Snippet:    snippet,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return results, nil
}

func (ft *FileTracker) searchFilesFallback(ctx context.Context, query string, tier int) ([]FileSearchResult, error) {
	sqlQuery := `
		SELECT id, file_path, tier, content, embedding, themes, created_at, updated_at, archived_at
		FROM memory_files
		WHERE content LIKE ? AND archived_at IS NULL
	`

	var args []interface{}
	args = append(args, "%"+query+"%")

	if tier > 0 {
		sqlQuery += " AND tier = ?"
		args = append(args, tier)
	}

	sqlQuery += " ORDER BY updated_at DESC"

	rows, err := ft.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}
	defer rows.Close()

	var results []FileSearchResult
	for rows.Next() {
		file, err := ft.scanFileRow(rows)
		if err != nil {
			continue
		}

		snippet := file.Content
		if len(snippet) > MaxSnippetLength {
			snippet = snippet[:MaxSnippetLength] + "..."
		}

		results = append(results, FileSearchResult{
			File:       *file,
			Similarity: 0.0,
			Snippet:    snippet,
		})
	}

	return results, nil
}

func (ft *FileTracker) scanFileRow(row scannable) (*MemoryFile, error) {
	var file MemoryFile
	var embeddingBytes []byte
	var themesJSON string
	var archivedAt sql.NullTime

	err := row.Scan(
		&file.ID,
		&file.FilePath,
		&file.Tier,
		&file.Content,
		&embeddingBytes,
		&themesJSON,
		&file.CreatedAt,
		&file.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(embeddingBytes) > 0 {
		embedding, err := deserializeEmbedding(embeddingBytes)
		if err == nil {
			file.Embedding = embedding
		}
	}

	if err := json.Unmarshal([]byte(themesJSON), &file.Themes); err != nil {
		file.Themes = []string{}
	}

	if archivedAt.Valid {
		file.ArchivedAt = &archivedAt.Time
	}

	return &file, nil
}
