package agent

import (
	"testing"
)

func TestIntentClassifier_ClassifyWithConfidence(t *testing.T) {
	ic := NewIntentClassifier()

	tests := []struct {
		name              string
		query             string
		expectedType      QueryType
		minConfidence     float64
		maxConfidence     float64
		expectedReasoning string
	}{
		{
			name:              "Empty query",
			query:             "",
			expectedType:      Simple,
			minConfidence:     1.0,
			maxConfidence:     1.0,
			expectedReasoning: "empty query",
		},
		{
			name:              "Simple greeting",
			query:             "hello",
			expectedType:      Simple,
			minConfidence:     0.9,
			maxConfidence:     1.0,
			expectedReasoning: "contains simple keyword",
		},
		{
			name:              "Complex code generation",
			query:             "write a Python function to calculate fibonacci",
			expectedType:      Complex,
			minConfidence:     0.8,
			maxConfidence:     1.0,
			expectedReasoning: "complex action indicator",
		},
		{
			name:              "Simple question",
			query:             "what is REST API",
			expectedType:      Simple,
			minConfidence:     0.7,
			maxConfidence:     0.8,
			expectedReasoning: "simple question pattern",
		},
		{
			name:              "Long complex query",
			query:             "build a complete e-commerce platform with user authentication, shopping cart, payment integration, and admin dashboard using React and Node.js",
			expectedType:      Complex,
			minConfidence:     0.9,
			maxConfidence:     1.0,
			expectedReasoning: "complex action indicator",
		},
		{
			name:              "Medium length query",
			query:             "this is a test query with exactly ten words here",
			expectedType:      Complex,
			minConfidence:     0.7,
			maxConfidence:     0.9,
			expectedReasoning: "Code generation",
		},
		{
			name:              "Ambiguous short query",
			query:             "help me",
			expectedType:      Simple,
			minConfidence:     0.9,
			maxConfidence:     0.95,
			expectedReasoning: "simple keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ic.ClassifyWithConfidence(tt.query)

			if result.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, result.Type)
			}

			if result.Confidence < tt.minConfidence || result.Confidence > tt.maxConfidence {
				t.Errorf("Expected confidence between %.2f and %.2f, got %.2f",
					tt.minConfidence, tt.maxConfidence, result.Confidence)
			}

			if result.Reasoning == "" {
				t.Error("Expected non-empty reasoning")
			}

			t.Logf("Query: %s", tt.query)
			t.Logf("Type: %v, Confidence: %.2f, Reasoning: %s",
				result.Type, result.Confidence, result.Reasoning)
		})
	}
}

func TestIntentClassifier_ConfidenceScores(t *testing.T) {
	ic := NewIntentClassifier()

	// Test that confidence scores are properly ordered
	tests := []struct {
		query                 string
		expectedMinConfidence float64
	}{
		{"hello", 0.9},               // Strong simple indicator (greeting)
		{"build something", 0.6},     // Action verb (updated for new implementation)
		{"what is", 0.6},             // Short factual question (updated for new implementation)
		{"a b c d e f g h i j", 0.5}, // Weak complex indicator (word count)
		{"random text", 0.4},         // Weakest (default)
	}

	for _, tt := range tests {
		result := ic.ClassifyWithConfidence(tt.query)
		if result.Confidence < tt.expectedMinConfidence {
			t.Errorf("Query '%s': expected confidence >= %.2f, got %.2f",
				tt.query, tt.expectedMinConfidence, result.Confidence)
		}
	}
}

func TestIntentClassifier_BackwardCompatibility(t *testing.T) {
	ic := NewIntentClassifier()

	// Ensure the old Classify method still works and returns same results
	testQueries := []string{
		"hello",
		"build a React app",
		"what is API",
		"",
	}

	for _, query := range testQueries {
		oldResult := ic.Classify(query)
		newResult := ic.ClassifyWithConfidence(query)

		if oldResult != newResult.Type {
			t.Errorf("Query '%s': Classify() returned %v but ClassifyWithConfidence() returned %v",
				query, oldResult, newResult.Type)
		}
	}
}

func BenchmarkClassifyWithConfidence(b *testing.B) {
	ic := NewIntentClassifier()
	query := "build a comprehensive web application with user authentication and database integration"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.ClassifyWithConfidence(query)
	}
}
