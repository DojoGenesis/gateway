package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	providerpkg "github.com/DojoGenesis/gateway/provider"
)

// buildMessagesWithContext builds a message array that includes conversation history
// from memory manager if available and UseMemory is enabled.
func (pa *PrimaryAgent) buildMessagesWithContext(ctx context.Context, systemPrompt string, query string, userID string, useMemory bool) ([]providerpkg.Message, error) {
	messages := []providerpkg.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	if !useMemory || pa.memoryManager == nil || userID == "" {
		messages = append(messages, providerpkg.Message{
			Role:    "user",
			Content: query,
		})
		return messages, nil
	}

	memories, err := pa.memoryManager.SearchByType(ctx, fmt.Sprintf("conversation:%s", userID), 5)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation history: %w", err)
	}

	for _, mem := range memories {
		var convMem ConversationMemory
		if err := json.Unmarshal([]byte(mem.Content), &convMem); err != nil {
			continue
		}

		messages = append(messages, providerpkg.Message{
			Role:    "user",
			Content: convMem.UserMessage,
		})
		messages = append(messages, providerpkg.Message{
			Role:    "assistant",
			Content: convMem.AssistantMessage,
		})
	}

	messages = append(messages, providerpkg.Message{
		Role:    "user",
		Content: query,
	})

	return messages, nil
}

// buildContextWithGarden builds context using the Memory Garden if available, otherwise falls back to buildMessagesWithContext
func (pa *PrimaryAgent) buildContextWithGarden(ctx context.Context, systemPrompt string, query string, userID string, useMemory bool) ([]providerpkg.Message, map[memory.ContextTier]int, error) {
	if pa.gardenManager != nil && pa.contextBuilder != nil && useMemory {
		sessionID := userID
		if sessionID == "" {
			sessionID = "default"
		}

		buildResult, err := pa.contextBuilder.BuildContext(ctx, query, sessionID, systemPrompt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build context with garden manager: %w", err)
		}
		// Convert memory.Message to provider.Message
		provMessages := make([]providerpkg.Message, len(buildResult.Messages))
		for i, m := range buildResult.Messages {
			provMessages[i] = providerpkg.Message{
				Role:    m.Role,
				Content: m.Content,
			}
		}
		return provMessages, buildResult.TiersUsed, nil
	}

	messages, err := pa.buildMessagesWithContext(ctx, systemPrompt, query, userID, useMemory)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build messages with context: %w", err)
	}
	return messages, nil, nil
}

// storeConversation stores a conversation turn in memory.
func (pa *PrimaryAgent) storeConversation(ctx context.Context, userID string, query string, response *Response) error {
	if pa.memoryManager == nil || userID == "" {
		return nil
	}

	convMem := ConversationMemory{
		UserMessage:      query,
		AssistantMessage: response.Content,
		Timestamp:        response.Timestamp,
		Model:            response.Model,
		Provider:         response.Provider,
	}

	contentJSON, err := json.Marshal(convMem)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	mem := memory.Memory{
		ID:      generateID(),
		Type:    fmt.Sprintf("conversation:%s", userID),
		Content: string(contentJSON),
		Metadata: map[string]interface{}{
			"user_id":  userID,
			"model":    response.Model,
			"provider": response.Provider,
		},
		CreatedAt: response.Timestamp,
		UpdatedAt: response.Timestamp,
	}

	return pa.memoryManager.Store(ctx, mem)
}

// ConversationMemory represents a stored conversation turn.
type ConversationMemory struct {
	UserMessage      string    `json:"user_message"`
	AssistantMessage string    `json:"assistant_message"`
	Timestamp        time.Time `json:"timestamp"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
}
