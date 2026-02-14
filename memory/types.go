package memory

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMemoryNotFound         = errors.New("memory not found")
	ErrProjectNotFound        = errors.New("project not found")
	ErrSeedNotFound           = errors.New("seed not found")
	ErrSeedNotEditable        = errors.New("seed is not editable")
	ErrCannotDeleteSystemSeed = errors.New("cannot delete system seed")
	ErrInvalidContent         = errors.New("invalid content")
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrPermissionDenied       = errors.New("permission denied")
)

// SourceType identifies the origin of a seed.
type SourceType string

const (
	SourceSystem     SourceType = "system"
	SourceUser       SourceType = "user"
	SourceCalibrated SourceType = "calibrated"
)

// Memory represents a single conversation memory entry.
type Memory struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	Embedding   []float32              `json:"embedding,omitempty"`
	ContextType string                 `json:"context_type"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SearchResult represents a memory search result with similarity scoring.
type SearchResult struct {
	Memory     Memory  `json:"memory"`
	Similarity float64 `json:"similarity"`
	Snippet    string  `json:"snippet"`
	SearchMode string  `json:"search_mode"`
}

// MemorySeed represents a reusable knowledge pattern extracted from conversations.
type MemorySeed struct {
	ID           string     `json:"id"`
	ProjectID    *string    `json:"project_id,omitempty"`
	Content      string     `json:"content"`
	SeedType     string     `json:"seed_type"`
	Source       SourceType `json:"source"`
	UserEditable bool       `json:"user_editable"`
	Confidence   float64    `json:"confidence"`
	UsageCount   int        `json:"usage_count"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedBy    *string    `json:"created_by,omitempty"`
	Version      int        `json:"version"`
}

// CompressedHistory represents a semantically compressed conversation history.
type CompressedHistory struct {
	ID                string    `json:"id"`
	SessionID         string    `json:"session_id"`
	OriginalTurnIDs   []string  `json:"original_turn_ids"`
	CompressedContent string    `json:"compressed_content"`
	CompressionRatio  float64   `json:"compression_ratio"`
	CreatedAt         time.Time `json:"created_at"`
}

// MemoryFilter specifies criteria for listing memories.
type MemoryFilter struct {
	Type         string
	ContextType  string
	CreatedAfter *time.Time
	Limit        int
}

// SeedFilter specifies criteria for listing seeds.
type SeedFilter struct {
	Type          string
	MinConfidence float64
	SortBy        string
	Limit         int
}

// MemoryFile represents a tracked file in the memory system.
type MemoryFile struct {
	ID         string     `json:"id"`
	FilePath   string     `json:"file_path"`
	Tier       int        `json:"tier"`
	Content    string     `json:"content"`
	Embedding  []float32  `json:"embedding,omitempty"`
	Themes     []string   `json:"themes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

// FileSearchResult represents a file search result with similarity scoring.
type FileSearchResult struct {
	File       MemoryFile `json:"file"`
	Similarity float64    `json:"similarity"`
	Snippet    string     `json:"snippet"`
}

// CompressionServiceInterface allows consumers to provide their own LLM-based compression.
type CompressionServiceInterface interface {
	// CompressHistory compresses conversation history using semantic abstraction.
	CompressHistory(ctx context.Context, sessionID string, memories []Memory) (*CompressedHistory, error)
	// ExtractSeeds extracts reusable patterns from memories.
	ExtractSeeds(ctx context.Context, memories []Memory) ([]*MemorySeed, error)
}
