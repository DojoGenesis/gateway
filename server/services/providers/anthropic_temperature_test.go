package providers

import "testing"

// TestModelSupportsTemperature locks in the temperature-deprecation boundary
// verified 2026-07-16 against live api.anthropic.com. Anthropic deprecated the
// `temperature` parameter for its newest models; sending it returns
// 400 "temperature is deprecated for this model", which makes the model
// unusable through the gateway. This is an allowlist: only models confirmed to
// accept temperature may return true. When Anthropic ships a new model, the
// safe default is false (omit) — add it here only after confirming acceptance.
func TestModelSupportsTemperature(t *testing.T) {
	accepts := []string{
		"claude-opus-4-6",
		"claude-sonnet-4-6",
		"claude-haiku-4-5",
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-haiku-4-20250414",
	}
	rejects := []string{
		"claude-fable-5",
		"claude-sonnet-5",
		"claude-opus-4-7",
		"claude-opus-4-8",
		// Unknown / future models must fail safe (omit temperature), never 400.
		"claude-opus-5",
		"claude-some-future-model",
		"",
	}

	for _, m := range accepts {
		if !modelSupportsTemperature(m) {
			t.Errorf("modelSupportsTemperature(%q) = false, want true (model accepts temperature)", m)
		}
	}
	for _, m := range rejects {
		if modelSupportsTemperature(m) {
			t.Errorf("modelSupportsTemperature(%q) = true, want false (temperature deprecated / unknown → omit)", m)
		}
	}
}
