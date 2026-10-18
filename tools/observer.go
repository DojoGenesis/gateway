package tools

import (
	"context"
	"sync/atomic"
	"time"
)

// Call observation for audit.
//
// Like the Authorizer, the audit log lives in a module this one cannot import,
// so the process that owns the log installs an observer here with
// SetCallObserver. Every path that runs a tool function reports to it through
// ObserveCall: InvokeToolWithTimeout does so itself, and code that calls
// ToolDefinition.Function directly must call ObserveCall alongside it, the
// same way it calls AuthorizeCall.
//
// A refused call is observed too, with the refusal as its error, so the log
// shows what was attempted and not only what ran.

// CallObserver is told about one tool call after it finishes. It must not
// modify params or result.
type CallObserver func(ctx context.Context, name string, params, result map[string]interface{}, err error, d time.Duration)

var callObserver atomic.Pointer[CallObserver]

// SetCallObserver installs o as the process-wide observer; nil removes it.
func SetCallObserver(o CallObserver) {
	if o == nil {
		callObserver.Store(nil)
		return
	}
	callObserver.Store(&o)
}

// ObserveCall reports a finished tool call to the installed observer, if any.
func ObserveCall(ctx context.Context, name string, params, result map[string]interface{}, err error, d time.Duration) {
	if o := callObserver.Load(); o != nil {
		(*o)(ctx, name, params, result, err, d)
	}
}
