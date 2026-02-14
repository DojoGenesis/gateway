package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShouldCompress_PackageLevel(t *testing.T) {
	t.Run("below threshold", func(t *testing.T) {
		memories := make([]Memory, 5)
		assert.False(t, ShouldCompress(memories, 10))
	})

	t.Run("at threshold", func(t *testing.T) {
		memories := make([]Memory, 10)
		assert.True(t, ShouldCompress(memories, 10))
	})

	t.Run("above threshold", func(t *testing.T) {
		memories := make([]Memory, 15)
		assert.True(t, ShouldCompress(memories, 10))
	})

	t.Run("empty memories", func(t *testing.T) {
		assert.False(t, ShouldCompress([]Memory{}, 1))
	})

	t.Run("zero threshold", func(t *testing.T) {
		assert.True(t, ShouldCompress([]Memory{{}}, 0))
	})
}

func TestGetOldMemories(t *testing.T) {
	now := time.Now()
	memories := []Memory{
		{ID: "m1", Content: "first", CreatedAt: now.Add(-3 * time.Hour)},
		{ID: "m2", Content: "second", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "m3", Content: "third", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "m4", Content: "fourth", CreatedAt: now},
	}

	t.Run("keep 2 of 4", func(t *testing.T) {
		old := GetOldMemories(memories, 2)
		assert.Equal(t, 2, len(old))
		assert.Equal(t, "m1", old[0].ID)
		assert.Equal(t, "m2", old[1].ID)
	})

	t.Run("keep all", func(t *testing.T) {
		old := GetOldMemories(memories, 4)
		assert.Equal(t, 0, len(old))
	})

	t.Run("keep more than available", func(t *testing.T) {
		old := GetOldMemories(memories, 10)
		assert.Equal(t, 0, len(old))
	})

	t.Run("keep zero", func(t *testing.T) {
		old := GetOldMemories(memories, 0)
		assert.Equal(t, 4, len(old))
	})

	t.Run("empty memories", func(t *testing.T) {
		old := GetOldMemories([]Memory{}, 5)
		assert.Equal(t, 0, len(old))
	})
}

func TestGetRecentMemories(t *testing.T) {
	memories := []Memory{
		{ID: "m1", Content: "first"},
		{ID: "m2", Content: "second"},
		{ID: "m3", Content: "third"},
		{ID: "m4", Content: "fourth"},
	}

	t.Run("get last 2", func(t *testing.T) {
		recent := GetRecentMemories(memories, 2)
		assert.Equal(t, 2, len(recent))
		assert.Equal(t, "m3", recent[0].ID)
		assert.Equal(t, "m4", recent[1].ID)
	})

	t.Run("get more than available", func(t *testing.T) {
		recent := GetRecentMemories(memories, 10)
		assert.Equal(t, 4, len(recent))
	})

	t.Run("get all", func(t *testing.T) {
		recent := GetRecentMemories(memories, 4)
		assert.Equal(t, 4, len(recent))
	})

	t.Run("empty memories", func(t *testing.T) {
		recent := GetRecentMemories([]Memory{}, 5)
		assert.Equal(t, 0, len(recent))
	})
}

func TestBuildOriginalContent(t *testing.T) {
	now := time.Now()
	memories := []Memory{
		{Type: "user", Content: "Hello", CreatedAt: now},
		{Type: "assistant", Content: "Hi there", CreatedAt: now},
	}

	t.Run("builds content string", func(t *testing.T) {
		content := BuildOriginalContent(memories)
		assert.Contains(t, content, "[user]")
		assert.Contains(t, content, "Hello")
		assert.Contains(t, content, "[assistant]")
		assert.Contains(t, content, "Hi there")
	})

	t.Run("empty memories", func(t *testing.T) {
		content := BuildOriginalContent([]Memory{})
		assert.Equal(t, "", content)
	})
}

func TestBuildCompressionPrompt(t *testing.T) {
	content := "Some conversation content"

	t.Run("contains 3-Month Rule", func(t *testing.T) {
		prompt := BuildCompressionPrompt(content)
		assert.Contains(t, prompt, "3-Month Rule")
	})

	t.Run("contains original content", func(t *testing.T) {
		prompt := BuildCompressionPrompt(content)
		assert.Contains(t, prompt, content)
	})

	t.Run("contains instructions", func(t *testing.T) {
		prompt := BuildCompressionPrompt(content)
		assert.True(t, strings.Contains(prompt, "decisions") || strings.Contains(prompt, "lessons"))
	})
}
