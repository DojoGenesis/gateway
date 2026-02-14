package agent

import (
	"strings"
	"testing"
	"time"
)

func TestIntentClassifier_Classify(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected QueryType
	}{
		{
			name:     "empty string",
			query:    "",
			expected: Simple,
		},
		{
			name:     "hello",
			query:    "hello",
			expected: Simple,
		},
		{
			name:     "hi",
			query:    "hi",
			expected: Simple,
		},
		{
			name:     "what can you do",
			query:    "what can you do?",
			expected: Complex,
		},
		{
			name:     "help",
			query:    "help",
			expected: Simple,
		},
		{
			name:     "thanks",
			query:    "thanks",
			expected: Simple,
		},
		{
			name:     "goodbye",
			query:    "goodbye",
			expected: Simple,
		},
		{
			name:     "Build a React todo app",
			query:    "Build a React todo app",
			expected: Complex,
		},
		{
			name:     "Create a REST API",
			query:    "Create a REST API with authentication",
			expected: Complex,
		},
		{
			name:     "Implement feature",
			query:    "Implement user authentication in my app",
			expected: Complex,
		},
		{
			name:     "Write code",
			query:    "Write code to sort an array",
			expected: Complex,
		},
		{
			name:     "Debug issue",
			query:    "Debug this error in my code",
			expected: Complex,
		},
		{
			name:     "Refactor code",
			query:    "Refactor this function to be more efficient",
			expected: Complex,
		},
		{
			name:     "Step by step tutorial",
			query:    "Explain step by step how to deploy to AWS",
			expected: Complex,
		},
		{
			name:     "Very long query",
			query:    strings.Repeat("word ", 101),
			expected: Complex,
		},
		{
			name:     "Simple question what is",
			query:    "what is docker",
			expected: Simple,
		},
		{
			name:     "Simple question who is",
			query:    "who is the creator",
			expected: Simple,
		},
		{
			name:     "Short greeting with caps",
			query:    "HELLO THERE",
			expected: Simple,
		},
		{
			name:     "Whitespace query",
			query:    "   hello   ",
			expected: Simple,
		},
		{
			name:     "Medium length without complex indicators",
			query:    "I want to know more about",
			expected: Simple,
		},
		{
			name:     "Design request",
			query:    "Design a database schema for e-commerce",
			expected: Complex,
		},
		{
			name:     "Architecture question",
			query:    "Explain microservices architecture",
			expected: Complex,
		},
		{
			name:     "System design",
			query:    "How to design a scalable system",
			expected: Complex,
		},
		{
			name:     "Optimize request",
			query:    "Optimize my SQL queries",
			expected: Complex,
		},
		{
			name:     "Develop feature",
			query:    "Develop a login system",
			expected: Complex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestIntentClassifier_ClassifyLatency(t *testing.T) {
	ic := NewIntentClassifier()

	queries := []string{
		"hello",
		"what can you do?",
		"Build a React todo app",
		"Create a REST API with authentication and role-based access control",
		strings.Repeat("word ", 50),
	}

	for _, query := range queries {
		start := time.Now()
		ic.Classify(query)
		duration := time.Since(start)

		if duration > 10*time.Millisecond {
			t.Errorf("Classify(%q) took %v, want <10ms", query, duration)
		}
	}
}

func BenchmarkIntentClassifier_Simple(b *testing.B) {
	ic := NewIntentClassifier()
	query := "hello"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Classify(query)
	}
}

func BenchmarkIntentClassifier_Complex(b *testing.B) {
	ic := NewIntentClassifier()
	query := "Build a React todo app with authentication and database"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Classify(query)
	}
}

func BenchmarkIntentClassifier_VeryLong(b *testing.B) {
	ic := NewIntentClassifier()
	query := strings.Repeat("word ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Classify(query)
	}
}

func TestQueryType_String(t *testing.T) {
	tests := []struct {
		name     string
		qt       QueryType
		expected string
	}{
		{
			name:     "Simple",
			qt:       Simple,
			expected: "Simple",
		},
		{
			name:     "Complex",
			qt:       Complex,
			expected: "Complex",
		},
		{
			name:     "Unknown",
			qt:       QueryType(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.qt.String()
			if result != tt.expected {
				t.Errorf("QueryType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIntentClassifier_EdgeCases(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected QueryType
	}{
		{
			name:     "Only whitespace",
			query:    "   \t\n  ",
			expected: Simple,
		},
		{
			name:     "Special characters",
			query:    "!@#$%^&*()",
			expected: Simple,
		},
		{
			name:     "Unicode characters",
			query:    "你好",
			expected: Simple,
		},
		{
			name:     "Mixed case hello",
			query:    "HeLLo ThErE",
			expected: Simple,
		},
		{
			name:     "Exactly 10 words simple",
			query:    "what is the best way to learn programming today online",
			expected: Complex, // Contains code term "programming", classified as Explanation
		},
		{
			name:     "Exactly 10 words with complex indicator",
			query:    "build a simple app to learn programming today online quickly",
			expected: Complex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestIntentClassifier_ConflictingSignals(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected QueryType
		reason   string
	}{
		{
			name:     "Simple keyword + complex indicator",
			query:    "hello, build me an app",
			expected: Complex,
			reason:   "Complex indicators take precedence over greetings when both present",
		},
		{
			name:     "Simple question + complex indicator (factual)",
			query:    "what is docker",
			expected: Simple,
			reason:   "Factual question without complex indicators",
		},
		{
			name:     "Simple question + complex indicator (implementation)",
			query:    "what is the best way to build a microservices system",
			expected: Complex,
			reason:   "Implementation guidance with 'build' indicator",
		},
		{
			name:     "Question pattern + complex action",
			query:    "how do you implement authentication",
			expected: Complex,
			reason:   "Implementation task despite question pattern",
		},
		{
			name:     "Multi-sentence: greeting + complex",
			query:    "Hello! Can you build me a todo app?",
			expected: Complex,
			reason:   "Complex indicators take precedence even after greetings",
		},
		{
			name:     "Word boundary test: 'this' contains 'hi'",
			query:    "Debug this error in my code",
			expected: Complex,
			reason:   "Word-boundary matching prevents false positive on 'this'",
		},
		{
			name:     "Exactly 9 words no indicators",
			query:    "I want to know more about this topic today",
			expected: Simple,
			reason:   "Under 10 words, no complex indicators",
		},
		{
			name:     "Exactly 11 words no indicators",
			query:    "I want to know more about this interesting topic we discussed today",
			expected: Complex,
			reason:   "10+ words triggers complex classification",
		},
		{
			name:     "Code snippet in query",
			query:    "fix this error: undefined is not a function",
			expected: Complex,
			reason:   "'fix' is a complex indicator",
		},
		{
			name:     "New greeting patterns",
			query:    "good morning",
			expected: Simple,
			reason:   "Expanded greeting coverage",
		},
		{
			name:     "New farewell patterns",
			query:    "see you later",
			expected: Simple,
			reason:   "Expanded farewell coverage",
		},
		{
			name:     "Why question pattern",
			query:    "why is this happening",
			expected: Simple,
			reason:   "Simple 'why is' question pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v (reason: %s)", tt.query, result, tt.expected, tt.reason)
			}
		})
	}
}

func TestIntentClassifier_NewKeywords(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected QueryType
	}{
		{
			name:     "Generate keyword",
			query:    "generate a REST API",
			expected: Complex,
		},
		{
			name:     "Configure keyword",
			query:    "configure nginx for production",
			expected: Complex,
		},
		{
			name:     "Setup keyword",
			query:    "setup docker containers",
			expected: Complex,
		},
		{
			name:     "Deploy keyword",
			query:    "deploy to AWS",
			expected: Complex,
		},
		{
			name:     "Fix keyword",
			query:    "fix this bug",
			expected: Complex,
		},
		{
			name:     "Test keyword",
			query:    "test this function",
			expected: Complex,
		},
		{
			name:     "Can you explain pattern",
			query:    "can you explain recursion",
			expected: Simple,
		},
		{
			name:     "Who are you pattern",
			query:    "who are you",
			expected: Complex,
		},
		{
			name:     "What date pattern",
			query:    "what date is it",
			expected: Simple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestClassifyMetaQueries(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected QueryType
	}{
		{
			name:     "what is dojo genesis",
			query:    "what is dojo genesis",
			expected: Complex,
		},
		{
			name:     "what are your capabilities",
			query:    "what are your capabilities",
			expected: Complex,
		},
		{
			name:     "your tools",
			query:    "what are your tools",
			expected: Complex,
		},
		{
			name:     "how do you work",
			query:    "how do you work",
			expected: Complex,
		},
		{
			name:     "how does this work",
			query:    "how does this work",
			expected: Complex,
		},
		{
			name:     "explain yourself",
			query:    "explain yourself",
			expected: Complex,
		},
		{
			name:     "what are you",
			query:    "what are you",
			expected: Complex,
		},
		{
			name:     "your architecture",
			query:    "describe your architecture",
			expected: Complex,
		},
		{
			name:     "your features",
			query:    "what are your features",
			expected: Complex,
		},
		{
			name:     "your system",
			query:    "tell me about your system",
			expected: Complex,
		},
		{
			name:     "hello should still be simple",
			query:    "hello",
			expected: Simple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestIntentClassifier_CoverageBoost(t *testing.T) {
	ic := NewIntentClassifier()

	t.Run("isSimpleQuestion coverage", func(t *testing.T) {
		simpleQuestions := []string{
			"what is golang",
			"who is the author",
			"when is the deadline",
			"where is the file",
			"how are you doing",
			"what's your name",
			"what time is it",
			"what day is today",
		}

		for _, q := range simpleQuestions {
			if !ic.isSimpleQuestion(q) {
				t.Errorf("Expected %q to be a simple question", q)
			}
		}
	})

	t.Run("hasComplexIndicators coverage", func(t *testing.T) {
		complexQueries := []string{
			"build something",
			"create a feature",
			"implement authentication",
			"develop an app",
			"write code for sorting",
			"debug this issue",
			"refactor my code",
			"optimize the performance",
			"explain how to deploy",
			"step by step guide",
			"tutorial on react",
			"design a system",
			"architecture patterns",
			"system design interview",
		}

		for _, q := range complexQueries {
			if !ic.hasComplexIndicators(q) {
				t.Errorf("Expected %q to have complex indicators", q)
			}
		}
	})

	t.Run("case insensitivity", func(t *testing.T) {
		queries := map[string]QueryType{
			"HELLO":           Simple,
			"HeLp":            Simple,
			"BUILD APP":       Complex,
			"CrEaTe FuNcTiOn": Complex,
			"WhAt CaN yOu Do": Complex,
		}

		for query, expected := range queries {
			result := ic.Classify(query)
			if result != expected {
				t.Errorf("Classify(%q) = %v, want %v", query, result, expected)
			}
		}
	})
}

func TestExtractFeatures(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name          string
		query         string
		checkFeatures func(*testing.T, QueryFeatures)
	}{
		{
			name:  "simple greeting",
			query: "hello",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if f.WordCount != 1 {
					t.Errorf("WordCount = %d, want 1", f.WordCount)
				}
				if f.HasQuestionMark {
					t.Error("HasQuestionMark should be false")
				}
				if f.StartsWithWH {
					t.Error("StartsWithWH should be false")
				}
			},
		},
		{
			name:  "question with WH word",
			query: "what is docker?",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if f.WordCount != 3 {
					t.Errorf("WordCount = %d, want 3", f.WordCount)
				}
				if !f.HasQuestionMark {
					t.Error("HasQuestionMark should be true")
				}
				if !f.StartsWithWH {
					t.Error("StartsWithWH should be true")
				}
			},
		},
		{
			name:  "code generation query",
			query: "write a function to sort an array",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasCodeTerms {
					t.Error("HasCodeTerms should be true")
				}
				if !f.HasActionVerbs {
					t.Error("HasActionVerbs should be true")
				}
			},
		},
		{
			name:  "math query",
			query: "what is 2+2?",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasMathTerms {
					t.Error("HasMathTerms should be true (contains arithmetic)")
				}
				if !f.HasQuestionMark {
					t.Error("HasQuestionMark should be true")
				}
			},
		},
		{
			name:  "calculation query",
			query: "calculate 15% of 200",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasMathTerms {
					t.Error("HasMathTerms should be true")
				}
			},
		},
		{
			name:  "multi-part query",
			query: "create a function and then test it",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasMultiPart {
					t.Error("HasMultiPart should be true")
				}
				if !f.HasActionVerbs {
					t.Error("HasActionVerbs should be true")
				}
			},
		},
		{
			name:  "query with constraints",
			query: "build an API that must support authentication",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasConstraints {
					t.Error("HasConstraints should be true")
				}
				if !f.HasActionVerbs {
					t.Error("HasActionVerbs should be true")
				}
			},
		},
		{
			name:  "comparison query",
			query: "what is the difference between REST and GraphQL",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasComparison {
					t.Error("HasComparison should be true")
				}
			},
		},
		{
			name:  "query with code block",
			query: "fix this `console.log()` error",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasCodeBlock {
					t.Error("HasCodeBlock should be true")
				}
			},
		},
		{
			name:  "query with URL",
			query: "check out https://example.com for details",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if !f.HasURL {
					t.Error("HasURL should be true")
				}
			},
		},
		{
			name:  "debugging query",
			query: "why am I getting a segmentation fault",
			checkFeatures: func(t *testing.T, f QueryFeatures) {
				if f.WordCount != 7 {
					t.Errorf("WordCount = %d, want 7", f.WordCount)
				}
				if !f.StartsWithWH {
					t.Error("StartsWithWH should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ic.extractFeatures(tt.query)

			if features.OriginalQuery != tt.query {
				t.Errorf("OriginalQuery = %q, want %q", features.OriginalQuery, tt.query)
			}

			tt.checkFeatures(t, features)
		})
	}
}

func TestContainsArithmetic(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "addition",
			query:    "what is 2+2",
			expected: true,
		},
		{
			name:     "subtraction",
			query:    "calculate 10-5",
			expected: true,
		},
		{
			name:     "multiplication",
			query:    "what is 3*4",
			expected: true,
		},
		{
			name:     "division",
			query:    "divide 100/5",
			expected: true,
		},
		{
			name:     "percentage",
			query:    "calculate 15% of 200",
			expected: true,
		},
		{
			name:     "equals sign",
			query:    "solve x = 10",
			expected: true,
		},
		{
			name:     "word-based addition",
			query:    "add 5 and 10",
			expected: true,
		},
		{
			name:     "word-based subtraction",
			query:    "subtract 3 from 10",
			expected: true,
		},
		{
			name:     "word-based multiplication",
			query:    "multiply 4 times 5",
			expected: true,
		},
		{
			name:     "no arithmetic",
			query:    "what is docker",
			expected: false,
		},
		{
			name:     "code-related minus (not arithmetic)",
			query:    "use snake-case naming",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.containsArithmetic(strings.ToLower(tt.query))
			if result != tt.expected {
				t.Errorf("containsArithmetic(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestIsGreeting(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "simple hi",
			query:    "hi",
			expected: true,
		},
		{
			name:     "simple hello",
			query:    "hello",
			expected: true,
		},
		{
			name:     "hey with space",
			query:    "hey there",
			expected: true,
		},
		{
			name:     "good morning",
			query:    "good morning",
			expected: true,
		},
		{
			name:     "good afternoon",
			query:    "good afternoon",
			expected: true,
		},
		{
			name:     "hello with exclamation",
			query:    "hello!",
			expected: true,
		},
		{
			name:     "greeting with extra text",
			query:    "hello how are you",
			expected: true,
		},
		{
			name:     "uppercase greeting",
			query:    "HELLO",
			expected: true,
		},
		{
			name:     "mixed case greeting",
			query:    "HeLLo",
			expected: true,
		},
		{
			name:     "not a greeting",
			query:    "what is docker",
			expected: false,
		},
		{
			name:     "contains 'hi' but not greeting",
			query:    "this is a test",
			expected: false,
		},
		{
			name:     "sup greeting",
			query:    "sup",
			expected: true,
		},
		{
			name:     "what's up greeting",
			query:    "what's up",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.isGreeting(tt.query)
			if result != tt.expected {
				t.Errorf("isGreeting(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestHasCodeBlock(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "triple backticks",
			query:    "fix this code ```console.log('hello')```",
			expected: true,
		},
		{
			name:     "single backtick",
			query:    "what does `console.log()` do",
			expected: true,
		},
		{
			name:     "no code block",
			query:    "what is docker",
			expected: false,
		},
		{
			name:     "inline code",
			query:    "use `npm install` to install packages",
			expected: true,
		},
		{
			name:     "markdown code block",
			query:    "```javascript\nfunction test() {}\n```",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.hasCodeBlock(tt.query)
			if result != tt.expected {
				t.Errorf("hasCodeBlock(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestHasURL(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "http URL",
			query:    "check http://example.com",
			expected: true,
		},
		{
			name:     "https URL",
			query:    "visit https://example.com",
			expected: true,
		},
		{
			name:     "www URL",
			query:    "go to www.example.com",
			expected: true,
		},
		{
			name:     "no URL",
			query:    "what is docker",
			expected: false,
		},
		{
			name:     "URL in sentence",
			query:    "the documentation is at https://docs.example.com/api",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.hasURL(tt.query)
			if result != tt.expected {
				t.Errorf("hasURL(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestScoreComplexity(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		minScore float64
		maxScore float64
	}{
		{
			name:     "short greeting",
			query:    "hello",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "short question",
			query:    "what is docker?",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "math query",
			query:    "what is 2+2?",
			minScore: 0.2,
			maxScore: 0.3,
		},
		{
			name:     "simple code query",
			query:    "explain python functions",
			minScore: 0.3,
			maxScore: 0.4,
		},
		{
			name:     "complex code generation",
			query:    "write a Python function to calculate fibonacci numbers using memoization",
			minScore: 0.7,
			maxScore: 1.0,
		},
		{
			name:     "multi-part with constraints",
			query:    "create a REST API with authentication and also add rate limiting",
			minScore: 0.7,
			maxScore: 1.0,
		},
		{
			name:     "code block present",
			query:    "debug this code: `function test() { return null; }`",
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name:     "very long query",
			query:    strings.Repeat("word ", 60),
			minScore: 0.6,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ic.extractFeatures(tt.query)
			score := ic.scoreComplexity(features)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("scoreComplexity(%q) = %.2f, want between %.2f and %.2f", tt.query, score, tt.minScore, tt.maxScore)
			}

			if score < 0.0 || score > 1.0 {
				t.Errorf("scoreComplexity(%q) = %.2f, must be in range [0.0, 1.0]", tt.query, score)
			}
		})
	}
}

func TestDetectCategory(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected IntentCategory
	}{
		{
			name:     "greeting - hello",
			query:    "hello",
			expected: Greeting,
		},
		{
			name:     "greeting - hi there",
			query:    "hi there",
			expected: Greeting,
		},
		{
			name:     "greeting - good morning",
			query:    "good morning",
			expected: Greeting,
		},
		{
			name:     "calculation - arithmetic",
			query:    "what is 2+2?",
			expected: Calculation,
		},
		{
			name:     "calculation - percentage",
			query:    "calculate 15% of 200",
			expected: Calculation,
		},
		{
			name:     "calculation - math terms",
			query:    "compute the square root of 16",
			expected: Calculation,
		},
		{
			name:     "factual - short what is",
			query:    "what is docker?",
			expected: Factual,
		},
		{
			name:     "factual - who is",
			query:    "who is the creator?",
			expected: Factual,
		},
		{
			name:     "explanation - long what is",
			query:    "what is the difference between docker and kubernetes and how do they work together?",
			expected: Explanation,
		},
		{
			name:     "code generation - write function",
			query:    "write a function to sort an array",
			expected: CodeGeneration,
		},
		{
			name:     "code generation - create class",
			query:    "create a Python class for user authentication",
			expected: CodeGeneration,
		},
		{
			name:     "debugging - fix error",
			query:    "fix this bug in my code",
			expected: Debugging,
		},
		{
			name:     "debugging - error message",
			query:    "why am I getting a null pointer error?",
			expected: Debugging,
		},
		{
			name:     "planning - design system",
			query:    "design a scalable microservices architecture",
			expected: Planning,
		},
		{
			name:     "planning - architecture",
			query:    "what is the best architecture for an e-commerce system?",
			expected: Planning,
		},
		{
			name:     "explanation - code concept",
			query:    "explain how recursion works",
			expected: Explanation,
		},
		{
			name:     "explanation - code with backticks",
			query:    "explain this code: `const x = 5`",
			expected: Explanation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ic.extractFeatures(tt.query)
			category := ic.detectCategory(features)

			if category != tt.expected {
				t.Errorf("detectCategory(%q) = %v, want %v", tt.query, category, tt.expected)
			}
		})
	}
}

func TestScoreIntent(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name              string
		query             string
		expectedCategory  IntentCategory
		minComplexity     float64
		maxComplexity     float64
		minCertainty      float64
		reasoningContains string
	}{
		{
			name:              "greeting",
			query:             "hello",
			expectedCategory:  Greeting,
			minComplexity:     0.0,
			maxComplexity:     0.1,
			minCertainty:      0.9,
			reasoningContains: "greeting pattern",
		},
		{
			name:              "calculation query",
			query:             "what is 2+2?",
			expectedCategory:  Calculation,
			minComplexity:     0.2,
			maxComplexity:     0.4,
			minCertainty:      0.9,
			reasoningContains: "arithmetic expression",
		},
		{
			name:              "complex code generation",
			query:             "write a Python function to calculate fibonacci using memoization",
			expectedCategory:  CodeGeneration,
			minComplexity:     0.7,
			maxComplexity:     1.0,
			minCertainty:      0.7,
			reasoningContains: "Code generation",
		},
		{
			name:              "debugging request",
			query:             "fix this error in my function",
			expectedCategory:  Debugging,
			minComplexity:     0.4,
			maxComplexity:     0.8,
			minCertainty:      0.7,
			reasoningContains: "Debugging",
		},
		{
			name:              "factual question",
			query:             "what is docker?",
			expectedCategory:  Factual,
			minComplexity:     0.0,
			maxComplexity:     0.2,
			minCertainty:      0.5,
			reasoningContains: "factual question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ic.extractFeatures(tt.query)
			score := ic.scoreIntent(features)

			if score.Category != tt.expectedCategory {
				t.Errorf("scoreIntent(%q).Category = %v, want %v", tt.query, score.Category, tt.expectedCategory)
			}

			if score.Complexity < tt.minComplexity || score.Complexity > tt.maxComplexity {
				t.Errorf("scoreIntent(%q).Complexity = %.2f, want between %.2f and %.2f", tt.query, score.Complexity, tt.minComplexity, tt.maxComplexity)
			}

			if score.Certainty < tt.minCertainty {
				t.Errorf("scoreIntent(%q).Certainty = %.2f, want >= %.2f", tt.query, score.Certainty, tt.minCertainty)
			}

			if len(score.ReasoningChain) == 0 {
				t.Errorf("scoreIntent(%q).ReasoningChain is empty, want at least one reason", tt.query)
			}

			hasExpectedReasoning := false
			for _, reason := range score.ReasoningChain {
				if strings.Contains(strings.ToLower(reason), strings.ToLower(tt.reasoningContains)) {
					hasExpectedReasoning = true
					break
				}
			}

			if !hasExpectedReasoning {
				t.Errorf("scoreIntent(%q).ReasoningChain = %v, want to contain %q", tt.query, score.ReasoningChain, tt.reasoningContains)
			}
		})
	}
}

func TestCategoryDetectionPrecedence(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected IntentCategory
		reason   string
	}{
		{
			name:     "greeting takes precedence over everything",
			query:    "hello",
			expected: Greeting,
			reason:   "Greeting should be detected first for efficiency",
		},
		{
			name:     "code terms with debug keyword",
			query:    "debug this function error",
			expected: Debugging,
			reason:   "Debug keywords should be detected before general code generation",
		},
		{
			name:     "code terms with action verbs",
			query:    "write a function to sort array",
			expected: CodeGeneration,
			reason:   "Action verbs with code terms indicate code generation",
		},
		{
			name:     "code terms without action verbs",
			query:    "what is a python function?",
			expected: Factual,
			reason:   "Short WH questions are factual even with code terms (backward compatibility)",
		},
		{
			name:     "arithmetic overrides factual pattern",
			query:    "what is 2+2?",
			expected: Calculation,
			reason:   "Arithmetic should be detected even in 'what is' questions",
		},
		{
			name:     "planning terms detected",
			query:    "design a scalable system architecture",
			expected: Planning,
			reason:   "Planning keywords should be detected",
		},
		{
			name:     "long WH question becomes explanation",
			query:    "what is the difference between docker and kubernetes and how do they work together?",
			expected: Explanation,
			reason:   "Long WH questions should be explanations, not factual",
		},
		{
			name:     "short WH question is factual",
			query:    "what is docker?",
			expected: Factual,
			reason:   "Short WH questions should be factual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ic.extractFeatures(tt.query)
			category := ic.detectCategory(features)

			if category != tt.expected {
				t.Errorf("detectCategory(%q) = %v, want %v (reason: %s)", tt.query, category, tt.expected, tt.reason)
			}
		})
	}
}

func BenchmarkScoreComplexity(b *testing.B) {
	ic := NewIntentClassifier()
	features := ic.extractFeatures("write a Python function to calculate fibonacci using memoization and explain time complexity")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.scoreComplexity(features)
	}
}

func BenchmarkDetectCategory(b *testing.B) {
	ic := NewIntentClassifier()
	features := ic.extractFeatures("write a Python function to calculate fibonacci using memoization")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.detectCategory(features)
	}
}

func BenchmarkScoreIntent(b *testing.B) {
	ic := NewIntentClassifier()
	features := ic.extractFeatures("write a Python function to calculate fibonacci using memoization")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.scoreIntent(features)
	}
}

func TestRouteGreeting(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple hello",
			query:    "hello",
			expected: "template",
		},
		{
			name:     "hi",
			query:    "hi",
			expected: "template",
		},
		{
			name:     "good morning",
			query:    "good morning",
			expected: "template",
		},
		{
			name:     "hey there",
			query:    "hey there",
			expected: "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler != tt.expected {
				t.Errorf("Route(%q).Handler = %v, want %v", tt.query, decision.Handler, tt.expected)
			}

			if decision.Category != Greeting {
				t.Errorf("Route(%q).Category = %v, want Greeting", tt.query, decision.Category)
			}

			if decision.Template != "greeting" {
				t.Errorf("Route(%q).Template = %v, want 'greeting'", tt.query, decision.Template)
			}
		})
	}
}

func TestRouteCalculation(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "what is 2+2",
			query:    "what is 2+2?",
			expected: "llm-fast",
		},
		{
			name:     "calculate percentage",
			query:    "calculate 15% of 200",
			expected: "llm-fast",
		},
		{
			name:     "arithmetic with multiply",
			query:    "what is 5 * 8?",
			expected: "llm-fast",
		},
		{
			name:     "solve equation",
			query:    "solve 2x + 5 = 15",
			expected: "llm-fast",
		},
		{
			name:     "add numbers",
			query:    "add 25 and 37",
			expected: "llm-fast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler != tt.expected {
				t.Errorf("Route(%q).Handler = %v, want %v", tt.query, decision.Handler, tt.expected)
			}

			if decision.Category != Calculation {
				t.Errorf("Route(%q).Category = %v, want Calculation", tt.query, decision.Category)
			}

			if decision.Provider != "llm-fast" {
				t.Errorf("Route(%q).Provider = %v, want 'llm-fast'", tt.query, decision.Provider)
			}
		})
	}
}

func TestRouteComplexCode(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name            string
		query           string
		expectedHandler string
		minComplexity   float64
	}{
		{
			name:            "complex code generation with constraints",
			query:           "write a Python function to implement a binary search tree with insert, delete, and find operations, ensuring O(log n) performance",
			expectedHandler: "llm-reasoning",
			minComplexity:   0.7,
		},
		{
			name:            "complex system design",
			query:           "design a microservices architecture for an e-commerce platform with user authentication, inventory management, payment processing, and order tracking",
			expectedHandler: "llm-reasoning",
			minComplexity:   0.7,
		},
		{
			name:            "moderate code generation",
			query:           "write a function to sort an array",
			expectedHandler: "llm-reasoning",
			minComplexity:   0.7,
		},
		{
			name:            "debugging request",
			query:           "debug this error in my code",
			expectedHandler: "llm-fast",
			minComplexity:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler != tt.expectedHandler {
				t.Errorf("Route(%q).Handler = %v, want %v (complexity: %v)", tt.query, decision.Handler, tt.expectedHandler, decision.Confidence)
			}

			features := ic.extractFeatures(tt.query)
			score := ic.scoreComplexity(features)

			if score < tt.minComplexity {
				t.Errorf("Route(%q) complexity = %v, want >= %v", tt.query, score, tt.minComplexity)
			}
		})
	}
}

func TestRouteFactual(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name            string
		query           string
		expectedHandler string
	}{
		{
			name:            "simple factual question",
			query:           "what is docker?",
			expectedHandler: "template",
		},
		{
			name:            "who question",
			query:           "who is the creator?",
			expectedHandler: "template",
		},
		{
			name:            "when question",
			query:           "when is the meeting?",
			expectedHandler: "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Handler != tt.expectedHandler {
				t.Errorf("Route(%q).Handler = %v, want %v", tt.query, decision.Handler, tt.expectedHandler)
			}

			if tt.expectedHandler == "template" && decision.Fallback != "llm-fast" {
				t.Errorf("Route(%q).Fallback = %v, want 'llm-fast'", tt.query, decision.Fallback)
			}
		})
	}
}

func TestRoutingEdgeCases(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name     string
		query    string
		category IntentCategory
		handler  string
	}{
		{
			name:     "empty query defaults to greeting (backward compatibility)",
			query:    "",
			category: Greeting,
			handler:  "template",
		},
		{
			name:     "very short query",
			query:    "hi",
			category: Greeting,
			handler:  "template",
		},
		{
			name:     "mixed greeting and complex",
			query:    "hello, build me an app",
			category: CodeGeneration,
			handler:  "llm-reasoning",
		},
		{
			name:     "planning query",
			query:    "design a scalable system",
			category: Planning,
			handler:  "llm-fast",
		},
		{
			name:     "explanation query",
			query:    "what is the difference between docker and kubernetes?",
			category: Explanation,
			handler:  "llm-reasoning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ic.Route(tt.query)

			if decision.Category != tt.category {
				t.Errorf("Route(%q).Category = %v, want %v", tt.query, decision.Category, tt.category)
			}

			if decision.Handler != tt.handler {
				t.Errorf("Route(%q).Handler = %v, want %v", tt.query, decision.Handler, tt.handler)
			}

			if decision.Reasoning == nil || len(decision.Reasoning) == 0 {
				t.Errorf("Route(%q).Reasoning should not be empty", tt.query)
			}
		})
	}
}

func BenchmarkRoute(b *testing.B) {
	ic := NewIntentClassifier()
	queries := []string{
		"hello",
		"what is 2+2?",
		"write a Python function to calculate fibonacci",
		"what is docker?",
		"design a microservices architecture",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		ic.Route(query)
	}
}

func BenchmarkRouteSimple(b *testing.B) {
	ic := NewIntentClassifier()
	query := "what is 2+2?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Route(query)
	}
}

func BenchmarkRouteComplex(b *testing.B) {
	ic := NewIntentClassifier()
	query := "Build a React todo app with authentication and database integration"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Route(query)
	}
}

func BenchmarkExtractFeatures(b *testing.B) {
	ic := NewIntentClassifier()
	query := "what is 2+2?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.extractFeatures(query)
	}
}

func BenchmarkExtractFeaturesComplex(b *testing.B) {
	ic := NewIntentClassifier()
	query := "Build a React todo app with authentication and database integration and deploy it to AWS"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.extractFeatures(query)
	}
}

func BenchmarkScoreIntentComplex(b *testing.B) {
	ic := NewIntentClassifier()
	features := QueryFeatures{
		WordCount:       15,
		HasQuestionMark: false,
		StartsWithWH:    false,
		HasCodeTerms:    true,
		HasActionVerbs:  true,
		HasMultiPart:    true,
		HasConstraints:  true,
		OriginalQuery:   "Build a React todo app with authentication and database integration and deploy it to AWS",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.scoreIntent(features)
	}
}

func BenchmarkClassify(b *testing.B) {
	ic := NewIntentClassifier()
	query := "what is 2+2?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Classify(query)
	}
}

func BenchmarkClassifyComplex(b *testing.B) {
	ic := NewIntentClassifier()
	query := "Build a React todo app with authentication and database integration"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.Classify(query)
	}
}
