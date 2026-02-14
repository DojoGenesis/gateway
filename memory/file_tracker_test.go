package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupFileTrackerTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS memory_files (
		id TEXT PRIMARY KEY,
		file_path TEXT NOT NULL UNIQUE,
		tier INTEGER NOT NULL,
		content TEXT NOT NULL,
		embedding BLOB,
		themes TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		archived_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_memory_files_tier ON memory_files(tier);
	CREATE INDEX IF NOT EXISTS idx_memory_files_archived ON memory_files(archived_at);
	CREATE INDEX IF NOT EXISTS idx_memory_files_path ON memory_files(file_path);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestFileTracker_TrackFile(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		path    string
		tier    int
		content string
		wantErr bool
	}{
		{
			name:    "valid tier 1 file",
			path:    "memory/2026-02-01.md",
			tier:    1,
			content: "Today we worked on the file tracker implementation.",
			wantErr: false,
		},
		{
			name:    "valid tier 2 file",
			path:    "MEMORY.md",
			tier:    2,
			content: "Core insights about the system architecture.",
			wantErr: false,
		},
		{
			name:    "valid tier 3 file",
			path:    "memory/archive/2026-01.jsonl.gz",
			tier:    3,
			content: "Compressed archive of January 2026.",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			tier:    1,
			content: "Content",
			wantErr: true,
		},
		{
			name:    "invalid tier (0)",
			path:    "test.md",
			tier:    0,
			content: "Content",
			wantErr: true,
		},
		{
			name:    "invalid tier (4)",
			path:    "test2.md",
			tier:    4,
			content: "Content",
			wantErr: true,
		},
		{
			name:    "empty content",
			path:    "test3.md",
			tier:    1,
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ft.TrackFile(ctx, tt.path, tt.tier, tt.content, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				file, err := ft.GetFile(ctx, tt.path)
				require.NoError(t, err)
				assert.Equal(t, tt.path, file.FilePath)
				assert.Equal(t, tt.tier, file.Tier)
				assert.Equal(t, tt.content, file.Content)
			}
		})
	}
}

func TestFileTracker_TrackFile_WithEmbedding(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 768.0
	}

	err := ft.TrackFile(ctx, "test.md", 1, "Test content", embedding)
	require.NoError(t, err)

	file, err := ft.GetFile(ctx, "test.md")
	require.NoError(t, err)
	assert.Equal(t, len(embedding), len(file.Embedding))
}

func TestFileTracker_TrackFile_Update(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	path := "test.md"
	content1 := "Original content"
	content2 := "Updated content"

	err := ft.TrackFile(ctx, path, 1, content1, nil)
	require.NoError(t, err)

	file1, err := ft.GetFile(ctx, path)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = ft.TrackFile(ctx, path, 2, content2, nil)
	require.NoError(t, err)

	file2, err := ft.GetFile(ctx, path)
	require.NoError(t, err)

	assert.Equal(t, content2, file2.Content)
	assert.Equal(t, 2, file2.Tier)
	assert.True(t, file2.UpdatedAt.After(file1.UpdatedAt) || file2.UpdatedAt.Equal(file1.UpdatedAt))
	assert.Equal(t, file1.ID, file2.ID)
}

func TestFileTracker_GetFile(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	path := "test.md"
	content := "Test content"

	err := ft.TrackFile(ctx, path, 1, content, nil)
	require.NoError(t, err)

	t.Run("existing file", func(t *testing.T) {
		file, err := ft.GetFile(ctx, path)
		assert.NoError(t, err)
		assert.Equal(t, path, file.FilePath)
		assert.Equal(t, content, file.Content)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ft.GetFile(ctx, "nonexistent.md")
		assert.Error(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := ft.GetFile(ctx, "")
		assert.Error(t, err)
	})
}

func TestFileTracker_ListFiles(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	files := []struct {
		path    string
		tier    int
		content string
	}{
		{"memory/2026-02-01.md", 1, "Daily note 1"},
		{"memory/2026-02-02.md", 1, "Daily note 2"},
		{"MEMORY.md", 2, "Curated wisdom"},
		{"memory/archive/2026-01.jsonl.gz", 3, "Archive"},
	}

	for _, f := range files {
		err := ft.TrackFile(ctx, f.path, f.tier, f.content, nil)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		tier      int
		wantCount int
	}{
		{"tier 1 files", 1, 2},
		{"tier 2 files", 2, 1},
		{"tier 3 files", 3, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ft.ListFiles(ctx, tt.tier, 100, 0, false)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCount, len(result))

			for _, f := range result {
				assert.Equal(t, tt.tier, f.Tier)
			}
		})
	}
}

func TestFileTracker_ListFiles_WithArchived(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "file1.md", 1, "Active file 1", nil)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "file2.md", 1, "File to be archived", nil)
	require.NoError(t, err)

	err = ft.ArchiveFile(ctx, "file2.md")
	require.NoError(t, err)

	t.Run("excludes archived files by default", func(t *testing.T) {
		result, err := ft.ListFiles(ctx, 1, 100, 0, false)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "file1.md", result[0].FilePath)
	})

	t.Run("includes archived files when requested", func(t *testing.T) {
		result, err := ft.ListFiles(ctx, 1, 100, 0, true)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
	})
}

func TestFileTracker_ListFiles_Pagination(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		err := ft.TrackFile(ctx, fmt.Sprintf("file%d.md", i), 1, fmt.Sprintf("Content %d", i), nil)
		require.NoError(t, err)
	}

	t.Run("first page", func(t *testing.T) {
		result, err := ft.ListFiles(ctx, 1, 5, 0, false)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(result))
	})

	t.Run("second page", func(t *testing.T) {
		result, err := ft.ListFiles(ctx, 1, 5, 5, false)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(result))
	})

	t.Run("third page (empty)", func(t *testing.T) {
		result, err := ft.ListFiles(ctx, 1, 5, 10, false)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})
}

func TestFileTracker_ListFiles_TierValidation(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	t.Run("invalid tier -1", func(t *testing.T) {
		_, err := ft.ListFiles(ctx, -1, 100, 0, false)
		assert.Error(t, err)
	})

	t.Run("invalid tier 4", func(t *testing.T) {
		_, err := ft.ListFiles(ctx, 4, 100, 0, false)
		assert.Error(t, err)
	})

	t.Run("valid tier 0 (all)", func(t *testing.T) {
		_, err := ft.ListFiles(ctx, 0, 100, 0, false)
		assert.NoError(t, err)
	})
}

func TestFileTracker_ArchiveFile(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	path := "test.md"
	err := ft.TrackFile(ctx, path, 1, "Test content", nil)
	require.NoError(t, err)

	t.Run("archive existing file", func(t *testing.T) {
		err := ft.ArchiveFile(ctx, path)
		assert.NoError(t, err)

		file, err := ft.GetFile(ctx, path)
		assert.NoError(t, err)
		assert.NotNil(t, file.ArchivedAt)
	})

	t.Run("archive non-existent file", func(t *testing.T) {
		err := ft.ArchiveFile(ctx, "nonexistent.md")
		assert.Error(t, err)
	})

	t.Run("archive with empty path", func(t *testing.T) {
		err := ft.ArchiveFile(ctx, "")
		assert.Error(t, err)
	})
}

func TestFileTracker_SearchFiles_Fallback(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	files := []struct {
		path    string
		tier    int
		content string
	}{
		{"file1.md", 1, "This document contains information about machine learning."},
		{"file2.md", 1, "This is a guide to web development using React."},
		{"file3.md", 2, "Machine learning and AI are transforming software."},
	}

	for _, f := range files {
		err := ft.TrackFile(ctx, f.path, f.tier, f.content, nil)
		require.NoError(t, err)
	}

	t.Run("search with keyword", func(t *testing.T) {
		results, err := ft.SearchFiles(ctx, "machine learning", 0, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})

	t.Run("search with tier filter", func(t *testing.T) {
		results, err := ft.SearchFiles(ctx, "machine learning", 1, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, 1, results[0].File.Tier)
	})

	t.Run("search with no results", func(t *testing.T) {
		results, err := ft.SearchFiles(ctx, "nonexistent keyword", 0, nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("search with empty query", func(t *testing.T) {
		_, err := ft.SearchFiles(ctx, "", 0, nil)
		assert.Error(t, err)
	})
}

func TestFileTracker_SearchFiles_WithEmbeddings(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	// Create embeddings
	emb1 := make([]float32, 768)
	emb2 := make([]float32, 768)
	for i := range emb1 {
		emb1[i] = float32(i) / 768.0
		emb2[i] = float32(768-i) / 768.0
	}

	err := ft.TrackFile(ctx, "file1.md", 1, "Machine learning content", emb1)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "file2.md", 1, "Web development content", emb2)
	require.NoError(t, err)

	queryEmb := make([]float32, 768)
	for i := range queryEmb {
		queryEmb[i] = float32(i) / 768.0
	}

	results, err := ft.SearchFiles(ctx, "search query", 0, queryEmb)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)

	for _, result := range results {
		assert.True(t, result.Similarity >= -1.0 && result.Similarity <= 1.0)
	}

	if len(results) > 1 {
		for i := 0; i < len(results)-1; i++ {
			assert.GreaterOrEqual(t, results[i].Similarity, results[i+1].Similarity)
		}
	}
}

func TestFileTracker_SearchFiles_ExcludesArchived(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "file1.md", 1, "This is an active file.", nil)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "file2.md", 1, "This file will be archived.", nil)
	require.NoError(t, err)

	err = ft.ArchiveFile(ctx, "file2.md")
	require.NoError(t, err)

	results, err := ft.SearchFiles(ctx, "file", 0, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "file1.md", results[0].File.FilePath)
}

func TestFileTracker_SearchFiles_WithTierFilter(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "tier1.md", 1, "tier one content", nil)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "tier2.md", 2, "tier two content", nil)
	require.NoError(t, err)

	results, err := ft.SearchFiles(ctx, "tier", 1, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
}

func TestFileTracker_Snippet(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	longContent := strings.Repeat("a", 300)

	err := ft.TrackFile(ctx, "test.md", 1, longContent, nil)
	require.NoError(t, err)

	results, err := ft.SearchFiles(ctx, "a", 0, nil)
	assert.NoError(t, err)
	require.Greater(t, len(results), 0)

	snippet := results[0].Snippet
	assert.LessOrEqual(t, len(snippet), MaxSnippetLength+3)
	assert.True(t, strings.Contains(snippet, "..."))
}

func TestFileTracker_ListFiles_DefaultLimit(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	for i := 1; i <= 60; i++ {
		err := ft.TrackFile(ctx, fmt.Sprintf("file%d.md", i), 1, fmt.Sprintf("Content %d", i), nil)
		require.NoError(t, err)
	}

	files, err := ft.ListFiles(ctx, 1, 0, 0, false)
	assert.NoError(t, err)
	assert.Equal(t, DefaultPageLimit, len(files))
}

func TestFileTracker_ListFiles_NegativeOffset(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "test.md", 1, "content", nil)
	require.NoError(t, err)

	files, err := ft.ListFiles(ctx, 1, 10, -5, false)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

func TestMemoryFile_ArchivedAt(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	path := "test.md"
	err := ft.TrackFile(ctx, path, 1, "content", nil)
	require.NoError(t, err)

	file1, err := ft.GetFile(ctx, path)
	require.NoError(t, err)
	assert.Nil(t, file1.ArchivedAt)

	err = ft.ArchiveFile(ctx, path)
	require.NoError(t, err)

	file2, err := ft.GetFile(ctx, path)
	require.NoError(t, err)
	assert.NotNil(t, file2.ArchivedAt)
}

func TestMemoryFile_JSON(t *testing.T) {
	now := time.Now()
	file := MemoryFile{
		ID:        "test-id",
		FilePath:  "test.md",
		Tier:      1,
		Content:   "Test content",
		Themes:    []string{"theme1", "theme2"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(file)
	require.NoError(t, err)

	var decoded MemoryFile
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, file.ID, decoded.ID)
	assert.Equal(t, file.FilePath, decoded.FilePath)
	assert.Equal(t, len(file.Themes), len(decoded.Themes))
}

func TestFileTracker_searchFilesFallback_WithTier(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "test1.md", 1, "searchable content here", nil)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "test2.md", 2, "searchable content here", nil)
	require.NoError(t, err)

	results, err := ft.searchFilesFallback(ctx, "searchable", 1)
	assert.NoError(t, err)

	for _, r := range results {
		assert.NotEqual(t, 2, r.File.Tier)
	}
}

func TestFileTracker_searchFilesFallback_ExcludesArchived(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "active.md", 1, "searchable content", nil)
	require.NoError(t, err)

	err = ft.TrackFile(ctx, "archived.md", 1, "searchable content", nil)
	require.NoError(t, err)

	err = ft.ArchiveFile(ctx, "archived.md")
	require.NoError(t, err)

	results, err := ft.searchFilesFallback(ctx, "searchable", 0)
	assert.NoError(t, err)

	for _, r := range results {
		assert.NotEqual(t, "archived.md", r.File.FilePath)
	}
}

func TestFileTracker_searchFilesFallback_NoTier(t *testing.T) {
	db, cleanup := setupFileTrackerTestDB(t)
	defer cleanup()

	ft := NewFileTracker(db)
	ctx := context.Background()

	err := ft.TrackFile(ctx, "test.md", 1, "searchable content", nil)
	require.NoError(t, err)

	results, err := ft.searchFilesFallback(ctx, "searchable", 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
}
