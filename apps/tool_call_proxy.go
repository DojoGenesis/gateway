package apps

import (
	"context"
	"fmt"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// ToolCallRequest represents a tool call from an app.
type ToolCallRequest struct {
	AppID     string                 `json:"app_id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	SessionID string                 `json:"session_id"`
}

// ToolCallResponse represents the result of a proxied tool call.
type ToolCallResponse struct {
	Result  map[string]interface{} `json:"result,omitempty"`
	IsError bool                   `json:"is_error"`
	Error   string                 `json:"error,omitempty"`
}

// ToolCallProxy proxies tool calls from apps through the gateway's tool registry.
type ToolCallProxy struct {
	registry       gateway.ToolRegistry
	policy         *AuthorizationPolicy
	defaultTimeout time.Duration
}

// NewToolCallProxy creates a proxy with the given registry and authorization policy.
func NewToolCallProxy(registry gateway.ToolRegistry, policy *AuthorizationPolicy) *ToolCallProxy {
	return &ToolCallProxy{
		registry:       registry,
		policy:         policy,
		defaultTimeout: 30 * time.Second,
	}
}

// SetDefaultTimeout overrides the default 30s timeout for tool calls.
func (p *ToolCallProxy) SetDefaultTimeout(d time.Duration) {
	p.defaultTimeout = d
}

// ProxyCall authorizes and executes a tool call on behalf of an app.
func (p *ToolCallProxy) ProxyCall(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	if req == nil {
		return &ToolCallResponse{
			IsError: true,
			Error:   "tool call request cannot be nil",
		}, nil
	}

	// Check authorization
	if !p.policy.CanCallTool(req.AppID, req.ToolName) {
		return &ToolCallResponse{
			IsError: true,
			Error:   fmt.Sprintf("app %s is not authorized to call tool %s", req.AppID, req.ToolName),
		}, nil
	}

	// Lookup tool
	tool, err := p.registry.Get(ctx, req.ToolName)
	if err != nil {
		return &ToolCallResponse{
			IsError: true,
			Error:   fmt.Sprintf("tool not found: %s", req.ToolName),
		}, nil
	}

	if tool.Function == nil {
		return &ToolCallResponse{
			IsError: true,
			Error:   fmt.Sprintf("tool %s has no function implementation", req.ToolName),
		}, nil
	}

	// Apply timeout
	timeout := p.defaultTimeout
	if tool.Timeout > 0 {
		timeout = tool.Timeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute tool
	args := req.Arguments
	if args == nil {
		args = make(map[string]interface{})
	}

	result, err := tool.Function(callCtx, args)
	if err != nil {
		return &ToolCallResponse{
			IsError: true,
			Error:   fmt.Sprintf("tool execution failed: %s", err.Error()),
		}, nil
	}

	return &ToolCallResponse{
		Result:  result,
		IsError: false,
	}, nil
}
