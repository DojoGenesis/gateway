package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindRelevantSeeds(t *testing.T) {
	seeds := []Seed{
		{ID: "s1", Name: "golang-patterns", Content: "Go patterns", Trigger: "golang,go,patterns"},
		{ID: "s2", Name: "python-data", Content: "Python data science", Trigger: "python,data science"},
		{ID: "s3", Name: "testing-guide", Content: "Testing practices", Trigger: "testing,test"},
		{ID: "s4", Name: "deployment", Content: "Deployment guide", Trigger: "deploy,ci/cd"},
	}

	t.Run("match by trigger", func(t *testing.T) {
		results := FindRelevantSeeds("help me with golang testing", seeds)
		assert.Equal(t, 2, len(results))
	})

	t.Run("match by name", func(t *testing.T) {
		results := FindRelevantSeeds("I need the deployment guide", seeds)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "s4", results[0].ID)
	})

	t.Run("case insensitive", func(t *testing.T) {
		results := FindRelevantSeeds("GOLANG patterns", seeds)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("no matches", func(t *testing.T) {
		results := FindRelevantSeeds("machine learning", seeds)
		assert.Equal(t, 0, len(results))
	})

	t.Run("empty query", func(t *testing.T) {
		results := FindRelevantSeeds("", seeds)
		assert.Equal(t, 0, len(results))
	})

	t.Run("empty seeds", func(t *testing.T) {
		results := FindRelevantSeeds("golang", []Seed{})
		assert.Equal(t, 0, len(results))
	})

	t.Run("seed without trigger matches by name", func(t *testing.T) {
		seedsNoTrigger := []Seed{
			{ID: "s1", Name: "golang", Content: "Go language"},
		}
		results := FindRelevantSeeds("tell me about golang", seedsNoTrigger)
		assert.Equal(t, 1, len(results))
	})

	t.Run("comma-separated triggers", func(t *testing.T) {
		results := FindRelevantSeeds("python scripts", seeds)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "s2", results[0].ID)
	})

	t.Run("multiple matches", func(t *testing.T) {
		results := FindRelevantSeeds("golang testing patterns go", seeds)
		assert.GreaterOrEqual(t, len(results), 2)
	})
}

func TestIsRelevant(t *testing.T) {
	t.Run("match by trigger", func(t *testing.T) {
		seed := Seed{Name: "test", Trigger: "golang,testing"}
		assert.True(t, isRelevant("help with golang", seed))
	})

	t.Run("match by name", func(t *testing.T) {
		seed := Seed{Name: "golang", Trigger: ""}
		assert.True(t, isRelevant("tell me about golang", seed))
	})

	t.Run("no match", func(t *testing.T) {
		seed := Seed{Name: "python", Trigger: "python,data"}
		assert.False(t, isRelevant("tell me about rust", seed))
	})

	t.Run("empty trigger and name no match", func(t *testing.T) {
		seed := Seed{Name: "", Trigger: ""}
		assert.False(t, isRelevant("anything", seed))
	})

	t.Run("whitespace in triggers", func(t *testing.T) {
		seed := Seed{Name: "test", Trigger: " golang , testing "}
		assert.True(t, isRelevant("golang programming", seed))
	})
}
