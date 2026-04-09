package skill

import (
	"strings"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"abc", "abcd", 1},
		{"a", "b", 1},

		// Case insensitive.
		{"ABC", "abc", 0},
		{"Strategic-Scout", "strategic-scout", 0},

		// Slopsquatting-relevant examples.
		{"strategic-scout", "strategic-scoot", 1},
		{"data-analyzer", "data-analyser", 1},
		{"data-analyzer", "data-analyzr", 1},
		{"code-review", "code-revew", 1},
		{"code-review", "cod-review", 1},

		// Longer distance — should NOT trigger at MaxNameDistance=2.
		{"hello", "world", 4},
		{"abcdef", "xyz", 6},
	}

	for _, tc := range tests {
		got := LevenshteinDistance(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckNameSafety_ExactReserved(t *testing.T) {
	reserved := map[string]bool{
		"strategic-scout": true,
		"dojo-platform":   true,
	}

	err := CheckNameSafety("strategic-scout", nil, reserved)
	if err == nil {
		t.Fatal("expected error for exact reserved name, got nil")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("expected 'reserved' in error, got: %q", err.Error())
	}
}

func TestCheckNameSafety_ExactReserved_CaseInsensitive(t *testing.T) {
	reserved := map[string]bool{"dojo-platform": true}

	err := CheckNameSafety("Dojo-Platform", nil, reserved)
	if err == nil {
		t.Fatal("expected error for case-insensitive reserved match, got nil")
	}
}

func TestCheckNameSafety_NearReserved(t *testing.T) {
	reserved := map[string]bool{"strategic-scout": true}

	// Distance 1 — should be rejected.
	err := CheckNameSafety("strategic-scoot", nil, reserved)
	if err == nil {
		t.Fatal("expected error for near-reserved name, got nil")
	}
	if !strings.Contains(err.Error(), "too similar") {
		t.Errorf("expected 'too similar' in error, got: %q", err.Error())
	}
}

func TestCheckNameSafety_NearExisting(t *testing.T) {
	existing := []string{"data-analyzer"}

	// Distance 1 (s→z) — should be rejected.
	err := CheckNameSafety("data-analyser", existing, map[string]bool{})
	if err == nil {
		t.Fatal("expected error for near-existing name, got nil")
	}
	if !strings.Contains(err.Error(), "too similar") {
		t.Errorf("expected 'too similar' in error, got: %q", err.Error())
	}
}

func TestCheckNameSafety_ExactExisting(t *testing.T) {
	existing := []string{"my-cool-skill"}

	err := CheckNameSafety("my-cool-skill", existing, map[string]bool{})
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %q", err.Error())
	}
}

func TestCheckNameSafety_SafeName(t *testing.T) {
	reserved := map[string]bool{"strategic-scout": true, "dojo-platform": true}
	existing := []string{"data-analyzer", "code-review"}

	err := CheckNameSafety("my-totally-unique-skill", existing, reserved)
	if err != nil {
		t.Errorf("expected safe name to pass, got: %v", err)
	}
}

func TestCheckNameSafety_DistanceBeyondThreshold(t *testing.T) {
	reserved := map[string]bool{"strategic-scout": true}

	// "hello-world" is far from "strategic-scout" — should pass.
	err := CheckNameSafety("hello-world", nil, reserved)
	if err != nil {
		t.Errorf("expected distant name to pass, got: %v", err)
	}
}
