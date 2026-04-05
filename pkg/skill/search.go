package skill

import (
	"context"
	"sort"
	"strings"
)

// SearchSkills searches installed skills by name or description substring.
//
// Matching is case-insensitive against both the Name and Description fields.
// If query is empty, all skills are returned.
// Results are sorted alphabetically by name.
func SearchSkills(ctx context.Context, store *SkillStore, query string) ([]SkillManifest, error) {
	all, err := store.List(ctx)
	if err != nil {
		return nil, err
	}

	if query == "" {
		sort.Slice(all, func(i, j int) bool {
			return all[i].Name < all[j].Name
		})
		return all, nil
	}

	q := strings.ToLower(query)
	var matched []SkillManifest
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.Name), q) ||
			strings.Contains(strings.ToLower(m.Description), q) {
			matched = append(matched, m)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Name < matched[j].Name
	})

	return matched, nil
}
