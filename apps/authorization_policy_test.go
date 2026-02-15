package apps

import (
	"testing"
)

func TestAuthorizationPolicy_GrantAndCheck(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantToolAccess("app-1", "tool-a")

	if !p.CanCallTool("app-1", "tool-a") {
		t.Error("expected app-1 to have access to tool-a")
	}
	if p.CanCallTool("app-1", "tool-b") {
		t.Error("expected app-1 to NOT have access to tool-b")
	}
	if p.CanCallTool("app-2", "tool-a") {
		t.Error("expected app-2 to NOT have access to tool-a")
	}
}

func TestAuthorizationPolicy_WildcardAccess(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantAllToolAccess("app-1")

	if !p.CanCallTool("app-1", "tool-a") {
		t.Error("expected wildcard access to tool-a")
	}
	if !p.CanCallTool("app-1", "any-tool-name") {
		t.Error("expected wildcard access to any-tool-name")
	}
	if p.CanCallTool("app-2", "tool-a") {
		t.Error("expected app-2 to NOT have wildcard access")
	}
}

func TestAuthorizationPolicy_DenyUngranted(t *testing.T) {
	p := NewAuthorizationPolicy()

	if p.CanCallTool("app-1", "tool-a") {
		t.Error("expected deny for ungranted app")
	}
}

func TestAuthorizationPolicy_RevokeAccess(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-b")

	p.RevokeToolAccess("app-1", "tool-a")

	if p.CanCallTool("app-1", "tool-a") {
		t.Error("expected tool-a to be revoked")
	}
	if !p.CanCallTool("app-1", "tool-b") {
		t.Error("expected tool-b to still be granted")
	}
}

func TestAuthorizationPolicy_IdempotentGrant(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-a")

	perms := p.ListPermissions("app-1")
	if len(perms) != 1 {
		t.Errorf("expected 1 permission after idempotent grants, got %d", len(perms))
	}
}

func TestAuthorizationPolicy_IdempotentWildcard(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantAllToolAccess("app-1")
	p.GrantAllToolAccess("app-1")

	perms := p.ListPermissions("app-1")
	wildcardCount := 0
	for _, perm := range perms {
		if perm == "*" {
			wildcardCount++
		}
	}
	if wildcardCount != 1 {
		t.Errorf("expected 1 wildcard after idempotent grants, got %d", wildcardCount)
	}
}

func TestAuthorizationPolicy_ListPermissions(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-b")
	p.GrantToolAccess("app-1", "tool-c")

	perms := p.ListPermissions("app-1")
	if len(perms) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(perms))
	}

	permSet := make(map[string]bool)
	for _, perm := range perms {
		permSet[perm] = true
	}
	for _, want := range []string{"tool-a", "tool-b", "tool-c"} {
		if !permSet[want] {
			t.Errorf("missing permission %q", want)
		}
	}
}

func TestAuthorizationPolicy_ListPermissionsEmpty(t *testing.T) {
	p := NewAuthorizationPolicy()

	perms := p.ListPermissions("nonexistent-app")
	if len(perms) != 0 {
		t.Errorf("expected 0 permissions for unknown app, got %d", len(perms))
	}
}

func TestAuthorizationPolicy_RevokeAll(t *testing.T) {
	p := NewAuthorizationPolicy()

	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-b")
	p.GrantAllToolAccess("app-1")

	p.RevokeAll("app-1")

	if p.CanCallTool("app-1", "tool-a") {
		t.Error("expected all access revoked")
	}
	if len(p.ListPermissions("app-1")) != 0 {
		t.Error("expected empty permissions after RevokeAll")
	}
}
