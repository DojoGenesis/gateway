package agent

import (
	"context"
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockEmbeddingBackend produces deterministic vectors of a fixed dimension.
// Vectors are designed so different semantic "families" land in distinct
// regions of the 16-dimensional space:
//
//	Greetings     → high values in dims [0..3]
//	Technical     → high values in dims [6..9]
//	Multi-step    → high values in dims [12..15]
//
// Any other text produces a uniform mid-range vector. The same input always
// produces the same vector (no randomness).
type mockEmbeddingBackend struct {
	dim int // vector length; defaults to 16
}

func newMockEmbeddingBackend() *mockEmbeddingBackend {
	return &mockEmbeddingBackend{dim: 16}
}

func (m *mockEmbeddingBackend) Embed(_ context.Context, text string) ([]float32, error) {
	d := m.dim
	if d == 0 {
		d = 16
	}
	v := make([]float32, d)

	// Fill with a low baseline so the vector is never zero.
	for i := range v {
		v[i] = 0.1
	}

	lower := text
	if len(text) > 0 {
		b := []byte(text)
		for i, c := range b {
			if c >= 'A' && c <= 'Z' {
				b[i] = c + 32
			}
		}
		lower = string(b)
	}

	// Greeting family: contains common greeting words.
	greetingWords := []string{"hi", "hello", "hey", "thanks", "thank", "bye", "goodbye", "howdy", "greet", "morning", "afternoon", "evening"}
	for _, w := range greetingWords {
		if hasSubstr(lower, w) {
			for i := 0; i < 4 && i < d; i++ {
				v[i] = 0.9
			}
			return normalize(v), nil
		}
	}

	// Multi-step / orchestration family.
	multiWords := []string{"then", "and then", "analyze", "create", "deploy", "orchestrat", "plan", "multi", "step", "pipeline", "refactor", "build", "setup"}
	for _, w := range multiWords {
		if hasSubstr(lower, w) {
			for i := 12; i < d; i++ {
				v[i] = 0.9
			}
			return normalize(v), nil
		}
	}

	// Technical / code family.
	techWords := []string{"code", "write", "debug", "function", "go ", "golang", "python", "sql", "api", "http", "test", "unit", "jwt", "refactor", "compil"}
	for _, w := range techWords {
		if hasSubstr(lower, w) {
			for i := 6; i < 10 && i < d; i++ {
				v[i] = 0.9
			}
			return normalize(v), nil
		}
	}

	// Default: uniform vector (different from all specialized families).
	for i := range v {
		v[i] = 1.0 / float32(d)
	}
	return normalize(v), nil
}

// hasSubstr is a simple substring check used by the mock embedding backend.
func hasSubstr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// normalize returns a unit-length copy of v. If v is the zero vector, it is
// returned unchanged to avoid division by zero.
func normalize(v []float32) []float32 {
	var sumSq float64
	for _, x := range v {
		sumSq += float64(x) * float64(x)
	}
	if sumSq == 0 {
		return v
	}
	norm := math.Sqrt(sumSq)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}

// errEmbeddingBackend always returns an error.
type errEmbeddingBackend struct{}

func (e *errEmbeddingBackend) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("embed: simulated backend failure")
}

// mockLLMRouter returns a pre-configured RoutingDecision or an error.
type mockLLMRouter struct {
	decision RoutingDecision
	err      error
	called   int
}

func (m *mockLLMRouter) ClassifyWithLLM(_ context.Context, _ string) (RoutingDecision, error) {
	m.called++
	if m.err != nil {
		return RoutingDecision{}, m.err
	}
	return m.decision, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// greetingRoute returns a RouteDefinition whose centroid is pre-set to a
// greeting-family vector (high in dims [0..3]). This avoids calling Initialize
// in tests that only need Tier 2 behaviour.
func greetingRouteWithCentroid(embedder *mockEmbeddingBackend) RouteDefinition {
	centroid, _ := embedder.Embed(context.Background(), "hello there")
	return RouteDefinition{
		Name:      "direct_response",
		Handler:   "template",
		Threshold: 0.70,
		Centroid:  centroid,
	}
}

// techRouteWithCentroid returns a RouteDefinition pre-computed for code/tech queries.
func techRouteWithCentroid(embedder *mockEmbeddingBackend) RouteDefinition {
	centroid, _ := embedder.Embed(context.Background(), "write code debug function")
	return RouteDefinition{
		Name:      "deep_inference",
		Handler:   "llm-reasoning",
		Threshold: 0.58,
		Centroid:  centroid,
	}
}

// lowSimilarityRoute returns a route whose centroid is orthogonal to any
// realistic query vector so Tier 2 will always miss.
func lowSimilarityRoute() RouteDefinition {
	// A centroid that is zero in all dims used by the mock embedder.
	centroid := make([]float32, 16)
	centroid[4] = 1.0 // dim 4 is never set high by the mock
	centroid[5] = 1.0
	return RouteDefinition{
		Name:      "unreachable_route",
		Handler:   "unreachable",
		Threshold: 0.99, // very high — will never match
		Centroid:  centroid,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSemanticRouter_Tier1_EmptyQuery(t *testing.T) {
	emb := newMockEmbeddingBackend()
	llm := &mockLLMRouter{}
	sr := NewSemanticRouter(emb, llm)

	for _, q := range []string{"", "   ", "\t", "\n"} {
		t.Run(fmt.Sprintf("query=%q", q), func(t *testing.T) {
			dec, err := sr.Route(context.Background(), q, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec.Handler != "template" {
				t.Errorf("handler: got %q, want %q", dec.Handler, "template")
			}
			if dec.Template != "greeting" {
				t.Errorf("template: got %q, want %q", dec.Template, "greeting")
			}
			if dec.Confidence != 1.0 {
				t.Errorf("confidence: got %v, want 1.0", dec.Confidence)
			}
			if dec.Category != Greeting {
				t.Errorf("category: got %v, want Greeting", dec.Category)
			}
			if llm.called != 0 {
				t.Errorf("LLM router should not be called for empty query")
			}
		})
	}
}

func TestSemanticRouter_Tier1_Greetings(t *testing.T) {
	cases := []struct {
		query   string
		handler string
	}{
		{"hello", "template"},
		{"hi", "template"},
		{"hey", "template"},
		{"thanks", "template"},
		{"thank you", "template"},
		{"bye", "template"},
		{"goodbye", "template"},
		{"good morning", "template"},
		{"good afternoon", "template"},
		{"good evening", "template"},
		{"good night", "template"},
		{"greetings", "template"},
		{"howdy", "template"},
		{"see you", "template"},
	}

	emb := newMockEmbeddingBackend()
	llm := &mockLLMRouter{}
	sr := NewSemanticRouter(emb, llm)

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			dec, err := sr.Route(context.Background(), tc.query, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec.Handler != tc.handler {
				t.Errorf("query %q: handler got %q, want %q", tc.query, dec.Handler, tc.handler)
			}
			if dec.Category != Greeting {
				t.Errorf("query %q: category got %v, want Greeting", tc.query, dec.Category)
			}
		})
	}
	// LLM must never be called — tier1 handles all of these.
	if llm.called != 0 {
		t.Errorf("LLM router called %d times, want 0", llm.called)
	}
}

func TestSemanticRouter_Tier1_ProviderOverride(t *testing.T) {
	cases := []struct {
		query    string
		provider string
	}{
		{"use openai: tell me a joke", "openai"},
		{"use anthropic: summarize this", "anthropic"},
		{"@anthropic what is entropy", "anthropic"},
		{"@openai write a haiku", "openai"},
		{"@gemini translate this", "gemini"},
	}

	emb := newMockEmbeddingBackend()
	llm := &mockLLMRouter{}
	sr := NewSemanticRouter(emb, llm)

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			dec, err := sr.Route(context.Background(), tc.query, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec.Provider != tc.provider {
				t.Errorf("query %q: provider got %q, want %q", tc.query, dec.Provider, tc.provider)
			}
			if dec.Handler != "llm-fast" {
				t.Errorf("query %q: handler got %q, want llm-fast", tc.query, dec.Handler)
			}
			if dec.Confidence != 1.0 {
				t.Errorf("query %q: confidence got %v, want 1.0", tc.query, dec.Confidence)
			}
			if dec.Category != MetaQuery {
				t.Errorf("query %q: category got %v, want MetaQuery", tc.query, dec.Category)
			}
		})
	}
	if llm.called != 0 {
		t.Errorf("LLM router called %d times, want 0", llm.called)
	}
}

func TestSemanticRouter_Tier2_MatchesRoute(t *testing.T) {
	emb := newMockEmbeddingBackend()
	llm := &mockLLMRouter{} // should not be called

	sr := NewSemanticRouter(emb, llm)

	// Pre-compute a centroid for the "code" family and add it as a route.
	techRoute := techRouteWithCentroid(emb)
	sr.AddRoute(techRoute)

	// A query that the mock embedder will map to the same tech-family vector.
	query := "write a Go HTTP handler and debug the function"
	dec, err := sr.Route(context.Background(), query, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dec.Handler != "llm-reasoning" {
		t.Errorf("handler: got %q, want %q", dec.Handler, "llm-reasoning")
	}
	if dec.Confidence < techRoute.Threshold {
		t.Errorf("confidence %v is below threshold %v", dec.Confidence, techRoute.Threshold)
	}
	// Tier 2 matched — LLM must not have been called.
	if llm.called != 0 {
		t.Errorf("LLM router called %d times, want 0", llm.called)
	}
}

func TestSemanticRouter_Tier2_BelowThreshold_FallsToTier3(t *testing.T) {
	emb := newMockEmbeddingBackend()
	wantDecision := RoutingDecision{
		Handler:    "llm-fast",
		Provider:   "anthropic",
		Confidence: 0.7,
		Category:   Factual,
		Reasoning:  []string{"mock llm decision"},
	}
	llm := &mockLLMRouter{decision: wantDecision}

	sr := NewSemanticRouter(emb, llm)
	sr.AddRoute(lowSimilarityRoute()) // threshold=0.99, never matches

	query := "a completely generic question about nothing specific"
	dec, err := sr.Route(context.Background(), query, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llm.called != 1 {
		t.Errorf("LLM router called %d times, want 1", llm.called)
	}
	// The tier3 wrapper prepends "tier3: LLM classification" to Reasoning.
	if dec.Handler != wantDecision.Handler {
		t.Errorf("handler: got %q, want %q", dec.Handler, wantDecision.Handler)
	}
	if dec.Provider != wantDecision.Provider {
		t.Errorf("provider: got %q, want %q", dec.Provider, wantDecision.Provider)
	}
}

func TestSemanticRouter_ModeLLM(t *testing.T) {
	emb := newMockEmbeddingBackend()
	wantDecision := RoutingDecision{
		Handler:    "llm-reasoning",
		Confidence: 0.9,
		Category:   Explanation,
	}
	llm := &mockLLMRouter{decision: wantDecision}

	sr := NewSemanticRouter(emb, llm)
	sr.SetMode(RoutingModeLLM)

	// Even a greeting should bypass tier1 and go straight to LLM.
	queries := []string{"hello", "write some code", "what is 2+2"}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			dec, err := sr.Route(context.Background(), q, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec.Handler != wantDecision.Handler {
				t.Errorf("handler: got %q, want %q", dec.Handler, wantDecision.Handler)
			}
		})
	}

	if llm.called != len(queries) {
		t.Errorf("LLM router called %d times, want %d", llm.called, len(queries))
	}
}

func TestSemanticRouter_ModeEmbedding_NoFallback(t *testing.T) {
	emb := newMockEmbeddingBackend()
	llm := &mockLLMRouter{} // must not be called

	sr := NewSemanticRouter(emb, llm)
	sr.SetMode(RoutingModeEmbedding)
	sr.AddRoute(lowSimilarityRoute()) // threshold=0.99, never matches

	query := "a question that matches no route"
	dec, err := sr.Route(context.Background(), query, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sub-threshold in embedding mode → low-confidence default, no LLM.
	if llm.called != 0 {
		t.Errorf("LLM router called %d times in embedding-only mode, want 0", llm.called)
	}
	if dec.Confidence != 0.0 {
		t.Errorf("confidence: got %v, want 0.0", dec.Confidence)
	}
	if dec.Handler != "llm-fast" {
		t.Errorf("handler: got %q, want llm-fast (default)", dec.Handler)
	}
}

func TestSemanticRouter_SetMode_HotSwitch(t *testing.T) {
	emb := newMockEmbeddingBackend()
	llmDecision := RoutingDecision{Handler: "llm-reasoning", Confidence: 0.8, Category: Explanation}
	llm := &mockLLMRouter{decision: llmDecision}

	sr := NewSemanticRouter(emb, llm)
	// Start in cascade — "hello" should be caught by tier1.
	dec, err := sr.Route(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("cascade mode: unexpected error: %v", err)
	}
	if dec.Handler != "template" {
		t.Errorf("cascade mode: handler got %q, want template", dec.Handler)
	}
	if llm.called != 0 {
		t.Errorf("cascade mode: LLM should not be called for greeting")
	}

	// Hot-switch to LLM mode.
	sr.SetMode(RoutingModeLLM)
	dec, err = sr.Route(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("llm mode: unexpected error: %v", err)
	}
	if dec.Handler != "llm-reasoning" {
		t.Errorf("llm mode after switch: handler got %q, want llm-reasoning", dec.Handler)
	}
	if llm.called != 1 {
		t.Errorf("llm mode after switch: LLM called %d times, want 1", llm.called)
	}

	// Hot-switch back to cascade.
	sr.SetMode(RoutingModeCascade)
	dec, err = sr.Route(context.Background(), "hello", nil)
	if err != nil {
		t.Fatalf("cascade mode (restored): unexpected error: %v", err)
	}
	if dec.Handler != "template" {
		t.Errorf("cascade mode (restored): handler got %q, want template", dec.Handler)
	}
	// LLM count should still be 1 (not incremented again).
	if llm.called != 1 {
		t.Errorf("cascade mode (restored): LLM called %d times, want 1", llm.called)
	}
}

func TestSemanticRouter_Initialize(t *testing.T) {
	emb := newMockEmbeddingBackend()
	sr := NewSemanticRouter(emb, &mockLLMRouter{})

	routes := DefaultRouteDefinitions()
	for _, r := range routes {
		r.Centroid = nil // ensure centroids start empty
		sr.AddRoute(r)
	}

	if err := sr.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: unexpected error: %v", err)
	}

	got := sr.GetRoutes()
	if len(got) != len(routes) {
		t.Fatalf("route count: got %d, want %d", len(got), len(routes))
	}
	for _, r := range got {
		if len(r.Centroid) == 0 {
			t.Errorf("route %q: centroid not populated after Initialize", r.Name)
			continue
		}
		// Centroid must have positive magnitude (non-zero vector).
		// Note: centroids are averages of unit vectors and are NOT
		// themselves unit-normalized, so we only check > 0.
		var sumSq float64
		for _, v := range r.Centroid {
			sumSq += float64(v) * float64(v)
		}
		if sumSq == 0 {
			t.Errorf("route %q: centroid is the zero vector after Initialize", r.Name)
		}
	}
}

func TestSemanticRouter_AddRoute_GetRoutes(t *testing.T) {
	emb := newMockEmbeddingBackend()
	sr := NewSemanticRouter(emb, &mockLLMRouter{})

	if len(sr.GetRoutes()) != 0 {
		t.Fatalf("expected empty routes initially")
	}

	r1 := RouteDefinition{Name: "route-a", Handler: "handler-a"}
	r2 := RouteDefinition{Name: "route-b", Handler: "handler-b"}
	sr.AddRoute(r1)
	sr.AddRoute(r2)

	got := sr.GetRoutes()
	if len(got) != 2 {
		t.Fatalf("route count: got %d, want 2", len(got))
	}
	if got[0].Name != "route-a" {
		t.Errorf("routes[0].Name: got %q, want %q", got[0].Name, "route-a")
	}
	if got[1].Name != "route-b" {
		t.Errorf("routes[1].Name: got %q, want %q", got[1].Name, "route-b")
	}

	// GetRoutes must return a snapshot (mutations must not affect internal state).
	got[0].Name = "mutated"
	snapshot2 := sr.GetRoutes()
	if snapshot2[0].Name == "mutated" {
		t.Error("GetRoutes returned a reference instead of a copy")
	}
}

func TestDefaultLLMRouter_ParseResponse(t *testing.T) {
	router := &DefaultLLMRouter{
		routes: DefaultRouteDescriptions,
		routeBy: func() map[string]RouteDescription {
			m := make(map[string]RouteDescription)
			for _, r := range DefaultRouteDescriptions {
				m[r.Name] = r
			}
			return m
		}(),
	}

	cases := []struct {
		name      string
		raw       string
		wantErr   bool
		wantRoute string
		wantConf  float64
	}{
		{
			name:      "valid_quick",
			raw:       `{"route":"quick","confidence":0.85,"reasoning":"simple factual query"}`,
			wantRoute: "llm-fast",
			wantConf:  0.85,
		},
		{
			name:      "valid_deep",
			raw:       `{"route":"deep","confidence":0.9,"reasoning":"complex code task"}`,
			wantRoute: "llm-reasoning",
			wantConf:  0.9,
		},
		{
			name:      "valid_trivial",
			raw:       `{"route":"trivial","confidence":1.0,"reasoning":"greeting"}`,
			wantRoute: "template",
			wantConf:  1.0,
		},
		{
			name:      "valid_orchestrate",
			raw:       `{"route":"orchestrate","confidence":0.75,"reasoning":"multi-step plan"}`,
			wantRoute: "orchestrate",
			wantConf:  0.75,
		},
		{
			name:      "markdown_fence_stripped",
			raw:       "```json\n{\"route\":\"quick\",\"confidence\":0.6,\"reasoning\":\"factual\"}\n```",
			wantRoute: "llm-fast",
			wantConf:  0.6,
		},
		{
			name:      "confidence_clamped_above_1",
			raw:       `{"route":"quick","confidence":1.5,"reasoning":"over-confident"}`,
			wantRoute: "llm-fast",
			wantConf:  1.0,
		},
		{
			name:      "confidence_clamped_below_0",
			raw:       `{"route":"quick","confidence":-0.5,"reasoning":"negative"}`,
			wantRoute: "llm-fast",
			wantConf:  0.0,
		},
		{
			name:    "invalid_json",
			raw:     `not json at all`,
			wantErr: true,
		},
		{
			name:    "unknown_route",
			raw:     `{"route":"nonexistent","confidence":0.8,"reasoning":"bad route"}`,
			wantErr: true,
		},
		{
			name:    "empty_string",
			raw:     ``,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dec, err := router.parseResponse(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (decision=%+v)", dec)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dec.Handler != tc.wantRoute {
				t.Errorf("handler: got %q, want %q", dec.Handler, tc.wantRoute)
			}
			if dec.Confidence != tc.wantConf {
				t.Errorf("confidence: got %v, want %v", dec.Confidence, tc.wantConf)
			}
		})
	}
}

func TestDefaultLLMRouter_DefaultDecision(t *testing.T) {
	router := &DefaultLLMRouter{}

	reason := "provider not available: connection refused"
	dec := router.defaultDecision(reason)

	if dec.Handler != "llm-fast" {
		t.Errorf("handler: got %q, want llm-fast", dec.Handler)
	}
	if dec.Confidence != 0.3 {
		t.Errorf("confidence: got %v, want 0.3", dec.Confidence)
	}
	if dec.Category != Factual {
		t.Errorf("category: got %v, want Factual", dec.Category)
	}
	if len(dec.Reasoning) < 2 {
		t.Errorf("reasoning: got %d entries, want >= 2", len(dec.Reasoning))
	}
	// The original reason should appear somewhere in Reasoning.
	found := false
	for _, r := range dec.Reasoning {
		if r == reason {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("reason %q not found in Reasoning %v", reason, dec.Reasoning)
	}
}

func TestDefaultRouteDefinitions(t *testing.T) {
	routes := DefaultRouteDefinitions()

	if len(routes) != 5 {
		t.Fatalf("route count: got %d, want 5", len(routes))
	}

	for _, r := range routes {
		if r.Name == "" {
			t.Errorf("route has empty Name")
		}
		if r.Handler == "" {
			t.Errorf("route %q has empty Handler", r.Name)
		}
		if len(r.Utterances) == 0 {
			t.Errorf("route %q has no utterances", r.Name)
		}
		if r.Threshold <= 0 {
			t.Errorf("route %q has non-positive threshold %v", r.Name, r.Threshold)
		}
	}

	// Verify expected route names are present.
	wantNames := map[string]bool{
		"direct_response":    false,
		"fast_inference":     false,
		"deep_inference":     false,
		"specialist_dispatch": false,
		"orchestrated_plan":  false,
	}
	for _, r := range routes {
		wantNames[r.Name] = true
	}
	for name, found := range wantNames {
		if !found {
			t.Errorf("expected route %q not found in DefaultRouteDefinitions", name)
		}
	}
}

func TestCosineSimilarityLocal(t *testing.T) {
	cases := []struct {
		name string
		a, b []float32
		want float64
		tol  float64
	}{
		{
			name: "identical_vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
			tol:  1e-9,
		},
		{
			name: "opposite_vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{-1, 0, 0},
			want: -1.0,
			tol:  1e-9,
		},
		{
			name: "orthogonal_vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 1, 0},
			want: 0.0,
			tol:  1e-9,
		},
		{
			name: "equal_45_degree_angle",
			a:    []float32{1, 1, 0},
			b:    []float32{1, 0, 0},
			want: 1.0 / math.Sqrt2,
			tol:  1e-6,
		},
		{
			name: "scaled_vectors_same_direction",
			a:    []float32{2, 4, 0},
			b:    []float32{1, 2, 0},
			want: 1.0,
			tol:  1e-6,
		},
		{
			name: "empty_a",
			a:    []float32{},
			b:    []float32{1, 0},
			want: 0.0,
			tol:  0,
		},
		{
			name: "mismatched_dims",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0.0,
			tol:  0,
		},
		{
			name: "zero_vector_a",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 0, 0},
			want: 0.0,
			tol:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cosineSimilarityLocal(tc.a, tc.b)
			diff := math.Abs(got - tc.want)
			if diff > tc.tol {
				t.Errorf("cosineSimilarityLocal(%v, %v) = %v, want %v (tol %v)", tc.a, tc.b, got, tc.want, tc.tol)
			}
		})
	}
}

func TestExtractProviderOverride(t *testing.T) {
	cases := []struct {
		name         string
		lower        string
		wantProvider string
		wantOk       bool
	}{
		{
			name:         "use_openai",
			lower:        "use openai: tell me a joke",
			wantProvider: "openai",
			wantOk:       true,
		},
		{
			name:         "use_anthropic",
			lower:        "use anthropic: summarize this document",
			wantProvider: "anthropic",
			wantOk:       true,
		},
		{
			name:         "at_anthropic",
			lower:        "@anthropic what is entropy",
			wantProvider: "anthropic",
			wantOk:       true,
		},
		{
			name:         "at_openai",
			lower:        "@openai write a haiku about clouds",
			wantProvider: "openai",
			wantOk:       true,
		},
		{
			name:         "at_gemini",
			lower:        "@gemini translate this to spanish",
			wantProvider: "gemini",
			wantOk:       true,
		},
		{
			name:   "no_override_plain_query",
			lower:  "what is the capital of france",
			wantOk: false,
		},
		{
			name:   "use_without_colon",
			lower:  "use openai tell me a joke",
			wantOk: false,
		},
		{
			name:   "at_without_space_after",
			lower:  "@openai",
			wantOk: false,
		},
		{
			name:   "use_with_space_in_provider",
			lower:  "use open ai: something",
			wantOk: false,
		},
		{
			name:   "empty_string",
			lower:  "",
			wantOk: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := extractProviderOverride(tc.lower)
			if ok != tc.wantOk {
				t.Errorf("ok: got %v, want %v (provider=%q)", ok, tc.wantOk, got)
			}
			if ok && got != tc.wantProvider {
				t.Errorf("provider: got %q, want %q", got, tc.wantProvider)
			}
		})
	}
}
