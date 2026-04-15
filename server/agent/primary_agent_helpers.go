package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// resolveBaseSystemPrompt returns the configured base system prompt.
// Precedence: SYSTEM_PROMPT env > SYSTEM_PROMPT_FILE > QWEN3_SYSTEM_PROMPT > built-in default.
func resolveBaseSystemPrompt() string {
	if v := os.Getenv("SYSTEM_PROMPT"); v != "" {
		return strings.TrimSpace(v)
	}
	if path := os.Getenv("SYSTEM_PROMPT_FILE"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
				return trimmed
			}
		}
	}
	if v := os.Getenv("QWEN3_SYSTEM_PROMPT"); v != "" {
		return v
	}
	return "You are a helpful AI coding assistant."
}

// buildSystemPrompt creates an intent-specific system prompt.
func (pa *PrimaryAgent) buildSystemPrompt(intent Intent) string {
	basePrompt := resolveBaseSystemPrompt()

	switch intent {
	case IntentThink:
		return basePrompt + "\n\nFocus on deep analysis, reasoning, and explaining complex concepts. Take time to think through problems systematically."
	case IntentSearch:
		return basePrompt + "\n\nFocus on finding and retrieving accurate information. Use search tools to gather comprehensive data before answering."
	case IntentBuild:
		return basePrompt + "\n\nFocus on generating clean, efficient, and well-documented code. Follow best practices and established patterns."
	case IntentDebug:
		return basePrompt + "\n\nFocus on identifying issues, validating code, and providing clear solutions. Be thorough in error analysis."
	case IntentGeneral:
		return basePrompt
	default:
		return basePrompt
	}
}

// generateID generates a random unique identifier.
func generateID() string {
	id, err := generateIDWithError()
	if err != nil {
		// crypto/rand failure is catastrophic; panic so it surfaces immediately.
		panic(fmt.Sprintf("generateID: %v", err))
	}
	return id
}

// generateIDWithError generates a random unique identifier, returning any error
// from the underlying crypto/rand source.
func generateIDWithError() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// truncateQuery truncates a query string for safe inclusion in log messages.
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

func getConfidenceLevel(confidence float64) string {
	if confidence >= 0.9 {
		return "high"
	} else if confidence >= 0.7 {
		return "medium"
	} else if confidence >= 0.5 {
		return "low"
	}
	return "very_low"
}
