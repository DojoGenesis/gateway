package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// ShouldCompress returns true if the number of memories exceeds the threshold.
func ShouldCompress(memories []Memory, turnThreshold int) bool {
	return len(memories) >= turnThreshold
}

// ShouldCompressWithDisposition returns true if memories should be compressed
// based on both the count threshold and the agent's disposition depth setting.
//
// Depth values and their compression behavior:
//   - "surface": Aggressive compression (threshold: 5 turns)
//   - "functional": Moderate compression (threshold: 10 turns)
//   - "thorough": Conservative compression (threshold: 20 turns)
//   - "exhaustive": Minimal compression (threshold: 50 turns)
//
// If agentConfig is nil, falls back to the provided turnThreshold.
func ShouldCompressWithDisposition(memories []Memory, turnThreshold int, agentConfig *gateway.AgentConfig) bool {
	if agentConfig == nil {
		return ShouldCompress(memories, turnThreshold)
	}

	// Map depth to compression threshold
	threshold := turnThreshold
	switch strings.ToLower(agentConfig.Depth) {
	case "surface":
		threshold = 5
	case "functional":
		threshold = 10
	case "thorough":
		threshold = 20
	case "exhaustive":
		threshold = 50
	default:
		// Unknown depth, use provided threshold
		threshold = turnThreshold
	}

	return len(memories) >= threshold
}

// GetRetentionDaysFromDepth returns the number of days to retain uncompressed memories
// based on the agent's depth configuration.
//
// Depth values and retention periods:
//   - "surface": 1 day (compress aggressively)
//   - "functional": 7 days (compress weekly)
//   - "thorough": 30 days (compress monthly)
//   - "exhaustive": 90 days (compress quarterly)
//
// Default: 7 days if depth is unknown or agentConfig is nil.
func GetRetentionDaysFromDepth(agentConfig *gateway.AgentConfig) int {
	if agentConfig == nil {
		return 7 // Default: 1 week
	}

	switch strings.ToLower(agentConfig.Depth) {
	case "surface":
		return 1
	case "functional":
		return 7
	case "thorough":
		return 30
	case "exhaustive":
		return 90
	default:
		return 7 // Default: 1 week
	}
}

// FilterMemoriesForCompression returns only memories older than the retention period
// defined by the agent's disposition. Recent memories are excluded from compression.
//
// This allows agents to control memory retention via their depth configuration:
//   - Surface: Compress memories older than 1 day
//   - Functional: Compress memories older than 7 days
//   - Thorough: Compress memories older than 30 days
//   - Exhaustive: Compress memories older than 90 days
func FilterMemoriesForCompression(memories []Memory, agentConfig *gateway.AgentConfig) []Memory {
	retentionDays := GetRetentionDaysFromDepth(agentConfig)
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	eligible := make([]Memory, 0)
	for _, mem := range memories {
		if mem.CreatedAt.Before(cutoffTime) {
			eligible = append(eligible, mem)
		}
	}

	return eligible
}

// GetOldMemories returns memories older than the most recent 'keep' count.
func GetOldMemories(memories []Memory, keep int) []Memory {
	if len(memories) <= keep {
		return []Memory{}
	}
	return memories[:len(memories)-keep]
}

// GetRecentMemories returns the most recent 'count' memories.
func GetRecentMemories(memories []Memory, count int) []Memory {
	if len(memories) <= count {
		return memories
	}
	return memories[len(memories)-count:]
}

// BuildOriginalContent concatenates memories into a single string for compression.
func BuildOriginalContent(memories []Memory) string {
	var builder strings.Builder
	for _, mem := range memories {
		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n", mem.Type, mem.CreatedAt.Format("2006-01-02 15:04:05"), mem.Content))
	}
	return builder.String()
}

// BuildCompressionPrompt creates a prompt for semantic compression using the "3-Month Rule".
func BuildCompressionPrompt(content string) string {
	return fmt.Sprintf(`You are a summarization agent applying the "3-Month Rule": if information wouldn't matter in 3 months, compress or discard it.

Analyze the following conversation. Identify the key decisions, lessons, patterns, and important facts. Discard all pleasantries, confirmations, and redundant explanations. Produce a dense, fact-based summary that preserves the core meaning.

Requirements:
- Keep: decisions, lessons learned, important patterns, actionable insights
- Remove: greetings, confirmations, redundant explanations, verbose language
- The summary should be no more than 20-30%% of the original length
- Use concise, technical language
- Maintain chronological flow where relevant

Original content:
%s

Compressed summary:`, content)
}
