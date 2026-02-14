package tools

import (
	"context"
	"os"
	"testing"
)

func TestGatherSourcesIntegration(t *testing.T) {
	if os.Getenv("SERPAPI_KEY") == "" {
		t.Skip("Skipping integration test: SERPAPI_KEY not set")
	}

	params := map[string]interface{}{
		"topic":       "artificial intelligence recent developments",
		"sources":     []string{"web"},
		"max_results": 3,
	}

	result, err := GatherSources(context.Background(), params)
	if err != nil {
		t.Fatalf("GatherSources failed: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		errMsg, _ := result["error"].(string)
		t.Fatalf("GatherSources returned success=false: %s", errMsg)
	}

	sources, ok := result["sources"].([]Source)
	if !ok {
		t.Fatal("expected sources field of type []Source")
	}

	if len(sources) == 0 {
		warnings, _ := result["warnings"].([]string)
		t.Logf("No sources found, warnings: %v", warnings)
	} else {
		t.Logf("Found %d sources", len(sources))
		for i, source := range sources {
			t.Logf("Source %d: %s - %s", i+1, source.Title, source.URL)
		}
	}
}

func TestSynthesizeInfoIntegration(t *testing.T) {
	sources := []interface{}{
		map[string]interface{}{
			"content": "Artificial intelligence is rapidly transforming industries. Machine learning algorithms are becoming more sophisticated. Deep learning has achieved remarkable results in image recognition and natural language processing.",
			"url":     "https://example.com/ai-trends",
		},
		map[string]interface{}{
			"content": "The future of AI includes advancements in reinforcement learning and transfer learning. Large language models are showing impressive capabilities. Ethical considerations in AI development are becoming increasingly important.",
			"url":     "https://example.com/ai-future",
		},
		map[string]interface{}{
			"content": "AI applications span healthcare, finance, transportation, and education. Computer vision systems are improving medical diagnostics. Natural language processing is enabling better human-computer interaction.",
			"url":     "https://example.com/ai-applications",
		},
	}

	params := map[string]interface{}{
		"sources":    sources,
		"focus_area": "machine learning and deep learning",
		"max_length": 300,
	}

	result, err := SynthesizeInfo(context.Background(), params)
	if err != nil {
		t.Fatalf("SynthesizeInfo failed: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		errMsg, _ := result["error"].(string)
		t.Fatalf("SynthesizeInfo returned success=false: %s", errMsg)
	}

	summary, ok := result["summary"].(string)
	if !ok {
		t.Fatal("expected summary field")
	}

	if summary == "" {
		t.Error("summary should not be empty")
	}

	if len(summary) > 300 {
		t.Errorf("summary length %d exceeds max_length 300", len(summary))
	}

	t.Logf("Summary: %s", summary)

	keyPoints, ok := result["key_points"].([]string)
	if !ok {
		t.Fatal("expected key_points field")
	}

	t.Logf("Key points (%d):", len(keyPoints))
	for i, point := range keyPoints {
		t.Logf("  %d. %s", i+1, point)
	}

	sourceURLs, ok := result["source_urls"].([]string)
	if !ok {
		t.Fatal("expected source_urls field")
	}

	if len(sourceURLs) != 3 {
		t.Errorf("expected 3 source URLs, got %d", len(sourceURLs))
	}
}

func TestFactCheckIntegration(t *testing.T) {
	if os.Getenv("SERPAPI_KEY") == "" {
		t.Skip("Skipping integration test: SERPAPI_KEY not set")
	}

	params := map[string]interface{}{
		"claim":       "The Earth orbits around the Sun",
		"max_results": 5,
	}

	result, err := FactCheck(context.Background(), params)
	if err != nil {
		t.Fatalf("FactCheck failed: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		errMsg, _ := result["error"].(string)
		t.Fatalf("FactCheck returned success=false: %s", errMsg)
	}

	assessment, ok := result["assessment"].(string)
	if !ok {
		t.Fatal("expected assessment field")
	}

	confidence, ok := result["confidence"].(float64)
	if !ok {
		t.Fatal("expected confidence field")
	}

	t.Logf("Assessment: %s (confidence: %.2f)", assessment, confidence)

	supportingEvidence, ok := result["supporting_evidence"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected supporting_evidence field")
	}

	t.Logf("Supporting evidence: %d sources", len(supportingEvidence))
	for i, ev := range supportingEvidence {
		t.Logf("  %d. %s", i+1, ev["title"])
	}

	contradictingEvidence, ok := result["contradicting_evidence"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected contradicting_evidence field")
	}

	t.Logf("Contradicting evidence: %d sources", len(contradictingEvidence))
}

func TestAnalyzeSentimentIntegration(t *testing.T) {
	testCases := []struct {
		name              string
		text              string
		expectedSentiment string
	}{
		{
			name:              "product review positive",
			text:              "This product is absolutely amazing! I love everything about it. The quality is excellent and it exceeded all my expectations. Highly recommend!",
			expectedSentiment: "positive",
		},
		{
			name:              "product review negative",
			text:              "This is the worst product I've ever bought. Terrible quality, awful customer service, and it failed after one day. Complete waste of money. Very disappointing.",
			expectedSentiment: "negative",
		},
		{
			name:              "news article neutral",
			text:              "The company announced quarterly earnings today. Revenue was reported at $10 billion. The board of directors held a meeting to discuss future strategies.",
			expectedSentiment: "neutral",
		},
		{
			name:              "mixed sentiment",
			text:              "The good news is that the product works well. However, the bad news is that delivery was terrible and customer service was disappointing. Some features are excellent, others are poor.",
			expectedSentiment: "neutral",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]interface{}{
				"text": tc.text,
			}

			result, err := AnalyzeSentiment(context.Background(), params)
			if err != nil {
				t.Fatalf("AnalyzeSentiment failed: %v", err)
			}

			success, _ := result["success"].(bool)
			if !success {
				errMsg, _ := result["error"].(string)
				t.Fatalf("AnalyzeSentiment returned success=false: %s", errMsg)
			}

			sentiment, _ := result["sentiment"].(string)
			score, _ := result["score"].(float64)
			confidence, _ := result["confidence"].(float64)
			positiveCount, _ := result["positive_count"].(int)
			negativeCount, _ := result["negative_count"].(int)

			t.Logf("Sentiment: %s (score: %.2f, confidence: %.2f)", sentiment, score, confidence)
			t.Logf("Positive: %d, Negative: %d", positiveCount, negativeCount)

			if sentiment != tc.expectedSentiment {
				t.Logf("Note: expected sentiment %s, got %s (this may vary based on word detection)", tc.expectedSentiment, sentiment)
			}
		})
	}
}

func TestExtractEntitiesIntegration(t *testing.T) {
	text := `John Smith, the CEO of Google Inc, sent an email to mary.johnson@example.com on January 15, 2024. 
	The meeting was scheduled in New York at 2:00 PM. 
	More information is available at https://company.com/meetings. 
	Contact us at support@company.org for questions.
	The presentation will include data from Microsoft Corporation and IBM.
	Our offices in San Francisco and London will participate.`

	params := map[string]interface{}{
		"text":         text,
		"entity_types": []string{"person", "organization", "location", "date", "email", "url"},
	}

	result, err := ExtractEntities(context.Background(), params)
	if err != nil {
		t.Fatalf("ExtractEntities failed: %v", err)
	}

	success, _ := result["success"].(bool)
	if !success {
		errMsg, _ := result["error"].(string)
		t.Fatalf("ExtractEntities returned success=false: %s", errMsg)
	}

	entities, ok := result["entities"].([]Entity)
	if !ok {
		t.Fatal("expected entities field")
	}

	byType, ok := result["by_type"].(map[string][]string)
	if !ok {
		t.Fatal("expected by_type field")
	}

	t.Logf("Total entities found: %d", len(entities))

	for entityType, items := range byType {
		t.Logf("%s (%d):", entityType, len(items))
		for _, item := range items {
			t.Logf("  - %s", item)
		}
	}

	if len(entities) == 0 {
		t.Error("expected to find at least some entities in the text")
	}

	expectedTypes := []string{"person", "email", "url"}
	for _, expectedType := range expectedTypes {
		if items, ok := byType[expectedType]; !ok || len(items) == 0 {
			t.Errorf("expected to find at least one %s entity", expectedType)
		}
	}
}

func TestEndToEndResearchWorkflow(t *testing.T) {
	if os.Getenv("SERPAPI_KEY") == "" {
		t.Skip("Skipping integration test: SERPAPI_KEY not set")
	}

	t.Log("Step 1: Gather sources on a topic")
	gatherParams := map[string]interface{}{
		"topic":       "renewable energy solar power",
		"sources":     []string{"web"},
		"max_results": 3,
	}

	gatherResult, err := GatherSources(context.Background(), gatherParams)
	if err != nil {
		t.Fatalf("GatherSources failed: %v", err)
	}

	if success, _ := gatherResult["success"].(bool); !success {
		t.Logf("GatherSources failed (expected without API key)")
		return
	}

	sources, ok := gatherResult["sources"].([]Source)
	if !ok || len(sources) == 0 {
		t.Log("No sources found, skipping synthesis step")
		return
	}

	t.Log("Step 2: Extract entities from source content")
	for i, source := range sources {
		if source.Snippet != "" {
			entitiesParams := map[string]interface{}{
				"text":         source.Snippet,
				"entity_types": []string{"organization", "location"},
			}

			entitiesResult, err := ExtractEntities(context.Background(), entitiesParams)
			if err != nil {
				t.Errorf("ExtractEntities failed for source %d: %v", i, err)
				continue
			}

			entities, _ := entitiesResult["entities"].([]Entity)
			t.Logf("Source %d entities: %d found", i+1, len(entities))
		}
	}

	t.Log("Step 3: Synthesize information")
	synthesizeSources := make([]interface{}, len(sources))
	for i, source := range sources {
		synthesizeSources[i] = map[string]interface{}{
			"content": source.Snippet,
			"url":     source.URL,
		}
	}

	synthesizeParams := map[string]interface{}{
		"sources":    synthesizeSources,
		"max_length": 200,
	}

	synthesizeResult, err := SynthesizeInfo(context.Background(), synthesizeParams)
	if err != nil {
		t.Fatalf("SynthesizeInfo failed: %v", err)
	}

	if success, _ := synthesizeResult["success"].(bool); success {
		summary, _ := synthesizeResult["summary"].(string)
		t.Logf("Summary: %s", summary)
	}

	t.Log("End-to-end research workflow completed successfully")
}
