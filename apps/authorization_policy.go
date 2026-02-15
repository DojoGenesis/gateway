package apps

import (
	"sync"
)

// AuthorizationPolicy controls which apps can call which tools.
type AuthorizationPolicy struct {
	mu     sync.RWMutex
	grants map[string][]string // appID → tool names ("*" for wildcard)
}

// NewAuthorizationPolicy creates an empty authorization policy.
func NewAuthorizationPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		grants: make(map[string][]string),
	}
}

// GrantToolAccess grants a specific tool to an app. Idempotent.
func (p *AuthorizationPolicy) GrantToolAccess(appID, toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tools := p.grants[appID]
	for _, t := range tools {
		if t == toolName {
			return // already granted
		}
	}
	p.grants[appID] = append(tools, toolName)
}

// GrantAllToolAccess grants wildcard access to an app.
func (p *AuthorizationPolicy) GrantAllToolAccess(appID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tools := p.grants[appID]
	for _, t := range tools {
		if t == "*" {
			return // already has wildcard
		}
	}
	p.grants[appID] = append(tools, "*")
}

// RevokeToolAccess removes a specific tool grant from an app.
func (p *AuthorizationPolicy) RevokeToolAccess(appID, toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tools := p.grants[appID]
	for i, t := range tools {
		if t == toolName {
			p.grants[appID] = append(tools[:i], tools[i+1:]...)
			return
		}
	}
}

// CanCallTool checks whether an app is authorized to call a tool.
func (p *AuthorizationPolicy) CanCallTool(appID, toolName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := p.grants[appID]
	for _, t := range tools {
		if t == "*" || t == toolName {
			return true
		}
	}
	return false
}

// ListPermissions returns all granted tool names for an app.
func (p *AuthorizationPolicy) ListPermissions(appID string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := p.grants[appID]
	result := make([]string, len(tools))
	copy(result, tools)
	return result
}

// RevokeAll removes all grants for an app.
func (p *AuthorizationPolicy) RevokeAll(appID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.grants, appID)
}
