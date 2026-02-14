package tools

import (
	"testing"
)

func TestVerifyAllTools(t *testing.T) {
	if err := VerifyAllTools(); err != nil {
		t.Fatalf("Tool verification failed: %v", err)
	}
}

func TestGetToolCount(t *testing.T) {
	count := GetToolCount()
	expectedCount := 32
	if count != expectedCount {
		t.Errorf("Expected %d tools, got %d", expectedCount, count)
	}
}

func TestListAllTools(t *testing.T) {
	categories := ListAllTools()

	expectedCounts := map[string]int{
		"file_operations": 4,
		"web_operations":  3,
		"web_advanced":    5,
		"computation":     2,
		"system":          1,
		"planning":        5,
		"research":        5,
		"meta":            2,
		"visual":          5,
	}

	for category, expectedCount := range expectedCounts {
		tools, exists := categories[category]
		if !exists {
			t.Errorf("Category %s not found", category)
			continue
		}
		if len(tools) != expectedCount {
			t.Errorf("Category %s: expected %d tools, got %d: %v",
				category, expectedCount, len(tools), tools)
		}
	}
}
