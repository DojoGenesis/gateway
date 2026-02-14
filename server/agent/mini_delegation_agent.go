package agent

import (
	"context"
	"strings"
	"sync"
)

// Intent represents the classified user intent for routing to specialized workflows.
type Intent string

const (
	// IntentThink indicates complex reasoning, planning, or analysis tasks.
	// Examples: "analyze this architecture", "explain why this pattern is better"
	IntentThink Intent = "THINK"

	// IntentSearch indicates information retrieval or research tasks.
	// Examples: "find documentation for X", "search for examples of Y"
	IntentSearch Intent = "SEARCH"

	// IntentBuild indicates code generation or creation tasks.
	// Examples: "create a new component", "write code for X"
	IntentBuild Intent = "BUILD"

	// IntentDebug indicates validation, error checking, or debugging tasks.
	// Examples: "fix this error", "debug this issue", "validate this code"
	IntentDebug Intent = "DEBUG"

	// IntentGeneral is the default fallback for queries that don't match specific intents.
	IntentGeneral Intent = "GENERAL"
)

// String returns the string representation of Intent.
func (i Intent) String() string {
	return string(i)
}

// IntentClassificationResult contains the intent classification with confidence score.
type IntentClassificationResult struct {
	Intent     Intent
	Confidence float64 // 0.0 to 1.0
	Reasoning  string  // Human-readable explanation
}

// MiniDelegationAgent is a fast, lightweight intent classifier optimized for
// sub-millisecond performance. It uses simple keyword-based classification
// without LLM calls.
//
// Thread-safe: All public methods use RLock for concurrent read access.
type MiniDelegationAgent struct {
	mu              sync.RWMutex
	thinkKeywords   []string
	searchKeywords  []string
	buildKeywords   []string
	debugKeywords   []string
	generalKeywords []string
}

// NewMiniDelegationAgent creates a new MiniDelegationAgent with default keyword lists.
// The classifier uses word-boundary matching and weighted scoring for accuracy.
//
// Performance target: < 1ms per classification (typically < 100μs).
func NewMiniDelegationAgent() *MiniDelegationAgent {
	return &MiniDelegationAgent{
		thinkKeywords: []string{
			"analyze", "explain why", "why does", "why is", "how does", "how works",
			"plan", "strategy", "reasoning", "compare", "evaluate", "assess",
			"consider", "understand", "help me understand", "think about", "implications", "trade-offs",
			"design", "design pattern", "architecture", "approach", "methodology",
			"philosophy", "principle", "concept", "theory",
		},
		searchKeywords: []string{
			"find", "search", "lookup", "what is", "who is", "research",
			"discover", "locate", "identify", "explore", "investigate",
			"get information", "learn about", "tell me about", "show me",
			"documentation", "example", "tutorial", "guide",
		},
		buildKeywords: []string{
			"create", "build", "generate", "write code", "implement",
			"develop", "construct", "make", "add", "new",
			"setup", "initialize", "configure", "scaffold",
			"write", "code", "program", "script", "function",
		},
		debugKeywords: []string{
			"fix", "fix this", "fix the", "debug", "debug this", "debug why",
			"error", "validate", "check", "test", "test this", "test the",
			"troubleshoot", "diagnose", "resolve", "repair",
			"bug", "issue", "problem", "broken", "failing",
			"verify", "review", "inspect", "examine",
		},
		generalKeywords: []string{
			"hello", "hi", "hey", "help", "thanks", "thank you",
			"what can you do", "capabilities", "bye", "goodbye",
		},
	}
}

// ClassifyIntent performs fast intent classification on a user query.
// This is the main public method that routes to ClassifyIntentWithConfidence.
//
// Performance: Typically < 100μs per classification.
func (m *MiniDelegationAgent) ClassifyIntent(ctx context.Context, query string) (Intent, float64) {
	result := m.ClassifyIntentWithConfidence(ctx, query)
	return result.Intent, result.Confidence
}

// ClassifyIntentWithConfidence performs intent classification with detailed results.
//
// Classification algorithm:
//  1. Normalize query (lowercase, trim)
//  2. Score each intent based on keyword matches
//  3. Apply bonus for multi-word phrase matches
//  4. Select highest scoring intent
//  5. Calculate confidence based on score distribution
//
// Performance: Typically < 100μs per classification.
func (m *MiniDelegationAgent) ClassifyIntentWithConfidence(ctx context.Context, query string) IntentClassificationResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.TrimSpace(strings.ToLower(query))

	if query == "" {
		return IntentClassificationResult{
			Intent:     IntentGeneral,
			Confidence: 1.0,
			Reasoning:  "empty query",
		}
	}

	// Build word set for fast lookup
	words := strings.Fields(query)
	wordSet := make(map[string]bool, len(words))
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		wordSet[word] = true
	}

	// Score each intent
	scores := map[Intent]int{
		IntentThink:   m.scoreKeywords(query, wordSet, m.thinkKeywords),
		IntentSearch:  m.scoreKeywords(query, wordSet, m.searchKeywords),
		IntentBuild:   m.scoreKeywords(query, wordSet, m.buildKeywords),
		IntentDebug:   m.scoreKeywords(query, wordSet, m.debugKeywords),
		IntentGeneral: m.scoreKeywords(query, wordSet, m.generalKeywords),
	}

	// Find highest scoring intent
	maxScore := 0
	bestIntent := IntentGeneral
	totalScore := 0

	for intent, score := range scores {
		totalScore += score
		if score > maxScore {
			maxScore = score
			bestIntent = intent
		}
	}

	// Calculate confidence based on score distribution
	// Higher confidence when one intent dominates
	confidence := 0.0
	if totalScore > 0 {
		confidence = float64(maxScore) / float64(totalScore)
	}

	// Adjust confidence based on absolute score
	// Low absolute scores get low confidence even if relative score is high
	if maxScore == 1 {
		confidence *= 0.6 // Single match = moderate confidence
	} else if maxScore == 2 {
		confidence *= 0.75
	} else if maxScore >= 3 {
		confidence *= 0.9
	}

	// Cap confidence at reasonable levels
	if confidence > 1.0 {
		confidence = 1.0
	}

	// If no matches, return general with low confidence
	if maxScore == 0 {
		return IntentClassificationResult{
			Intent:     IntentGeneral,
			Confidence: 0.3,
			Reasoning:  "no keyword matches, defaulting to general",
		}
	}

	reasoning := m.buildReasoning(bestIntent, maxScore, totalScore)

	return IntentClassificationResult{
		Intent:     bestIntent,
		Confidence: confidence,
		Reasoning:  reasoning,
	}
}

// scoreKeywords calculates a score for a keyword list against the query.
// Scoring:
//   - Single word match: +1 point
//   - Multi-word phrase match: +2 points (bonus for phrase matching)
func (m *MiniDelegationAgent) scoreKeywords(query string, wordSet map[string]bool, keywords []string) int {
	score := 0

	for _, keyword := range keywords {
		if strings.Contains(keyword, " ") {
			// Multi-word phrase - bonus points
			if strings.Contains(query, keyword) {
				score += 2
			}
		} else {
			// Single word
			if wordSet[keyword] {
				score++
			}
		}
	}

	return score
}

// buildReasoning generates a human-readable explanation for the classification.
func (m *MiniDelegationAgent) buildReasoning(intent Intent, score, totalScore int) string {
	var reasoning strings.Builder

	reasoning.WriteString("matched ")
	reasoning.WriteString(intent.String())
	reasoning.WriteString(" keywords (score: ")

	// Use a simple approach to convert int to string
	scoreStr := ""
	if score == 0 {
		scoreStr = "0"
	} else {
		// Convert score to string manually
		temp := score
		digits := []rune{}
		for temp > 0 {
			digits = append([]rune{rune('0' + temp%10)}, digits...)
			temp /= 10
		}
		scoreStr = string(digits)
	}
	reasoning.WriteString(scoreStr)

	reasoning.WriteString("/")

	// Convert totalScore to string
	totalScoreStr := ""
	if totalScore == 0 {
		totalScoreStr = "0"
	} else {
		temp := totalScore
		digits := []rune{}
		for temp > 0 {
			digits = append([]rune{rune('0' + temp%10)}, digits...)
			temp /= 10
		}
		totalScoreStr = string(digits)
	}
	reasoning.WriteString(totalScoreStr)
	reasoning.WriteString(")")

	return reasoning.String()
}

// SetKeywords allows customization of keyword lists for specific domains.
// This is useful for specialized applications or multilingual support.
func (m *MiniDelegationAgent) SetKeywords(intent Intent, keywords []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch intent {
	case IntentThink:
		m.thinkKeywords = keywords
	case IntentSearch:
		m.searchKeywords = keywords
	case IntentBuild:
		m.buildKeywords = keywords
	case IntentDebug:
		m.debugKeywords = keywords
	case IntentGeneral:
		m.generalKeywords = keywords
	}
}

// GetKeywords returns the current keyword list for an intent.
// Useful for debugging and testing.
func (m *MiniDelegationAgent) GetKeywords(intent Intent) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch intent {
	case IntentThink:
		return append([]string{}, m.thinkKeywords...)
	case IntentSearch:
		return append([]string{}, m.searchKeywords...)
	case IntentBuild:
		return append([]string{}, m.buildKeywords...)
	case IntentDebug:
		return append([]string{}, m.debugKeywords...)
	case IntentGeneral:
		return append([]string{}, m.generalKeywords...)
	default:
		return []string{}
	}
}
