// Package reflection provides disposition-driven introspection and session
// reflection capabilities for agents.
//
// It determines when to generate reflections based on frequency (never,
// session-end, daily, weekly) and formats output as structured, narrative,
// or bullet-point summaries.
//
// Usage:
//
//	engine := reflection.NewEngine(reflection.WithDisposition(cfg))
//	engine.LogEvent(event)
//	if engine.ShouldReflect(ctx, "session_end") {
//	    output, _ := engine.GenerateReflection(ctx, sessionData)
//	}
package reflection
