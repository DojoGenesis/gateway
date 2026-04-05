// Package main provides the DIP scoring logic as a standalone WASM module.
//
// This file contains the pure Go scoring logic with no WASM-specific code,
// making it testable with standard Go tooling. The WASM glue lives in main.go.
package main

// ScoringRequest is the input payload for the WASM scorer.
type ScoringRequest struct {
	DimensionScores []DimensionScore `json:"dimension_scores"`
	Weights         []float64        `json:"weights"`
	LicenseType     string           `json:"license_type"`
}

// DimensionScore represents a single dimension's score.
type DimensionScore struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// ScoringResult is the output payload from the WASM scorer.
type ScoringResult struct {
	AggregateScore float64          `json:"aggregate_score"`
	DeviationState string           `json:"deviation_state"`
	PerDimension   []DimensionScore `json:"per_dimension"`
}

// ComputeScore computes a weighted aggregate score and classifies deviation.
// This matches DIP's ComputeAggregateDimensionScore logic:
//   - If no dimensions are provided, aggregate is 0.
//   - If weights are nil/empty or length mismatches, equal weighting is used.
//   - Each dimension's weight defaults to 1.0 when not explicitly set.
func ComputeScore(req ScoringRequest) ScoringResult {
	result := ScoringResult{
		PerDimension: make([]DimensionScore, len(req.DimensionScores)),
	}

	// Copy per-dimension scores to output.
	copy(result.PerDimension, req.DimensionScores)

	if len(req.DimensionScores) == 0 {
		result.AggregateScore = 0
		result.DeviationState = ClassifyDeviation(0, req.LicenseType)
		return result
	}

	// Determine whether to use equal weighting.
	// Use equal weighting when: no weights provided, or length mismatch.
	useEqual := len(req.Weights) == 0 || len(req.Weights) != len(req.DimensionScores)

	var weightedSum, totalWeight float64
	for i, d := range req.DimensionScores {
		w := 1.0
		if !useEqual {
			w = req.Weights[i]
			// If an explicit weight is zero or negative, fall back to 1.0
			// to match DIP's behavior where missing weights default to 1.0.
			if w <= 0 {
				w = 1.0
			}
		}
		weightedSum += d.Score * w
		totalWeight += w
	}

	if totalWeight == 0 {
		result.AggregateScore = 0
	} else {
		result.AggregateScore = weightedSum / totalWeight
	}

	result.DeviationState = ClassifyDeviation(result.AggregateScore, req.LicenseType)
	return result
}

// ClassifyDeviation determines the deviation state for a score given the license type.
// This matches DIP's ClassifyDeviation logic exactly:
//
//   - "strict":     score >= 0   -> "conforming", else "intentional"
//   - "guided":     score >= -0.3 -> "conforming", else "intentional"
//   - "expressive": always "intentional" (deviation is the norm)
//   - default:      always "intentional" (unknown license, treat conservatively)
func ClassifyDeviation(score float64, licenseType string) string {
	switch licenseType {
	case "strict":
		if score >= 0 {
			return "conforming"
		}
		return "intentional"

	case "guided":
		if score >= -0.3 {
			return "conforming"
		}
		return "intentional"

	case "expressive":
		return "intentional"

	default:
		return "intentional"
	}
}
