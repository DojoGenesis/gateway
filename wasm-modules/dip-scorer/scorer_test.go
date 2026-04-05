package main

import (
	"math"
	"testing"
)

// floatClose returns true if a and b differ by less than 1e-9.
func floatClose(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// ---------------------------------------------------------------------------
// ComputeScore: weighted average computation
// ---------------------------------------------------------------------------

func TestComputeScore_EqualWeighting(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "spacing", Score: 0.8},
			{Name: "typography", Score: 0.4},
			{Name: "color", Score: 0.6},
		},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	want := (0.8 + 0.4 + 0.6) / 3.0
	if !floatClose(result.AggregateScore, want) {
		t.Errorf("equal weighting: got %g, want %g", result.AggregateScore, want)
	}
}

func TestComputeScore_ExplicitWeights(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "spacing", Score: 1.0},
			{Name: "typography", Score: 0.0},
		},
		Weights:     []float64{3.0, 1.0},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	// (1.0*3 + 0.0*1) / (3+1) = 0.75
	want := 0.75
	if !floatClose(result.AggregateScore, want) {
		t.Errorf("explicit weights: got %g, want %g", result.AggregateScore, want)
	}
}

func TestComputeScore_WeightLengthMismatch_FallsBackToEqual(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "spacing", Score: 0.6},
			{Name: "typography", Score: 0.4},
		},
		Weights:     []float64{2.0}, // length mismatch: 1 weight for 2 dimensions
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	// Falls back to equal weighting: (0.6+0.4)/2 = 0.5
	want := 0.5
	if !floatClose(result.AggregateScore, want) {
		t.Errorf("weight mismatch fallback: got %g, want %g", result.AggregateScore, want)
	}
}

func TestComputeScore_EmptyDimensions_ReturnsZero(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: nil,
		LicenseType:     "strict",
	}

	result := ComputeScore(req)
	if result.AggregateScore != 0 {
		t.Errorf("empty dims: got %g, want 0", result.AggregateScore)
	}
}

func TestComputeScore_EmptySliceDimensions_ReturnsZero(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{},
		LicenseType:     "guided",
	}

	result := ComputeScore(req)
	if result.AggregateScore != 0 {
		t.Errorf("empty slice dims: got %g, want 0", result.AggregateScore)
	}
}

func TestComputeScore_NegativeScores(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "spacing", Score: -0.5},
			{Name: "typography", Score: -0.3},
		},
		LicenseType: "guided",
	}

	result := ComputeScore(req)
	want := (-0.5 + -0.3) / 2.0 // = -0.4
	if !floatClose(result.AggregateScore, want) {
		t.Errorf("negative scores: got %g, want %g", result.AggregateScore, want)
	}
}

func TestComputeScore_PerDimensionPreserved(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "spacing", Score: 0.8},
			{Name: "color", Score: 0.2},
		},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	if len(result.PerDimension) != 2 {
		t.Fatalf("expected 2 per-dimension scores, got %d", len(result.PerDimension))
	}
	if result.PerDimension[0].Name != "spacing" || result.PerDimension[0].Score != 0.8 {
		t.Errorf("per-dimension[0]: got %+v, want spacing/0.8", result.PerDimension[0])
	}
	if result.PerDimension[1].Name != "color" || result.PerDimension[1].Score != 0.2 {
		t.Errorf("per-dimension[1]: got %+v, want color/0.2", result.PerDimension[1])
	}
}

func TestComputeScore_SingleDimension(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "only", Score: 0.42},
		},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	if !floatClose(result.AggregateScore, 0.42) {
		t.Errorf("single dim: got %g, want 0.42", result.AggregateScore)
	}
}

// ---------------------------------------------------------------------------
// ClassifyDeviation: license type classification
// ---------------------------------------------------------------------------

func TestClassifyDeviation_Strict(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{1.0, "conforming"},
		{0.5, "conforming"},
		{0.0, "conforming"},
		{-0.01, "intentional"},
		{-0.5, "intentional"},
		{-1.0, "intentional"},
	}
	for _, tc := range tests {
		got := ClassifyDeviation(tc.score, "strict")
		if got != tc.want {
			t.Errorf("strict score=%g: got %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestClassifyDeviation_Guided(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{1.0, "conforming"},
		{0.0, "conforming"},
		{-0.3, "conforming"},  // boundary: -0.3 is conforming
		{-0.31, "intentional"},
		{-0.5, "intentional"},
		{-1.0, "intentional"},
	}
	for _, tc := range tests {
		got := ClassifyDeviation(tc.score, "guided")
		if got != tc.want {
			t.Errorf("guided score=%g: got %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestClassifyDeviation_Expressive(t *testing.T) {
	scores := []float64{1.0, 0.5, 0.0, -0.5, -1.0}
	for _, score := range scores {
		got := ClassifyDeviation(score, "expressive")
		if got != "intentional" {
			t.Errorf("expressive score=%g: got %q, want intentional", score, got)
		}
	}
}

func TestClassifyDeviation_Unknown(t *testing.T) {
	scores := []float64{1.0, 0.0, -1.0}
	for _, score := range scores {
		got := ClassifyDeviation(score, "unknown")
		if got != "intentional" {
			t.Errorf("unknown license score=%g: got %q, want intentional", score, got)
		}
	}
}

func TestClassifyDeviation_EmptyLicense(t *testing.T) {
	got := ClassifyDeviation(0.5, "")
	if got != "intentional" {
		t.Errorf("empty license: got %q, want intentional", got)
	}
}

// ---------------------------------------------------------------------------
// Integration: ComputeScore sets correct deviation state
// ---------------------------------------------------------------------------

func TestComputeScore_SetsDeviationState_Strict_Conforming(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: 0.5},
			{Name: "b", Score: 0.5},
		},
		LicenseType: "strict",
	}
	result := ComputeScore(req)
	if result.DeviationState != "conforming" {
		t.Errorf("strict positive: got %q, want conforming", result.DeviationState)
	}
}

func TestComputeScore_SetsDeviationState_Strict_Intentional(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: -0.5},
			{Name: "b", Score: -0.5},
		},
		LicenseType: "strict",
	}
	result := ComputeScore(req)
	if result.DeviationState != "intentional" {
		t.Errorf("strict negative: got %q, want intentional", result.DeviationState)
	}
}

func TestComputeScore_SetsDeviationState_Guided_Boundary(t *testing.T) {
	// Score of exactly -0.3 should be conforming under guided.
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: -0.3},
		},
		LicenseType: "guided",
	}
	result := ComputeScore(req)
	if result.DeviationState != "conforming" {
		t.Errorf("guided boundary -0.3: got %q, want conforming", result.DeviationState)
	}
}

func TestComputeScore_SetsDeviationState_Expressive(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: 1.0},
		},
		LicenseType: "expressive",
	}
	result := ComputeScore(req)
	if result.DeviationState != "intentional" {
		t.Errorf("expressive: got %q, want intentional", result.DeviationState)
	}
}

// ---------------------------------------------------------------------------
// Matches DIP's existing scoring logic
// ---------------------------------------------------------------------------

func TestComputeScore_MatchesDIPWeightedAverage(t *testing.T) {
	// This test mirrors DIP's TestComputeAggregateDimensionScore_WeightedAverage.
	// DIP uses map[uuid.UUID]float64 weights; here we use positional []float64.
	// The mathematical result must be identical.
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "dim-1", Score: 1.0},
			{Name: "dim-2", Score: 0.0},
		},
		Weights:     []float64{3.0, 1.0},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	// DIP: (1.0*3 + 0.0*1) / (3+1) = 0.75
	if !floatClose(result.AggregateScore, 0.75) {
		t.Errorf("DIP parity: got %g, want 0.75", result.AggregateScore)
	}
}

func TestComputeScore_ZeroWeights_TreatedAsOne(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: 0.8},
			{Name: "b", Score: 0.4},
		},
		Weights:     []float64{0, 0},
		LicenseType: "guided",
	}
	result := ComputeScore(req)
	// Zero weights should be treated as 1.0, so equal weighting: (0.8+0.4)/2 = 0.6
	if math.Abs(result.AggregateScore-0.6) > 0.001 {
		t.Errorf("got %g, want 0.6", result.AggregateScore)
	}
}

func TestComputeScore_NegativeWeights_TreatedAsOne(t *testing.T) {
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "a", Score: 0.8},
			{Name: "b", Score: 0.4},
		},
		Weights:     []float64{-1.0, -2.0},
		LicenseType: "guided",
	}
	result := ComputeScore(req)
	// Negative weights should be treated as 1.0
	if math.Abs(result.AggregateScore-0.6) > 0.001 {
		t.Errorf("got %g, want 0.6", result.AggregateScore)
	}
}

func TestComputeScore_MatchesDIPMissingWeightFallback(t *testing.T) {
	// DIP: when a dimension has no entry in the weights map, weight defaults to 1.0.
	// WASM module: when weights slice has length mismatch, falls back to equal weighting.
	// When weights are provided and correct length, all values are used directly.
	req := ScoringRequest{
		DimensionScores: []DimensionScore{
			{Name: "dim-1", Score: 0.5},
			{Name: "dim-2", Score: 1.0},
		},
		Weights:     []float64{2.0, 1.0},
		LicenseType: "strict",
	}

	result := ComputeScore(req)
	// (0.5*2 + 1.0*1) / (2+1) = 2/3
	want := 2.0 / 3.0
	if !floatClose(result.AggregateScore, want) {
		t.Errorf("DIP missing weight parity: got %g, want %g", result.AggregateScore, want)
	}
}
