// Package memory provides functionality for managing memory seeds in the collaborative memory system.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// SeedManager handles CRUD operations for memory seeds.
// Memory seeds are project-scoped knowledge snippets that can be user-editable
// or system-generated, allowing collaborative customization of AI behavior.
type SeedManager struct {
	db *sql.DB
}

// NewSeedManager creates a new SeedManager instance.
// Returns an error if the database connection is nil.
func NewSeedManager(db *sql.DB) (*SeedManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	sm := &SeedManager{db: db}
	if err := sm.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return sm, nil
}

func (sm *SeedManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memory_seeds (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		content TEXT NOT NULL CHECK(length(content) > 0 AND length(content) <= 10240),
		seed_type TEXT DEFAULT 'knowledge' CHECK(seed_type IN ('knowledge', 'pattern', 'preference', 'constraint')),
		source TEXT NOT NULL DEFAULT 'system' CHECK(source IN ('system', 'user', 'calibrated')),
		user_editable BOOLEAN DEFAULT FALSE,
		confidence REAL DEFAULT 1.0 CHECK(confidence >= 0.0 AND confidence <= 1.0),
		usage_count INTEGER DEFAULT 0,
		last_used_at DATETIME,
		deleted_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		created_by TEXT,
		version INTEGER DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_memory_seeds_project ON memory_seeds(project_id);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_type ON memory_seeds(seed_type);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_source ON memory_seeds(source);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_editable ON memory_seeds(user_editable);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_deleted ON memory_seeds(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_memory_seeds_created ON memory_seeds(created_at);
	`

	_, err := sm.db.Exec(schema)
	return err
}

// GetSeeds retrieves memory seeds with optional filtering.
// If projectID is provided, returns only seeds for that project.
// Filters can include: seed_type, source, user_editable, global_only.
// Automatically excludes soft-deleted seeds.
// Returns seeds ordered by creation time (newest first).
func (sm *SeedManager) GetSeeds(ctx context.Context, projectID *string, filters map[string]interface{}) ([]MemorySeed, error) {
	slog.Info("GetSeeds called", "project_id", projectID, "filters", filters)

	query := `
		SELECT id, project_id, content, seed_type, source, user_editable, 
		       confidence, usage_count, last_used_at, deleted_at, created_at, updated_at, created_by, version
		FROM memory_seeds
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if projectID != nil {
		query += " AND project_id = ?"
		args = append(args, *projectID)
	} else if filters != nil {
		if globalOnly, ok := filters["global_only"].(bool); ok && globalOnly {
			query += " AND project_id IS NULL"
		}
	}

	if filters != nil {
		if seedType, ok := filters["seed_type"].(string); ok && seedType != "" {
			query += " AND seed_type = ?"
			args = append(args, seedType)
		}
		if source, ok := filters["source"].(string); ok && source != "" {
			query += " AND source = ?"
			args = append(args, source)
		}
		if userEditable, ok := filters["user_editable"].(bool); ok {
			query += " AND user_editable = ?"
			args = append(args, userEditable)
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := sm.db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error("failed to query seeds", "error", err)
		return nil, fmt.Errorf("failed to query seeds: %w", err)
	}
	defer rows.Close()

	seeds, err := sm.scanSeedRows(rows)
	if err != nil {
		return nil, err
	}

	slog.Info("retrieved seeds", "count", len(seeds))
	return seeds, nil
}

// GetSeedByID retrieves a single memory seed by its ID.
// Returns ErrSeedNotFound if the seed doesn't exist or has been soft-deleted.
func (sm *SeedManager) GetSeedByID(ctx context.Context, id string) (*MemorySeed, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	slog.Info("GetSeedByID called", "seed_id", id)

	query := `
		SELECT id, project_id, content, seed_type, source, user_editable, 
		       confidence, usage_count, last_used_at, deleted_at, created_at, updated_at, created_by, version
		FROM memory_seeds
		WHERE id = ? AND deleted_at IS NULL
	`

	var seed MemorySeed
	var projectID, createdBy sql.NullString
	var lastUsedAt, deletedAt sql.NullTime

	err := sm.db.QueryRowContext(ctx, query, id).Scan(
		&seed.ID,
		&projectID,
		&seed.Content,
		&seed.SeedType,
		&seed.Source,
		&seed.UserEditable,
		&seed.Confidence,
		&seed.UsageCount,
		&lastUsedAt,
		&deletedAt,
		&seed.CreatedAt,
		&seed.UpdatedAt,
		&createdBy,
		&seed.Version,
	)

	if err == sql.ErrNoRows {
		slog.Warn("seed not found", "seed_id", id)
		return nil, ErrSeedNotFound
	}
	if err != nil {
		slog.Error("failed to get seed", "seed_id", id, "error", err)
		return nil, fmt.Errorf("failed to get seed: %w", err)
	}

	if projectID.Valid {
		seed.ProjectID = &projectID.String
	}
	if lastUsedAt.Valid {
		seed.LastUsedAt = &lastUsedAt.Time
	}
	if deletedAt.Valid {
		seed.DeletedAt = &deletedAt.Time
	}
	if createdBy.Valid {
		seed.CreatedBy = &createdBy.String
	}

	return &seed, nil
}

// UpdateSeed updates the content of a memory seed.
// Only user-editable seeds can be updated.
// Implements optimistic locking to prevent concurrent modification conflicts.
// Automatically increments the version number on successful update.
// Returns ErrSeedNotEditable if the seed cannot be modified.
// Returns ErrConcurrentModification if the seed was modified by another process.
func (sm *SeedManager) UpdateSeed(ctx context.Context, id, content string, userID *string) (*MemorySeed, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	if err := validateContent(content); err != nil {
		slog.Error("invalid content for seed", "seed_id", id, "error", err)
		return nil, err
	}

	seed, err := sm.GetSeedByID(ctx, id)
	if err != nil {
		slog.Error("failed to get seed for update", "seed_id", id, "error", err)
		return nil, err
	}

	if !seed.UserEditable {
		slog.Warn("attempt to update non-editable seed", "seed_id", id)
		return nil, ErrSeedNotEditable
	}

	tx, err := sm.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var currentUpdatedAt time.Time
	var currentVersion int
	err = tx.QueryRowContext(ctx, "SELECT updated_at, version FROM memory_seeds WHERE id = ?", id).Scan(&currentUpdatedAt, &currentVersion)
	if err != nil {
		slog.Error("failed to check current version", "seed_id", id, "error", err)
		return nil, fmt.Errorf("failed to check current version: %w", err)
	}

	if !currentUpdatedAt.Equal(seed.UpdatedAt) || currentVersion != seed.Version {
		slog.Warn("concurrent modification detected", "seed_id", id)
		return nil, ErrConcurrentModification
	}

	now := time.Now()
	newVersion := seed.Version + 1

	query := `
		UPDATE memory_seeds
		SET content = ?, updated_at = ?, source = ?, version = ?
		WHERE id = ? AND user_editable = TRUE
	`

	result, err := tx.ExecContext(ctx, query, content, now, SourceUser, newVersion, id)
	if err != nil {
		slog.Error("failed to update seed", "seed_id", id, "error", err)
		return nil, fmt.Errorf("failed to update seed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		slog.Error("no rows affected when updating seed", "seed_id", id)
		return nil, fmt.Errorf("seed update failed")
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	seed.Content = content
	seed.UpdatedAt = now
	seed.Source = SourceUser
	seed.Version = newVersion

	slog.Info("updated seed", "seed_id", id, "content_length", len(content), "version", newVersion)
	return seed, nil
}

// CreateUserSeed creates a new user-generated memory seed.
// User seeds are always editable and have source='user'.
// If projectID is provided, validates that the project exists.
// Returns ErrProjectNotFound if the project doesn't exist.
func (sm *SeedManager) CreateUserSeed(ctx context.Context, projectID *string, content, seedType, userID string) (*MemorySeed, error) {
	if err := validateContent(content); err != nil {
		slog.Error("invalid content for new seed", "error", err)
		return nil, err
	}

	if seedType == "" {
		return nil, fmt.Errorf("seed_type is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if projectID != nil && *projectID != "" {
		exists, err := sm.projectExists(ctx, *projectID)
		if err != nil {
			slog.Error("failed to validate project existence", "error", err)
			return nil, err
		}
		if !exists {
			slog.Warn("attempt to create seed for non-existent project", "project_id", *projectID)
			return nil, ErrProjectNotFound
		}
	}

	now := time.Now()
	seed := &MemorySeed{
		ID:           uuid.New().String(),
		ProjectID:    projectID,
		Content:      content,
		SeedType:     seedType,
		Source:       SourceUser,
		UserEditable: true,
		Confidence:   1.0,
		UsageCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    &userID,
		Version:      1,
	}

	query := `
		INSERT INTO memory_seeds 
		(id, project_id, content, seed_type, source, user_editable, confidence, 
		 usage_count, created_at, updated_at, created_by, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var projectIDVal interface{}
	if projectID != nil {
		projectIDVal = *projectID
	} else {
		projectIDVal = nil
	}

	_, err := sm.db.ExecContext(ctx, query,
		seed.ID,
		projectIDVal,
		seed.Content,
		seed.SeedType,
		seed.Source,
		seed.UserEditable,
		seed.Confidence,
		seed.UsageCount,
		seed.CreatedAt,
		seed.UpdatedAt,
		seed.CreatedBy,
		seed.Version,
	)

	if err != nil {
		slog.Error("failed to create seed", "error", err)
		return nil, fmt.Errorf("failed to create seed: %w", err)
	}

	slog.Info("created user seed", "seed_id", seed.ID, "project_id", projectID, "seed_type", seedType, "content_length", len(content))
	return seed, nil
}

// CreateSystemSeed creates a new system-sourced memory seed.
// System seeds are non-editable by users and have source='system'.
// The createdBy parameter is optional — system seeds may not have a human creator.
// If projectID is provided, validates that the project exists.
func (sm *SeedManager) CreateSystemSeed(ctx context.Context, projectID *string, content, seedType string, createdBy *string) (*MemorySeed, error) {
	if err := validateContent(content); err != nil {
		slog.Error("invalid content for new system seed", "error", err)
		return nil, err
	}

	if seedType == "" {
		return nil, fmt.Errorf("seed_type is required")
	}

	if projectID != nil && *projectID != "" {
		exists, err := sm.projectExists(ctx, *projectID)
		if err != nil {
			slog.Error("failed to validate project existence", "error", err)
			return nil, err
		}
		if !exists {
			slog.Warn("attempt to create system seed for non-existent project", "project_id", *projectID)
			return nil, ErrProjectNotFound
		}
	}

	now := time.Now()
	seed := &MemorySeed{
		ID:           uuid.New().String(),
		ProjectID:    projectID,
		Content:      content,
		SeedType:     seedType,
		Source:       SourceSystem,
		UserEditable: false,
		Confidence:   1.0,
		UsageCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    createdBy,
		Version:      1,
	}

	query := `
		INSERT INTO memory_seeds
		(id, project_id, content, seed_type, source, user_editable, confidence,
		 usage_count, created_at, updated_at, created_by, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var projectIDVal interface{}
	if projectID != nil {
		projectIDVal = *projectID
	} else {
		projectIDVal = nil
	}

	var createdByVal interface{}
	if createdBy != nil {
		createdByVal = *createdBy
	} else {
		createdByVal = nil
	}

	_, err := sm.db.ExecContext(ctx, query,
		seed.ID,
		projectIDVal,
		seed.Content,
		seed.SeedType,
		seed.Source,
		seed.UserEditable,
		seed.Confidence,
		seed.UsageCount,
		seed.CreatedAt,
		seed.UpdatedAt,
		createdByVal,
		seed.Version,
	)

	if err != nil {
		slog.Error("failed to create system seed", "error", err)
		return nil, fmt.Errorf("failed to create system seed: %w", err)
	}

	slog.Info("created system seed", "seed_id", seed.ID, "project_id", projectID, "seed_type", seedType, "content_length", len(content))
	return seed, nil
}

// DeleteSeed soft-deletes a memory seed by setting its deleted_at timestamp.
// Only user-created seeds (source='user' AND user_editable=true) can be deleted.
// If userID is provided, verifies that the user owns the seed.
// Returns ErrCannotDeleteSystemSeed if attempting to delete a system seed.
// Returns ErrPermissionDenied if the user doesn't own the seed.
func (sm *SeedManager) DeleteSeed(ctx context.Context, id string, userID *string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	seed, err := sm.GetSeedByID(ctx, id)
	if err != nil {
		slog.Error("failed to get seed for deletion", "seed_id", id, "error", err)
		return err
	}

	if seed.Source != SourceUser || !seed.UserEditable {
		slog.Warn("attempt to delete non-user seed", "seed_id", id, "source", seed.Source, "user_editable", seed.UserEditable)
		return ErrCannotDeleteSystemSeed
	}

	if userID != nil && seed.CreatedBy != nil && *seed.CreatedBy != *userID {
		slog.Warn("permission denied to delete seed", "user_id", *userID, "seed_id", id, "owner", *seed.CreatedBy)
		return ErrPermissionDenied
	}

	now := time.Now()
	query := `
		UPDATE memory_seeds
		SET deleted_at = ?
		WHERE id = ? AND source = ? AND user_editable = TRUE
	`

	result, err := sm.db.ExecContext(ctx, query, now, id, SourceUser)
	if err != nil {
		slog.Error("failed to soft-delete seed", "seed_id", id, "error", err)
		return fmt.Errorf("failed to delete seed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rowsAffected == 0 {
		slog.Error("no rows affected when deleting seed", "seed_id", id)
		return fmt.Errorf("seed not found: %s", id)
	}

	slog.Info("soft-deleted seed", "seed_id", id)
	return nil
}

// IncrementUsage increments the usage counter for a seed and updates its last_used_at timestamp.
// This helps track which seeds are actively used by the AI.
func (sm *SeedManager) IncrementUsage(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	now := time.Now()
	query := `
		UPDATE memory_seeds
		SET usage_count = usage_count + 1, last_used_at = ?
		WHERE id = ?
	`

	result, err := sm.db.ExecContext(ctx, query, now, id)
	if err != nil {
		slog.Error("failed to increment usage", "seed_id", id, "error", err)
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("seed not found when incrementing usage", "seed_id", id)
		return fmt.Errorf("seed not found: %s", id)
	}

	slog.Info("incremented usage for seed", "seed_id", id)
	return nil
}

// SearchSeeds performs a full-text search on memory seeds within a project.
// Returns seeds whose content contains the search query (case-insensitive).
// Excludes soft-deleted seeds and orders results by creation time.
// Limit parameter controls maximum number of results (default: 20, max: 100).
func (sm *SeedManager) SearchSeeds(ctx context.Context, projectID *string, query string, limit int) ([]MemorySeed, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Trim and validate query
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, fmt.Errorf("search query must be at least 2 characters")
	}

	// Set default and max limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	slog.Info("SearchSeeds called", "project_id", projectID, "query", query, "limit", limit)

	sqlQuery := `
		SELECT id, project_id, content, seed_type, source, user_editable, 
		       confidence, usage_count, last_used_at, deleted_at, created_at, updated_at, created_by, version
		FROM memory_seeds
		WHERE deleted_at IS NULL
		  AND content LIKE ?
	`
	args := []interface{}{"%" + query + "%"}

	if projectID != nil {
		sqlQuery += " AND project_id = ?"
		args = append(args, *projectID)
	}

	sqlQuery += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := sm.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		slog.Error("failed to search seeds", "error", err)
		return nil, fmt.Errorf("failed to search seeds: %w", err)
	}
	defer rows.Close()

	seeds, err := sm.scanSeedRows(rows)
	if err != nil {
		return nil, err
	}

	slog.Info("found seeds matching query", "count", len(seeds))
	return seeds, nil
}

func (sm *SeedManager) scanSeedRows(rows *sql.Rows) ([]MemorySeed, error) {
	seeds := []MemorySeed{}

	for rows.Next() {
		var seed MemorySeed
		var projectID, createdBy sql.NullString
		var lastUsedAt, deletedAt sql.NullTime

		err := rows.Scan(
			&seed.ID,
			&projectID,
			&seed.Content,
			&seed.SeedType,
			&seed.Source,
			&seed.UserEditable,
			&seed.Confidence,
			&seed.UsageCount,
			&lastUsedAt,
			&deletedAt,
			&seed.CreatedAt,
			&seed.UpdatedAt,
			&createdBy,
			&seed.Version,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan seed row: %w", err)
		}

		if projectID.Valid {
			seed.ProjectID = &projectID.String
		}
		if lastUsedAt.Valid {
			seed.LastUsedAt = &lastUsedAt.Time
		}
		if deletedAt.Valid {
			seed.DeletedAt = &deletedAt.Time
		}
		if createdBy.Valid {
			seed.CreatedBy = &createdBy.String
		}

		seeds = append(seeds, seed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating seed rows: %w", err)
	}

	return seeds, nil
}

func validateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrInvalidContent
	}

	if len(content) > 10240 {
		return fmt.Errorf("%w: content exceeds maximum length of 10KB (got %d bytes)", ErrInvalidContent, len(content))
	}

	if !utf8.ValidString(content) {
		return fmt.Errorf("%w: content must be valid UTF-8", ErrInvalidContent)
	}

	if strings.Contains(content, "\x00") {
		return fmt.Errorf("%w: content cannot contain null bytes", ErrInvalidContent)
	}

	return nil
}

func (sm *SeedManager) projectExists(ctx context.Context, projectID string) (bool, error) {
	query := `SELECT COUNT(*) FROM projects WHERE id = ?`
	var count int
	err := sm.db.QueryRowContext(ctx, query, projectID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return count > 0, nil
}
