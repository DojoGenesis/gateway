package agent

import (
	"context"
	"strings"
	"testing"
)

func TestMiniDelegationAgent_ClassifyIntent_ThinkQueries(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedIntent Intent
		minConfidence  float64
	}{
		{
			name:           "analyze architecture",
			query:          "analyze the architecture of this system",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "explain concept",
			query:          "explain why microservices are better",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "plan strategy",
			query:          "plan a migration strategy for our database",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "compare approaches",
			query:          "compare REST vs GraphQL approaches",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "evaluate trade-offs",
			query:          "evaluate the trade-offs of using Redis",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "design system",
			query:          "design a scalable notification system",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
		{
			name:           "understand concept",
			query:          "help me understand CQRS pattern",
			expectedIntent: IntentThink,
			minConfidence:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent != tt.expectedIntent {
				t.Errorf("ClassifyIntent() intent = %v, want %v", intent, tt.expectedIntent)
			}

			if confidence < tt.minConfidence {
				t.Errorf("ClassifyIntent() confidence = %v, want >= %v", confidence, tt.minConfidence)
			}
		})
	}
}

func TestMiniDelegationAgent_ClassifyIntent_SearchQueries(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedIntent Intent
		minConfidence  float64
	}{
		{
			name:           "find documentation",
			query:          "find documentation for React hooks",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "search examples",
			query:          "search for examples of async/await",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "what is",
			query:          "what is Kubernetes",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "who is",
			query:          "who is the maintainer of Vue.js",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "research topic",
			query:          "research best practices for API versioning",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "lookup information",
			query:          "lookup the latest Python version",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
		{
			name:           "show me tutorial",
			query:          "show me a tutorial on Docker",
			expectedIntent: IntentSearch,
			minConfidence:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent != tt.expectedIntent {
				t.Errorf("ClassifyIntent() intent = %v, want %v", intent, tt.expectedIntent)
			}

			if confidence < tt.minConfidence {
				t.Errorf("ClassifyIntent() confidence = %v, want >= %v", confidence, tt.minConfidence)
			}
		})
	}
}

func TestMiniDelegationAgent_ClassifyIntent_BuildQueries(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedIntent Intent
		minConfidence  float64
	}{
		{
			name:           "create component",
			query:          "create a new React component for user profile",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "build feature",
			query:          "build an authentication system",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "generate code",
			query:          "generate code for a REST API",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "write code",
			query:          "write code to parse JSON",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "implement function",
			query:          "implement a binary search function",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "develop app",
			query:          "develop a todo list application",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
		{
			name:           "setup project",
			query:          "setup a new Node.js project",
			expectedIntent: IntentBuild,
			minConfidence:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent != tt.expectedIntent {
				t.Errorf("ClassifyIntent() intent = %v, want %v", intent, tt.expectedIntent)
			}

			if confidence < tt.minConfidence {
				t.Errorf("ClassifyIntent() confidence = %v, want >= %v", confidence, tt.minConfidence)
			}
		})
	}
}

func TestMiniDelegationAgent_ClassifyIntent_DebugQueries(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedIntent Intent
		minConfidence  float64
	}{
		{
			name:           "fix error",
			query:          "fix this TypeError in my code",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "debug issue",
			query:          "debug why this function returns undefined",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "validate code",
			query:          "validate this SQL query",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "check correctness",
			query:          "check if this algorithm is correct",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "test function",
			query:          "test this authentication function",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "troubleshoot problem",
			query:          "troubleshoot the failing tests",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
		{
			name:           "resolve bug",
			query:          "resolve the memory leak bug",
			expectedIntent: IntentDebug,
			minConfidence:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent != tt.expectedIntent {
				t.Errorf("ClassifyIntent() intent = %v, want %v", intent, tt.expectedIntent)
			}

			if confidence < tt.minConfidence {
				t.Errorf("ClassifyIntent() confidence = %v, want >= %v", confidence, tt.minConfidence)
			}
		})
	}
}

func TestMiniDelegationAgent_ClassifyIntent_GeneralQueries(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedIntent Intent
		minConfidence  float64
	}{
		{
			name:           "greeting hello",
			query:          "hello",
			expectedIntent: IntentGeneral,
			minConfidence:  0.5,
		},
		{
			name:           "greeting hi",
			query:          "hi there",
			expectedIntent: IntentGeneral,
			minConfidence:  0.5,
		},
		{
			name:           "help request",
			query:          "help",
			expectedIntent: IntentGeneral,
			minConfidence:  0.5,
		},
		{
			name:           "thanks",
			query:          "thank you",
			expectedIntent: IntentGeneral,
			minConfidence:  0.5,
		},
		{
			name:           "goodbye",
			query:          "goodbye",
			expectedIntent: IntentGeneral,
			minConfidence:  0.5,
		},
		{
			name:           "empty query",
			query:          "",
			expectedIntent: IntentGeneral,
			minConfidence:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent != tt.expectedIntent {
				t.Errorf("ClassifyIntent() intent = %v, want %v", intent, tt.expectedIntent)
			}

			if confidence < tt.minConfidence {
				t.Errorf("ClassifyIntent() confidence = %v, want >= %v", confidence, tt.minConfidence)
			}
		})
	}
}

func TestMiniDelegationAgent_ClassifyIntentWithConfidence(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name              string
		query             string
		expectedIntent    Intent
		expectedReasoning string
	}{
		{
			name:              "think with reasoning",
			query:             "analyze and compare these two approaches",
			expectedIntent:    IntentThink,
			expectedReasoning: "matched THINK keywords",
		},
		{
			name:              "search with reasoning",
			query:             "find and research documentation",
			expectedIntent:    IntentSearch,
			expectedReasoning: "matched SEARCH keywords",
		},
		{
			name:              "empty query reasoning",
			query:             "",
			expectedIntent:    IntentGeneral,
			expectedReasoning: "empty query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.ClassifyIntentWithConfidence(ctx, tt.query)

			if result.Intent != tt.expectedIntent {
				t.Errorf("ClassifyIntentWithConfidence() intent = %v, want %v", result.Intent, tt.expectedIntent)
			}

			if result.Reasoning == "" {
				t.Errorf("ClassifyIntentWithConfidence() reasoning is empty")
			}
		})
	}
}

func TestMiniDelegationAgent_SetAndGetKeywords(t *testing.T) {
	agent := NewMiniDelegationAgent()

	customKeywords := []string{"custom1", "custom2", "custom3"}
	agent.SetKeywords(IntentThink, customKeywords)

	retrieved := agent.GetKeywords(IntentThink)

	if len(retrieved) != len(customKeywords) {
		t.Errorf("GetKeywords() length = %v, want %v", len(retrieved), len(customKeywords))
	}

	for i, keyword := range customKeywords {
		if retrieved[i] != keyword {
			t.Errorf("GetKeywords()[%d] = %v, want %v", i, retrieved[i], keyword)
		}
	}
}

func TestMiniDelegationAgent_ThreadSafety(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	queries := []string{
		"analyze this code",
		"find documentation",
		"create a function",
		"fix this bug",
		"hello",
	}

	done := make(chan bool, len(queries)*10)

	for i := 0; i < 10; i++ {
		for _, query := range queries {
			go func(q string) {
				agent.ClassifyIntent(ctx, q)
				done <- true
			}(query)
		}
	}

	for i := 0; i < len(queries)*10; i++ {
		<-done
	}
}

func TestMiniDelegationAgent_EdgeCases(t *testing.T) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "very long query",
			query: strings.Repeat("analyze this code ", 100),
		},
		{
			name:  "special characters",
			query: "fix this @#$% error!!!",
		},
		{
			name:  "unicode characters",
			query: "créer une fonction 函数",
		},
		{
			name:  "whitespace only",
			query: "   \t\n   ",
		},
		{
			name:  "mixed case",
			query: "FiNd DoCuMeNtAtIoN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := agent.ClassifyIntent(ctx, tt.query)

			if intent == "" {
				t.Errorf("ClassifyIntent() returned empty intent")
			}

			if confidence < 0 || confidence > 1 {
				t.Errorf("ClassifyIntent() confidence = %v, want 0.0-1.0", confidence)
			}
		})
	}
}

func TestIntent_String(t *testing.T) {
	tests := []struct {
		intent   Intent
		expected string
	}{
		{IntentThink, "THINK"},
		{IntentSearch, "SEARCH"},
		{IntentBuild, "BUILD"},
		{IntentDebug, "DEBUG"},
		{IntentGeneral, "GENERAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.intent.String(); got != tt.expected {
				t.Errorf("Intent.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMiniDelegationAgent_ClassifyIntent(b *testing.B) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()
	query := "analyze the architecture of this microservices system"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.ClassifyIntent(ctx, query)
	}
}

func BenchmarkMiniDelegationAgent_ClassifyIntent_Short(b *testing.B) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()
	query := "hello"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.ClassifyIntent(ctx, query)
	}
}

func BenchmarkMiniDelegationAgent_ClassifyIntent_Long(b *testing.B) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()
	query := strings.Repeat("analyze and explain the reasoning behind this architectural decision and compare it with alternative approaches ", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.ClassifyIntent(ctx, query)
	}
}

func BenchmarkMiniDelegationAgent_ClassifyIntentWithConfidence(b *testing.B) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()
	query := "create a new authentication system with OAuth2"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.ClassifyIntentWithConfidence(ctx, query)
	}
}

func BenchmarkMiniDelegationAgent_Parallel(b *testing.B) {
	agent := NewMiniDelegationAgent()
	ctx := context.Background()
	queries := []string{
		"analyze this code",
		"find documentation",
		"create a function",
		"fix this bug",
		"hello",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			query := queries[i%len(queries)]
			agent.ClassifyIntent(ctx, query)
			i++
		}
	})
}
