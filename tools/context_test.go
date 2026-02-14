package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithProjectID(t *testing.T) {
	ctx := WithProjectID(context.Background(), "project-123")
	projectID := GetProjectIDFromContext(ctx)
	assert.Equal(t, "project-123", projectID)
}

func TestGetProjectIDFromContext_NotFound(t *testing.T) {
	projectID := GetProjectIDFromContext(context.Background())
	assert.Equal(t, "", projectID)
}

func TestGetProjectIDFromContext_EmptyString(t *testing.T) {
	ctx := WithProjectID(context.Background(), "")
	projectID := GetProjectIDFromContext(ctx)
	assert.Equal(t, "", projectID)
}

func TestInjectProjectIDIntoParams(t *testing.T) {
	t.Run("injects when not present", func(t *testing.T) {
		ctx := WithProjectID(context.Background(), "proj-abc")
		params := map[string]interface{}{"query": "test"}

		result := injectProjectIDIntoParams(ctx, params)
		assert.Equal(t, "proj-abc", result["project_id"])
		assert.Equal(t, "test", result["query"])
	})

	t.Run("does not override existing project_id", func(t *testing.T) {
		ctx := WithProjectID(context.Background(), "from-context")
		params := map[string]interface{}{
			"query":      "test",
			"project_id": "from-params",
		}

		result := injectProjectIDIntoParams(ctx, params)
		assert.Equal(t, "from-params", result["project_id"])
	})

	t.Run("no injection when context has no project_id", func(t *testing.T) {
		params := map[string]interface{}{"query": "test"}
		result := injectProjectIDIntoParams(context.Background(), params)
		_, exists := result["project_id"]
		assert.False(t, exists)
	})

	t.Run("does not mutate original params", func(t *testing.T) {
		ctx := WithProjectID(context.Background(), "proj-xyz")
		original := map[string]interface{}{"query": "test"}

		result := injectProjectIDIntoParams(ctx, original)

		// Result should have project_id
		assert.Equal(t, "proj-xyz", result["project_id"])
		// Original should NOT have project_id
		_, exists := original["project_id"]
		assert.False(t, exists)
	})
}

func TestProjectIDContextThreadSafety(t *testing.T) {
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := WithProjectID(context.Background(), "project-"+string(rune('A'+id)))
			pid := GetProjectIDFromContext(ctx)
			assert.NotEmpty(t, pid)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
