# Refactoring Needed: ADA Contract Alignment

**Status:** Documented Technical Debt  
**Priority:** High (but not blocking for v0.2.0)  
**Effort:** 2-3 hours  
**Target:** v0.3.0 or Phase 2.5

## Issue

The `pkg/disposition/` package was implemented based on inferred requirements from Prompt 2B before the actual ADA contract specifications were available. Now that `docs/v0.2.0/contract-gateway-ada.md` and `docs/v0.2.0/gateway-ada-finalization.md` exist, there's a **schema mismatch**.

## Current Implementation (Inferred)

```go
type DispositionConfig struct {
    AgentID       string
    Name          string
    Validation    ValidationConfig    // Complex nested struct
    ErrorHandling ErrorHandlingConfig // with many fields
    Collaboration CollaborationConfig
    Reflection    ReflectionConfig
    Pacing        PacingConfig        // Nested struct with ms delays
    Depth         DepthConfig         // Nested struct with compression
}
```

## Actual ADA Contract Schema

```go
type DispositionConfig struct {
    // Core behavioral dimensions (REQUIRED - currently missing!)
    Pacing     string // deliberate | measured | responsive | rapid
    Depth      string // surface | functional | thorough | exhaustive
    Tone       string // formal | professional | conversational | casual
    Initiative string // reactive | responsive | proactive | autonomous
    
    // Nested configs (simpler than current implementation)
    Validation    ValidationConfig
    ErrorHandling ErrorHandlingConfig
    Collaboration CollaborationConfig
    Reflection    ReflectionConfig
}

type ValidationConfig struct {
    Strategy     string // none | spot-check | thorough | exhaustive
    RequireTests bool
    RequireDocs  bool
}

type ErrorHandlingConfig struct {
    Strategy   string // fail-fast | log-and-continue | retry | escalate
    RetryCount int    // 0-10
}

type CollaborationConfig struct {
    Style            string // independent | consultative | collaborative | delegating
    CheckInFrequency string // never | rarely | regularly | constantly
}

type ReflectionConfig struct {
    Frequency string   // never | session-end | daily | weekly
    Format    string   // structured | narrative | bullets
    Triggers  []string
}
```

## Impact Analysis

### What Works (No Changes Needed)
- ✅ Test infrastructure (93.5% coverage)
- ✅ File resolution and caching logic
- ✅ Mode merging algorithm
- ✅ Validation error formatting
- ✅ AgentInitializer interface implementation

### What Needs Refactoring
- ❌ DispositionConfig struct fields
- ❌ Nested config struct definitions
- ❌ Validator enum values
- ❌ Test fixtures (YAML files)
- ❌ Gateway type definitions
- ❌ Documentation and examples

## Refactoring Checklist

### Phase 1: Type Definitions (30 min)
- [ ] Add core fields to DispositionConfig: Pacing, Depth, Tone, Initiative
- [ ] Simplify ValidationConfig to match contract
- [ ] Simplify ErrorHandlingConfig to match contract
- [ ] Simplify CollaborationConfig to match contract
- [ ] Simplify ReflectionConfig to match contract
- [ ] Remove PacingConfig and DepthConfig structs (now string enums)

### Phase 2: Validators (30 min)
- [ ] Update validateValidationConfig for new enum values
- [ ] Update validateErrorHandlingConfig for new enum values
- [ ] Update validateCollaborationConfig for new enum values
- [ ] Update validateReflectionConfig for new enum values
- [ ] Add validators for Pacing, Depth, Tone, Initiative enums

### Phase 3: Defaults (15 min)
- [ ] Update DefaultDisposition() with new field values
- [ ] Update MinimalDisposition() with new field values

### Phase 4: Gateway Types (15 min)
- [ ] Update pkg/gateway/types.go to match contract
- [ ] Update convertToAgentConfig mapping

### Phase 5: Tests (45 min)
- [ ] Update all test fixtures to new schema
- [ ] Update test assertions for new field names
- [ ] Add tests for new core dimension fields
- [ ] Update benchmarks if needed

### Phase 6: Documentation (15 min)
- [ ] Update package documentation
- [ ] Add schema version documentation
- [ ] Update YAML examples in comments

## Migration Strategy

### Option A: Breaking Change in v0.3.0 (Recommended)
- Keep current implementation for v0.2.0
- Mark current types as deprecated
- Create new types matching contract in v0.3.0
- Provide migration guide

### Option B: Immediate Refactor (Risky)
- Refactor now before Phase 3 integration
- Risk: delays v0.2.0 release
- Benefit: correct from the start

### Option C: Hybrid Approach
- Add core dimension fields now (Pacing, Depth, Tone, Initiative)
- Keep complex nested structs as-is
- Gradually align in v0.3.0

## Recommendation

**Defer to v0.3.0** with Option A:
1. Current implementation is functional and well-tested
2. Phase 3 can proceed with current interface
3. Breaking changes are better for major version bumps
4. Gives time to align with upstream ADA spec changes

## Files Affected

```
pkg/disposition/disposition.go       - Type definitions
pkg/disposition/validator.go         - Enum validation
pkg/disposition/defaults.go          - Default values
pkg/disposition/resolver.go          - Minor (merge logic)
pkg/disposition/agent_initializer.go - Mapping function
pkg/gateway/types.go                 - Gateway type alignment
pkg/disposition/testdata/*.yaml      - Test fixtures
pkg/disposition/*_test.go            - All tests
```

## Contract Compliance Tracking

| Requirement | Current | Contract | Status |
|-------------|---------|----------|--------|
| Core dimensions | Missing | Required | ❌ Not Compliant |
| ValidationConfig | Complex | Simple | ⚠️ Superset |
| ErrorHandlingConfig | Complex | Simple | ⚠️ Superset |
| CollaborationConfig | Complex | Simple | ⚠️ Superset |
| ReflectionConfig | Complex | Simple | ⚠️ Superset |
| File resolution | ✅ | ✅ | ✅ Compliant |
| Mode merging | ✅ | ✅ | ✅ Compliant |
| Performance <100ms | ✅ (0.071ms) | ✅ | ✅ Compliant |

## Notes

- Current implementation is a **superset** of the contract (more features)
- Contract alignment can be done incrementally
- No runtime bugs or functional issues with current code
- This is a **structural/interface** issue, not a logic issue

---

**Decision:** Document as technical debt, proceed with v0.2.0, refactor in v0.3.0
**Tracking Issue:** Create GitHub issue for v0.3.0 milestone
