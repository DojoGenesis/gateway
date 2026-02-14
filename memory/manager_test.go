package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*MemoryManager, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)

	return mm, dbPath
}

func TestNewMemoryManager(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)
	require.NotNil(t, mm)
	defer mm.Close()

	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestNewMemoryManager_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)
	defer mm.Close()

	assert.NotNil(t, mm)
}

func TestStoreMemory(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	t.Run("valid memory", func(t *testing.T) {
		mem := &Memory{
			ID:        "mem-1",
			Type:      "conversation",
			Content:   "Hello world",
			Metadata:  map[string]interface{}{"key": "value"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := mm.StoreMemory(ctx, mem)
		assert.NoError(t, err)
	})

	t.Run("default context_type", func(t *testing.T) {
		mem := &Memory{
			ID:        "mem-2",
			Type:      "conversation",
			Content:   "Test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := mm.StoreMemory(ctx, mem)
		assert.NoError(t, err)
		assert.Equal(t, "conversation", mem.ContextType)
	})

	t.Run("nil memory", func(t *testing.T) {
		err := mm.StoreMemory(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("empty ID", func(t *testing.T) {
		mem := &Memory{Type: "conversation", Content: "test"}
		err := mm.StoreMemory(ctx, mem)
		assert.Error(t, err)
	})

	t.Run("empty type", func(t *testing.T) {
		mem := &Memory{ID: "x", Content: "test"}
		err := mm.StoreMemory(ctx, mem)
		assert.Error(t, err)
	})

	t.Run("upsert on conflict", func(t *testing.T) {
		mem := &Memory{
			ID:        "mem-upsert",
			Type:      "conversation",
			Content:   "original",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))

		mem.Content = "updated"
		mem.UpdatedAt = time.Now()
		require.NoError(t, mm.StoreMemory(ctx, mem))

		got, err := mm.GetMemory(ctx, "mem-upsert")
		require.NoError(t, err)
		assert.Equal(t, "updated", got.Content)
	})

	t.Run("store with embedding", func(t *testing.T) {
		emb := make([]float32, 768)
		for i := range emb {
			emb[i] = float32(i) / 768.0
		}
		mem := &Memory{
			ID:        "mem-emb",
			Type:      "conversation",
			Content:   "embedded",
			Embedding: emb,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))

		got, err := mm.GetMemory(ctx, "mem-emb")
		require.NoError(t, err)
		assert.Equal(t, 768, len(got.Embedding))
	})
}

func TestGetMemory(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	now := time.Now()
	mem := &Memory{
		ID:        "get-1",
		Type:      "conversation",
		Content:   "retrievable",
		Metadata:  map[string]interface{}{"foo": "bar"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, mm.StoreMemory(ctx, mem))

	t.Run("existing", func(t *testing.T) {
		got, err := mm.GetMemory(ctx, "get-1")
		assert.NoError(t, err)
		assert.Equal(t, "get-1", got.ID)
		assert.Equal(t, "conversation", got.Type)
		assert.Equal(t, "retrievable", got.Content)
		assert.Equal(t, "bar", got.Metadata["foo"])
	})

	t.Run("not found", func(t *testing.T) {
		_, err := mm.GetMemory(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrMemoryNotFound)
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := mm.GetMemory(ctx, "")
		assert.Error(t, err)
	})
}

func TestUpdateMemory(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	mem := &Memory{
		ID:        "update-1",
		Type:      "conversation",
		Content:   "original",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, mm.StoreMemory(ctx, mem))

	t.Run("update existing", func(t *testing.T) {
		mem.Content = "updated content"
		mem.Metadata = map[string]interface{}{"new": "meta"}
		err := mm.UpdateMemory(ctx, mem)
		assert.NoError(t, err)

		got, err := mm.GetMemory(ctx, "update-1")
		require.NoError(t, err)
		assert.Equal(t, "updated content", got.Content)
		assert.Equal(t, "meta", got.Metadata["new"])
	})

	t.Run("update nonexistent", func(t *testing.T) {
		nonexistent := &Memory{
			ID:      "nonexistent",
			Type:    "conversation",
			Content: "test",
		}
		err := mm.UpdateMemory(ctx, nonexistent)
		assert.ErrorIs(t, err, ErrMemoryNotFound)
	})

	t.Run("nil memory", func(t *testing.T) {
		err := mm.UpdateMemory(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("empty ID", func(t *testing.T) {
		err := mm.UpdateMemory(ctx, &Memory{Content: "test"})
		assert.Error(t, err)
	})
}

func TestDeleteMemory(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	mem := &Memory{
		ID:        "delete-1",
		Type:      "conversation",
		Content:   "to be deleted",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, mm.StoreMemory(ctx, mem))

	t.Run("delete existing", func(t *testing.T) {
		err := mm.DeleteMemory(ctx, "delete-1")
		assert.NoError(t, err)

		_, err = mm.GetMemory(ctx, "delete-1")
		assert.ErrorIs(t, err, ErrMemoryNotFound)
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		err := mm.DeleteMemory(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrMemoryNotFound)
	})

	t.Run("empty id", func(t *testing.T) {
		err := mm.DeleteMemory(ctx, "")
		assert.Error(t, err)
	})
}

func TestSearchMemories(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	memories := []*Memory{
		{ID: "s1", Type: "conversation", Content: "golang programming tips", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "s2", Type: "conversation", Content: "python data science", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "s3", Type: "conversation", Content: "golang performance tuning", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, m := range memories {
		require.NoError(t, mm.StoreMemory(ctx, m))
	}

	t.Run("keyword search", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "golang", 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(results))
		for _, r := range results {
			assert.Equal(t, "text", r.SearchMode)
		}
	})

	t.Run("with limit", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "golang", 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
	})

	t.Run("empty query returns all", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "", 10)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(results))
	})

	t.Run("no matches", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "nonexistent_keyword_xyz", 10)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("snippet truncation", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 300; i++ {
			longContent += "word "
		}
		longMem := &Memory{
			ID:        "long",
			Type:      "conversation",
			Content:   longContent,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, mm.StoreMemory(ctx, longMem))

		results, err := mm.SearchMemories(ctx, "word", 10)
		assert.NoError(t, err)
		for _, r := range results {
			if r.Memory.ID == "long" {
				assert.LessOrEqual(t, len(r.Snippet), 204)
			}
		}
	})

	t.Run("default limit", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "golang", 0)
		assert.NoError(t, err)
		assert.NotNil(t, results)
	})
}

func TestListMemories(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	now := time.Now()
	memories := []*Memory{
		{ID: "l1", Type: "conversation", Content: "c1", ContextType: "session", CreatedAt: now, UpdatedAt: now},
		{ID: "l2", Type: "note", Content: "c2", ContextType: "session", CreatedAt: now, UpdatedAt: now},
		{ID: "l3", Type: "conversation", Content: "c3", ContextType: "global", CreatedAt: now, UpdatedAt: now},
	}
	for _, m := range memories {
		require.NoError(t, mm.StoreMemory(ctx, m))
	}

	t.Run("filter by type", func(t *testing.T) {
		result, err := mm.ListMemories(ctx, MemoryFilter{Type: "conversation"})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
	})

	t.Run("filter by context_type", func(t *testing.T) {
		result, err := mm.ListMemories(ctx, MemoryFilter{ContextType: "session"})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
	})

	t.Run("filter by created_after", func(t *testing.T) {
		past := now.Add(-time.Hour)
		result, err := mm.ListMemories(ctx, MemoryFilter{CreatedAfter: &past})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(result))

		future := now.Add(time.Hour)
		result, err = mm.ListMemories(ctx, MemoryFilter{CreatedAfter: &future})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))
	})

	t.Run("with limit", func(t *testing.T) {
		result, err := mm.ListMemories(ctx, MemoryFilter{Limit: 2})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))
	})

	t.Run("default limit", func(t *testing.T) {
		result, err := mm.ListMemories(ctx, MemoryFilter{})
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(result), 100)
	})
}

func TestSearchMemoriesSemantic(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	embA := make([]float32, 768)
	embB := make([]float32, 768)
	for i := range embA {
		embA[i] = float32(i) / 768.0
		embB[i] = float32(768-i) / 768.0
	}

	require.NoError(t, mm.StoreMemory(ctx, &Memory{
		ID: "sem-a", Type: "conversation", Content: "golang programming",
		Embedding: embA, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, mm.StoreMemory(ctx, &Memory{
		ID: "sem-b", Type: "conversation", Content: "python scripting",
		Embedding: embB, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, mm.StoreMemory(ctx, &Memory{
		ID: "sem-c", Type: "conversation", Content: "no embedding",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	t.Run("search with embedding", func(t *testing.T) {
		results, err := mm.SearchMemoriesSemantic(ctx, embA, 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(results))
		if len(results) > 1 {
			assert.GreaterOrEqual(t, results[0].Similarity, results[1].Similarity)
		}
		for _, r := range results {
			assert.Equal(t, "semantic", r.SearchMode)
		}
	})

	t.Run("limit results", func(t *testing.T) {
		results, err := mm.SearchMemoriesSemantic(ctx, embA, 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
	})

	t.Run("empty embedding error", func(t *testing.T) {
		_, err := mm.SearchMemoriesSemantic(ctx, []float32{}, 10)
		assert.Error(t, err)
	})
}

func TestEstimateTokens(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.Equal(t, 0, EstimateTokens(""))
	})

	t.Run("short text", func(t *testing.T) {
		tokens := EstimateTokens("hello world")
		assert.Greater(t, tokens, 0)
	})

	t.Run("long text", func(t *testing.T) {
		text := ""
		for i := 0; i < 100; i++ {
			text += "word "
		}
		tokens := EstimateTokens(text)
		assert.Greater(t, tokens, 50)
	})
}

func TestMemoryManagerClose(t *testing.T) {
	mm, _ := setupTestDB(t)
	err := mm.Close()
	assert.NoError(t, err)
}

func TestMemoryManagerDB(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	db := mm.DB()
	assert.NotNil(t, db)
}
