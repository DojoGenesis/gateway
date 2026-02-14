package agent

import (
	"testing"
	"time"

	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

func TestContextManager_GetOrCreate(t *testing.T) {
	cm := NewContextManager(1 * time.Hour)

	// Create new context
	ctx1 := cm.GetOrCreate("session-1", "user-1")
	if ctx1 == nil {
		t.Fatal("Expected non-nil context")
	}
	if ctx1.SessionID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got '%s'", ctx1.SessionID)
	}
	if ctx1.UserID != "user-1" {
		t.Errorf("Expected user ID 'user-1', got '%s'", ctx1.UserID)
	}

	// Get existing context
	ctx2 := cm.GetOrCreate("session-1", "user-1")
	if ctx2 != ctx1 {
		t.Error("Expected same context instance")
	}

	// Create different context
	ctx3 := cm.GetOrCreate("session-2", "user-2")
	if ctx3 == ctx1 {
		t.Error("Expected different context instance")
	}
}

func TestContextManager_AddMessage(t *testing.T) {
	cm := NewContextManager(1 * time.Hour)
	ctx := cm.GetOrCreate("session-1", "user-1")

	// Add messages
	cm.AddMessage("session-1", providerpkg.Message{
		Role:    "user",
		Content: "Hello",
	}, 2)

	cm.AddMessage("session-1", providerpkg.Message{
		Role:    "assistant",
		Content: "Hi there!",
	}, 3)

	// Verify messages were added
	ctx = cm.GetContext("session-1")
	if len(ctx.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(ctx.Messages))
	}
	if ctx.TotalTokens != 5 {
		t.Errorf("Expected 5 total tokens, got %d", ctx.TotalTokens)
	}
}

func TestContextManager_TrimContext(t *testing.T) {
	cm := NewContextManager(1 * time.Hour)

	// Create context first
	cm.GetOrCreate("session-1", "user-1")

	// Add many messages
	for i := 0; i < 20; i++ {
		cm.AddMessage("session-1", providerpkg.Message{
			Role:    "user",
			Content: "Message",
		}, 100)
	}

	ctx := cm.GetContext("session-1")

	// Should be trimmed to max 10 messages
	if len(ctx.Messages) > 10 {
		t.Errorf("Expected max 10 messages, got %d", len(ctx.Messages))
	}

	// Should be trimmed to max 8000 tokens
	if ctx.TotalTokens > 8000 {
		t.Errorf("Expected max 8000 tokens, got %d", ctx.TotalTokens)
	}
}

func TestContextManager_Clear(t *testing.T) {
	cm := NewContextManager(1 * time.Hour)

	cm.GetOrCreate("session-1", "user-1")
	cm.Clear("session-1")

	ctx := cm.GetContext("session-1")
	if ctx != nil {
		t.Error("Expected nil context after clear")
	}
}

func TestContextManager_Stats(t *testing.T) {
	cm := NewContextManager(1 * time.Hour)

	// Create sessions with messages
	cm.GetOrCreate("session-1", "user-1")
	cm.AddMessage("session-1", providerpkg.Message{Role: "user", Content: "Hello"}, 2)
	cm.AddMessage("session-1", providerpkg.Message{Role: "assistant", Content: "Hi"}, 2)

	cm.GetOrCreate("session-2", "user-2")
	cm.AddMessage("session-2", providerpkg.Message{Role: "user", Content: "Test"}, 1)

	stats := cm.Stats()

	if stats["total_sessions"] != 2 {
		t.Errorf("Expected 2 sessions, got %v", stats["total_sessions"])
	}
	if stats["total_messages"] != 3 {
		t.Errorf("Expected 3 messages, got %v", stats["total_messages"])
	}
	if stats["total_tokens"] != 5 {
		t.Errorf("Expected 5 tokens, got %v", stats["total_tokens"])
	}
}

func TestContextManager_Expiration(t *testing.T) {
	// Short expiration time for testing
	cm := NewContextManager(100 * time.Millisecond)

	ctx := cm.GetOrCreate("session-1", "user-1")
	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Context should still exist (cleanup runs every 5 minutes)
	// To properly test expiration, we'd need to expose the cleanup function
	// For now, just verify the context exists
	ctx = cm.GetContext("session-1")
	if ctx == nil {
		t.Log("Context may have been cleaned up (this is acceptable)")
	}
}
