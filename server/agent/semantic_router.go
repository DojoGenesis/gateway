package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/DojoGenesis/gateway/provider"
)

// defaultSimilarityThreshold is used for routes that do not specify their own
// Threshold. A score at or above this value triggers a Tier 2 match.
const defaultSimilarityThreshold = 0.65

// greetings is the allowlist used by Tier 1 deterministic matching.
// Exact lower-cased match only; keep this list small and unambiguous.
var greetings = []string{
	"hi", "hello", "hey", "howdy", "greetings",
	"good morning", "good afternoon", "good evening", "good night",
	"thanks", "thank you", "bye", "goodbye", "see you",
}

// EmbeddingBackend is the interface SemanticRouter uses to obtain vector
// representations of text. Implementations may call a remote embedding API,
// an in-process model, or a test stub.
type EmbeddingBackend interface {
	// Embed returns a dense float32 vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// LLMRouter is implemented by the component responsible for Tier 3 (LLM)
// classification. It is defined here so semantic_router.go depends only on
// this interface; the concrete implementation lives in a separate file.
type LLMRouter interface {
	// ClassifyWithLLM uses an LLM to determine the best RoutingDecision for
	// the given query when deterministic and embedding-based routing have
	// failed or been bypassed.
	ClassifyWithLLM(ctx context.Context, query string) (RoutingDecision, error)
}

// ProviderEmbeddingBackend implements EmbeddingBackend by delegating to a
// named provider loaded in the gateway's PluginManager.
type ProviderEmbeddingBackend struct {
	// Manager is the gateway PluginManager used to resolve the provider.
	Manager *provider.PluginManager

	// EmbeddingProvider is the name of the provider to use (e.g. "openai").
	EmbeddingProvider string
}

// Embed resolves the configured provider from the PluginManager and calls
// GenerateEmbedding with the supplied text.
func (b *ProviderEmbeddingBackend) Embed(ctx context.Context, text string) ([]float32, error) {
	if b.Manager == nil {
		return nil, fmt.Errorf("semantic router: plugin manager is nil")
	}
	p, err := b.Manager.GetProvider(b.EmbeddingProvider)
	if err != nil {
		return nil, fmt.Errorf("semantic router: get embedding provider %q: %w", b.EmbeddingProvider, err)
	}
	vec, err := p.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("semantic router: generate embedding: %w", err)
	}
	return vec, nil
}

// SemanticRouter classifies incoming queries and produces RoutingDecisions
// using a three-tier cascade: deterministic rules → embedding similarity → LLM
// fallback. The active mode can be hot-switched at runtime without restart.
//
// Thread-safety: route definitions are protected by a RWMutex; the routing
// mode is stored as an atomic int32 so reads and writes are lock-free.
// initialized is an atomic bool so IsInitialized and TryInitialize are safe
// to call from any goroutine.
type SemanticRouter struct {
	mu          sync.RWMutex
	routes      []RouteDefinition
	mode        atomic.Int32 // stores RoutingMode
	initialized atomic.Bool
	embedder    EmbeddingBackend
	llmRouter   LLMRouter
}

// NewSemanticRouter creates a SemanticRouter with the given EmbeddingBackend
// and LLMRouter. The router starts in RoutingModeCascade. Call Initialize
// before invoking Route.
func NewSemanticRouter(embedder EmbeddingBackend, llmRouter LLMRouter) *SemanticRouter {
	sr := &SemanticRouter{
		embedder:  embedder,
		llmRouter: llmRouter,
	}
	sr.mode.Store(int32(RoutingModeCascade))
	return sr
}

// AddRoute appends a RouteDefinition to the router. Utterances are not
// embedded until Initialize is called; call Initialize again after adding
// routes if the router is already running.
func (sr *SemanticRouter) AddRoute(route RouteDefinition) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.routes = append(sr.routes, route)
}

// GetRoutes returns a snapshot of the current route definitions.
func (sr *SemanticRouter) GetRoutes() []RouteDefinition {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	out := make([]RouteDefinition, len(sr.routes))
	copy(out, sr.routes)
	return out
}

// GetMode returns the current routing mode.
func (sr *SemanticRouter) GetMode() RoutingMode {
	return RoutingMode(sr.mode.Load())
}

// SetMode hot-switches the routing mode. The change takes effect on the next
// call to Route; in-flight calls continue with the mode they observed.
func (sr *SemanticRouter) SetMode(mode RoutingMode) {
	sr.mode.Store(int32(mode))
	slog.Info("semantic router: mode changed", "mode", mode.String())
}

// SetRouteThreshold updates the similarity threshold for a named route.
// Returns an error if no route with that name exists.
func (sr *SemanticRouter) SetRouteThreshold(name string, threshold float64) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	for i := range sr.routes {
		if sr.routes[i].Name == name {
			sr.routes[i].Threshold = threshold
			slog.Info("semantic router: threshold updated", "route", name, "threshold", threshold)
			return nil
		}
	}
	return fmt.Errorf("semantic router: route %q not found", name)
}

// Initialize computes centroids for all registered routes by embedding each
// utterance and averaging the resulting vectors. This must be called before
// Route; it may also be called again after AddRoute to update centroids.
func (sr *SemanticRouter) Initialize(ctx context.Context) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	slog.Info("semantic router: initializing", "routes", len(sr.routes))

	for i := range sr.routes {
		route := &sr.routes[i]
		if len(route.Utterances) == 0 {
			slog.Warn("semantic router: route has no utterances, skipping centroid", "route", route.Name)
			continue
		}

		centroid, err := sr.computeCentroid(ctx, route.Utterances)
		if err != nil {
			return fmt.Errorf("semantic router: compute centroid for route %q: %w", route.Name, err)
		}
		route.Centroid = centroid
		slog.Debug("semantic router: computed centroid", "route", route.Name, "dim", len(centroid))
	}

	slog.Info("semantic router: initialization complete", "routes", len(sr.routes))
	sr.initialized.Store(true)
	return nil
}

// IsInitialized reports whether centroids have been successfully computed.
// Safe to call from any goroutine.
func (sr *SemanticRouter) IsInitialized() bool {
	return sr.initialized.Load()
}

// TryInitialize attempts to initialize the router if it has not been
// initialized yet. Returns (true, nil) when initialization succeeds,
// (false, nil) when the router was already initialized, and (false, err)
// when initialization fails. Thread-safe: concurrent calls are safe because
// Initialize holds the write lock, but only one caller will observe the
// centroids being computed — subsequent callers after the first success
// will return (false, nil).
func (sr *SemanticRouter) TryInitialize(ctx context.Context) (bool, error) {
	if sr.initialized.Load() {
		return false, nil
	}
	if err := sr.Initialize(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// Route classifies query and returns a RoutingDecision. The pipeline executed
// depends on the active RoutingMode:
//   - RoutingModeCascade: Tier1 → Tier2 → Tier3
//   - RoutingModeLLM: Tier3 only
//   - RoutingModeEmbedding: Tier2 only (no LLM fallback)
func (sr *SemanticRouter) Route(ctx context.Context, query string) (RoutingDecision, error) {
	mode := RoutingMode(sr.mode.Load())

	switch mode {
	case RoutingModeLLM:
		slog.Debug("semantic router: LLM-only mode", "query_len", len(query))
		return sr.tier3LLM(ctx, query)

	case RoutingModeEmbedding:
		slog.Debug("semantic router: embedding-only mode", "query_len", len(query))
		decision, matched, err := sr.tier2Embedding(ctx, query)
		if err != nil {
			return RoutingDecision{}, err
		}
		if matched {
			return decision, nil
		}
		// No match and no LLM fallback in this mode — return low-confidence default.
		return RoutingDecision{
			Handler:    "llm-fast",
			Provider:   "llm-fast",
			Confidence: 0.0,
			Category:   Factual,
			Reasoning:  []string{"embedding-only mode: no route matched similarity threshold"},
		}, nil

	default: // RoutingModeCascade
		// Tier 1: deterministic fast-path.
		if decision, matched := sr.tier1Deterministic(query); matched {
			slog.Debug("semantic router: tier1 match", "handler", decision.Handler, "confidence", decision.Confidence)
			return decision, nil
		}

		// Tier 2: embedding similarity.
		decision, matched, err := sr.tier2Embedding(ctx, query)
		if err != nil {
			slog.Warn("semantic router: tier2 error, falling through to tier3", "error", err)
		} else if matched {
			slog.Debug("semantic router: tier2 match", "handler", decision.Handler, "confidence", decision.Confidence)
			return decision, nil
		}

		// Tier 3: LLM fallback.
		slog.Debug("semantic router: falling through to tier3", "query_len", len(query))
		return sr.tier3LLM(ctx, query)
	}
}

// tier1Deterministic handles trivially classifiable queries without any I/O.
// Returns (decision, true) when a definitive match is found.
func (sr *SemanticRouter) tier1Deterministic(query string) (RoutingDecision, bool) {
	trimmed := strings.TrimSpace(query)

	// Empty query → greeting template.
	if trimmed == "" {
		return RoutingDecision{
			Handler:    "template",
			Template:   "greeting",
			Confidence: 1.0,
			Category:   Greeting,
			Reasoning:  []string{"tier1: empty query"},
		}, true
	}

	lower := strings.ToLower(trimmed)

	// Exact greeting match.
	for _, g := range greetings {
		if lower == g {
			return RoutingDecision{
				Handler:    "template",
				Template:   "greeting",
				Confidence: 0.98,
				Category:   Greeting,
				Reasoning:  []string{"tier1: exact greeting match"},
			}, true
		}
	}

	// Explicit provider override: "use <provider>: <rest>" or "@<provider> <rest>".
	if p, ok := extractProviderOverride(lower); ok {
		return RoutingDecision{
			Handler:    "llm-fast",
			Provider:   p,
			Confidence: 1.0,
			Category:   MetaQuery,
			Reasoning:  []string{"tier1: explicit provider override", "provider=" + p},
		}, true
	}

	return RoutingDecision{}, false
}

// tier2Embedding embeds the query and compares it against route centroids.
// Returns (decision, true, nil) on a match; (zero, false, nil) when no route
// exceeds its threshold; (zero, false, err) on embedding failure.
func (sr *SemanticRouter) tier2Embedding(ctx context.Context, query string) (RoutingDecision, bool, error) {
	sr.mu.RLock()
	routes := make([]RouteDefinition, len(sr.routes))
	copy(routes, sr.routes)
	sr.mu.RUnlock()

	if len(routes) == 0 {
		return RoutingDecision{}, false, nil
	}

	vec, err := sr.embedder.Embed(ctx, query)
	if err != nil {
		return RoutingDecision{}, false, fmt.Errorf("tier2: embed query: %w", err)
	}

	allScores := make(map[string]float64, len(routes))
	var best RouteMatch
	found := false
	for _, route := range routes {
		if len(route.Centroid) == 0 {
			continue
		}
		sim := cosineSimilarityLocal(vec, route.Centroid)
		allScores[route.Name] = sim
		if !found || sim > best.Similarity {
			best = RouteMatch{Route: route, Similarity: sim}
			found = true
		}
	}

	if !found {
		return RoutingDecision{}, false, nil
	}

	threshold := defaultSimilarityThreshold
	if best.Route.Threshold > 0 {
		threshold = best.Route.Threshold
	}

	slog.Debug("semantic router: tier2 scores",
		"query_len", len(query),
		"best_route", best.Route.Name,
		"best_similarity", best.Similarity,
		"threshold", threshold,
		"matched", best.Similarity >= threshold,
	)

	if best.Similarity < threshold {
		return RoutingDecision{}, false, nil
	}

	category := routeNameToCategory(best.Route.Name)
	return RoutingDecision{
		Handler:    best.Route.Handler,
		Provider:   best.Route.ProviderAlias,
		Fallback:   best.Route.Fallback,
		Confidence: best.Similarity,
		Category:   category,
		Scores:     allScores,
		Reasoning: []string{
			fmt.Sprintf("tier2: embedding match route=%s similarity=%.4f threshold=%.4f",
				best.Route.Name, best.Similarity, threshold),
		},
	}, true, nil
}

// tier3LLM delegates classification to the injected LLMRouter.
func (sr *SemanticRouter) tier3LLM(ctx context.Context, query string) (RoutingDecision, error) {
	if sr.llmRouter == nil {
		return RoutingDecision{
			Handler:    "llm-fast",
			Provider:   "llm-fast",
			Confidence: 0.5,
			Category:   Explanation,
			Reasoning:  []string{"tier3: LLMRouter not configured, using llm-fast default"},
		}, nil
	}
	decision, err := sr.llmRouter.ClassifyWithLLM(ctx, query)
	if err != nil {
		return RoutingDecision{}, fmt.Errorf("tier3: LLM classification: %w", err)
	}
	decision.Reasoning = append([]string{"tier3: LLM classification"}, decision.Reasoning...)
	return decision, nil
}

// computeCentroid embeds each utterance and returns their element-wise average.
// Requires at least one utterance; returns an error if all embeddings fail.
func (sr *SemanticRouter) computeCentroid(ctx context.Context, utterances []string) ([]float32, error) {
	var (
		sum   []float32
		count int
	)

	for _, u := range utterances {
		vec, err := sr.embedder.Embed(ctx, u)
		if err != nil {
			slog.Warn("semantic router: failed to embed utterance, skipping", "utterance", u, "error", err)
			continue
		}
		if len(vec) == 0 {
			continue
		}
		if sum == nil {
			sum = make([]float32, len(vec))
		}
		if len(vec) != len(sum) {
			slog.Warn("semantic router: embedding dimension mismatch, skipping", "expected", len(sum), "got", len(vec))
			continue
		}
		for i, v := range vec {
			sum[i] += v
		}
		count++
	}

	if count == 0 {
		return nil, fmt.Errorf("no valid embeddings produced for %d utterances", len(utterances))
	}

	centroid := make([]float32, len(sum))
	fCount := float32(count)
	for i, v := range sum {
		centroid[i] = v / fCount
	}
	return centroid, nil
}

// cosineSimilarityLocal computes the cosine similarity between two float32
// vectors. Returns 0 for empty or mismatched-dimension inputs.
// This is a local reimplementation to avoid importing the memory package
// (which would create a circular dependency).
func cosineSimilarityLocal(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0.0
	}

	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// routeNameToCategory maps a route name to the closest IntentCategory.
// Exact matches for the 5 canonical routes are checked first; substring
// fallback handles custom routes. Unknown names default to Explanation.
func routeNameToCategory(name string) IntentCategory {
	// Exact matches for canonical route names from DefaultRouteDefinitions.
	switch name {
	case "direct_response":
		return Greeting
	case "fast_inference":
		return Factual
	case "deep_inference":
		return Explanation
	case "specialist_dispatch":
		return CodeGeneration
	case "orchestrated_plan":
		return Planning
	}

	// Substring fallback for custom routes.
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "greeting") || strings.Contains(lower, "hello") || strings.Contains(lower, "direct"):
		return Greeting
	case strings.Contains(lower, "factual") || strings.Contains(lower, "fact") || strings.Contains(lower, "fast"):
		return Factual
	case strings.Contains(lower, "calc") || strings.Contains(lower, "math"):
		return Calculation
	case strings.Contains(lower, "code") || strings.Contains(lower, "generat"):
		return CodeGeneration
	case strings.Contains(lower, "debug") || strings.Contains(lower, "fix") || strings.Contains(lower, "error"):
		return Debugging
	case strings.Contains(lower, "plan") || strings.Contains(lower, "architect") || strings.Contains(lower, "orchestrat"):
		return Planning
	case strings.Contains(lower, "meta") || strings.Contains(lower, "system") || strings.Contains(lower, "specialist"):
		return MetaQuery
	default:
		return Explanation
	}
}

// extractProviderOverride detects explicit provider-override syntax in a
// lower-cased query. Supported forms:
//   - "use <provider>: ..."   (e.g. "use openai: tell me a joke")
//   - "@<provider> ..."       (e.g. "@anthropic what is entropy")
//
// Returns (providerName, true) when an override is detected.
func extractProviderOverride(lower string) (string, bool) {
	// "use <word>:" prefix
	if strings.HasPrefix(lower, "use ") {
		rest := lower[4:]
		if idx := strings.Index(rest, ":"); idx > 0 {
			candidate := strings.TrimSpace(rest[:idx])
			if !strings.Contains(candidate, " ") && candidate != "" {
				return candidate, true
			}
		}
	}

	// "@<word> " prefix
	if strings.HasPrefix(lower, "@") {
		rest := lower[1:]
		if idx := strings.Index(rest, " "); idx > 0 {
			candidate := rest[:idx]
			if candidate != "" {
				return candidate, true
			}
		}
	}

	return "", false
}
