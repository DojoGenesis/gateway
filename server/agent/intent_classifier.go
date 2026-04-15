// DEPRECATION NOTICE (v0.0.30): The intent classification approach in this file
// is being superseded by the new Planner-based orchestration system.
// This classifier will be maintained for backward compatibility but new
// development should use the orchestration engine for complex multi-step tasks.
// Expected removal: v1.0.0
package agent

import (
	"strings"
	"sync"
)

// IntentClassifier performs lightweight, rule-based classification of user queries
// into Simple or Complex categories to optimize routing and response latency.
//
// Classification logic:
//  1. Simple keywords only (no complex indicators) → Simple
//  2. Complex indicators present → Complex (takes precedence)
//  3. Simple question patterns → Simple
//  4. Word count >= 10 → Complex
//  5. Default → Simple
//
// Thread-safe: All public methods use RLock for concurrent read access.
type IntentClassifier struct {
	mu                 sync.RWMutex
	simpleKeywords     []string
	simplePatterns     []string
	complexIndicators  []string
	codeTerms          []string
	mathTerms          []string
	actionVerbs        []string
	planningTerms      []string
	debugKeywords      []string
	greetingPatterns   []string
	arithmeticPatterns []string
	numberWords        []string
	whWords            []string
	metaKeywords       []string
}

// NewIntentClassifier creates a new IntentClassifier with default keyword lists.
// The classifier uses word-boundary matching to prevent false positives and
// maintains separate lists for greetings, question patterns, and action indicators.
//
// TODO(post-v1): Make keyword lists configurable via config.yaml to allow:
//   - Custom domain-specific keywords
//   - Multilingual support
//   - Tunable word count thresholds
//   - Project-specific classification rules
func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{
		simpleKeywords: []string{
			"hello", "hi", "hey", "greetings", "howdy",
			"good morning", "good afternoon", "good evening", "good night",
			"help", "what can you do", "capabilities",
			"thanks", "thank you", "bye", "goodbye", "see you", "later",
		},
		simplePatterns: []string{
			"what is", "who is", "when is", "where is", "why is",
			"how are you", "what's your name", "who are you",
			"what time", "what day", "what date",
			"how do you", "can you explain",
		},
		complexIndicators: []string{
			// Actions
			"write", "make", "create", "build", "implement", "develop",
			"generate", "construct", "configure", "setup",

			// Code operations
			"code", "function", "class", "method", "algorithm",
			"debug", "refactor", "optimize", "fix", "deploy",
			"test this", "test the", "test my",

			// Learning/Explanation
			"explain how to", "explain how", "step by step", "tutorial", "show me",
			"help me write", "help me build", "help me create", "demonstrate",

			// Design and architecture
			"design", "architecture", "system", "pattern",

			// Math/Calculations (to override "what is" simple pattern)
			"calculate", "compute", "solve", "equation", "formula",
			"2+2", "add", "subtract", "multiply", "divide",
		},
		codeTerms: []string{
			"function", "class", "method", "variable", "constant", "array", "list", "dict", "map", "struct",
			"loop", "iteration", "recursion", "algorithm", "data structure", "api", "endpoint",
			"import", "export", "module", "package", "library", "framework",
			"compile", "build", "deploy", "test", "lint", "format",
		},
		mathTerms: []string{
			"calculate", "compute", "solve", "equation", "formula", "expression",
			"add", "subtract", "multiply", "divide", "sum", "difference", "product", "quotient",
			"square root", "power", "exponent", "logarithm", "derivative", "integral",
			"percentage", "ratio", "proportion", "probability",
		},
		actionVerbs: []string{
			"create", "build", "make", "generate", "construct", "implement", "develop",
			"write", "code", "program", "script",
			"design", "architect", "plan", "model",
			"debug", "fix", "resolve", "troubleshoot",
			"refactor", "optimize", "improve", "enhance",
			"test", "validate", "verify",
			"deploy", "launch", "publish", "release",
			"setup", "configure", "install", "initialize",
		},
		planningTerms: []string{
			"design", "architecture", "system", "structure", "pattern", "approach", "strategy",
			"plan", "roadmap", "milestone", "phase", "stage",
			"requirements", "specifications", "constraints",
			"scalability", "performance", "reliability", "security",
		},
		debugKeywords: []string{
			"error", "bug", "issue", "problem", "fail", "failure",
			"debug", "fix", "resolve", "troubleshoot",
			"why am i getting", "why does", "doesn't work", "not working",
			"segmentation fault", "null pointer", "undefined",
		},
		greetingPatterns: []string{
			"hi", "hello", "hey", "greetings", "good morning", "good afternoon", "good evening",
			"howdy", "yo", "sup", "what's up",
			"help", "thanks", "thank you", "bye", "goodbye", "see you", "later",
			"what can you do", "capabilities",
		},
		arithmeticPatterns: []string{
			"+", "*", "/", "×", "÷", "=", "^", "%",
		},
		numberWords: []string{
			"add", "subtract", "multiply", "divide", "plus", "minus", "times",
		},
		whWords: []string{
			"what", "when", "where", "why", "which", "who", "how",
		},
		metaKeywords: []string{
			"dojo genesis", "dojo",
			"your capabilities", "your tools", "your features",
			"how do you work", "how does this work", "explain yourself",
			"what can you do", "what are you",
			"tell me about yourself", "who are you",
			"your system", "your architecture",
		},
	}
}

// Classify determines whether a query is Simple or Complex based on keyword
// matching, question patterns, and word count heuristics.
//
// DEPRECATED (v0.0.30): This method is deprecated and will be removed in v1.0.0.
// Use the new orchestration engine (go_backend/orchestration) for complex multi-step tasks
// that require planning and execution coordination. This method is maintained for
// backward compatibility with simple single-step queries.
//
// Classification precedence (highest to lowest):
//  1. Empty query → Simple
//  2. Simple keyword only (no complex indicators) → Simple
//  3. Complex indicator + simple question → Complex
//  4. Complex indicator → Complex
//  5. Simple keyword → Simple
//  6. Simple question pattern → Simple
//  7. Word count > 100 → Complex
//  8. Word count >= 10 → Complex
//  9. Default → Simple
//
// Performance: Typically <10μs per classification.
func (ic *IntentClassifier) Classify(query string) QueryType {
	result := ic.ClassifyWithConfidence(query)
	return result.Type
}

// ClassifyWithConfidence determines query type with confidence score and reasoning.
// This provides more detailed classification information for debugging and monitoring.
//
// DEPRECATED (v0.0.30): This method is deprecated and will be removed in v1.0.0.
// Use the new orchestration engine (go_backend/orchestration) for complex multi-step tasks
// that require planning and execution coordination. For simple routing decisions, use Route()
// instead. This method is maintained for backward compatibility and internally uses the Route() method.
func (ic *IntentClassifier) ClassifyWithConfidence(query string) ClassificationResult {
	decision := ic.Route(query)

	queryType := ic.mapRoutingDecisionToQueryType(decision)

	reasoning := ""
	if len(decision.Reasoning) > 0 {
		reasoning = decision.Reasoning[0]
		for i := 1; i < len(decision.Reasoning); i++ {
			reasoning += "; " + decision.Reasoning[i]
		}
	}

	return ClassificationResult{
		Type:       queryType,
		Confidence: decision.Confidence,
		Reasoning:  reasoning,
	}
}

func (ic *IntentClassifier) mapRoutingDecisionToQueryType(decision RoutingDecision) QueryType {
	if decision.Handler == "template" {
		return Simple
	}

	return Complex
}

func (ic *IntentClassifier) isMetaQuery(query string) bool {
	lowerQuery := strings.ToLower(query)
	for _, keyword := range ic.metaKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return true
		}
	}
	return false
}

func (ic *IntentClassifier) isSimpleQuestion(query string) bool {
	for _, pattern := range ic.simplePatterns {
		if strings.HasPrefix(query, pattern) {
			return true
		}
	}

	return false
}

func (ic *IntentClassifier) hasComplexIndicators(query string) bool {
	words := strings.Fields(query)
	wordSet := make(map[string]bool)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		wordSet[word] = true
	}

	for _, indicator := range ic.complexIndicators {
		if strings.Contains(indicator, " ") {
			if strings.Contains(query, indicator) {
				return true
			}
		} else {
			if wordSet[indicator] {
				return true
			}
		}
	}

	return false
}

func (ic *IntentClassifier) containsAny(query string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}
	return false
}

func (ic *IntentClassifier) containsArithmetic(query string) bool {
	for _, pattern := range ic.arithmeticPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	words := strings.Fields(query)
	for i, word := range words {
		word = strings.Trim(word, ".,!?;:")
		if strings.Contains(word, "-") && len(word) >= 3 {
			hasDigitBefore := i > 0 && len(words[i-1]) > 0 && (words[i-1][0] >= '0' && words[i-1][0] <= '9')
			hasDigitAfter := i < len(words)-1 && len(words[i+1]) > 0 && (words[i+1][0] >= '0' && words[i+1][0] <= '9')
			containsDigit := false
			for _, ch := range word {
				if ch >= '0' && ch <= '9' {
					containsDigit = true
					break
				}
			}
			if hasDigitBefore || hasDigitAfter || containsDigit {
				return true
			}
		}
	}

	queryLower := strings.ToLower(query)
	for _, word := range ic.numberWords {
		if strings.Contains(queryLower, word) {
			return true
		}
	}

	return false
}

func (ic *IntentClassifier) isGreeting(query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))

	for _, pattern := range ic.greetingPatterns {
		if query == pattern || strings.HasPrefix(query, pattern+" ") || strings.HasPrefix(query, pattern+"!") {
			return true
		}
	}

	return false
}

func (ic *IntentClassifier) hasCodeBlock(query string) bool {
	return strings.Contains(query, "```") || strings.Contains(query, "`")
}

func (ic *IntentClassifier) hasURL(query string) bool {
	return strings.Contains(query, "http://") || strings.Contains(query, "https://") || strings.Contains(query, "www.")
}

func (ic *IntentClassifier) extractFeatures(query string) QueryFeatures {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	features := QueryFeatures{
		WordCount:       len(words),
		HasQuestionMark: strings.Contains(query, "?"),
		StartsWithWH:    false,
		HasCodeTerms:    ic.containsAny(queryLower, ic.codeTerms),
		HasMathTerms:    ic.containsAny(queryLower, ic.mathTerms) || ic.containsArithmetic(queryLower),
		HasActionVerbs:  ic.containsAny(queryLower, ic.actionVerbs),
		HasMultiPart:    strings.Contains(queryLower, " and ") || strings.Contains(queryLower, " then ") || strings.Contains(queryLower, " also "),
		HasConstraints:  strings.Contains(queryLower, " must ") || strings.Contains(queryLower, " should ") || strings.Contains(queryLower, " need "),
		HasComparison:   strings.Contains(queryLower, " vs ") || strings.Contains(queryLower, " versus ") || strings.Contains(queryLower, " difference between "),
		IsFollowUp:      false,
		HasCodeBlock:    ic.hasCodeBlock(query),
		HasURL:          ic.hasURL(query),
		OriginalQuery:   query,
	}

	for _, wh := range ic.whWords {
		if strings.HasPrefix(queryLower, wh+" ") || strings.HasPrefix(queryLower, wh+"'") {
			features.StartsWithWH = true
			break
		}
	}

	return features
}

func (ic *IntentClassifier) scoreComplexity(features QueryFeatures) float64 {
	score := 0.0

	if features.WordCount < 5 {
		score += 0.0
	} else if features.WordCount < 15 {
		score += 0.2
	} else if features.WordCount < 50 {
		score += 0.4
	} else {
		score += 0.6
	}

	if features.HasCodeTerms {
		score += 0.3
	}
	if features.HasMathTerms {
		score += 0.2
	}
	if features.HasActionVerbs {
		score += 0.3
	}

	if features.HasMultiPart {
		score += 0.2
	}
	if features.HasConstraints {
		score += 0.2
	}
	if features.HasComparison {
		score += 0.1
	}

	if features.HasCodeBlock {
		score += 0.3
	}
	if features.HasURL {
		score += 0.1
	}

	if score > 1.0 {
		return 1.0
	}
	return score
}

func (ic *IntentClassifier) detectCategory(features QueryFeatures) IntentCategory {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	queryLower := strings.ToLower(features.OriginalQuery)

	// Meta queries about the system itself get highest priority
	if ic.isMetaQuery(features.OriginalQuery) {
		return MetaQuery
	}

	if ic.isGreeting(features.OriginalQuery) && !features.HasActionVerbs {
		return Greeting
	}

	hasDebugKeyword := ic.containsAny(queryLower, ic.debugKeywords)
	if hasDebugKeyword {
		return Debugging
	}

	hasPlanningKeyword := ic.containsAny(queryLower, ic.planningTerms)
	if hasPlanningKeyword {
		return Planning
	}

	// Check for code generation BEFORE math/calculation
	// to prioritize "write a function to calculate X" as CodeGeneration
	// BUT allow short WH questions to fall through for Factual classification
	if features.HasCodeTerms || features.HasCodeBlock {
		// Short WH questions without action verbs should be Factual (e.g., "what is a python function?")
		if features.StartsWithWH && !features.HasActionVerbs && features.WordCount <= 10 && !features.HasCodeBlock {
			// Fall through to WH question handling
		} else {
			hasExplainVerb := strings.Contains(queryLower, "explain") ||
				strings.Contains(queryLower, "what is") ||
				strings.Contains(queryLower, "what does") ||
				strings.Contains(queryLower, "how does")

			if hasExplainVerb {
				return Explanation
			}
			if features.HasActionVerbs {
				return CodeGeneration
			}
			if !features.StartsWithWH {
				return Explanation
			}
			// Fall through to WH question handling for short queries
		}
	}

	hasArithmetic := ic.containsArithmetic(features.OriginalQuery)
	if hasArithmetic && !features.HasCodeBlock {
		return Calculation
	}

	hasMathKeywords := features.HasMathTerms
	hasContextualMathOnly := hasMathKeywords &&
		!strings.Contains(queryLower, "difference between") &&
		!strings.Contains(queryLower, "compare") &&
		!features.HasCodeBlock

	if hasContextualMathOnly {
		return Calculation
	}

	if features.StartsWithWH && !features.HasActionVerbs {
		if features.WordCount > 10 || features.HasMultiPart {
			return Explanation
		}
		return Factual
	}

	if features.StartsWithWH {
		if features.WordCount > 10 || features.HasMultiPart {
			return Explanation
		}
		return Factual
	}

	return Explanation
}

func (ic *IntentClassifier) scoreIntent(features QueryFeatures) IntentScore {
	complexity := ic.scoreComplexity(features)
	category := ic.detectCategory(features)

	reasoningChain := []string{}

	switch category {
	case MetaQuery:
		reasoningChain = append(reasoningChain, "Meta query about the system itself")
	case Greeting:
		reasoningChain = append(reasoningChain, "Detected greeting pattern")
	case Calculation:
		if features.HasMathTerms {
			reasoningChain = append(reasoningChain, "Contains math terms")
		}
		if ic.containsArithmetic(features.OriginalQuery) {
			reasoningChain = append(reasoningChain, "Contains arithmetic expression")
		}
	case CodeGeneration:
		reasoningChain = append(reasoningChain, "Code generation request with action verbs")
		if features.HasCodeTerms {
			reasoningChain = append(reasoningChain, "Contains code-related terms")
		}
	case Debugging:
		reasoningChain = append(reasoningChain, "Debugging request detected")
	case Planning:
		reasoningChain = append(reasoningChain, "Planning/architecture query detected")
	case Factual:
		reasoningChain = append(reasoningChain, "Short factual question")
	case Explanation:
		if features.HasCodeTerms {
			reasoningChain = append(reasoningChain, "Explanation of code concept")
		} else {
			reasoningChain = append(reasoningChain, "General explanation request")
		}
	}

	if complexity > 0.7 {
		reasoningChain = append(reasoningChain, "High complexity score")
	} else if complexity < 0.3 {
		reasoningChain = append(reasoningChain, "Low complexity score")
	}

	certainty := 0.8
	if category == Greeting || category == Calculation || category == MetaQuery {
		certainty = 0.95
	} else if features.WordCount < 3 {
		certainty = 0.6
	} else if complexity > 0.8 && (category == CodeGeneration || category == Planning || category == Debugging) {
		// Very high complexity technical queries have high certainty
		certainty = 0.95
	}

	return IntentScore{
		Complexity:     complexity,
		Certainty:      certainty,
		Category:       category,
		ReasoningChain: reasoningChain,
	}
}

func (ic *IntentClassifier) Route(query string) RoutingDecision {
	query = strings.TrimSpace(query)
	if query == "" {
		return RoutingDecision{
			Handler:    "template",
			Template:   "greeting",
			Provider:   "",
			Fallback:   "",
			Confidence: 1.0,
			Category:   Greeting,
			Reasoning:  []string{"Empty query", "Routing to template handler for greeting"},
		}
	}

	features := ic.extractFeatures(query)

	if features.WordCount == 0 {
		return RoutingDecision{
			Handler:    "template",
			Template:   "greeting",
			Provider:   "",
			Fallback:   "",
			Confidence: 0.5,
			Category:   Greeting,
			Reasoning:  []string{"No words in query", "Routing to template handler for greeting"},
		}
	}

	score := ic.scoreIntent(features)

	decision := RoutingDecision{
		Category:   score.Category,
		Confidence: score.Certainty,
		Reasoning:  score.ReasoningChain,
	}

	switch score.Category {
	case MetaQuery:
		decision.Handler = "llm-fast"
		decision.Template = ""
		decision.Provider = "llm-fast"
		decision.Fallback = ""
		decision.Reasoning = append(decision.Reasoning, "Routing to LLM for meta query about the system")

	case Greeting:
		decision.Handler = "template"
		decision.Template = "greeting"
		decision.Provider = ""
		decision.Fallback = ""
		decision.Reasoning = append(decision.Reasoning, "Routing to template handler for greeting")

	case Calculation:
		if features.WordCount <= 2 && !strings.ContainsAny(features.OriginalQuery, "0123456789") {
			decision.Handler = "template"
			decision.Template = "greeting"
			decision.Provider = ""
			decision.Fallback = ""
			decision.Confidence = 0.5
			decision.Reasoning = append(decision.Reasoning, "Very short query without numbers, routing to template")
		} else {
			decision.Handler = "llm-fast"
			decision.Template = ""
			decision.Provider = "llm-fast"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-fast for calculation")
		}

	case Factual:
		if score.Complexity <= 0.3 || (features.WordCount <= 10 && score.Complexity < 0.8 && !features.HasActionVerbs) {
			decision.Handler = "template"
			decision.Template = "factual"
			decision.Provider = ""
			decision.Fallback = "llm-fast"
			decision.Reasoning = append(decision.Reasoning, "Routing to template with llm-fast fallback for simple factual query")
		} else {
			decision.Handler = "llm-fast"
			decision.Template = ""
			decision.Provider = "llm-fast"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-fast for complex factual query")
		}

	case CodeGeneration, Debugging, Planning:
		if score.Complexity > 0.7 {
			decision.Handler = "llm-reasoning"
			decision.Template = ""
			decision.Provider = "llm-reasoning"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-reasoning for high complexity task")
		} else {
			decision.Handler = "llm-fast"
			decision.Template = ""
			decision.Provider = "llm-fast"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-fast for moderate complexity task")
		}

	case Explanation:
		if score.Complexity > 0.7 {
			decision.Handler = "llm-reasoning"
			decision.Template = ""
			decision.Provider = "llm-reasoning"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-reasoning for complex explanation")
		} else if (score.Complexity <= 0.3 && features.WordCount < 10 && !features.HasActionVerbs) || (features.WordCount <= 2 && score.Complexity < 0.3) {
			decision.Handler = "template"
			decision.Template = "factual"
			decision.Provider = ""
			decision.Fallback = "llm-fast"
			decision.Reasoning = append(decision.Reasoning, "Routing to template with llm-fast fallback for simple query")
		} else {
			decision.Handler = "llm-fast"
			decision.Template = ""
			decision.Provider = "llm-fast"
			decision.Fallback = ""
			decision.Reasoning = append(decision.Reasoning, "Routing to llm-fast for explanation")
		}

	default:
		decision.Handler = "llm-fast"
		decision.Template = ""
		decision.Provider = "llm-fast"
		decision.Fallback = ""
		decision.Reasoning = append(decision.Reasoning, "Default routing to llm-fast")
	}

	return decision
}
