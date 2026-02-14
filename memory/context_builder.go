package memory

import (
	"context"
	"fmt"
	"strings"
)

// ContextTier represents the priority tier for context building.
type ContextTier int

const (
	Tier1AlwaysOn   ContextTier = 1
	Tier2OnDemand   ContextTier = 2
	Tier3Referenced ContextTier = 3
	Tier4Pruned     ContextTier = 4
)

const (
	DefaultContextCapacity = 8000
	Tier1Capacity          = 1000
	Tier2Capacity          = 2000
	Tier3Capacity          = 3000
	Tier4Capacity          = 2000

	PruneThresholdTier4 = 0.80
	PruneThresholdTier3 = 0.90
	AlertThreshold      = 0.95
)

// ContextMessage represents a chat message for context building.
// Renamed from Message to avoid shadowing shared.Message.
type ContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ContextBuilder constructs tiered context from memory and seeds.
type ContextBuilder struct {
	gardenManager   *GardenManager
	contextCapacity int
}

// ContextBuildResult contains the built context with tier information.
type ContextBuildResult struct {
	Messages        []ContextMessage    `json:"messages"`
	TiersUsed       map[ContextTier]int `json:"tiers_used"`
	TotalTokens     int                 `json:"total_tokens"`
	CapacityPercent float64             `json:"capacity_percent"`
	Pruned          []ContextTier       `json:"pruned"`
	Alert           bool                `json:"alert"`
}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder(gardenManager *GardenManager) *ContextBuilder {
	return &ContextBuilder{
		gardenManager:   gardenManager,
		contextCapacity: DefaultContextCapacity,
	}
}

// SetContextCapacity sets the maximum context capacity in tokens.
func (cb *ContextBuilder) SetContextCapacity(capacity int) {
	cb.contextCapacity = capacity
}

// BuildContext builds a tiered context from seeds, recent memories, and compressed history.
func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, sessionID string, systemPrompt string) (*ContextBuildResult, error) {
	result := &ContextBuildResult{
		Messages:  []ContextMessage{},
		TiersUsed: make(map[ContextTier]int),
		Pruned:    []ContextTier{},
	}

	// Tier 1: System prompt + user query (always included)
	tier1Messages := cb.buildTier1(systemPrompt, query)
	tier1Tokens := cb.estimateTokens(tier1Messages)
	result.Messages = append(result.Messages, tier1Messages...)
	result.TiersUsed[Tier1AlwaysOn] = tier1Tokens
	result.TotalTokens += tier1Tokens

	// Tier 2: Relevant seeds
	availableForTier2 := cb.contextCapacity - result.TotalTokens
	tier2Messages, tier2Tokens, err := cb.buildTier2(ctx, query, availableForTier2)
	if err != nil {
		return nil, fmt.Errorf("failed to build tier 2: %w", err)
	}
	result.Messages = append(result.Messages, tier2Messages...)
	result.TiersUsed[Tier2OnDemand] = tier2Tokens
	result.TotalTokens += tier2Tokens

	// Tier 3: Recent memories
	availableForTier3 := cb.contextCapacity - result.TotalTokens
	tier3Messages, tier3Tokens, err := cb.buildTier3(ctx, sessionID, availableForTier3)
	if err != nil {
		return nil, fmt.Errorf("failed to build tier 3: %w", err)
	}

	capacityUsed := float64(result.TotalTokens+tier3Tokens) / float64(cb.contextCapacity)
	if capacityUsed > PruneThresholdTier3 {
		result.Pruned = append(result.Pruned, Tier3Referenced)
		tier3Messages = []ContextMessage{}
		tier3Tokens = 0
	}

	result.Messages = append(result.Messages, tier3Messages...)
	result.TiersUsed[Tier3Referenced] = tier3Tokens
	result.TotalTokens += tier3Tokens

	// Tier 4: Compressed history
	availableForTier4 := cb.contextCapacity - result.TotalTokens
	tier4Messages, tier4Tokens, err := cb.buildTier4(ctx, sessionID, availableForTier4)
	if err != nil {
		return nil, fmt.Errorf("failed to build tier 4: %w", err)
	}

	capacityUsed = float64(result.TotalTokens+tier4Tokens) / float64(cb.contextCapacity)
	if capacityUsed > PruneThresholdTier4 {
		result.Pruned = append(result.Pruned, Tier4Pruned)
		tier4Tokens = 0
		tier4Messages = []ContextMessage{}
	}

	result.Messages = append(result.Messages, tier4Messages...)
	result.TiersUsed[Tier4Pruned] = tier4Tokens
	result.TotalTokens += tier4Tokens

	result.CapacityPercent = float64(result.TotalTokens) / float64(cb.contextCapacity)
	result.Alert = result.CapacityPercent >= AlertThreshold

	return result, nil
}

func (cb *ContextBuilder) buildTier1(systemPrompt string, query string) []ContextMessage {
	messages := []ContextMessage{}

	if systemPrompt != "" {
		messages = append(messages, ContextMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, ContextMessage{
		Role:    "user",
		Content: query,
	})

	return messages
}

func (cb *ContextBuilder) buildTier2(ctx context.Context, query string, availableTokens int) ([]ContextMessage, int, error) {
	messages := []ContextMessage{}

	allSeeds, err := cb.gardenManager.ListSeeds(ctx, 100)
	if err != nil {
		return messages, 0, err
	}

	relevantSeeds := FindRelevantSeeds(query, allSeeds)

	if len(relevantSeeds) == 0 {
		return messages, 0, nil
	}

	seedsContent := cb.formatSeeds(relevantSeeds)
	tokens := estimateTokensFromText(seedsContent)

	if tokens > availableTokens {
		seedsContent = truncateContent(seedsContent, availableTokens)
		tokens = availableTokens
	}

	if seedsContent != "" {
		messages = append(messages, ContextMessage{
			Role:    "system",
			Content: fmt.Sprintf("Relevant knowledge seeds:\n%s", seedsContent),
		})
	}

	return messages, tokens, nil
}

func (cb *ContextBuilder) buildTier3(ctx context.Context, sessionID string, availableTokens int) ([]ContextMessage, int, error) {
	messages := []ContextMessage{}

	recentMemories, err := cb.gardenManager.memManager.ListMemories(ctx, MemoryFilter{
		Limit: 10,
	})
	if err != nil {
		return messages, 0, err
	}

	if len(recentMemories) == 0 {
		return messages, 0, nil
	}

	for _, mem := range recentMemories {
		tokens := estimateTokensFromText(mem.Content)
		if tokens > availableTokens {
			break
		}

		role := "user"
		if mem.Type == "assistant" || mem.Type == "response" {
			role = "assistant"
		}

		messages = append(messages, ContextMessage{
			Role:    role,
			Content: mem.Content,
		})

		availableTokens -= tokens
	}

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += estimateTokensFromText(msg.Content)
	}

	return messages, totalTokens, nil
}

func (cb *ContextBuilder) buildTier4(ctx context.Context, sessionID string, availableTokens int) ([]ContextMessage, int, error) {
	messages := []ContextMessage{}

	compressedHistories, err := cb.gardenManager.RetrieveCompressedHistory(ctx, sessionID)
	if err != nil {
		return messages, 0, err
	}

	if len(compressedHistories) == 0 {
		return messages, 0, nil
	}

	var combinedContent strings.Builder
	totalTokens := 0

	for _, history := range compressedHistories {
		tokens := estimateTokensFromText(history.CompressedContent)
		if totalTokens+tokens > availableTokens {
			break
		}

		combinedContent.WriteString(history.CompressedContent)
		combinedContent.WriteString("\n\n")
		totalTokens += tokens
	}

	if combinedContent.Len() > 0 {
		messages = append(messages, ContextMessage{
			Role:    "system",
			Content: fmt.Sprintf("Compressed history:\n%s", combinedContent.String()),
		})
	}

	return messages, totalTokens, nil
}

func (cb *ContextBuilder) formatSeeds(seeds []Seed) string {
	var builder strings.Builder
	for i, seed := range seeds {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(fmt.Sprintf("**%s**: %s", seed.Name, seed.Content))
	}
	return builder.String()
}

func (cb *ContextBuilder) estimateTokens(messages []ContextMessage) int {
	total := 0
	for _, msg := range messages {
		total += estimateTokensFromText(msg.Content)
		total += 4 // overhead per message
	}
	return total
}

func estimateTokensFromText(text string) int {
	words := len(strings.Fields(text))
	chars := len(text)

	tokensFromWords := float64(words) * 1.3
	tokensFromChars := float64(chars) / 4.0

	estimate := (tokensFromWords + tokensFromChars) / 2.0
	return int(estimate)
}

func truncateContent(content string, maxTokens int) string {
	estimatedChars := maxTokens * 4
	if len(content) <= estimatedChars {
		return content
	}
	return content[:estimatedChars] + "..."
}
