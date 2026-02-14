// Package intelligence provides proactive suggestion and auto-execution
// capabilities driven by the agent's disposition initiative setting.
//
// Initiative levels (reactive, responsive, proactive, autonomous) control
// when the engine suggests actions, auto-executes low-risk tasks, or defers
// to the user.
//
// Usage:
//
//	engine := intelligence.NewProactiveEngine(intelligence.WithDisposition(cfg))
//	if engine.ShouldSuggest(ctx, event) {
//	    suggestions, _ := engine.GenerateSuggestions(ctx, state)
//	}
package intelligence
