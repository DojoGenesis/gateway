// Package errors provides disposition-driven error handling for the Agentic Gateway.
//
// It decides how to handle errors based on the agent's disposition configuration,
// supporting four strategies: fail-fast, log-and-continue, retry, and escalate.
//
// Usage:
//
//	handler := errors.NewHandler(errors.WithDisposition(cfg))
//	decision := handler.HandleError(ctx, err, attemptCount)
//	if decision.ShouldRetry() {
//	    // retry the operation
//	}
package errors
