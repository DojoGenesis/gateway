package memory

import (
	"strings"
)

// FindRelevantSeeds returns seeds whose triggers or names match the query.
func FindRelevantSeeds(query string, seeds []Seed) []Seed {
	queryLower := strings.ToLower(query)
	relevant := []Seed{}

	for _, seed := range seeds {
		if isRelevant(queryLower, seed) {
			relevant = append(relevant, seed)
		}
	}

	return relevant
}

func isRelevant(queryLower string, seed Seed) bool {
	if seed.Trigger != "" {
		triggers := strings.Split(seed.Trigger, ",")
		for _, trigger := range triggers {
			trigger = strings.TrimSpace(strings.ToLower(trigger))
			if trigger != "" && strings.Contains(queryLower, trigger) {
				return true
			}
		}
	}

	nameLower := strings.ToLower(seed.Name)
	if nameLower != "" && strings.Contains(queryLower, nameLower) {
		return true
	}

	return false
}
