package skill

import (
	"context"
	"fmt"
)

// contextKey is the type for skill-related context keys
type contextKey string

const (
	// callDepthKey is the context key for tracking skill invocation depth
	callDepthKey contextKey = "skill_call_depth"

	// MaxMetaSkillDepth is the maximum allowed depth for meta-skill invocations
	MaxMetaSkillDepth = 3
)

// GetCallDepth returns the current call depth from context.
// Returns 0 if depth is not set in the context.
func GetCallDepth(ctx context.Context) int {
	depth, ok := ctx.Value(callDepthKey).(int)
	if !ok {
		return 0
	}
	return depth
}

// WithIncrementedDepth returns a new context with call depth incremented by 1.
func WithIncrementedDepth(ctx context.Context) context.Context {
	currentDepth := GetCallDepth(ctx)
	return context.WithValue(ctx, callDepthKey, currentDepth+1)
}

// CheckDepthLimit returns ErrMaxDepthExceeded if the current depth >= maxDepth.
func CheckDepthLimit(ctx context.Context, maxDepth int) error {
	currentDepth := GetCallDepth(ctx)
	if currentDepth >= maxDepth {
		return fmt.Errorf("%w: current depth %d exceeds maximum %d", ErrMaxDepthExceeded, currentDepth, maxDepth)
	}
	return nil
}
