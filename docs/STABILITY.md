# API Stability Policy

## Current Phase: v0.x (Experimental)

- The gateway's public API surfaces (`/v1/chat/completions`, `/v1/gateway/*`, `/admin/*`) are subject to breaking changes as we gather integrator feedback
- The ten Go module interfaces (`ToolRegistry`, `SkillExecutor`, `OrchestrationEngine`, `MemoryStore`, `MCPHostManager`, etc.) are experimental
- The codebase is production-ready; the APIs are experimental
- Integrators should monitor releases for breaking changes

## Planned v1.0.0 Release

**Target Date:** Q3 2026 (est. June–September)

- HTTP API endpoints will be frozen and stable
- Go module interfaces will be frozen and stable
- All v1.x.y releases will maintain backward compatibility
- Breaking changes will only occur in v2.0.0+

## What This Means

- **If you're using v0.x:** Expect breaking changes in minor releases. Use explicit versions in `go.mod`.
- **When v1.0.0 releases:** Upgrade confidently. All v1.x versions will be compatible.
- **If you need v1 before our timeline:** Contact the team; we'll prioritize.

## Making Changes

### While in v0.x (Before v1.0)

**You can freely:**
- Add new methods to existing interfaces
- Change method signatures
- Remove interfaces entirely
- Restructure packages
- Add, rename, or remove HTTP endpoints

**You should document:**
- What changed in release notes
- How to migrate (if not obvious)
- When it will stabilize (point to this file's timeline)

### After v1.0.0

**Safe changes (v1.x.y):**
- Add new interfaces
- Add new standalone functions
- Add struct fields (if zero-value is sensible)
- Add new HTTP endpoints (preserving existing ones)

**Forbidden changes (requires v2.0.0):**
- Change existing method signatures
- Add methods to existing interfaces
- Remove exported types or functions
- Change HTTP endpoint paths or response shapes

## Upgrading from v0 to v1

See MIGRATION.md (published with v1.0.0) for detailed migration guide.
