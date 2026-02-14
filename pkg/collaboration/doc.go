// Package collaboration provides disposition-driven check-in behavior for
// multi-agent and human-in-the-loop workflows.
//
// It determines when an agent should check in with the user based on
// collaboration style (independent, consultative, collaborative, delegating)
// and frequency (never, rarely, regularly, constantly).
//
// Usage:
//
//	mgr := collaboration.NewManager(collaboration.WithDisposition(cfg))
//	if mgr.ShouldCheckIn(ctx, event) {
//	    // prompt the user for feedback
//	}
package collaboration
