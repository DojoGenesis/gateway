package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GardenManager provides garden-based memory management with compression and seed extraction.
// It delegates LLM-dependent operations to CompressionServiceInterface.
type GardenManager struct {
	memManager         *MemoryManager
	compressionService CompressionServiceInterface
	db                 *sql.DB
}

// Seed represents a reusable knowledge pattern in the garden.
type Seed struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Trigger     string     `json:"trigger"`
	Content     string     `json:"content"`
	UsageCount  int        `json:"usage_count"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MemorySnapshot represents a point-in-time snapshot of memory state.
type MemorySnapshot struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	SnapshotName string                 `json:"snapshot_name"`
	SnapshotData map[string]interface{} `json:"snapshot_data"`
	CreatedAt    time.Time              `json:"created_at"`
}

// NewGardenManager creates a new GardenManager.
// compressionService may be nil; compression and seed extraction will use simple fallbacks.
func NewGardenManager(memManager *MemoryManager, compressionService CompressionServiceInterface) (*GardenManager, error) {
	gm := &GardenManager{
		memManager:         memManager,
		compressionService: compressionService,
		db:                 memManager.db,
	}

	if err := gm.initGardenSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize garden schema: %w", err)
	}

	return gm, nil
}

func (gm *GardenManager) initGardenSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS seeds (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		trigger_text TEXT,
		content TEXT NOT NULL,
		usage_count INTEGER DEFAULT 0,
		last_used DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_seeds_name ON seeds(name);
	CREATE INDEX IF NOT EXISTS idx_seeds_trigger ON seeds(trigger_text);
	CREATE INDEX IF NOT EXISTS idx_seeds_usage_count ON seeds(usage_count DESC);
	CREATE INDEX IF NOT EXISTS idx_seeds_last_used ON seeds(last_used DESC);

	CREATE TABLE IF NOT EXISTS compressed_history (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		original_turn_ids TEXT NOT NULL,
		compressed_content TEXT NOT NULL,
		compression_ratio REAL,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_compressed_history_session ON compressed_history(session_id);
	CREATE INDEX IF NOT EXISTS idx_compressed_history_created_at ON compressed_history(created_at);

	CREATE TABLE IF NOT EXISTS memory_snapshots (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		snapshot_name TEXT,
		snapshot_data TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_memory_snapshots_session ON memory_snapshots(session_id);
	CREATE INDEX IF NOT EXISTS idx_memory_snapshots_created_at ON memory_snapshots(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_memory_snapshots_name ON memory_snapshots(snapshot_name);
	`

	_, err := gm.db.Exec(schema)
	return err
}

// CompressHistory compresses conversation memories using the configured compression service.
// Falls back to a simple concatenation-based compression if no service is configured.
func (gm *GardenManager) CompressHistory(ctx context.Context, sessionID string, memories []Memory) (*CompressedHistory, error) {
	if len(memories) == 0 {
		return nil, fmt.Errorf("no memories to compress")
	}

	if gm.compressionService != nil {
		return gm.compressionService.CompressHistory(ctx, sessionID, memories)
	}

	return gm.simpleCompress(sessionID, memories)
}

// simpleCompress provides a fallback compression when no LLM service is available.
func (gm *GardenManager) simpleCompress(sessionID string, memories []Memory) (*CompressedHistory, error) {
	var builder strings.Builder
	for _, mem := range memories {
		builder.WriteString(fmt.Sprintf("[%s] %s\n", mem.Type, mem.Content))
	}

	originalContent := builder.String()
	compressed := originalContent
	if len(compressed) > 500 {
		compressed = compressed[:500] + "..."
	}

	ratio := float64(len(compressed)) / float64(len(originalContent))
	if ratio > 1.0 {
		ratio = 1.0
	}

	turnIDs := make([]string, len(memories))
	for i, mem := range memories {
		turnIDs[i] = mem.ID
	}

	return &CompressedHistory{
		ID:                uuid.New().String(),
		SessionID:         sessionID,
		OriginalTurnIDs:   turnIDs,
		CompressedContent: compressed,
		CompressionRatio:  ratio,
		CreatedAt:         time.Now(),
	}, nil
}

// ExtractSeeds extracts reusable patterns from memories using the compression service.
// Falls back to returning empty results if no service is configured.
func (gm *GardenManager) ExtractSeeds(ctx context.Context, memories []Memory) ([]*MemorySeed, error) {
	if gm.compressionService != nil {
		return gm.compressionService.ExtractSeeds(ctx, memories)
	}
	return []*MemorySeed{}, nil
}

// StoreSeed stores or updates a seed in the database.
func (gm *GardenManager) StoreSeed(ctx context.Context, seed *Seed) error {
	if seed.Name == "" {
		return fmt.Errorf("seed name cannot be empty")
	}
	if seed.Content == "" {
		return fmt.Errorf("seed content cannot be empty")
	}

	if seed.ID == "" {
		seed.ID = uuid.New().String()
	}

	query := `
		INSERT INTO seeds (id, name, description, trigger_text, content, usage_count, last_used, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			trigger_text = excluded.trigger_text,
			content = excluded.content,
			usage_count = excluded.usage_count,
			last_used = excluded.last_used,
			updated_at = excluded.updated_at
	`

	_, err := gm.db.ExecContext(ctx, query,
		seed.ID,
		seed.Name,
		seed.Description,
		seed.Trigger,
		seed.Content,
		seed.UsageCount,
		seed.LastUsed,
		seed.CreatedAt,
		seed.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store seed: %w", err)
	}

	return nil
}

// RetrieveSeed retrieves a seed by ID.
func (gm *GardenManager) RetrieveSeed(ctx context.Context, id string) (*Seed, error) {
	query := `SELECT id, name, description, trigger_text, content, usage_count, last_used, created_at, updated_at FROM seeds WHERE id = ?`

	var seed Seed
	var lastUsed sql.NullTime

	err := gm.db.QueryRowContext(ctx, query, id).Scan(
		&seed.ID,
		&seed.Name,
		&seed.Description,
		&seed.Trigger,
		&seed.Content,
		&seed.UsageCount,
		&lastUsed,
		&seed.CreatedAt,
		&seed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSeedNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve seed: %w", err)
	}

	if lastUsed.Valid {
		seed.LastUsed = &lastUsed.Time
	}

	return &seed, nil
}

// StoreCompressedHistory stores a compressed conversation history.
func (gm *GardenManager) StoreCompressedHistory(ctx context.Context, history *CompressedHistory) error {
	if history.SessionID == "" {
		return fmt.Errorf("session_id cannot be empty")
	}
	if history.CompressionRatio < 0 || history.CompressionRatio > 1 {
		return fmt.Errorf("compression_ratio must be between 0 and 1, got %f", history.CompressionRatio)
	}

	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	originalTurnIDsJSON, err := json.Marshal(history.OriginalTurnIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal original_turn_ids: %w", err)
	}

	query := `
		INSERT INTO compressed_history (id, session_id, original_turn_ids, compressed_content, compression_ratio, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = gm.db.ExecContext(ctx, query,
		history.ID,
		history.SessionID,
		string(originalTurnIDsJSON),
		history.CompressedContent,
		history.CompressionRatio,
		history.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store compressed history: %w", err)
	}

	return nil
}

// RetrieveCompressedHistory retrieves compressed histories for a session.
func (gm *GardenManager) RetrieveCompressedHistory(ctx context.Context, sessionID string) ([]CompressedHistory, error) {
	query := `
		SELECT id, session_id, original_turn_ids, compressed_content, compression_ratio, created_at
		FROM compressed_history
		WHERE session_id = ?
		ORDER BY created_at DESC
	`

	rows, err := gm.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve compressed history: %w", err)
	}
	defer rows.Close()

	histories := []CompressedHistory{}
	for rows.Next() {
		var history CompressedHistory
		var originalTurnIDsJSON string

		err := rows.Scan(
			&history.ID,
			&history.SessionID,
			&originalTurnIDsJSON,
			&history.CompressedContent,
			&history.CompressionRatio,
			&history.CreatedAt,
		)

		if err != nil {
			log.Printf("warning: failed to scan compressed history row: %v", err)
			continue
		}

		if err := json.Unmarshal([]byte(originalTurnIDsJSON), &history.OriginalTurnIDs); err != nil {
			log.Printf("warning: failed to unmarshal original_turn_ids for history %s: %v", history.ID, err)
			continue
		}

		histories = append(histories, history)
	}

	return histories, nil
}

// StoreSnapshot stores a memory snapshot.
func (gm *GardenManager) StoreSnapshot(ctx context.Context, snapshot *MemorySnapshot) error {
	if snapshot.SessionID == "" {
		return fmt.Errorf("session_id cannot be empty")
	}

	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}

	snapshotDataJSON, err := json.Marshal(snapshot.SnapshotData)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot_data: %w", err)
	}

	query := `
		INSERT INTO memory_snapshots (id, session_id, snapshot_name, snapshot_data, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = gm.db.ExecContext(ctx, query,
		snapshot.ID,
		snapshot.SessionID,
		snapshot.SnapshotName,
		string(snapshotDataJSON),
		snapshot.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store snapshot: %w", err)
	}

	return nil
}

// RetrieveSnapshot retrieves a snapshot by ID.
func (gm *GardenManager) RetrieveSnapshot(ctx context.Context, id string) (*MemorySnapshot, error) {
	query := `SELECT id, session_id, snapshot_name, snapshot_data, created_at FROM memory_snapshots WHERE id = ?`

	var snapshot MemorySnapshot
	var snapshotDataJSON string

	err := gm.db.QueryRowContext(ctx, query, id).Scan(
		&snapshot.ID,
		&snapshot.SessionID,
		&snapshot.SnapshotName,
		&snapshotDataJSON,
		&snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	if err := json.Unmarshal([]byte(snapshotDataJSON), &snapshot.SnapshotData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot_data: %w", err)
	}

	return &snapshot, nil
}

// ListSnapshots lists snapshots for a session.
func (gm *GardenManager) ListSnapshots(ctx context.Context, sessionID string) ([]MemorySnapshot, error) {
	query := `
		SELECT id, session_id, snapshot_name, snapshot_data, created_at
		FROM memory_snapshots
		WHERE session_id = ?
		ORDER BY created_at DESC
	`

	rows, err := gm.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := []MemorySnapshot{}
	for rows.Next() {
		var snapshot MemorySnapshot
		var snapshotDataJSON string

		err := rows.Scan(
			&snapshot.ID,
			&snapshot.SessionID,
			&snapshot.SnapshotName,
			&snapshotDataJSON,
			&snapshot.CreatedAt,
		)

		if err != nil {
			log.Printf("warning: failed to scan snapshot row: %v", err)
			continue
		}

		if err := json.Unmarshal([]byte(snapshotDataJSON), &snapshot.SnapshotData); err != nil {
			log.Printf("warning: failed to unmarshal snapshot_data for snapshot %s: %v", snapshot.ID, err)
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// ListSeeds lists seeds ordered by usage count.
func (gm *GardenManager) ListSeeds(ctx context.Context, limit int) ([]Seed, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, name, description, trigger_text, content, usage_count, last_used, created_at, updated_at
		FROM seeds
		ORDER BY usage_count DESC, last_used DESC
		LIMIT ?
	`

	rows, err := gm.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list seeds: %w", err)
	}
	defer rows.Close()

	seeds := []Seed{}
	for rows.Next() {
		var seed Seed
		var lastUsed sql.NullTime

		err := rows.Scan(
			&seed.ID,
			&seed.Name,
			&seed.Description,
			&seed.Trigger,
			&seed.Content,
			&seed.UsageCount,
			&lastUsed,
			&seed.CreatedAt,
			&seed.UpdatedAt,
		)

		if err != nil {
			log.Printf("warning: failed to scan seed row in ListSeeds: %v", err)
			continue
		}

		if lastUsed.Valid {
			seed.LastUsed = &lastUsed.Time
		}

		seeds = append(seeds, seed)
	}

	return seeds, nil
}

// SearchSeedsByTrigger searches seeds by trigger text.
func (gm *GardenManager) SearchSeedsByTrigger(ctx context.Context, trigger string, limit int) ([]Seed, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, name, description, trigger_text, content, usage_count, last_used, created_at, updated_at
		FROM seeds
		WHERE trigger_text LIKE ?
		ORDER BY usage_count DESC
		LIMIT ?
	`

	rows, err := gm.db.QueryContext(ctx, query, "%"+trigger+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search seeds: %w", err)
	}
	defer rows.Close()

	seeds := []Seed{}
	for rows.Next() {
		var seed Seed
		var lastUsed sql.NullTime

		err := rows.Scan(
			&seed.ID,
			&seed.Name,
			&seed.Description,
			&seed.Trigger,
			&seed.Content,
			&seed.UsageCount,
			&lastUsed,
			&seed.CreatedAt,
			&seed.UpdatedAt,
		)

		if err != nil {
			log.Printf("warning: failed to scan seed row: %v", err)
			continue
		}

		if lastUsed.Valid {
			seed.LastUsed = &lastUsed.Time
		}

		seeds = append(seeds, seed)
	}

	return seeds, nil
}

// DeleteSeed deletes a seed by ID.
func (gm *GardenManager) DeleteSeed(ctx context.Context, seedID string) error {
	query := `DELETE FROM seeds WHERE id = ?`

	result, err := gm.db.ExecContext(ctx, query, seedID)
	if err != nil {
		return fmt.Errorf("failed to delete seed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSeedNotFound
	}

	return nil
}

// DeleteCompressedHistory deletes a compressed history entry by ID.
func (gm *GardenManager) DeleteCompressedHistory(ctx context.Context, historyID string) error {
	query := `DELETE FROM compressed_history WHERE id = ?`

	result, err := gm.db.ExecContext(ctx, query, historyID)
	if err != nil {
		return fmt.Errorf("failed to delete compressed history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("compressed history not found: %s", historyID)
	}

	return nil
}

// DeleteSnapshot deletes a memory snapshot by ID.
func (gm *GardenManager) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	query := `DELETE FROM memory_snapshots WHERE id = ?`

	result, err := gm.db.ExecContext(ctx, query, snapshotID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	return nil
}

// IncrementSeedUsage increments the usage counter for a seed.
func (gm *GardenManager) IncrementSeedUsage(ctx context.Context, seedID string) error {
	query := `
		UPDATE seeds
		SET usage_count = usage_count + 1, last_used = ?
		WHERE id = ?
	`

	result, err := gm.db.ExecContext(ctx, query, time.Now(), seedID)
	if err != nil {
		return fmt.Errorf("failed to increment seed usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSeedNotFound
	}

	return nil
}

// ShouldCompress returns true if the number of memories exceeds the threshold.
func (gm *GardenManager) ShouldCompress(memories []Memory, turnThreshold int) bool {
	return len(memories) >= turnThreshold
}
