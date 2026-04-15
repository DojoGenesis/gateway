package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

// RouteDescription describes a single named route available for classification.
// The LLM prompt is built from a slice of these so route definitions are
// injectable without changing the router implementation.
type RouteDescription struct {
	// Name is the short identifier the LLM should emit in the "route" JSON field.
	Name string
	// Handler is the internal handler key (e.g. "template", "llm-fast", "llm-reasoning").
	Handler string
	// Description is the human-readable explanation shown to the LLM.
	Description string
	// ProviderAlias is the preferred provider for this route (empty = use router's default).
	ProviderAlias string
	// Fallback is the handler key to use when this route fails.
	Fallback string
}

// LLMRouterOptions configures a DefaultLLMRouter.
type LLMRouterOptions struct {
	// ProviderName selects which registered provider to use for classification
	// (e.g. "openai", "anthropic").
	ProviderName string
	// Model is the model ID to pass to the provider (e.g. "gpt-4o-mini").
	Model string
	// RouteDescriptions is the ordered list of routes the LLM may choose from.
	// If empty, DefaultRouteDescriptions are used.
	RouteDescriptions []RouteDescription
	// ClassifyTimeout overrides the per-call context deadline (default 10s).
	ClassifyTimeout time.Duration
}

// DefaultRouteDescriptions is the built-in route table used when
// LLMRouterOptions.RouteDescriptions is not provided.
var DefaultRouteDescriptions = []RouteDescription{
	{
		Name:        "trivial",
		Handler:     "template",
		Description: "Static responses for greetings, thanks, capabilities questions.",
		Fallback:    "llm-fast",
	},
	{
		Name:        "quick",
		Handler:     "llm-fast",
		Description: "Simple factual answers, short explanations, arithmetic.",
		Fallback:    "llm-reasoning",
	},
	{
		Name:        "deep",
		Handler:     "llm-reasoning",
		Description: "Complex analysis, code generation, debugging, architecture design.",
		Fallback:    "llm-fast",
	},
	{
		Name:        "specialist",
		Handler:     "specialist",
		Description: "Domain-specific tasks requiring specialized tools.",
		Fallback:    "llm-reasoning",
	},
	{
		Name:        "orchestrate",
		Handler:     "orchestrate",
		Description: "Multi-step tasks requiring decomposition and planning.",
		Fallback:    "llm-reasoning",
	},
}

// classifyResponse is the JSON structure expected from the LLM.
type classifyResponse struct {
	Route      string  `json:"route"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// DefaultLLMRouter implements LLMRouter by sending a short classification
// prompt to a fast LLM and parsing the JSON response.
type DefaultLLMRouter struct {
	pm      *provider.PluginManager
	opts    LLMRouterOptions
	routes  []RouteDescription
	routeBy map[string]RouteDescription // keyed by RouteDescription.Name
}

// NewDefaultLLMRouter constructs a DefaultLLMRouter with the supplied options.
// RouteDescriptions defaults to DefaultRouteDescriptions when not set.
// ClassifyTimeout defaults to 10 seconds when not set.
func NewDefaultLLMRouter(pm *provider.PluginManager, opts LLMRouterOptions) *DefaultLLMRouter {
	routes := opts.RouteDescriptions
	if len(routes) == 0 {
		routes = DefaultRouteDescriptions
	}

	if opts.ClassifyTimeout == 0 {
		opts.ClassifyTimeout = 10 * time.Second
	}

	byName := make(map[string]RouteDescription, len(routes))
	for _, r := range routes {
		byName[r.Name] = r
	}

	return &DefaultLLMRouter{
		pm:      pm,
		opts:    opts,
		routes:  routes,
		routeBy: byName,
	}
}

// ClassifyWithLLM sends the query to the configured LLM provider and maps the
// JSON response to a RoutingDecision. On any error (provider unavailable,
// network failure, unparseable JSON, unknown route name) it returns a safe
// default: handler="llm-fast", confidence=0.3.
func (r *DefaultLLMRouter) ClassifyWithLLM(ctx context.Context, query string) (RoutingDecision, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.opts.ClassifyTimeout)
	defer cancel()

	llmProvider, err := r.pm.GetProvider(r.opts.ProviderName)
	if err != nil {
		slog.Warn("llm_router: provider unavailable, using default route",
			"provider", r.opts.ProviderName,
			"error", err,
		)
		return r.defaultDecision("provider not available: " + err.Error()), nil
	}

	prompt := r.buildPrompt(query)

	req := &provider.CompletionRequest{
		Model:       r.opts.Model,
		Temperature: 0,
		MaxTokens:   150,
		Messages: []provider.Message{
			{Role: "user", Content: prompt},
		},
	}

	slog.Debug("llm_router: calling provider for classification",
		"provider", r.opts.ProviderName,
		"model", r.opts.Model,
		"query_len", len(query),
	)

	resp, err := llmProvider.GenerateCompletion(callCtx, req)
	if err != nil {
		slog.Warn("llm_router: completion failed, using default route",
			"provider", r.opts.ProviderName,
			"model", r.opts.Model,
			"error", err,
		)
		return r.defaultDecision("LLM call failed: " + err.Error()), nil
	}

	decision, parseErr := r.parseResponse(resp.Content)
	if parseErr != nil {
		slog.Warn("llm_router: parse error, using default route",
			"raw", resp.Content,
			"error", parseErr,
		)
		return r.defaultDecision("LLM classification failed, defaulting to llm-fast"), nil
	}

	slog.Debug("llm_router: classification complete",
		"route", decision.Handler,
		"confidence", decision.Confidence,
		"category", decision.Category,
	)

	return decision, nil
}

// buildPrompt constructs the classification prompt for the LLM.
func (r *DefaultLLMRouter) buildPrompt(query string) string {
	var sb strings.Builder

	sb.WriteString("You are a query router for an AI agent gateway. ")
	sb.WriteString("Classify the following user query into exactly one route.\n\n")
	sb.WriteString("Available routes:\n")

	for _, rd := range r.routes {
		fmt.Fprintf(&sb, "- %s: %s Handler: %s\n", rd.Name, rd.Description, rd.Handler)
	}

	sb.WriteString("\nUser query: \"")
	sb.WriteString(query)
	sb.WriteString("\"\n\n")
	sb.WriteString("Respond with JSON only:\n")
	sb.WriteString(`{"route": "...", "confidence": 0.0-1.0, "reasoning": "..."}`)

	return sb.String()
}

// parseResponse extracts a classifyResponse from the raw LLM output and maps
// it to a RoutingDecision. Returns an error when the JSON is malformed or the
// route name is not in the configured route table.
func (r *DefaultLLMRouter) parseResponse(raw string) (RoutingDecision, error) {
	// Strip markdown code fences and surrounding whitespace.
	content := strings.TrimSpace(raw)
	if idx := strings.Index(content, "{"); idx > 0 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "}"); idx >= 0 {
		content = content[:idx+1]
	}
	content = strings.TrimSpace(content)

	var cr classifyResponse
	if err := json.Unmarshal([]byte(content), &cr); err != nil {
		return RoutingDecision{}, fmt.Errorf("json unmarshal: %w (raw=%q)", err, raw)
	}

	rd, ok := r.routeBy[cr.Route]
	if !ok {
		return RoutingDecision{}, fmt.Errorf("unknown route %q in LLM response", cr.Route)
	}

	// Clamp confidence to [0, 1].
	conf := cr.Confidence
	if conf < 0 {
		conf = 0
	} else if conf > 1 {
		conf = 1
	}

	return RoutingDecision{
		Handler:    rd.Handler,
		Provider:   rd.ProviderAlias,
		Fallback:   rd.Fallback,
		Confidence: conf,
		Category:   r.handlerToCategory(rd.Handler),
		Reasoning:  []string{cr.Reasoning},
	}, nil
}

// defaultDecision returns the safe fallback RoutingDecision used whenever the
// LLM call fails or produces unparseable output.
func (r *DefaultLLMRouter) defaultDecision(reason string) RoutingDecision {
	return RoutingDecision{
		Handler:    "llm-fast",
		Confidence: 0.3,
		Category:   Factual,
		Reasoning:  []string{"LLM classification failed, defaulting to llm-fast", reason},
	}
}

// handlerToCategory maps a handler key to the nearest IntentCategory for
// downstream consumers that rely on category-based branching.
func (r *DefaultLLMRouter) handlerToCategory(handler string) IntentCategory {
	switch handler {
	case "template":
		return Greeting
	case "llm-fast":
		return Factual
	case "llm-reasoning":
		return Explanation
	case "specialist":
		return Debugging
	case "orchestrate":
		return Planning
	default:
		return Factual
	}
}
