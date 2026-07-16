package memory

import (
	"context"
	"fmt"
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

// TestSearchMemories_TokenBased reproduces the live defect (a substring-only
// `content LIKE '%query%'` search contracted as semantic) using the exact shape of
// the real-world entry that surfaced it: an IP-strategy decision whose words are
// all present but never contiguous the way a natural-language question phrases
// them.
func TestSearchMemories_TokenBased(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	tradeSecret := &Memory{
		ID:   "trade-secret-decision",
		Type: "general",
		Content: "DECISION 2026-07-01 (IP strategy - LOCKED default): ALL TresPies software IP is kept " +
			"as a TRADE SECRET by default, until it is either deliberately open-sourced or patented.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, mm.StoreMemory(ctx, tradeSecret))

	unrelated := &Memory{
		ID:        "unrelated",
		Type:      "general",
		Content:   "Reminder: renew the domain registration before it expires next month.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, mm.StoreMemory(ctx, unrelated))

	t.Run("exact substring still matches (regression guard)", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "trade secret", 10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "trade-secret-decision", results[0].Memory.ID)
	})

	t.Run("exact substring match is case-insensitive", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "TRADE", 10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "trade-secret-decision", results[0].Memory.ID)

		results, err = mm.SearchMemories(ctx, "trade secret", 10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "trade-secret-decision", results[0].Memory.ID)
	})

	t.Run("natural language query matches scattered tokens (the reported bug)", func(t *testing.T) {
		// None of these words are contiguous in the stored content in this
		// order -- the old `content LIKE '%query%'` implementation returned
		// zero hits for exactly this query against exactly this content.
		results, err := mm.SearchMemories(ctx, "IP strategy trade secret locked default", 10)
		require.NoError(t, err)
		require.NotEmpty(t, results, "natural-language query should match despite no contiguous phrase match")
		assert.Equal(t, "trade-secret-decision", results[0].Memory.ID, "the fully-covered entry should rank first")
		assert.Greater(t, results[0].Similarity, 0.0, "RelevanceScore/Similarity must no longer be hardcoded to zero")
	})

	t.Run("unrelated content is never returned for trade-secret queries", func(t *testing.T) {
		results, err := mm.SearchMemories(ctx, "IP strategy trade secret locked default", 10)
		require.NoError(t, err)
		for _, r := range results {
			assert.NotEqual(t, "unrelated", r.Memory.ID)
		}
	})

	t.Run("query with zero lexical overlap still misses (semantic gap, by design)", func(t *testing.T) {
		// "intellectual property" shares no substring with "IP" in the stored
		// text. Token matching is still exact/lexical, not semantic --
		// closing this specific gap needs embeddings, which this codebase
		// does not have and which are explicitly out of scope for this fix.
		results, err := mm.SearchMemories(ctx, "intellectual property", 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("partial token overlap still matches, ranked below full coverage", func(t *testing.T) {
		// A single shared token ("default") is present in the trade-secret
		// entry; requiring the whole phrase (old behavior) or zero tokens
		// (a no-op search) are the two failure modes this guards against.
		results, err := mm.SearchMemories(ctx, "default", 10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "trade-secret-decision", results[0].Memory.ID)
	})
}

func TestCountMemories(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 5; i++ {
		mem := &Memory{
			ID:          fmt.Sprintf("count-%d", i),
			Type:        "conversation",
			Content:     fmt.Sprintf("entry %d", i),
			ContextType: "session",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))
	}

	t.Run("true total ignores limit (the reported list-total bug)", func(t *testing.T) {
		// GET /v1/memory's TotalCount used to equal len(page): a caller
		// listing with limit=2 saw TotalCount=2 even though the store held
		// 5, indistinguishable from "there are only 2 memories total".
		page, err := mm.ListMemories(ctx, MemoryFilter{Limit: 2})
		require.NoError(t, err)
		require.Len(t, page, 2)

		total, err := mm.CountMemories(ctx, MemoryFilter{})
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.NotEqual(t, len(page), total, "total must not silently equal the page size")
	})

	t.Run("total respects filter predicates", func(t *testing.T) {
		total, err := mm.CountMemories(ctx, MemoryFilter{ContextType: "session"})
		require.NoError(t, err)
		assert.Equal(t, 5, total)

		total, err = mm.CountMemories(ctx, MemoryFilter{ContextType: "nonexistent"})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
	})
}

func TestListMemories_Offset(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	base := time.Now()
	ids := []string{"o0", "o1", "o2", "o3", "o4"}
	for i, id := range ids {
		mem := &Memory{
			ID:        id,
			Type:      "conversation",
			Content:   "offset test " + id,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
			UpdatedAt: base.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))
	}

	t.Run("offset actually skips rows instead of being echoed and ignored", func(t *testing.T) {
		firstPage, err := mm.ListMemories(ctx, MemoryFilter{Limit: 2, Offset: 0})
		require.NoError(t, err)
		require.Len(t, firstPage, 2)

		secondPage, err := mm.ListMemories(ctx, MemoryFilter{Limit: 2, Offset: 2})
		require.NoError(t, err)
		require.Len(t, secondPage, 2)

		assert.NotEqual(t, firstPage[0].ID, secondPage[0].ID)
		assert.NotEqual(t, firstPage[1].ID, secondPage[0].ID)
		assert.NotEqual(t, firstPage[0].ID, secondPage[1].ID)
	})

	t.Run("offset past the end returns empty, not an error", func(t *testing.T) {
		page, err := mm.ListMemories(ctx, MemoryFilter{Limit: 2, Offset: 100})
		require.NoError(t, err)
		assert.Empty(t, page)
	})
}

func TestSearchMemoriesPage(t *testing.T) {
	mm, _ := setupTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		mem := &Memory{
			ID:        fmt.Sprintf("page-%d", i),
			Type:      "conversation",
			Content:   "golang tips and tricks",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))
	}

	t.Run("total reflects all matches, not just the page", func(t *testing.T) {
		page, total, err := mm.SearchMemoriesPage(ctx, "golang", 1, 0)
		require.NoError(t, err)
		assert.Len(t, page, 1)
		assert.Equal(t, 3, total)
	})

	t.Run("offset pages through matches without gaps or repeats", func(t *testing.T) {
		seen := make(map[string]bool)
		for offset := 0; offset < 3; offset++ {
			page, total, err := mm.SearchMemoriesPage(ctx, "golang", 1, offset)
			require.NoError(t, err)
			require.Len(t, page, 1)
			assert.Equal(t, 3, total)
			seen[page[0].Memory.ID] = true
		}
		assert.Len(t, seen, 3, "each offset should surface a distinct entry")
	})

	t.Run("empty query behaves like plain listing", func(t *testing.T) {
		page, total, err := mm.SearchMemoriesPage(ctx, "", 2, 0)
		require.NoError(t, err)
		assert.Len(t, page, 2)
		assert.Equal(t, 3, total)
	})
}

func TestTokenizeQuery(t *testing.T) {
	t.Run("splits on punctuation and whitespace", func(t *testing.T) {
		tokens := tokenizeQuery("IP strategy, trade-secret! locked_default")
		assert.Equal(t, []string{"ip", "strategy", "trade", "secret", "locked", "default"}, tokens)
	})

	t.Run("lowercases", func(t *testing.T) {
		tokens := tokenizeQuery("TRADE Secret")
		assert.Equal(t, []string{"trade", "secret"}, tokens)
	})

	t.Run("de-duplicates repeated words", func(t *testing.T) {
		tokens := tokenizeQuery("golang golang programming")
		assert.Equal(t, []string{"golang", "programming"}, tokens)
	})

	t.Run("empty or punctuation-only query yields no tokens", func(t *testing.T) {
		assert.Empty(t, tokenizeQuery("   "))
		assert.Empty(t, tokenizeQuery("..."))
	})
}
