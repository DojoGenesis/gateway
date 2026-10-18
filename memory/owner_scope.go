package memory

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ─── Owner scoping (DGS-163) ────────────────────────────────────────────────
//
// Every memory row carries owner_id, the authenticated subject (JWT sub) that
// wrote it. The HTTP layer and anything else acting for a caller must go
// through the *Scoped methods below with an OwnerScope; the unscoped methods
// (GetMemory, Retrieve, Delete, SearchMemoriesPage, ...) remain for internal
// callers such as maintenance and compression that work across the store.

// LegacyOwnerID is the owner_id of rows written before DGS-163 and of rows
// stored by internal callers with no subject. No authenticated subject is
// empty, so a ForOwner scope never matches these rows.
const LegacyOwnerID = ""

// ErrInvalidScope is returned for a zero OwnerScope or ForOwner(""). It makes
// a forgotten scope fail closed instead of reading every owner's rows.
var ErrInvalidScope = errors.New("memory: owner scope is not set")

// OwnerScope says whose memories an operation may touch. Build it with
// ForOwner or AllOwners; the zero value is invalid and every scoped method
// rejects it.
type OwnerScope struct {
	ownerID string
	all     bool
	set     bool
}

// ForOwner scopes an operation to one subject's memories.
func ForOwner(ownerID string) OwnerScope {
	return OwnerScope{ownerID: ownerID, set: ownerID != ""}
}

// AllOwners scopes an operation to every owner's memories, legacy rows
// included. Only an explicit admin request should construct it.
func AllOwners() OwnerScope {
	return OwnerScope{all: true, set: true}
}

// IsAllOwners reports whether s is the cross-owner scope.
func (s OwnerScope) IsAllOwners() bool { return s.all }

// OwnerID returns the subject s is scoped to, or "" for AllOwners.
func (s OwnerScope) OwnerID() string { return s.ownerID }

// predicate converts s to the *string the internal queries take: nil for
// AllOwners, the owner otherwise.
func (s OwnerScope) predicate() (*string, error) {
	if !s.set {
		return nil, ErrInvalidScope
	}
	if s.all {
		return nil, nil
	}
	owner := s.ownerID
	return &owner, nil
}

func (s OwnerScope) auditMeta(meta map[string]interface{}) map[string]interface{} {
	if s.all {
		meta["owner_scope"] = "all"
	} else {
		meta["owner_id"] = s.ownerID
	}
	return meta
}

// GetMemoryScoped retrieves a memory by ID within scope. Another owner's
// memory reads as ErrMemoryNotFound, so callers cannot probe for IDs.
func (m *MemoryManager) GetMemoryScoped(ctx context.Context, scope OwnerScope, id string) (*Memory, error) {
	start := time.Now()
	owner, err := scope.predicate()
	var mem *Memory
	if err == nil {
		mem, err = m.getMemory(ctx, id, owner)
	}
	m.observe(ctx, "get", scope.auditMeta(map[string]interface{}{"memory_id": id}), err, start)
	return mem, err
}

// UpdateScoped patches content and/or metadata of a memory within scope. It
// never changes the memory's owner.
func (m *MemoryManager) UpdateScoped(ctx context.Context, scope OwnerScope, id, content string, metadata map[string]interface{}) (*Memory, error) {
	start := time.Now()
	owner, err := scope.predicate()
	var mem *Memory
	if err == nil {
		mem, err = m.getMemory(ctx, id, owner)
	}
	if err == nil {
		if content != "" {
			mem.Content = content
		}
		if metadata != nil {
			mem.Metadata = metadata
		}
		err = m.updateMemory(ctx, mem, owner)
	}
	meta := map[string]interface{}{"memory_id": id}
	if err == nil {
		meta["content_sha256"] = hashText(mem.Content)
	}
	m.observe(ctx, "update", scope.auditMeta(meta), err, start)
	if err != nil {
		return nil, err
	}
	return mem, nil
}

// DeleteMemoryScoped deletes a memory by ID within scope. Another owner's
// memory reads as ErrMemoryNotFound.
func (m *MemoryManager) DeleteMemoryScoped(ctx context.Context, scope OwnerScope, id string) error {
	start := time.Now()
	owner, err := scope.predicate()
	if err == nil {
		err = m.deleteMemory(ctx, id, owner)
	}
	m.observe(ctx, "delete", scope.auditMeta(map[string]interface{}{"memory_id": id}), err, start)
	return err
}

// SearchMemoriesPageScoped is SearchMemoriesPage within scope; total counts
// only matches inside the scope.
func (m *MemoryManager) SearchMemoriesPageScoped(ctx context.Context, scope OwnerScope, query string, limit, offset int) ([]SearchResult, int, error) {
	start := time.Now()
	owner, err := scope.predicate()
	var results []SearchResult
	var total int
	if err == nil {
		results, total, err = m.searchMemoriesPage(ctx, query, limit, offset, owner)
	}
	m.observe(ctx, "search", scope.auditMeta(map[string]interface{}{
		"query_sha256": hashText(query),
		"returned":     len(results),
		"total":        total,
	}), err, start)
	return results, total, err
}

// ListMemoriesScoped is ListMemories within scope. scope replaces any
// filter.OwnerID the caller set.
func (m *MemoryManager) ListMemoriesScoped(ctx context.Context, scope OwnerScope, filter MemoryFilter) ([]*Memory, error) {
	owner, err := scope.predicate()
	if err != nil {
		m.observe(ctx, "list", scope.auditMeta(map[string]interface{}{}), err, time.Now())
		return nil, err
	}
	filter.OwnerID = owner
	return m.ListMemories(ctx, filter)
}

// CountMemoriesScoped is CountMemories within scope. scope replaces any
// filter.OwnerID the caller set.
func (m *MemoryManager) CountMemoriesScoped(ctx context.Context, scope OwnerScope, filter MemoryFilter) (int, error) {
	owner, err := scope.predicate()
	if err != nil {
		return 0, err
	}
	filter.OwnerID = owner
	return m.CountMemories(ctx, filter)
}

// DeleteAllForOwner removes every memory owned by ownerID and returns how many
// rows went. It is the account-deletion path (DGS-150 kata-account-delete).
// An empty ownerID is refused, so it can never wipe legacy rows or the store.
func (m *MemoryManager) DeleteAllForOwner(ctx context.Context, ownerID string) (int64, error) {
	start := time.Now()
	var deleted int64
	err := func() error {
		if ownerID == LegacyOwnerID {
			return ErrInvalidScope
		}
		result, err := m.db.ExecContext(ctx, `DELETE FROM memories WHERE owner_id = ?`, ownerID)
		if err != nil {
			return fmt.Errorf("failed to delete memories for owner: %w", err)
		}
		deleted, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		return nil
	}()
	m.observe(ctx, "delete_all_for_owner", map[string]interface{}{
		"owner_id": ownerID,
		"deleted":  deleted,
	}, err, start)
	return deleted, err
}

type ownerCtxKey struct{}

// WithOwner returns a context carrying the authenticated subject, for code
// paths whose interface has no owner parameter (GatewayMemoryStore).
func WithOwner(ctx context.Context, ownerID string) context.Context {
	return context.WithValue(ctx, ownerCtxKey{}, ownerID)
}

// ScopeFromContext returns ForOwner of the subject WithOwner put on ctx. With
// no subject on ctx it returns the zero scope, which every scoped method
// rejects with ErrInvalidScope.
func ScopeFromContext(ctx context.Context) OwnerScope {
	ownerID, _ := ctx.Value(ownerCtxKey{}).(string)
	return ForOwner(ownerID)
}
