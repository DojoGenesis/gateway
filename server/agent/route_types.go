package agent

// RoutingMode controls which classification pipeline the SemanticRouter uses
// when evaluating an incoming query. The active mode can be hot-switched via
// SetMode without restarting the server.
type RoutingMode int

const (
	// RoutingModeCascade runs the full three-tier pipeline:
	// Tier 1 (deterministic) → Tier 2 (embedding similarity) → Tier 3 (LLM fallback).
	// This is the default mode and provides the best balance of latency and accuracy.
	RoutingModeCascade RoutingMode = iota

	// RoutingModeLLM routes every query through LLM classification (Tier 3 only).
	// Use this when embedding quality is uncertain or for evaluation purposes.
	RoutingModeLLM

	// RoutingModeEmbedding routes every query through embedding similarity only (Tier 2).
	// Queries that fall below the similarity threshold receive a zero-confidence decision
	// instead of escalating to LLM fallback.
	RoutingModeEmbedding
)

// String returns the human-readable name of the RoutingMode.
func (rm RoutingMode) String() string {
	switch rm {
	case RoutingModeCascade:
		return "cascade"
	case RoutingModeLLM:
		return "llm"
	case RoutingModeEmbedding:
		return "embedding"
	default:
		return "unknown"
	}
}

// RouteDefinition describes a single named route that the SemanticRouter can
// match queries against. Utterances are embedded at initialization time and
// their centroid is used for similarity comparisons at runtime.
type RouteDefinition struct {
	// Name is the unique identifier for this route (e.g. "code-generation").
	Name string

	// Handler is the downstream handler key that should process matched queries
	// (e.g. "llm-reasoning", "llm-fast", "template").
	Handler string

	// ProviderAlias is the provider name to use when forwarding to an LLM handler
	// (e.g. "openai", "anthropic"). Empty string means use the gateway default.
	ProviderAlias string

	// Fallback is the handler to use if the primary handler is unavailable.
	Fallback string

	// Utterances are representative example queries for this route.
	// They are embedded during Initialize() and averaged into Centroid.
	Utterances []string

	// Centroid is the average embedding of all Utterances.
	// Populated by SemanticRouter.Initialize(); callers should not set this directly.
	Centroid []float32

	// Threshold is the minimum cosine similarity score required to match this
	// route during Tier 2 evaluation. Overrides the router-level default when > 0.
	Threshold float64
}

// RouteMatch pairs a matched RouteDefinition with the similarity score that
// triggered the match. Returned by internal Tier 2 scoring helpers.
type RouteMatch struct {
	// Route is the winning RouteDefinition.
	Route RouteDefinition

	// Similarity is the cosine similarity between the query embedding and the
	// route centroid, in the range [−1, 1].
	Similarity float64
}
