package maintenance

import "time"

type Insight struct {
	Theme       string    `json:"theme"`
	Summary     string    `json:"summary"`
	Source      string    `json:"source"`
	Importance  float64   `json:"importance"`
	ExtractedAt time.Time `json:"extracted_at"`
}

type MaintenanceReport struct {
	StartTime            time.Time         `json:"start_time"`
	EndTime              time.Time         `json:"end_time"`
	Duration             time.Duration     `json:"duration"`
	FilesProcessed       int               `json:"files_processed"`
	FilesArchived        int               `json:"files_archived"`
	InsightsExtracted    int               `json:"insights_extracted"`
	EntriesComposted     int               `json:"entries_composted"`
	EmbeddingsBackfilled int               `json:"embeddings_backfilled"`
	EmbeddingsRemaining  int               `json:"embeddings_remaining"`
	BackfillProgress     float64           `json:"backfill_progress"`
	MemorySize           MemorySizeMetrics `json:"memory_size"`
	Errors               []string          `json:"errors,omitempty"`
	Success              bool              `json:"success"`
}

type MemorySizeMetrics struct {
	BeforeBytes int64 `json:"before_bytes"`
	AfterBytes  int64 `json:"after_bytes"`
	Reduction   int64 `json:"reduction"`
}
