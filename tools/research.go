package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Type    string `json:"type"`
}

type Entity struct {
	Text   string `json:"text"`
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

func GatherSources(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	topic, ok := params["topic"].(string)
	if !ok || topic == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "topic parameter is required",
		}, nil
	}

	sources := GetStringSliceParam(params, "sources", []string{"web"})
	maxResults := GetIntParam(params, "max_results", 10)

	allSources := []Source{}
	errors := []string{}

	for _, sourceType := range sources {
		switch sourceType {
		case "web":
			webResults, err := WebSearch(ctx, map[string]interface{}{
				"query":       topic,
				"max_results": maxResults,
			})
			if err != nil {
				errors = append(errors, fmt.Sprintf("web search error: %v", err))
				continue
			}

			if success, _ := webResults["success"].(bool); success {
				results, _ := webResults["results"].([]map[string]interface{})
				for _, r := range results {
					allSources = append(allSources, Source{
						Title:   getString(r, "title"),
						URL:     getString(r, "link"),
						Snippet: getString(r, "snippet"),
						Type:    "web",
					})
				}
			} else {
				errors = append(errors, fmt.Sprintf("web search failed: %v", webResults["error"]))
			}

		case "academic":
			errors = append(errors, "academic search not yet implemented")

		default:
			errors = append(errors, fmt.Sprintf("unknown source type: %s", sourceType))
		}
	}

	result := map[string]interface{}{
		"success": len(allSources) > 0,
		"topic":   topic,
		"sources": allSources,
		"count":   len(allSources),
	}

	if len(errors) > 0 {
		result["warnings"] = errors
	}

	return result, nil
}

func SynthesizeInfo(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	sourcesParam, ok := params["sources"]
	if !ok {
		return map[string]interface{}{
			"success": false,
			"error":   "sources parameter is required",
		}, nil
	}

	var sources []map[string]interface{}
	switch v := sourcesParam.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				sources = append(sources, m)
			}
		}
	case []map[string]interface{}:
		sources = v
	default:
		return map[string]interface{}{
			"success": false,
			"error":   "sources must be an array of objects",
		}, nil
	}

	if len(sources) == 0 {
		return map[string]interface{}{
			"success": false,
			"error":   "sources array cannot be empty",
		}, nil
	}

	focusArea := GetStringParam(params, "focus_area", "")
	maxLength := GetIntParam(params, "max_length", 500)

	var allText strings.Builder
	sourceURLs := make([]string, 0, len(sources))
	keyPoints := make([]string, 0)

	for _, source := range sources {
		content := getString(source, "content")
		snippet := getString(source, "snippet")
		url := getString(source, "url")

		if url != "" {
			sourceURLs = append(sourceURLs, url)
		}

		text := content
		if text == "" {
			text = snippet
		}

		if text != "" {
			allText.WriteString(text)
			allText.WriteString(" ")

			sentences := splitIntoSentences(text)
			if len(sentences) > 0 {
				keyPoints = append(keyPoints, sentences[0])
			}
		}
	}

	fullText := allText.String()

	if focusArea != "" {
		fullText = extractRelevantSections(fullText, focusArea)
	}

	summary := summarizeText(fullText, maxLength)

	return map[string]interface{}{
		"success":      true,
		"summary":      summary,
		"key_points":   keyPoints,
		"source_urls":  sourceURLs,
		"source_count": len(sources),
	}, nil
}

func FactCheck(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	claim, ok := params["claim"].(string)
	if !ok || claim == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "claim parameter is required",
		}, nil
	}

	maxResults := GetIntParam(params, "max_results", 5)

	searchResults, err := WebSearch(ctx, map[string]interface{}{
		"query":       claim + " fact check",
		"max_results": maxResults,
	})

	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("search failed: %v", err),
		}, nil
	}

	if success, _ := searchResults["success"].(bool); !success {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("search failed: %v", searchResults["error"]),
		}, nil
	}

	results, _ := searchResults["results"].([]map[string]interface{})

	supportingEvidence := []map[string]interface{}{}
	contradictingEvidence := []map[string]interface{}{}

	claimLower := strings.ToLower(claim)
	claimWords := strings.Fields(claimLower)

	for _, r := range results {
		snippet := strings.ToLower(getString(r, "snippet"))
		title := strings.ToLower(getString(r, "title"))

		evidence := map[string]interface{}{
			"title":   getString(r, "title"),
			"url":     getString(r, "link"),
			"snippet": getString(r, "snippet"),
		}

		matchCount := 0
		for _, word := range claimWords {
			if len(word) > 3 && (strings.Contains(snippet, word) || strings.Contains(title, word)) {
				matchCount++
			}
		}

		if strings.Contains(snippet, "false") || strings.Contains(snippet, "incorrect") ||
			strings.Contains(snippet, "debunk") || strings.Contains(title, "false") {
			contradictingEvidence = append(contradictingEvidence, evidence)
		} else if matchCount > len(claimWords)/2 {
			supportingEvidence = append(supportingEvidence, evidence)
		}
	}

	assessment := "uncertain"
	confidence := 0.0

	if len(contradictingEvidence) > len(supportingEvidence) {
		assessment = "likely_false"
		confidence = float64(len(contradictingEvidence)) / float64(len(results))
	} else if len(supportingEvidence) > len(contradictingEvidence) && len(supportingEvidence) > 0 {
		assessment = "likely_true"
		confidence = float64(len(supportingEvidence)) / float64(len(results))
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return map[string]interface{}{
		"success":                true,
		"claim":                  claim,
		"assessment":             assessment,
		"confidence":             confidence,
		"supporting_evidence":    supportingEvidence,
		"contradicting_evidence": contradictingEvidence,
		"total_sources_checked":  len(results),
	}, nil
}

func AnalyzeSentiment(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "text parameter is required",
		}, nil
	}

	positiveWords := []string{
		"good", "great", "excellent", "amazing", "wonderful", "fantastic", "love",
		"best", "perfect", "beautiful", "awesome", "brilliant", "happy", "joy",
		"pleased", "delighted", "satisfied", "positive", "success", "successful",
	}

	negativeWords := []string{
		"bad", "terrible", "awful", "horrible", "worst", "hate", "poor", "negative",
		"disappointing", "disappointed", "angry", "sad", "unhappy", "frustrated",
		"fail", "failed", "failure", "problem", "issue", "error", "wrong",
	}

	textLower := strings.ToLower(text)
	words := strings.Fields(textLower)

	positiveCount := 0
	negativeCount := 0

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")

		for _, pos := range positiveWords {
			if word == pos {
				positiveCount++
			}
		}

		for _, neg := range negativeWords {
			if word == neg {
				negativeCount++
			}
		}
	}

	totalSentimentWords := positiveCount + negativeCount
	sentiment := "neutral"
	score := 0.0

	if totalSentimentWords > 0 {
		score = float64(positiveCount-negativeCount) / float64(totalSentimentWords)

		if score > 0.2 {
			sentiment = "positive"
		} else if score < -0.2 {
			sentiment = "negative"
		}
	}

	confidence := 0.5
	if totalSentimentWords > 0 {
		confidence = float64(totalSentimentWords) / float64(len(words))
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	return map[string]interface{}{
		"success":        true,
		"text":           text,
		"sentiment":      sentiment,
		"score":          score,
		"confidence":     confidence,
		"positive_count": positiveCount,
		"negative_count": negativeCount,
		"word_count":     len(words),
	}, nil
}

func ExtractEntities(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "text parameter is required",
		}, nil
	}

	entityTypes := GetStringSliceParam(params, "entity_types", []string{"person", "organization", "location", "date", "email", "url"})

	entities := []Entity{}

	for _, entityType := range entityTypes {
		switch entityType {
		case "person":
			entities = append(entities, extractPersons(text)...)
		case "organization":
			entities = append(entities, extractOrganizations(text)...)
		case "location":
			entities = append(entities, extractLocations(text)...)
		case "date":
			entities = append(entities, extractDates(text)...)
		case "email":
			entities = append(entities, extractEmails(text)...)
		case "url":
			entities = append(entities, extractURLs(text)...)
		}
	}

	entityMap := make(map[string][]string)
	for _, entity := range entities {
		entityMap[entity.Type] = append(entityMap[entity.Type], entity.Text)
	}

	return map[string]interface{}{
		"success":      true,
		"text":         text,
		"entities":     entities,
		"entity_count": len(entities),
		"by_type":      entityMap,
	}, nil
}

func extractPersons(text string) []Entity {
	personPattern := regexp.MustCompile(`\b([A-Z][a-z]+\s+[A-Z][a-z]+)\b`)
	matches := personPattern.FindAllStringIndex(text, -1)

	entities := []Entity{}
	for _, match := range matches {
		entities = append(entities, Entity{
			Text:   text[match[0]:match[1]],
			Type:   "person",
			Offset: match[0],
			Length: match[1] - match[0],
		})
	}
	return entities
}

func extractOrganizations(text string) []Entity {
	orgPatterns := []string{
		`\b([A-Z][a-z]+\s+(Inc|Corp|LLC|Ltd|Company|Corporation))\b`,
		`\b([A-Z][A-Z]+)\b`,
	}

	entities := []Entity{}
	for _, pattern := range orgPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringIndex(text, -1)

		for _, match := range matches {
			orgText := text[match[0]:match[1]]
			if len(orgText) > 1 {
				entities = append(entities, Entity{
					Text:   orgText,
					Type:   "organization",
					Offset: match[0],
					Length: match[1] - match[0],
				})
			}
		}
	}
	return entities
}

func extractLocations(text string) []Entity {
	commonLocations := []string{
		"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
		"San Francisco", "Seattle", "Boston", "Washington", "Miami",
		"London", "Paris", "Tokyo", "Beijing", "Berlin", "Moscow",
		"USA", "UK", "Canada", "China", "Japan", "Germany", "France",
	}

	entities := []Entity{}
	textLower := strings.ToLower(text)

	for _, location := range commonLocations {
		locationLower := strings.ToLower(location)
		index := 0
		for {
			idx := strings.Index(textLower[index:], locationLower)
			if idx == -1 {
				break
			}

			actualIndex := index + idx
			entities = append(entities, Entity{
				Text:   text[actualIndex : actualIndex+len(location)],
				Type:   "location",
				Offset: actualIndex,
				Length: len(location),
			})
			index = actualIndex + len(location)
		}
	}

	return entities
}

func extractDates(text string) []Entity {
	datePatterns := []string{
		`\b(\d{1,2}/\d{1,2}/\d{2,4})\b`,
		`\b(\d{4}-\d{2}-\d{2})\b`,
		`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}\b`,
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},?\s+\d{4}\b`,
	}

	entities := []Entity{}
	for _, pattern := range datePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringIndex(text, -1)

		for _, match := range matches {
			entities = append(entities, Entity{
				Text:   text[match[0]:match[1]],
				Type:   "date",
				Offset: match[0],
				Length: match[1] - match[0],
			})
		}
	}
	return entities
}

func extractEmails(text string) []Entity {
	emailPattern := regexp.MustCompile(`\b([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})\b`)
	matches := emailPattern.FindAllStringIndex(text, -1)

	entities := []Entity{}
	for _, match := range matches {
		entities = append(entities, Entity{
			Text:   text[match[0]:match[1]],
			Type:   "email",
			Offset: match[0],
			Length: match[1] - match[0],
		})
	}
	return entities
}

func extractURLs(text string) []Entity {
	urlPattern := regexp.MustCompile(`\b(https?://[^\s]+)\b`)
	matches := urlPattern.FindAllStringIndex(text, -1)

	entities := []Entity{}
	for _, match := range matches {
		entities = append(entities, Entity{
			Text:   text[match[0]:match[1]],
			Type:   "url",
			Offset: match[0],
			Length: match[1] - match[0],
		})
	}
	return entities
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func splitIntoSentences(text string) []string {
	sentencePattern := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentencePattern.Split(text, -1)

	result := []string{}
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func extractRelevantSections(text, focusArea string) string {
	sentences := splitIntoSentences(text)
	focusLower := strings.ToLower(focusArea)
	focusWords := strings.Fields(focusLower)

	relevant := []string{}
	for _, sentence := range sentences {
		sentenceLower := strings.ToLower(sentence)
		matchCount := 0
		for _, word := range focusWords {
			if len(word) > 3 && strings.Contains(sentenceLower, word) {
				matchCount++
			}
		}

		if matchCount > 0 {
			relevant = append(relevant, sentence)
		}
	}

	if len(relevant) == 0 {
		return text
	}

	return strings.Join(relevant, ". ")
}

func summarizeText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	sentences := splitIntoSentences(text)
	if len(sentences) == 0 {
		if len(text) > maxLength {
			return text[:maxLength] + "..."
		}
		return text
	}

	var summary strings.Builder
	for _, sentence := range sentences {
		if summary.Len()+len(sentence) > maxLength {
			break
		}
		if summary.Len() > 0 {
			summary.WriteString(". ")
		}
		summary.WriteString(sentence)
	}

	result := summary.String()
	if result == "" && len(sentences) > 0 {
		result = sentences[0]
		if len(result) > maxLength {
			result = result[:maxLength] + "..."
		}
	}

	return result
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "gather_sources",
		Description: "Gather information from multiple sources (web, academic) on a given topic",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"topic": map[string]interface{}{
					"type":        "string",
					"description": "The research topic or query",
				},
				"sources": map[string]interface{}{
					"type":        "array",
					"description": "Types of sources to gather (web, academic)",
					"items": map[string]interface{}{
						"type": "string",
					},
					"default": []string{"web"},
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results per source (default: 10)",
					"default":     10,
				},
			},
			"required": []string{"topic"},
		},
		Function: GatherSources,
	})

	RegisterTool(&ToolDefinition{
		Name:        "synthesize_info",
		Description: "Synthesize information from multiple sources into a coherent summary",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sources": map[string]interface{}{
					"type":        "array",
					"description": "Array of source objects with content/snippet",
					"items": map[string]interface{}{
						"type": "object",
					},
				},
				"focus_area": map[string]interface{}{
					"type":        "string",
					"description": "Optional focus area to extract relevant information",
				},
				"max_length": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum length of summary in characters (default: 500)",
					"default":     500,
				},
			},
			"required": []string{"sources"},
		},
		Function: SynthesizeInfo,
	})

	RegisterTool(&ToolDefinition{
		Name:        "fact_check",
		Description: "Check the veracity of a claim by searching for supporting and contradicting evidence",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"claim": map[string]interface{}{
					"type":        "string",
					"description": "The claim to fact-check",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of sources to check (default: 5)",
					"default":     5,
				},
			},
			"required": []string{"claim"},
		},
		Function: FactCheck,
	})

	RegisterTool(&ToolDefinition{
		Name:        "analyze_sentiment",
		Description: "Analyze the sentiment of a given text (positive, negative, neutral)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to analyze for sentiment",
				},
			},
			"required": []string{"text"},
		},
		Function: AnalyzeSentiment,
	})

	RegisterTool(&ToolDefinition{
		Name:        "extract_entities",
		Description: "Extract named entities (person, organization, location, date, email, url) from text",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to extract entities from",
				},
				"entity_types": map[string]interface{}{
					"type":        "array",
					"description": "Types of entities to extract",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"person", "organization", "location", "date", "email", "url"},
					},
					"default": []string{"person", "organization", "location", "date", "email", "url"},
				},
			},
			"required": []string{"text"},
		},
		Function: ExtractEntities,
	})
}
