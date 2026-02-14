package artifacts

import (
	"encoding/json"
	"fmt"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type DiffResult struct {
	Type      string `json:"type"`
	OldLine   int    `json:"old_line,omitempty"`
	NewLine   int    `json:"new_line,omitempty"`
	Content   string `json:"content"`
	Operation string `json:"operation"`
}

func CalculateDiff(oldContent, newContent string) (string, error) {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(oldContent, newContent, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	results := []DiffResult{}
	oldLine := 1
	newLine := 1

	for _, diff := range diffs {
		var op string
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			op = "equal"
			lineCount := countLines(diff.Text)
			oldLine += lineCount
			newLine += lineCount
		case diffmatchpatch.DiffDelete:
			op = "delete"
			results = append(results, DiffResult{
				Type:      "deletion",
				OldLine:   oldLine,
				Content:   diff.Text,
				Operation: op,
			})
			oldLine += countLines(diff.Text)
		case diffmatchpatch.DiffInsert:
			op = "insert"
			results = append(results, DiffResult{
				Type:      "insertion",
				NewLine:   newLine,
				Content:   diff.Text,
				Operation: op,
			})
			newLine += countLines(diff.Text)
		}
	}

	diffJSON, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal diff results: %w", err)
	}

	return string(diffJSON), nil
}

func countLines(text string) int {
	if text == "" {
		return 0
	}

	count := 0
	for _, char := range text {
		if char == '\n' {
			count++
		}
	}

	if len(text) > 0 && text[len(text)-1] != '\n' {
		count++
	}

	return count
}

func GetUnifiedDiff(oldContent, newContent, fileName string) string {
	dmp := diffmatchpatch.New()

	chars1, chars2, lineArray := dmp.DiffLinesToChars(oldContent, newContent)
	diffs := dmp.DiffMain(chars1, chars2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)
	diffs = dmp.DiffCleanupSemantic(diffs)

	return dmp.DiffPrettyText(diffs)
}
