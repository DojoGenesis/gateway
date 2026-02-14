package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContextTestDB(t *testing.T) (*MemoryManager, *GardenManager) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "context_test.db")

	mm, err := NewMemoryManager(dbPath)
	require.NoError(t, err)

	gm, err := NewGardenManager(mm, nil)
	require.NoError(t, err)

	return mm, gm
}

func TestNewContextBuilder(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()

	cb := NewContextBuilder(gm)
	assert.NotNil(t, cb)
}

func TestContextBuilder_SetContextCapacity(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()

	cb := NewContextBuilder(gm)
	cb.SetContextCapacity(16000)
	assert.Equal(t, 16000, cb.contextCapacity)
}

func TestContextBuilder_BuildContext(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	cb := NewContextBuilder(gm)

	t.Run("basic context build", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "What is golang?", "session-1", "You are a helpful assistant")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.Messages), 0)
		assert.Greater(t, result.TotalTokens, 0)
	})

	t.Run("tier 1 always included", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "Hello", "session-1", "System prompt")
		assert.NoError(t, err)

		// Tier 1 should have system prompt + user query
		tier1Tokens, exists := result.TiersUsed[Tier1AlwaysOn]
		assert.True(t, exists)
		assert.Greater(t, tier1Tokens, 0)

		// First message should be system
		assert.Equal(t, "system", result.Messages[0].Role)
		assert.Equal(t, "System prompt", result.Messages[0].Content)

		// Second message should be user
		assert.Equal(t, "user", result.Messages[1].Role)
		assert.Equal(t, "Hello", result.Messages[1].Content)
	})

	t.Run("without system prompt", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "Hello", "session-1", "")
		assert.NoError(t, err)
		// First message should be user query
		assert.Equal(t, "user", result.Messages[0].Role)
	})

	t.Run("capacity percentage calculated", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "Test", "session-1", "System")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.CapacityPercent, 0.0)
		assert.LessOrEqual(t, result.CapacityPercent, 1.0)
	})
}

func TestContextBuilder_BuildContext_WithSeeds(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	// Store seeds
	seed := &Seed{
		Name:      "golang-tips",
		Content:   "Always use go vet before committing. Use interfaces for testing.",
		Trigger:   "golang,go",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, gm.StoreSeed(ctx, seed))

	cb := NewContextBuilder(gm)

	t.Run("includes relevant seeds", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "help with golang", "session-1", "")
		assert.NoError(t, err)

		// Should have tier 2 tokens from seeds
		tier2Tokens, exists := result.TiersUsed[Tier2OnDemand]
		assert.True(t, exists)
		assert.Greater(t, tier2Tokens, 0)

		// Find seed message
		foundSeedMsg := false
		for _, msg := range result.Messages {
			if msg.Role == "system" && len(msg.Content) > 0 {
				if assert.ObjectsAreEqual("system", msg.Role) {
					foundSeedMsg = true
				}
			}
		}
		assert.True(t, foundSeedMsg)
	})

	t.Run("no seeds for unrelated query", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "what is the weather", "session-1", "")
		assert.NoError(t, err)

		tier2Tokens := result.TiersUsed[Tier2OnDemand]
		assert.Equal(t, 0, tier2Tokens)
	})
}

func TestContextBuilder_BuildContext_WithMemories(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	// Store some memories
	for i := 0; i < 5; i++ {
		mem := &Memory{
			ID:        "mem-" + string(rune('a'+i)),
			Type:      "user",
			Content:   "Previous message " + string(rune('A'+i)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, mm.StoreMemory(ctx, mem))
	}

	cb := NewContextBuilder(gm)

	t.Run("includes recent memories in tier 3", func(t *testing.T) {
		result, err := cb.BuildContext(ctx, "continue our discussion", "session-1", "")
		assert.NoError(t, err)

		tier3Tokens := result.TiersUsed[Tier3Referenced]
		assert.Greater(t, tier3Tokens, 0)
	})
}

func TestContextBuilder_BuildContext_WithCompressedHistory(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	// Store compressed history
	history := &CompressedHistory{
		SessionID:         "session-1",
		OriginalTurnIDs:   []string{"t1", "t2"},
		CompressedContent: "Previously discussed golang architecture patterns.",
		CompressionRatio:  0.3,
		CreatedAt:         time.Now(),
	}
	require.NoError(t, gm.StoreCompressedHistory(ctx, history))

	cb := NewContextBuilder(gm)

	result, err := cb.BuildContext(ctx, "continue", "session-1", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	tier4Tokens := result.TiersUsed[Tier4Pruned]
	assert.Greater(t, tier4Tokens, 0)
}

func TestContextBuilder_BuildContext_Pruning(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	// Create a very small capacity context builder
	cb := NewContextBuilder(gm)
	cb.SetContextCapacity(50) // Very small to force pruning

	result, err := cb.BuildContext(ctx, "This is a long query that takes up capacity", "session-1", "This is a system prompt")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestContextBuilder_BuildContext_Alert(t *testing.T) {
	mm, gm := setupContextTestDB(t)
	defer mm.Close()
	ctx := context.Background()

	cb := NewContextBuilder(gm)
	cb.SetContextCapacity(10) // Very small capacity

	result, err := cb.BuildContext(ctx, "query", "session-1", "system prompt that is long")
	assert.NoError(t, err)
	// With such small capacity, alert should be triggered
	assert.True(t, result.Alert)
}

func TestEstimateTokensFromText(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.Equal(t, 0, estimateTokensFromText(""))
	})

	t.Run("single word", func(t *testing.T) {
		tokens := estimateTokensFromText("hello")
		assert.Greater(t, tokens, 0)
	})

	t.Run("multiple words", func(t *testing.T) {
		tokens := estimateTokensFromText("hello world how are you")
		assert.Greater(t, tokens, 3)
	})
}

func TestTruncateContent(t *testing.T) {
	t.Run("short content not truncated", func(t *testing.T) {
		result := truncateContent("short", 100)
		assert.Equal(t, "short", result)
	})

	t.Run("long content truncated", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 1000; i++ {
			longContent += "a"
		}
		result := truncateContent(longContent, 10)
		assert.LessOrEqual(t, len(result), 44) // 10*4 + 3 for "..."
	})
}

func TestMessageType(t *testing.T) {
	msg := ContextMessage{
		Role:    "user",
		Content: "Hello",
	}
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content)
}

func TestContextBuildResultType(t *testing.T) {
	result := &ContextBuildResult{
		Messages:        []ContextMessage{{Role: "user", Content: "test"}},
		TiersUsed:       map[ContextTier]int{Tier1AlwaysOn: 10},
		TotalTokens:     10,
		CapacityPercent: 0.5,
		Pruned:          []ContextTier{},
		Alert:           false,
	}
	assert.Equal(t, 1, len(result.Messages))
	assert.Equal(t, 10, result.TiersUsed[Tier1AlwaysOn])
}

func TestContextTierConstants(t *testing.T) {
	assert.Equal(t, ContextTier(1), Tier1AlwaysOn)
	assert.Equal(t, ContextTier(2), Tier2OnDemand)
	assert.Equal(t, ContextTier(3), Tier3Referenced)
	assert.Equal(t, ContextTier(4), Tier4Pruned)
}
