package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCallDepth_NotSet(t *testing.T) {
	ctx := context.Background()
	depth := GetCallDepth(ctx)
	assert.Equal(t, 0, depth)
}

func TestGetCallDepth_Set(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, callDepthKey, 2)
	depth := GetCallDepth(ctx)
	assert.Equal(t, 2, depth)
}

func TestWithIncrementedDepth(t *testing.T) {
	ctx := context.Background()

	// Start at depth 0
	assert.Equal(t, 0, GetCallDepth(ctx))

	// Increment to 1
	ctx = WithIncrementedDepth(ctx)
	assert.Equal(t, 1, GetCallDepth(ctx))

	// Increment to 2
	ctx = WithIncrementedDepth(ctx)
	assert.Equal(t, 2, GetCallDepth(ctx))

	// Increment to 3
	ctx = WithIncrementedDepth(ctx)
	assert.Equal(t, 3, GetCallDepth(ctx))
}

func TestCheckDepthLimit_BelowLimit(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, callDepthKey, 2)

	err := CheckDepthLimit(ctx, 3)
	assert.NoError(t, err)
}

func TestCheckDepthLimit_AtLimit(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, callDepthKey, 3)

	err := CheckDepthLimit(ctx, 3)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	assert.Contains(t, err.Error(), "current depth 3 exceeds maximum 3")
}

func TestCheckDepthLimit_AboveLimit(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, callDepthKey, 5)

	err := CheckDepthLimit(ctx, 3)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
}

func TestMaxMetaSkillDepth_Constant(t *testing.T) {
	assert.Equal(t, 3, MaxMetaSkillDepth)
}
