package tools

import (
	"context"
	"errors"
	"log/slog"
)

// Caller authorization for tool invocation.
//
// The registry in this package is deliberately policy-free: it knows how to
// find and run a tool, not who may ask. Policy belongs to whoever exposes the
// registry to a principal (the HTTP gateway), and that code lives in a
// different module that this one cannot import.
//
// So the policy travels as a value on the context instead. An entry point that
// has authenticated a caller binds an Authorizer to the request context with
// WithAuthorizer; InvokeTool and InvokeToolWithTimeout then consult it before
// any tool function runs. Every call that descends from that context (an agent
// loop, an orchestration node, a direct invoke route) is covered without each
// call site having to remember to ask.
//
// A context with NO authorizer is unrestricted. That is the right default for
// callers that are not acting for a remote principal (tests, an embedding
// process, the CLI), and it is also the sharp edge: code that detaches from a
// request context (context.Background) drops the policy with it. Use
// InheritAuthorizer to carry it across such a boundary.

// Authorizer reports whether the caller bound to a context may invoke the named
// tool. It returns nil to allow and a *ToolAccessError to refuse.
type Authorizer func(toolName string) error

type authorizerKey struct{}

// ToolAccessError is the refusal returned when an Authorizer denies a call.
//
// It carries a stable machine-readable Code and a message that is safe to show
// to the caller. It never carries the call's arguments.
type ToolAccessError struct {
	Tool    string
	Code    string
	Message string
}

func (e *ToolAccessError) Error() string { return e.Message }

// AsToolAccessError reports whether err is (or wraps) a *ToolAccessError.
func AsToolAccessError(err error) (*ToolAccessError, bool) {
	var ae *ToolAccessError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// WithAuthorizer returns a context carrying a. A nil authorizer returns ctx
// unchanged, so it cannot be used to lift a restriction already in force.
func WithAuthorizer(ctx context.Context, a Authorizer) context.Context {
	if a == nil {
		return ctx
	}
	return context.WithValue(ctx, authorizerKey{}, a)
}

func authorizerFrom(ctx context.Context) Authorizer {
	a, _ := ctx.Value(authorizerKey{}).(Authorizer)
	return a
}

// InheritAuthorizer copies the authorizer bound to src (if any) onto dst.
// Use it where work outlives its request and so runs on a fresh context.
func InheritAuthorizer(dst, src context.Context) context.Context {
	return WithAuthorizer(dst, authorizerFrom(src))
}

// AuthorizeCall consults the authorizer bound to ctx. It returns nil when none
// is bound or when the call is permitted. A refusal is logged with the tool name
// and reason; the call's arguments are never logged.
func AuthorizeCall(ctx context.Context, toolName string) error {
	err := authorize(ctx, toolName)
	if err != nil {
		slog.Warn("tool call refused", "tool", toolName, "reason", err.Error())
	}
	return err
}

// authorize is AuthorizeCall without the log line, for FilterAuthorized, which
// probes every registered tool and would otherwise log one refusal per tool per
// request.
func authorize(ctx context.Context, toolName string) error {
	if a := authorizerFrom(ctx); a != nil {
		return a(toolName)
	}
	return nil
}

// FilterAuthorized returns the subset of defs the caller bound to ctx may
// invoke. Offering a model only tools it can use avoids spending iterations on
// calls that are certain to be refused; it is a convenience, and the refusal in
// InvokeToolWithTimeout remains the control.
func FilterAuthorized(ctx context.Context, defs []*ToolDefinition) []*ToolDefinition {
	if authorizerFrom(ctx) == nil {
		return defs
	}
	out := make([]*ToolDefinition, 0, len(defs))
	for _, d := range defs {
		if authorize(ctx, d.Name) == nil {
			out = append(out, d)
		}
	}
	return out
}
