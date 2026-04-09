package skill

import (
	"fmt"
	"strings"
)

// MaxNameDistance is the maximum Levenshtein edit distance between a proposed
// skill name and any existing or reserved name before the publish is rejected.
// ADR-020 specifies distance <= 2 as the slopsquatting threshold.
const MaxNameDistance = 2

// LevenshteinDistance computes the minimum number of single-character edits
// (insertions, deletions, substitutions) to transform a into b.
//
// Uses the single-row DP optimization: O(min(m,n)) space, O(m*n) time.
// Both strings are lowercased before comparison for case-insensitive matching.
func LevenshteinDistance(a, b string) int {
	ar := []rune(strings.ToLower(a))
	br := []rune(strings.ToLower(b))

	// Ensure ar is the shorter string for O(min(m,n)) space.
	if len(ar) > len(br) {
		ar, br = br, ar
	}

	m := len(ar)
	n := len(br)

	if m == 0 {
		return n
	}

	// prev holds the previous row of the DP matrix.
	prev := make([]int, m+1)
	for i := 0; i <= m; i++ {
		prev[i] = i
	}

	for j := 1; j <= n; j++ {
		curr := make([]int, m+1)
		curr[0] = j
		for i := 1; i <= m; i++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			ins := curr[i-1] + 1
			del := prev[i] + 1
			sub := prev[i-1] + cost
			curr[i] = min3(ins, del, sub)
		}
		prev = curr
	}

	return prev[m]
}

// min3 returns the smallest of three ints.
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// CheckNameSafety validates a proposed skill name against reserved names and
// existing published names using both exact match and edit-distance checks.
//
// It subsumes IsReservedName — callers should use CheckNameSafety instead of
// calling both separately.
//
// Returns nil if the name is safe; returns an error describing the conflict
// if the name is reserved or too similar to an existing name.
func CheckNameSafety(name string, existingNames []string, reserved map[string]bool) error {
	lower := strings.ToLower(name)

	// Fast path: exact reserved match.
	if reserved[lower] {
		return fmt.Errorf("skill name %q is reserved (see MARKETPLACE_POLICY.md)", name)
	}

	// Fast path: exact match against existing names.
	for _, existing := range existingNames {
		if strings.ToLower(existing) == lower {
			return fmt.Errorf("skill name %q already exists", name)
		}
	}

	// Edit-distance check against reserved names.
	for reserved := range reserved {
		if d := LevenshteinDistance(lower, reserved); d > 0 && d <= MaxNameDistance {
			return fmt.Errorf("skill name %q is too similar to reserved name %q (edit distance %d)", name, reserved, d)
		}
	}

	// Edit-distance check against existing names.
	for _, existing := range existingNames {
		if d := LevenshteinDistance(lower, strings.ToLower(existing)); d > 0 && d <= MaxNameDistance {
			return fmt.Errorf("skill name %q is too similar to existing skill %q (edit distance %d)", name, existing, d)
		}
	}

	return nil
}
