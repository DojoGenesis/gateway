package tools

import (
	"context"
	"strings"
	"testing"
)

func TestGatherSources(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "missing topic",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "topic parameter is required",
		},
		{
			name: "empty topic",
			params: map[string]interface{}{
				"topic": "",
			},
			wantSuccess: false,
			wantError:   "topic parameter is required",
		},
		{
			name: "valid topic with web source",
			params: map[string]interface{}{
				"topic":       "artificial intelligence",
				"sources":     []string{"web"},
				"max_results": 5,
			},
			wantSuccess: false,
		},
		{
			name: "multiple source types",
			params: map[string]interface{}{
				"topic":       "climate change",
				"sources":     []string{"web", "academic"},
				"max_results": 3,
			},
			wantSuccess: false,
		},
		{
			name: "unknown source type",
			params: map[string]interface{}{
				"topic":   "test",
				"sources": []string{"unknown"},
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GatherSources(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("GatherSources returned error: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Logf("success = %v, want %v", success, tt.wantSuccess)
			}

			if !tt.wantSuccess && tt.wantError != "" {
				errMsg, _ := result["error"].(string)
				if errMsg != tt.wantError {
					t.Errorf("error = %v, want %v", errMsg, tt.wantError)
				}
			}
		})
	}
}

func TestSynthesizeInfo(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
		wantError   string
		checkResult func(*testing.T, map[string]interface{})
	}{
		{
			name:        "missing sources",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "sources parameter is required",
		},
		{
			name: "empty sources array",
			params: map[string]interface{}{
				"sources": []interface{}{},
			},
			wantSuccess: false,
			wantError:   "sources array cannot be empty",
		},
		{
			name: "invalid sources type",
			params: map[string]interface{}{
				"sources": "not an array",
			},
			wantSuccess: false,
			wantError:   "sources must be an array of objects",
		},
		{
			name: "valid sources with content",
			params: map[string]interface{}{
				"sources": []interface{}{
					map[string]interface{}{
						"content": "Artificial intelligence is transforming the world. Machine learning is a subset of AI.",
						"url":     "https://example.com/1",
					},
					map[string]interface{}{
						"content": "Deep learning has made significant progress. Neural networks are becoming more powerful.",
						"url":     "https://example.com/2",
					},
				},
				"max_length": 200,
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				summary, ok := result["summary"].(string)
				if !ok {
					t.Error("expected summary field")
					return
				}
				if summary == "" {
					t.Error("summary should not be empty")
				}
				if len(summary) > 200 {
					t.Errorf("summary length %d exceeds max_length 200", len(summary))
				}

				keyPoints, ok := result["key_points"].([]string)
				if !ok {
					t.Error("expected key_points field")
					return
				}
				if len(keyPoints) != 2 {
					t.Errorf("expected 2 key points, got %d", len(keyPoints))
				}

				sourceURLs, ok := result["source_urls"].([]string)
				if !ok {
					t.Error("expected source_urls field")
					return
				}
				if len(sourceURLs) != 2 {
					t.Errorf("expected 2 source URLs, got %d", len(sourceURLs))
				}
			},
		},
		{
			name: "sources with focus area",
			params: map[string]interface{}{
				"sources": []interface{}{
					map[string]interface{}{
						"content": "Machine learning uses algorithms to learn from data. Deep learning is a powerful technique.",
					},
				},
				"focus_area": "deep learning",
				"max_length": 100,
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				summary, _ := result["summary"].(string)
				if summary == "" {
					t.Error("summary should not be empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SynthesizeInfo(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("SynthesizeInfo returned error: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", success, tt.wantSuccess)
			}

			if !tt.wantSuccess && tt.wantError != "" {
				errMsg, _ := result["error"].(string)
				if errMsg != tt.wantError {
					t.Errorf("error = %v, want %v", errMsg, tt.wantError)
				}
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestFactCheck(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
		wantError   string
	}{
		{
			name:        "missing claim",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "claim parameter is required",
		},
		{
			name: "empty claim",
			params: map[string]interface{}{
				"claim": "",
			},
			wantSuccess: false,
			wantError:   "claim parameter is required",
		},
		{
			name: "valid claim",
			params: map[string]interface{}{
				"claim":       "The Earth is flat",
				"max_results": 5,
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FactCheck(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("FactCheck returned error: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Logf("success = %v, want %v", success, tt.wantSuccess)
			}

			if !tt.wantSuccess && tt.wantError != "" {
				errMsg, _ := result["error"].(string)
				if errMsg != tt.wantError {
					t.Errorf("error = %v, want %v", errMsg, tt.wantError)
				}
			}
		})
	}
}

func TestAnalyzeSentiment(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]interface{}
		wantSuccess   bool
		wantError     string
		wantSentiment string
		checkResult   func(*testing.T, map[string]interface{})
	}{
		{
			name:        "missing text",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "text parameter is required",
		},
		{
			name: "empty text",
			params: map[string]interface{}{
				"text": "",
			},
			wantSuccess: false,
			wantError:   "text parameter is required",
		},
		{
			name: "positive sentiment",
			params: map[string]interface{}{
				"text": "This is a wonderful day! I love this amazing product. It's absolutely fantastic and excellent.",
			},
			wantSuccess:   true,
			wantSentiment: "positive",
			checkResult: func(t *testing.T, result map[string]interface{}) {
				sentiment, _ := result["sentiment"].(string)
				if sentiment != "positive" {
					t.Errorf("sentiment = %v, want positive", sentiment)
				}
				score, _ := result["score"].(float64)
				if score <= 0 {
					t.Errorf("score = %v, want positive score", score)
				}
				positiveCount, _ := result["positive_count"].(int)
				if positiveCount == 0 {
					t.Error("expected positive_count > 0")
				}
			},
		},
		{
			name: "negative sentiment",
			params: map[string]interface{}{
				"text": "This is terrible! I hate this awful product. It's horrible and disappointing. Complete failure.",
			},
			wantSuccess:   true,
			wantSentiment: "negative",
			checkResult: func(t *testing.T, result map[string]interface{}) {
				sentiment, _ := result["sentiment"].(string)
				if sentiment != "negative" {
					t.Errorf("sentiment = %v, want negative", sentiment)
				}
				score, _ := result["score"].(float64)
				if score >= 0 {
					t.Errorf("score = %v, want negative score", score)
				}
				negativeCount, _ := result["negative_count"].(int)
				if negativeCount == 0 {
					t.Error("expected negative_count > 0")
				}
			},
		},
		{
			name: "neutral sentiment",
			params: map[string]interface{}{
				"text": "The weather is moderate today. The temperature is average.",
			},
			wantSuccess:   true,
			wantSentiment: "neutral",
			checkResult: func(t *testing.T, result map[string]interface{}) {
				sentiment, _ := result["sentiment"].(string)
				if sentiment != "neutral" {
					t.Errorf("sentiment = %v, want neutral", sentiment)
				}
			},
		},
		{
			name: "mixed sentiment",
			params: map[string]interface{}{
				"text": "I love the good features, but I hate the bad performance.",
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				positiveCount, _ := result["positive_count"].(int)
				negativeCount, _ := result["negative_count"].(int)
				if positiveCount == 0 || negativeCount == 0 {
					t.Error("expected both positive and negative counts > 0")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeSentiment(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("AnalyzeSentiment returned error: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", success, tt.wantSuccess)
			}

			if !tt.wantSuccess && tt.wantError != "" {
				errMsg, _ := result["error"].(string)
				if errMsg != tt.wantError {
					t.Errorf("error = %v, want %v", errMsg, tt.wantError)
				}
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestExtractEntities(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
		wantError   string
		checkResult func(*testing.T, map[string]interface{})
	}{
		{
			name:        "missing text",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "text parameter is required",
		},
		{
			name: "empty text",
			params: map[string]interface{}{
				"text": "",
			},
			wantSuccess: false,
			wantError:   "text parameter is required",
		},
		{
			name: "extract persons",
			params: map[string]interface{}{
				"text":         "John Smith and Mary Johnson met yesterday.",
				"entity_types": []string{"person"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) != 2 {
					t.Errorf("expected 2 person entities, got %d", len(entities))
				}
				for _, e := range entities {
					if e.Type != "person" {
						t.Errorf("expected type person, got %s", e.Type)
					}
				}
			},
		},
		{
			name: "extract emails",
			params: map[string]interface{}{
				"text":         "Contact me at john@example.com or mary.smith@company.org",
				"entity_types": []string{"email"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) != 2 {
					t.Errorf("expected 2 email entities, got %d", len(entities))
				}
				for _, e := range entities {
					if e.Type != "email" {
						t.Errorf("expected type email, got %s", e.Type)
					}
				}
			},
		},
		{
			name: "extract URLs",
			params: map[string]interface{}{
				"text":         "Visit https://example.com or http://test.org for more info.",
				"entity_types": []string{"url"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) != 2 {
					t.Errorf("expected 2 URL entities, got %d", len(entities))
				}
				for _, e := range entities {
					if e.Type != "url" {
						t.Errorf("expected type url, got %s", e.Type)
					}
				}
			},
		},
		{
			name: "extract dates",
			params: map[string]interface{}{
				"text":         "The meeting is on 12/25/2023 and January 15, 2024.",
				"entity_types": []string{"date"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) < 1 {
					t.Error("expected at least 1 date entity")
				}
				for _, e := range entities {
					if e.Type != "date" {
						t.Errorf("expected type date, got %s", e.Type)
					}
				}
			},
		},
		{
			name: "extract organizations",
			params: map[string]interface{}{
				"text":         "Google Inc and Microsoft Corporation are tech companies. IBM and NASA are also mentioned.",
				"entity_types": []string{"organization"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) < 1 {
					t.Error("expected at least 1 organization entity")
				}
			},
		},
		{
			name: "extract locations",
			params: map[string]interface{}{
				"text":         "I traveled from New York to San Francisco and then to Tokyo.",
				"entity_types": []string{"location"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) < 1 {
					t.Error("expected at least 1 location entity")
				}
			},
		},
		{
			name: "extract multiple entity types",
			params: map[string]interface{}{
				"text":         "John Smith from Google Inc sent an email to mary@example.com on 12/25/2023 from New York.",
				"entity_types": []string{"person", "organization", "email", "date", "location"},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				entities, ok := result["entities"].([]Entity)
				if !ok {
					t.Error("expected entities field")
					return
				}
				if len(entities) < 3 {
					t.Errorf("expected at least 3 entities, got %d", len(entities))
				}

				byType, ok := result["by_type"].(map[string][]string)
				if !ok {
					t.Error("expected by_type field")
					return
				}
				if len(byType) < 2 {
					t.Errorf("expected at least 2 entity types, got %d", len(byType))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractEntities(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("ExtractEntities returned error: %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", success, tt.wantSuccess)
			}

			if !tt.wantSuccess && tt.wantError != "" {
				errMsg, _ := result["error"].(string)
				if errMsg != tt.wantError {
					t.Errorf("error = %v, want %v", errMsg, tt.wantError)
				}
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		want  int
		check func(*testing.T, []string)
	}{
		{
			name: "single sentence",
			text: "This is a single sentence",
			want: 1,
		},
		{
			name: "multiple sentences with periods",
			text: "First sentence. Second sentence. Third sentence.",
			want: 3,
		},
		{
			name: "mixed punctuation",
			text: "What is this? This is a test! And here's another sentence.",
			want: 3,
		},
		{
			name: "empty text",
			text: "",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoSentences(tt.text)
			if len(result) != tt.want {
				t.Errorf("got %d sentences, want %d", len(result), tt.want)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestSummarizeText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		check     func(*testing.T, string)
	}{
		{
			name:      "text shorter than max length",
			text:      "Short text.",
			maxLength: 100,
			check: func(t *testing.T, result string) {
				if result != "Short text." {
					t.Errorf("got %q, want %q", result, "Short text.")
				}
			},
		},
		{
			name:      "text longer than max length",
			text:      "This is the first sentence. This is the second sentence. This is the third sentence.",
			maxLength: 30,
			check: func(t *testing.T, result string) {
				if len(result) > 30 {
					t.Errorf("result length %d exceeds max length 30", len(result))
				}
				if result == "" {
					t.Error("result should not be empty")
				}
			},
		},
		{
			name:      "multiple sentences fit in max length",
			text:      "Short one. Short two. Short three.",
			maxLength: 100,
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "Short one") {
					t.Error("expected first sentence in summary")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeText(tt.text, tt.maxLength)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
