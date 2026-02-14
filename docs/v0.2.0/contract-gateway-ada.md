# Gateway ↔ Agent Identity Contract

**Contract Version:** 1.0.0
**Effective Date:** 2026-02-12
**Layer Boundary:** Runtime (AgenticGateway) ↔ Identity (Agent Identities by Dojo Genesis)

---

## Purpose

This contract defines the formal interface between the AgenticGateway runtime and the Agent Identities by Dojo Genesis (ADA) system. It specifies:

1. How the gateway loads agent disposition at initialization
2. The Go type system for disposition configuration
3. The runtime behavior guarantees the gateway provides
4. The validation and error handling requirements

This contract ensures that any runtime implementing this interface can correctly consume ADA identity files and provide consistent behavioral configuration to agents.

---

## Contract Parties

**Provider:** Agent Identities by Dojo Genesis
**Consumer:** AgenticGateway (and other Go-based runtimes)

**Specification Source:** `AgentIdentitiesByDojoGenesis/docs/agent-yaml-bridge-spec.md` v1.0.0

---

## Type Definitions

### DispositionConfig

The primary data structure representing resolved agent disposition:

```go
package disposition

// DispositionConfig represents the effective disposition configuration
// after resolving the active mode and applying overrides.
type DispositionConfig struct {
	// Core behavioral dimensions (required)
	Pacing     string `json:"pacing" yaml:"pacing"`         // deliberate | measured | responsive | rapid
	Depth      string `json:"depth" yaml:"depth"`           // surface | functional | thorough | exhaustive
	Tone       string `json:"tone" yaml:"tone"`             // formal | professional | conversational | casual
	Initiative string `json:"initiative" yaml:"initiative"` // reactive | responsive | proactive | autonomous

	// Validation preferences
	Validation ValidationConfig `json:"validation" yaml:"validation"`

	// Error handling configuration
	ErrorHandling ErrorHandlingConfig `json:"error_handling" yaml:"error_handling"`

	// Collaboration settings
	Collaboration CollaborationConfig `json:"collaboration" yaml:"collaboration"`

	// Reflection configuration
	Reflection ReflectionConfig `json:"reflection" yaml:"reflection"`
}

// ValidationConfig defines quality assurance preferences
type ValidationConfig struct {
	Strategy     string `json:"strategy" yaml:"strategy"`           // none | spot-check | thorough | exhaustive
	RequireTests bool   `json:"require_tests" yaml:"require_tests"` // default: true
	RequireDocs  bool   `json:"require_docs" yaml:"require_docs"`   // default: false
}

// ErrorHandlingConfig defines error response strategy
type ErrorHandlingConfig struct {
	Strategy   string `json:"strategy" yaml:"strategy"`       // fail-fast | log-and-continue | retry | escalate
	RetryCount int    `json:"retry_count" yaml:"retry_count"` // 0-10, default: 3
}

// CollaborationConfig defines multi-agent/human interaction
type CollaborationConfig struct {
	Style            string `json:"style" yaml:"style"`                           // independent | consultative | collaborative | delegating
	CheckInFrequency string `json:"check_in_frequency" yaml:"check_in_frequency"` // never | rarely | regularly | constantly
}

// ReflectionConfig defines introspection behavior
type ReflectionConfig struct {
	Frequency string   `json:"frequency" yaml:"frequency"` // never | session-end | daily | weekly
	Format    string   `json:"format" yaml:"format"`       // structured | narrative | bullets
	Triggers  []string `json:"triggers" yaml:"triggers"`   // e.g., ["error", "milestone", "learning"]
}
```

---

## Primary Interface

### ResolveDisposition

**Function Signature:**

```go
func ResolveDisposition(workspaceRoot string, activeMode string) (*DispositionConfig, error)
```

**Contract:**

**Input:**
- `workspaceRoot` (string): Absolute path to workspace root directory
- `activeMode` (string): Mode name to activate (empty string = use file's mode field)

**Output:**
- `*DispositionConfig`: Resolved disposition configuration
- `error`: Validation or parsing errors

**Guarantees:**

1. **File Resolution Order:**
   - First checks for full identity directory: `<workspaceRoot>/identity.yaml`, `<workspaceRoot>/disposition.yaml`, `<workspaceRoot>/modes/`
   - If not found, checks for bridge file: `<workspaceRoot>/agent.yaml`
   - If neither found, returns error OR default disposition (implementation choice)

2. **Validation:**
   - File size must be < 1 MB
   - File encoding must be UTF-8
   - `schema_version` must be valid semver
   - All required fields must be present (pacing, depth, tone, initiative)
   - All enum fields must use schema-defined values
   - `error_handling.retry_count` must be 0-10

3. **Mode Resolution:**
   - If `activeMode` parameter provided, uses that mode
   - Else if file has `mode` field, uses that mode
   - Else defaults to "action" mode
   - If mode not found in `modes` map, logs warning and uses base disposition

4. **Mode Merging:**
   - Starts with base `disposition` from file
   - Applies mode overrides (non-nil pointer fields replace base)
   - Returns merged result

5. **Error Handling:**
   - File not found → error (or default, implementation-defined)
   - Parse error → error with line number
   - Validation error → error with field path and constraint
   - Unknown mode → warning logged, uses base disposition

**Performance:**
- MUST complete in < 100ms for typical files (< 100 KB)
- MAY cache parsed results (cache invalidation is implementation-defined)

---

## Gateway Integration Requirements

The gateway MUST use `DispositionConfig` to configure these modules:

### 1. Orchestration Engine

**Field:** `disposition.Pacing`

**Required Behavior:**

| Pacing Value | Gateway MUST |
|--------------|--------------|
| `deliberate` | Add 2-5 second delays between tool calls |
| `measured` | Use standard timing (1-2 second delays) |
| `responsive` | Reduce delays to 0.5-1 second |
| `rapid` | Minimize delays, execute tools in parallel where possible |

**Module:** `orchestration/engine.go`

### 2. Memory Manager

**Field:** `disposition.Depth`

**Required Behavior:**

| Depth Value | Gateway MUST |
|-------------|--------------|
| `surface` | Keep only decisions and final outputs |
| `functional` | Keep decisions, actions, key observations |
| `thorough` | Keep decisions, actions, observations, alternatives |
| `exhaustive` | Keep full conversation history |

**Module:** `memory/compression.go`

### 3. Proactive Intelligence

**Field:** `disposition.Initiative`

**Required Behavior:**

| Initiative Value | Gateway MUST |
|------------------|--------------|
| `reactive` | Wait for explicit commands, no suggestions |
| `responsive` | Answer questions, suggest when asked |
| `proactive` | Suggest next steps automatically |
| `autonomous` | Execute anticipated tasks without approval |

**Module:** `intelligence/proactive.go`

### 4. Error Handler

**Fields:** `disposition.ErrorHandling.Strategy`, `disposition.ErrorHandling.RetryCount`

**Required Behavior:**

| Strategy Value | Gateway MUST |
|----------------|--------------|
| `fail-fast` | Stop on first error, return to user |
| `log-and-continue` | Log error, continue with remaining tasks |
| `retry` | Retry N times (from RetryCount), then fail |
| `escalate` | Ask user for guidance on error |

**Module:** `errors/handler.go`

### 5. Collaboration Manager

**Fields:** `disposition.Collaboration.Style`, `disposition.Collaboration.CheckInFrequency`

**Required Behavior:**

| Style Value | Gateway MUST |
|-------------|--------------|
| `independent` | Complete tasks without check-ins |
| `consultative` | Check in at decision points |
| `collaborative` | Frequent dialogue with user |
| `delegating` | Coordinate with other agents |

**Check-In Frequency:**

| Frequency Value | Gateway MUST |
|-----------------|--------------|
| `never` | No automatic check-ins |
| `rarely` | Check in at major milestones |
| `regularly` | Check in every 5-10 actions |
| `constantly` | Check in before significant actions |

**Module:** `collaboration/manager.go`

### 6. Validator

**Fields:** `disposition.Validation.Strategy`, `disposition.Validation.RequireTests`, `disposition.Validation.RequireDocs`

**Required Behavior:**

| Strategy Value | Gateway MUST |
|----------------|--------------|
| `none` | Skip validation |
| `spot-check` | Quick syntax validation, sample tests |
| `thorough` | Full test suite, linting, type checking |
| `exhaustive` | Tests + coverage + security scanning |

**Flags:**
- `RequireTests: true` → Fail validation if tests missing/failing
- `RequireDocs: false` → Warn if docs missing, don't fail

**Module:** `validation/validator.go`

### 7. Reflection Engine

**Fields:** `disposition.Reflection.Frequency`, `disposition.Reflection.Format`, `disposition.Reflection.Triggers`

**Required Behavior:**

| Frequency Value | Gateway MUST |
|-----------------|--------------|
| `never` | Disable automatic reflection |
| `session-end` | Trigger reflection at session completion |
| `daily` | Trigger reflection at end of day |
| `weekly` | Trigger reflection at end of week |

**Format:**
- `structured` → YAML template with sections
- `narrative` → Markdown freeform
- `bullets` → Concise bullet points

**Triggers:** Array of event types triggering reflection (implementation-defined event matching)

**Module:** `reflection/engine.go`

---

## Validation Contract

### Required Validations

The gateway MUST validate:

1. **File existence** – fail if neither full identity nor bridge file found (or use defaults)
2. **File size** – fail if > 1 MB
3. **Encoding** – fail if not UTF-8
4. **Schema version** – fail if unsupported version
5. **Required fields** – fail if pacing/depth/tone/initiative missing
6. **Enum values** – fail if invalid enum value
7. **Integer ranges** – fail if retry_count outside 0-10

### Error Reporting

Errors MUST include:
- Field path (e.g., `disposition.error_handling.retry_count`)
- Invalid value (if applicable)
- Valid constraints or options
- Line number (if parse error)

**Example:**
```
Invalid enum value at disposition.pacing: "lightning"
  Valid values: deliberate, measured, responsive, rapid
  File: /workspace/agent.yaml, line 12
```

---

## Immutability Guarantee

The gateway MUST treat `DispositionConfig` as **immutable** after agent initialization:

- Disposition is loaded once at agent creation
- Changes to agent.yaml do NOT affect running agent
- Agent restart required to pick up changes

**Rationale:** Prevents mid-session behavioral inconsistency and simplifies reasoning about agent behavior.

---

## Backward Compatibility

The gateway MUST support:

- **v1.0.0 files** (current)
- **v0.1.0 files** (deprecated, 4-month EOL period)

Version detection via `schema_version` field.

**Forward Compatibility:**

The gateway MUST ignore unknown fields (enables future schema additions without breaking older runtimes).

---

## Testing Requirements

Implementations MUST pass these tests:

1. **Valid file parsing** – Parse all example files from `AgentIdentitiesByDojoGenesis/schema/examples/`
2. **Mode merging** – Correctly merge base + mode overrides
3. **Validation rejection** – Reject files with invalid enum values
4. **Size limit enforcement** – Reject files > 1 MB
5. **Encoding validation** – Reject non-UTF-8 files
6. **Error messages** – Produce correctly formatted error messages
7. **Module integration** – Verify all 7 modules consume disposition fields

---

## Contract Versioning

**Contract Version:** 1.0.0

**Change Policy:**
- **Patch (1.0.x):** Documentation clarifications, no behavior changes
- **Minor (1.x.0):** Add optional fields or features, backward compatible
- **Major (x.0.0):** Breaking changes to interface or guarantees

**Deprecation Policy:**
- Features marked deprecated in minor version
- Deprecated features removed in next major version
- Minimum 6-month deprecation period

---

## References

**Primary Specification:**
- `AgentIdentitiesByDojoGenesis/docs/agent-yaml-bridge-spec.md` v1.0.0

**Schema Files:**
- `AgentIdentitiesByDojoGenesis/schema/disposition.schema.yaml`
- `AgentIdentitiesByDojoGenesis/schema/mode.schema.yaml`
- `AgentIdentitiesByDojoGenesis/schema/identity.schema.yaml`
- `AgentIdentitiesByDojoGenesis/schema/lineage.schema.yaml`

**Pattern Documentation:**
- `AgentIdentitiesByDojoGenesis/patterns/02-disposition.md`

---

## Change Log

### 1.0.0 (2026-02-12) - Initial Contract

- Established formal contract between Gateway and ADA
- Defined DispositionConfig type system
- Specified ResolveDisposition interface
- Defined module integration requirements for 7 gateway modules
- Established validation contract
- Defined immutability guarantee
- Specified backward compatibility requirements
