package agent

import (
	"sync"
	"time"

	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

// ConversationContext stores the state of a multi-turn conversation
type ConversationContext struct {
	SessionID     string
	UserID        string
	Messages      []providerpkg.Message
	TotalTokens   int
	CreatedAt     time.Time
	LastUpdatedAt time.Time
}

// ContextManager manages conversation contexts across sessions
type ContextManager struct {
	contexts map[string]*ConversationContext
	mu       sync.RWMutex
	maxAge   time.Duration
}

// NewContextManager creates a new context manager
func NewContextManager(maxAge time.Duration) *ContextManager {
	cm := &ContextManager{
		contexts: make(map[string]*ConversationContext),
		maxAge:   maxAge,
	}

	// Start cleanup goroutine
	go cm.cleanupExpired()

	return cm
}

// GetOrCreate retrieves an existing context or creates a new one
func (cm *ContextManager) GetOrCreate(sessionID, userID string) *ConversationContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if ctx, exists := cm.contexts[sessionID]; exists {
		ctx.LastUpdatedAt = time.Now()
		return ctx
	}

	ctx := &ConversationContext{
		SessionID:     sessionID,
		UserID:        userID,
		Messages:      []providerpkg.Message{},
		TotalTokens:   0,
		CreatedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
	}

	cm.contexts[sessionID] = ctx
	return ctx
}

// AddMessage adds a message to the conversation context
func (cm *ContextManager) AddMessage(sessionID string, message providerpkg.Message, tokens int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if ctx, exists := cm.contexts[sessionID]; exists {
		ctx.Messages = append(ctx.Messages, message)
		ctx.TotalTokens += tokens
		ctx.LastUpdatedAt = time.Now()

		// Limit context window to prevent unbounded growth
		// Keep last 10 messages or 8000 tokens, whichever comes first
		cm.trimContext(ctx)
	}
}

// trimContext removes old messages if context is too large
func (cm *ContextManager) trimContext(ctx *ConversationContext) {
	const maxMessages = 10
	const maxTokens = 8000

	// Trim by message count
	if len(ctx.Messages) > maxMessages {
		ctx.Messages = ctx.Messages[len(ctx.Messages)-maxMessages:]
	}

	// Trim by token count (estimate)
	for ctx.TotalTokens > maxTokens && len(ctx.Messages) > 2 {
		// Remove oldest message (but keep at least 2 for context)
		removed := ctx.Messages[0]
		ctx.Messages = ctx.Messages[1:]
		// Rough estimate: 1 token ≈ 4 characters
		ctx.TotalTokens -= len(removed.Content) / 4
	}
}

// GetContext retrieves a conversation context
func (cm *ContextManager) GetContext(sessionID string) *ConversationContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.contexts[sessionID]
}

// Clear removes a conversation context
func (cm *ContextManager) Clear(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.contexts, sessionID)
}

// cleanupExpired removes old conversation contexts
func (cm *ContextManager) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for sessionID, ctx := range cm.contexts {
			if now.Sub(ctx.LastUpdatedAt) > cm.maxAge {
				delete(cm.contexts, sessionID)
			}
		}
		cm.mu.Unlock()
	}
}

// Stats returns statistics about conversation contexts
func (cm *ContextManager) Stats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalSessions := len(cm.contexts)
	totalMessages := 0
	totalTokens := 0

	for _, ctx := range cm.contexts {
		totalMessages += len(ctx.Messages)
		totalTokens += ctx.TotalTokens
	}

	avgMessages := 0.0
	avgTokens := 0.0
	if totalSessions > 0 {
		avgMessages = float64(totalMessages) / float64(totalSessions)
		avgTokens = float64(totalTokens) / float64(totalSessions)
	}

	return map[string]interface{}{
		"total_sessions": totalSessions,
		"total_messages": totalMessages,
		"total_tokens":   totalTokens,
		"avg_messages":   avgMessages,
		"avg_tokens":     avgTokens,
	}
}
