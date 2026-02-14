package memory

import (
	"strings"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

// MemoryCategory classifies the type of information in a memory entry.
// Per Gateway-ADA Contract §3.2, categories determine retention based on depth.
type MemoryCategory string

const (
	CategoryDecision    MemoryCategory = "decision"
	CategoryAction      MemoryCategory = "action"
	CategoryObservation MemoryCategory = "observation"
	CategoryAlternative MemoryCategory = "alternative"
	CategoryReasoning   MemoryCategory = "reasoning"
	CategoryFinalOutput MemoryCategory = "final_output"
)

// CategorizedMemory wraps a Memory with its category classification.
type CategorizedMemory struct {
	Memory   Memory
	Category MemoryCategory
}

// RetentionPolicy returns a map indicating which categories should be retained
// based on the disposition depth setting.
//
// Per Gateway-ADA Contract §3.2:
//   - surface: decisions + final outputs only
//   - functional: decisions, actions, key observations
//   - thorough: decisions, actions, observations, alternatives
//   - exhaustive: everything (all categories)
func RetentionPolicy(depth string) map[MemoryCategory]bool {
	switch strings.ToLower(depth) {
	case "surface":
		return map[MemoryCategory]bool{
			CategoryDecision:    true,
			CategoryFinalOutput: true,
		}
	case "functional":
		return map[MemoryCategory]bool{
			CategoryDecision:    true,
			CategoryAction:      true,
			CategoryObservation: true,
		}
	case "thorough":
		return map[MemoryCategory]bool{
			CategoryDecision:    true,
			CategoryAction:      true,
			CategoryObservation: true,
			CategoryAlternative: true,
		}
	case "exhaustive":
		return map[MemoryCategory]bool{
			CategoryDecision:    true,
			CategoryAction:      true,
			CategoryObservation: true,
			CategoryAlternative: true,
			CategoryReasoning:   true,
			CategoryFinalOutput: true,
		}
	default:
		// Default to thorough if unknown depth
		return RetentionPolicy("thorough")
	}
}

// CategorizeMemo determines the category of a memory entry based on its content and type.
// This is a heuristic categorization that can be improved with more sophisticated analysis.
// Note: Order matters - check reasoning BEFORE decision, as "because" often appears with decisions.
func CategorizeMemory(mem Memory) MemoryCategory {
	content := strings.ToLower(mem.Content)
	memType := strings.ToLower(mem.Type)

	// Final output indicators (check FIRST - most specific patterns with colons/prefixes)
	finalOutputKeywords := []string{
		"result:", "output:", "final answer:", "conclusion:",
		"summary:", "final result", "completed task", "final output",
	}
	for _, keyword := range finalOutputKeywords {
		if strings.Contains(content, keyword) {
			return CategoryFinalOutput
		}
	}

	// Reasoning indicators (check second - "because" etc. often appear with decisions)
	reasoningKeywords := []string{
		"because", "therefore", "since", "due to", "given that",
		"reasoning", "rationale", "why", "analysis", "tradeoff",
	}
	for _, keyword := range reasoningKeywords {
		if strings.Contains(content, keyword) {
			return CategoryReasoning
		}
	}

	// Decision indicators
	decisionKeywords := []string{
		"decided", "chose", "selected", "picked", "concluded",
		"determined", "resolved", "opted for", "committed to",
	}
	for _, keyword := range decisionKeywords {
		if strings.Contains(content, keyword) {
			return CategoryDecision
		}
	}

	// Action indicators
	actionKeywords := []string{
		"executed", "ran", "performed", "completed", "called",
		"invoked", "triggered", "processed", "created", "modified",
	}
	for _, keyword := range actionKeywords {
		if strings.Contains(content, keyword) {
			return CategoryAction
		}
	}

	// Alternative indicators
	alternativeKeywords := []string{
		"alternatively", "could also", "another option", "instead",
		"or we could", "considered", "option", "alternative",
	}
	for _, keyword := range alternativeKeywords {
		if strings.Contains(content, keyword) {
			return CategoryAlternative
		}
	}

	// Type-based classification fallback
	switch memType {
	case "tool_call", "tool_result":
		return CategoryAction
	case "observation":
		return CategoryObservation
	case "decision":
		return CategoryDecision
	case "final_output", "output", "result":
		return CategoryFinalOutput
	default:
		return CategoryObservation // Default to observation
	}
}

// FilterMemoriesByDepth filters memories based on disposition depth setting.
// Returns only memories whose category should be retained per the retention policy.
func FilterMemoriesByDepth(memories []Memory, disp *disposition.DispositionConfig) []CategorizedMemory {
	if disp == nil {
		// No disposition - keep all memories
		result := make([]CategorizedMemory, len(memories))
		for i, mem := range memories {
			result[i] = CategorizedMemory{
				Memory:   mem,
				Category: CategorizeMemory(mem),
			}
		}
		return result
	}

	policy := RetentionPolicy(disp.Depth)
	filtered := make([]CategorizedMemory, 0, len(memories))

	for _, mem := range memories {
		category := CategorizeMemory(mem)
		if policy[category] {
			filtered = append(filtered, CategorizedMemory{
				Memory:   mem,
				Category: category,
			})
		}
	}

	return filtered
}

// WithDepthStrategy returns a function that can be used to configure
// a memory store with disposition-aware depth filtering.
type DepthStrategyOption func(*CompressionStore)

// CompressionStore is a placeholder for the actual memory store implementation.
// It will be enhanced to use depth-based retention.
type CompressionStore struct {
	disposition *disposition.DispositionConfig
	memories    []CategorizedMemory
}

// WithDepthStrategy configures the compression store with disposition-aware depth filtering.
func WithDepthStrategy(disp *disposition.DispositionConfig) DepthStrategyOption {
	return func(cs *CompressionStore) {
		cs.disposition = disp
	}
}

// NewCompressionStore creates a new compression store with optional configuration.
func NewCompressionStore(opts ...DepthStrategyOption) *CompressionStore {
	cs := &CompressionStore{
		disposition: disposition.DefaultDisposition(),
		memories:    make([]CategorizedMemory, 0),
	}

	for _, opt := range opts {
		opt(cs)
	}

	return cs
}

// Add adds a memory to the store with automatic categorization.
func (cs *CompressionStore) Add(mem Memory) {
	category := CategorizeMemory(mem)
	cs.memories = append(cs.memories, CategorizedMemory{
		Memory:   mem,
		Category: category,
	})
}

// Compress filters memories based on disposition depth and returns retained memories.
func (cs *CompressionStore) Compress() []CategorizedMemory {
	if cs.disposition == nil {
		return cs.memories
	}

	policy := RetentionPolicy(cs.disposition.Depth)
	filtered := make([]CategorizedMemory, 0, len(cs.memories))

	for _, catMem := range cs.memories {
		if policy[catMem.Category] {
			filtered = append(filtered, catMem)
		}
	}

	return filtered
}

// GetRetainedCount returns the number of memories retained for a given depth.
func (cs *CompressionStore) GetRetainedCount() int {
	return len(cs.Compress())
}
