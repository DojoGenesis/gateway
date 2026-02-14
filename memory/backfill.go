package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// EmbeddingGeneratorInterface abstracts embedding generation for backfill.
type EmbeddingGeneratorInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// BackfillService manages embedding backfill for memories without embeddings.
type BackfillService struct {
	manager   *MemoryManager
	generator EmbeddingGeneratorInterface
}

// BackfillStatus reports the current state of embedding coverage.
type BackfillStatus struct {
	TotalMemories         int     `json:"total_memories"`
	MemoriesWithEmbedding int     `json:"memories_with_embedding"`
	ProgressPercent       float64 `json:"progress_percent"`
}

// BackfillResult reports the outcome of a backfill operation.
type BackfillResult struct {
	ProcessedCount int      `json:"processed_count"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors,omitempty"`
	Duration       string   `json:"duration"`
}

// NewBackfillService creates a new BackfillService.
// generator may be nil; ProcessBackfill will return an error if called without one.
func NewBackfillService(manager *MemoryManager, generator EmbeddingGeneratorInterface) *BackfillService {
	return &BackfillService{
		manager:   manager,
		generator: generator,
	}
}

// GetBackfillStatus returns the current embedding coverage status.
func (bs *BackfillService) GetBackfillStatus(ctx context.Context) (*BackfillStatus, error) {
	if bs.manager == nil || bs.manager.db == nil {
		return nil, fmt.Errorf("memory manager not initialized")
	}

	var totalCount int
	err := bs.manager.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count total memories: %w", err)
	}

	var withEmbedding int
	err = bs.manager.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM memories WHERE embedding IS NOT NULL AND embedding != ''").Scan(&withEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to count memories with embeddings: %w", err)
	}

	progress := 0.0
	if totalCount > 0 {
		progress = (float64(withEmbedding) / float64(totalCount)) * 100
	}

	return &BackfillStatus{
		TotalMemories:         totalCount,
		MemoriesWithEmbedding: withEmbedding,
		ProgressPercent:       progress,
	}, nil
}

// ProcessBackfill generates embeddings for memories that don't have them.
func (bs *BackfillService) ProcessBackfill(ctx context.Context, batchSize int, dryRun bool, verbose bool) (*BackfillResult, error) {
	startTime := time.Now()
	result := &BackfillResult{
		Errors: []string{},
	}

	if bs.manager == nil || bs.manager.db == nil {
		return nil, fmt.Errorf("memory manager not initialized")
	}

	if bs.generator == nil {
		return nil, fmt.Errorf("embedding generator not configured")
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	query := `
		SELECT id, type, content, metadata, context_type, created_at, updated_at
		FROM memories
		WHERE embedding IS NULL OR embedding = ''
		LIMIT ?
	`

	rows, err := bs.manager.db.QueryContext(ctx, query, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	memories := []Memory{}
	for rows.Next() {
		var memory Memory
		var metadataJSON string
		var createdAt, updatedAt string

		err := rows.Scan(
			&memory.ID,
			&memory.Type,
			&memory.Content,
			&metadataJSON,
			&memory.ContextType,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("failed to scan memory: %v", err))
			continue
		}

		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &memory.Metadata); err != nil {
				memory.Metadata = make(map[string]interface{})
			}
		} else {
			memory.Metadata = make(map[string]interface{})
		}

		memory.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		memory.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		memories = append(memories, memory)
		result.ProcessedCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if len(memories) == 0 {
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	if verbose {
		slog.Info("found memories without embeddings", "count", len(memories))
	}

	if dryRun {
		if verbose {
			slog.Info("dry run mode - skipping embedding generation")
		}
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	for i, memory := range memories {
		if verbose && i > 0 && i%10 == 0 {
			slog.Info("backfill progress", "processed", i, "total", len(memories))
		}

		embeddingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		embedding, err := bs.generator.GenerateEmbedding(embeddingCtx, memory.Content)
		cancel()

		if err != nil {
			result.FailedCount++
			errMsg := fmt.Sprintf("failed to generate embedding for %s: %v", memory.ID, err)
			result.Errors = append(result.Errors, errMsg)
			if verbose {
				slog.Error("failed to generate embedding", "memory_id", memory.ID, "error", err)
			}
			continue
		}

		embeddingBytes, err := serializeEmbedding(embedding)
		if err != nil {
			result.FailedCount++
			errMsg := fmt.Sprintf("failed to serialize embedding for %s: %v", memory.ID, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		updateQuery := `UPDATE memories SET embedding = ?, updated_at = ? WHERE id = ?`
		_, err = bs.manager.db.ExecContext(ctx, updateQuery, embeddingBytes, time.Now(), memory.ID)
		if err != nil {
			result.FailedCount++
			errMsg := fmt.Sprintf("failed to update embedding for %s: %v", memory.ID, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		result.SuccessCount++

		time.Sleep(10 * time.Millisecond)
	}

	result.Duration = time.Since(startTime).String()

	if verbose {
		slog.Info("backfill complete", "processed", result.ProcessedCount, "succeeded", result.SuccessCount, "failed", result.FailedCount, "duration", result.Duration)
	}

	return result, nil
}
