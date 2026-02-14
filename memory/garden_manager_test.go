package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCompressionService implements CompressionServiceInterface for testing.
type mockCompressionService struct {
	compressResult *CompressedHistory
	compressErr    error
	extractResult  []*MemorySeed
	extractErr     error
}

func (m *mockCompressionService) CompressHistory(ctx context.Context, sessionID string, memories []Memory) (*CompressedHistory, error) {
	if m.compressErr != nil {
		return nil, m.compressErr
	}
	if m.compressResult != nil {
		return m.compressResult, nil
	}
	return &CompressedHistory{
		ID:                "compressed-1",
		SessionID:         sessionID,
		OriginalTurnIDs:   []string{"t1", "t2"},
		CompressedContent: "compressed content",
		CompressionRatio:  0.3,
		CreatedAt:         time.Now(),
	}, nil
}

func (m *mockCompressionService) ExtractSeeds(ctx context.Context, memories []Memory) ([]*MemorySeed, error) {
	if m.extractErr != nil {
		return nil, m.extractErr
	}
	if m.extractResult != nil {
		return m.extractResult, nil
	}
	return []*MemorySeed{}, nil
}

func setupGardenTest(t *testing.T) (*GardenManager, *MemoryManager) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "garden_test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)

	gm, err := NewGardenManager(mm, nil)
	require.NoError(t, err)

	return gm, mm
}

func setupGardenTestWithCompression(t *testing.T) (*GardenManager, *MemoryManager, *mockCompressionService) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "garden_test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)

	cs := &mockCompressionService{}
	gm, err := NewGardenManager(mm, cs)
	require.NoError(t, err)

	return gm, mm, cs
}

func TestNewGardenManager(t *testing.T) {
	t.Run("without compression service", func(t *testing.T) {
		gm, mm := setupGardenTest(t)
		defer mm.Close()
		assert.NotNil(t, gm)
	})

	t.Run("with compression service", func(t *testing.T) {
		gm, mm, _ := setupGardenTestWithCompression(t)
		defer mm.Close()
		assert.NotNil(t, gm)
	})
}

func TestGardenManager_StoreSeed(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	t.Run("valid seed", func(t *testing.T) {
		seed := &Seed{
			Name:      "test-seed",
			Content:   "Test content",
			Trigger:   "golang,testing",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := gm.StoreSeed(ctx, seed)
		assert.NoError(t, err)
		assert.NotEmpty(t, seed.ID)
	})

	t.Run("empty name", func(t *testing.T) {
		seed := &Seed{Content: "content"}
		err := gm.StoreSeed(ctx, seed)
		assert.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		seed := &Seed{Name: "name"}
		err := gm.StoreSeed(ctx, seed)
		assert.Error(t, err)
	})

	t.Run("upsert existing seed", func(t *testing.T) {
		seed := &Seed{
			ID:        "fixed-id",
			Name:      "original",
			Content:   "original content",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, gm.StoreSeed(ctx, seed))

		seed.Name = "updated"
		seed.Content = "updated content"
		seed.UpdatedAt = time.Now()
		require.NoError(t, gm.StoreSeed(ctx, seed))

		got, err := gm.RetrieveSeed(ctx, "fixed-id")
		require.NoError(t, err)
		assert.Equal(t, "updated", got.Name)
		assert.Equal(t, "updated content", got.Content)
	})
}

func TestGardenManager_RetrieveSeed(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	seed := &Seed{
		Name:      "retrieve-test",
		Content:   "Retrieve me",
		Trigger:   "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, gm.StoreSeed(ctx, seed))

	t.Run("existing seed", func(t *testing.T) {
		got, err := gm.RetrieveSeed(ctx, seed.ID)
		assert.NoError(t, err)
		assert.Equal(t, "retrieve-test", got.Name)
		assert.Equal(t, "Retrieve me", got.Content)
		assert.Equal(t, "test", got.Trigger)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := gm.RetrieveSeed(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrSeedNotFound)
	})
}

func TestGardenManager_ListSeeds(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seed := &Seed{
			Name:       "seed-" + string(rune('A'+i)),
			Content:    "content",
			UsageCount: i,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, gm.StoreSeed(ctx, seed))
	}

	t.Run("list all", func(t *testing.T) {
		seeds, err := gm.ListSeeds(ctx, 100)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(seeds))
	})

	t.Run("with limit", func(t *testing.T) {
		seeds, err := gm.ListSeeds(ctx, 3)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(seeds))
	})

	t.Run("default limit", func(t *testing.T) {
		seeds, err := gm.ListSeeds(ctx, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, seeds)
	})

	t.Run("ordered by usage_count desc", func(t *testing.T) {
		seeds, err := gm.ListSeeds(ctx, 100)
		assert.NoError(t, err)
		for i := 0; i < len(seeds)-1; i++ {
			assert.GreaterOrEqual(t, seeds[i].UsageCount, seeds[i+1].UsageCount)
		}
	})
}

func TestGardenManager_SearchSeedsByTrigger(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	seeds := []*Seed{
		{Name: "go-patterns", Content: "Go design patterns", Trigger: "golang,design", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "py-data", Content: "Python data science", Trigger: "python,data", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "go-testing", Content: "Go testing guide", Trigger: "golang,testing", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, s := range seeds {
		require.NoError(t, gm.StoreSeed(ctx, s))
	}

	t.Run("search golang", func(t *testing.T) {
		results, err := gm.SearchSeedsByTrigger(ctx, "golang", 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})

	t.Run("search python", func(t *testing.T) {
		results, err := gm.SearchSeedsByTrigger(ctx, "python", 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
	})

	t.Run("no results", func(t *testing.T) {
		results, err := gm.SearchSeedsByTrigger(ctx, "rust", 10)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("with limit", func(t *testing.T) {
		results, err := gm.SearchSeedsByTrigger(ctx, "golang", 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
	})
}

func TestGardenManager_DeleteSeed(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	seed := &Seed{
		Name:      "to-delete",
		Content:   "delete me",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, gm.StoreSeed(ctx, seed))

	t.Run("delete existing", func(t *testing.T) {
		err := gm.DeleteSeed(ctx, seed.ID)
		assert.NoError(t, err)

		_, err = gm.RetrieveSeed(ctx, seed.ID)
		assert.ErrorIs(t, err, ErrSeedNotFound)
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := gm.DeleteSeed(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrSeedNotFound)
	})
}

func TestGardenManager_IncrementSeedUsage(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	seed := &Seed{
		Name:      "usage-test",
		Content:   "test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, gm.StoreSeed(ctx, seed))

	t.Run("increment", func(t *testing.T) {
		err := gm.IncrementSeedUsage(ctx, seed.ID)
		assert.NoError(t, err)

		got, err := gm.RetrieveSeed(ctx, seed.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.UsageCount)
		assert.NotNil(t, got.LastUsed)
	})

	t.Run("increment multiple", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			require.NoError(t, gm.IncrementSeedUsage(ctx, seed.ID))
		}
		got, err := gm.RetrieveSeed(ctx, seed.ID)
		require.NoError(t, err)
		assert.Equal(t, 6, got.UsageCount)
	})

	t.Run("nonexistent seed", func(t *testing.T) {
		err := gm.IncrementSeedUsage(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrSeedNotFound)
	})
}

func TestGardenManager_CompressHistory(t *testing.T) {
	t.Run("with compression service", func(t *testing.T) {
		gm, mm, _ := setupGardenTestWithCompression(t)
		defer mm.Close()
		ctx := context.Background()

		memories := []Memory{
			{ID: "m1", Type: "user", Content: "Hello"},
			{ID: "m2", Type: "assistant", Content: "Hi there"},
		}

		result, err := gm.CompressHistory(ctx, "session-1", memories)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "session-1", result.SessionID)
	})

	t.Run("without compression service (fallback)", func(t *testing.T) {
		gm, mm := setupGardenTest(t)
		defer mm.Close()
		ctx := context.Background()

		memories := []Memory{
			{ID: "m1", Type: "user", Content: "Hello"},
			{ID: "m2", Type: "assistant", Content: "Hi there"},
		}

		result, err := gm.CompressHistory(ctx, "session-1", memories)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "session-1", result.SessionID)
		assert.NotEmpty(t, result.CompressedContent)
		assert.Equal(t, 2, len(result.OriginalTurnIDs))
	})

	t.Run("empty memories", func(t *testing.T) {
		gm, mm := setupGardenTest(t)
		defer mm.Close()
		ctx := context.Background()

		_, err := gm.CompressHistory(ctx, "session-1", []Memory{})
		assert.Error(t, err)
	})

	t.Run("long content truncation in fallback", func(t *testing.T) {
		gm, mm := setupGardenTest(t)
		defer mm.Close()
		ctx := context.Background()

		longContent := ""
		for i := 0; i < 200; i++ {
			longContent += "word "
		}

		memories := []Memory{
			{ID: "m1", Type: "user", Content: longContent},
		}

		result, err := gm.CompressHistory(ctx, "session-1", memories)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(result.CompressedContent), 504)
	})
}

func TestGardenManager_ExtractSeeds(t *testing.T) {
	t.Run("with compression service", func(t *testing.T) {
		gm, mm, cs := setupGardenTestWithCompression(t)
		defer mm.Close()
		ctx := context.Background()

		cs.extractResult = []*MemorySeed{
			{ID: "s1", Content: "extracted seed"},
		}

		result, err := gm.ExtractSeeds(ctx, []Memory{{ID: "m1", Content: "test"}})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result))
	})

	t.Run("without compression service", func(t *testing.T) {
		gm, mm := setupGardenTest(t)
		defer mm.Close()
		ctx := context.Background()

		result, err := gm.ExtractSeeds(ctx, []Memory{{ID: "m1", Content: "test"}})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})
}

func TestGardenManager_ShouldCompress(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()

	t.Run("below threshold", func(t *testing.T) {
		memories := make([]Memory, 5)
		assert.False(t, gm.ShouldCompress(memories, 10))
	})

	t.Run("at threshold", func(t *testing.T) {
		memories := make([]Memory, 10)
		assert.True(t, gm.ShouldCompress(memories, 10))
	})

	t.Run("above threshold", func(t *testing.T) {
		memories := make([]Memory, 15)
		assert.True(t, gm.ShouldCompress(memories, 10))
	})
}

func TestGardenManager_StoreCompressedHistory(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	t.Run("valid history", func(t *testing.T) {
		history := &CompressedHistory{
			SessionID:         "session-1",
			OriginalTurnIDs:   []string{"t1", "t2"},
			CompressedContent: "compressed",
			CompressionRatio:  0.3,
			CreatedAt:         time.Now(),
		}
		err := gm.StoreCompressedHistory(ctx, history)
		assert.NoError(t, err)
		assert.NotEmpty(t, history.ID)
	})

	t.Run("empty session_id", func(t *testing.T) {
		history := &CompressedHistory{
			OriginalTurnIDs:   []string{"t1"},
			CompressedContent: "test",
			CompressionRatio:  0.5,
			CreatedAt:         time.Now(),
		}
		err := gm.StoreCompressedHistory(ctx, history)
		assert.Error(t, err)
	})

	t.Run("invalid compression_ratio", func(t *testing.T) {
		history := &CompressedHistory{
			SessionID:         "s1",
			OriginalTurnIDs:   []string{"t1"},
			CompressedContent: "test",
			CompressionRatio:  1.5,
			CreatedAt:         time.Now(),
		}
		err := gm.StoreCompressedHistory(ctx, history)
		assert.Error(t, err)
	})
}

func TestGardenManager_RetrieveCompressedHistory(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	history := &CompressedHistory{
		SessionID:         "session-1",
		OriginalTurnIDs:   []string{"t1", "t2"},
		CompressedContent: "compressed content",
		CompressionRatio:  0.3,
		CreatedAt:         time.Now(),
	}
	require.NoError(t, gm.StoreCompressedHistory(ctx, history))

	t.Run("retrieve existing", func(t *testing.T) {
		histories, err := gm.RetrieveCompressedHistory(ctx, "session-1")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(histories))
		assert.Equal(t, "compressed content", histories[0].CompressedContent)
		assert.Equal(t, 2, len(histories[0].OriginalTurnIDs))
	})

	t.Run("retrieve nonexistent session", func(t *testing.T) {
		histories, err := gm.RetrieveCompressedHistory(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(histories))
	})
}

func TestGardenManager_DeleteCompressedHistory(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	history := &CompressedHistory{
		SessionID:         "session-1",
		OriginalTurnIDs:   []string{"t1"},
		CompressedContent: "test",
		CompressionRatio:  0.5,
		CreatedAt:         time.Now(),
	}
	require.NoError(t, gm.StoreCompressedHistory(ctx, history))

	t.Run("delete existing", func(t *testing.T) {
		err := gm.DeleteCompressedHistory(ctx, history.ID)
		assert.NoError(t, err)
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := gm.DeleteCompressedHistory(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

func TestGardenManager_StoreSnapshot(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	t.Run("valid snapshot", func(t *testing.T) {
		snapshot := &MemorySnapshot{
			SessionID:    "session-1",
			SnapshotName: "checkpoint-1",
			SnapshotData: map[string]interface{}{
				"memories": 10,
				"seeds":    5,
			},
			CreatedAt: time.Now(),
		}
		err := gm.StoreSnapshot(ctx, snapshot)
		assert.NoError(t, err)
		assert.NotEmpty(t, snapshot.ID)
	})

	t.Run("empty session_id", func(t *testing.T) {
		snapshot := &MemorySnapshot{
			SnapshotData: map[string]interface{}{"test": true},
			CreatedAt:    time.Now(),
		}
		err := gm.StoreSnapshot(ctx, snapshot)
		assert.Error(t, err)
	})
}

func TestGardenManager_RetrieveSnapshot(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	snapshot := &MemorySnapshot{
		SessionID:    "session-1",
		SnapshotName: "test-snap",
		SnapshotData: map[string]interface{}{"key": "value"},
		CreatedAt:    time.Now(),
	}
	require.NoError(t, gm.StoreSnapshot(ctx, snapshot))

	t.Run("retrieve existing", func(t *testing.T) {
		got, err := gm.RetrieveSnapshot(ctx, snapshot.ID)
		assert.NoError(t, err)
		assert.Equal(t, "session-1", got.SessionID)
		assert.Equal(t, "test-snap", got.SnapshotName)
		assert.Equal(t, "value", got.SnapshotData["key"])
	})

	t.Run("retrieve nonexistent", func(t *testing.T) {
		_, err := gm.RetrieveSnapshot(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

func TestGardenManager_ListSnapshots(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		snapshot := &MemorySnapshot{
			SessionID:    "session-1",
			SnapshotName: "snap-" + string(rune('A'+i)),
			SnapshotData: map[string]interface{}{"index": i},
			CreatedAt:    time.Now(),
		}
		require.NoError(t, gm.StoreSnapshot(ctx, snapshot))
	}

	t.Run("list snapshots", func(t *testing.T) {
		snapshots, err := gm.ListSnapshots(ctx, "session-1")
		assert.NoError(t, err)
		assert.Equal(t, 3, len(snapshots))
	})

	t.Run("empty session", func(t *testing.T) {
		snapshots, err := gm.ListSnapshots(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(snapshots))
	})
}

func TestGardenManager_DeleteSnapshot(t *testing.T) {
	gm, mm := setupGardenTest(t)
	defer mm.Close()
	ctx := context.Background()

	snapshot := &MemorySnapshot{
		SessionID:    "session-1",
		SnapshotData: map[string]interface{}{"key": "val"},
		CreatedAt:    time.Now(),
	}
	require.NoError(t, gm.StoreSnapshot(ctx, snapshot))

	t.Run("delete existing", func(t *testing.T) {
		err := gm.DeleteSnapshot(ctx, snapshot.ID)
		assert.NoError(t, err)
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := gm.DeleteSnapshot(ctx, "nonexistent")
		assert.Error(t, err)
	})
}
