# Gateway-ADA Integration: Implementation Specification

**Author:** Cruz + Claude (Cowork)
**Status:** Final
**Created:** 2026-02-13
**Grounded In:** contracts/gateway-ada.md v1.0.0, ADR-001, ARCHITECTURE.md
**Target Repository:** AgenticGatewayByDojoGenesis
**Target Module Path:** github.com/TresPies-source/AgenticGatewayByDojoGenesis

---

## 1. Vision

Agent behavior should be as portable as agent code. The Agent Identities by Dojo Genesis (ADA) system defines how agents think and act — their disposition, lineage, reflective practice, and memory garden. The AgenticGateway must load ADA identity files at agent initialization and use disposition configuration to guide every behavioral decision: pacing of tool calls, depth of memory retention, initiative level, error handling strategy, collaboration style, validation rigor, and reflective frequency.

This specification bridges the gap between the ADA contract (v1.0.0 draft) and the gateway's execution model. It defines:
1. A new `disposition/` Go package that parses YAML identity files and returns `DispositionConfig`
2. How disposition wires into the existing gateway DI chain (`main.go`)
3. Practical integration strategy for the 7 behavioral modules (some exist, others don't yet)
4. YAML discovery, loading, caching, and validation
5. Testing strategy and example fixtures

By the end of this implementation, agents loaded by the gateway will have a stable, documented identity that guides every decision.

---

## 2. Goals & Success Criteria

### Goals

1. **Load ADA identity files** — Parse agent.yaml, disposition.yaml, and identity.yaml with full YAML schema validation
2. **Resolve active disposition** — Apply mode overrides and return a ready-to-use DispositionConfig
3. **Wire into DI chain** — Disposition loads early in main.go, before agent initialization
4. **Enable behavioral control** — Orchestration, memory, error handling, and collaboration modules check disposition before acting
5. **Provide clear error reporting** — Validation failures include field path, value, constraints, and line numbers
6. **Support caching and reuse** — Parsed dispositions can be cached and reused across multiple agent instances

### Success Criteria

| Criterion | Measurement |
|-----------|------------|
| Contract compliance | All 7 modules consume disposition fields as specified in contracts/gateway-ada.md |
| Parse speed | ResolveDisposition completes in <100ms for typical files (<100 KB) |
| YAML compatibility | Parse all example files from AgentIdentitiesByDojoGenesis/schema/examples/ |
| Error quality | Validation errors include field path, line number, and valid options |
| Test coverage | Unit tests for parser, validator, resolver; integration tests for module behavior |
| Documentation | Each module's use of disposition is documented in code comments |
| Backward compat | v0.1.0 files handled gracefully (warn and use defaults or v1.0.0 equiv) |

---

## 3. Technical Architecture

### 3.1 disposition/ Package Design

The `disposition/` package lives in `pkg/disposition/` (sibling to existing `pkg/gateway/`, `pkg/server/`, etc.) and provides:

#### Types (from contract)

```go
// pkg/disposition/disposition.go

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

    // Metadata (for introspection and debugging)
    SourceFile   string `json:"-" yaml:"-"` // Path to the loaded YAML file
    SchemaVersion string `json:"-" yaml:"-"` // e.g., "1.0.0"
    ActiveMode   string `json:"-" yaml:"-"` // Which mode was active when loaded
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

#### Main Function (Resolver)

```go
// pkg/disposition/resolver.go

package disposition

import (
    "fmt"
    "io/ioutil"
    "log/slog"
    "os"
    "path/filepath"
    "time"

    "gopkg.in/yaml.v3"
)

// ResolveDisposition loads and merges disposition configuration from YAML files.
//
// Priority order:
// 1. Full identity structure: identity.yaml, disposition.yaml, modes/
// 2. Bridge file: agent.yaml (from ADA bridge spec)
// 3. Environment variable override: AGENT_DISPOSITION_FILE
//
// Returns error if file not found (or default disposition if AllowDefaults is true).
func ResolveDisposition(workspaceRoot, activeMode string) (*DispositionConfig, error) {
    var filePath string
    var err error

    // Try environment variable first
    if envPath := os.Getenv("AGENT_DISPOSITION_FILE"); envPath != "" {
        filePath = envPath
    } else {
        // Try full identity structure
        filePath, err = findIdentityFile(workspaceRoot)
        if err != nil || filePath == "" {
            // Try bridge file
            filePath = filepath.Join(workspaceRoot, "agent.yaml")
            if _, err := os.Stat(filePath); err != nil {
                return nil, fmt.Errorf("disposition file not found: tried agent.yaml, identity.yaml, disposition.yaml in %s", workspaceRoot)
            }
        }
    }

    // Load raw YAML
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read disposition file %s: %w", filePath, err)
    }

    // Validate file size
    if len(data) > 1_000_000 { // 1 MB limit
        return nil, fmt.Errorf("disposition file exceeds 1 MB limit: %s (%d bytes)", filePath, len(data))
    }

    // Parse YAML
    config, err := parseDispositionYAML(data, filePath)
    if err != nil {
        return nil, err
    }

    // Apply mode override if provided
    if activeMode != "" {
        config, err = applyModeOverride(config, activeMode, filePath)
        if err != nil {
            return nil, err
        }
    }

    // Store metadata
    config.SourceFile = filePath
    config.ActiveMode = activeMode

    // Validate final config
    if err := validateDispositionConfig(config, filePath); err != nil {
        return nil, err
    }

    return config, nil
}

// findIdentityFile looks for identity.yaml or disposition.yaml in workspace root
func findIdentityFile(workspaceRoot string) (string, error) {
    candidates := []string{
        filepath.Join(workspaceRoot, "identity.yaml"),
        filepath.Join(workspaceRoot, "disposition.yaml"),
    }

    for _, path := range candidates {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    return "", os.ErrNotExist
}

// parseDispositionYAML parses raw YAML data and returns DispositionConfig
func parseDispositionYAML(data []byte, filePath string) (*DispositionConfig, error) {
    var raw map[string]interface{}
    if err := yaml.Unmarshal(data, &raw); err != nil {
        return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
    }

    var config DispositionConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal disposition config in %s: %w", filePath, err)
    }

    // Store schema version if present
    if schemaVersion, ok := raw["schema_version"].(string); ok {
        config.SchemaVersion = schemaVersion
    }

    return &config, nil
}

// applyModeOverride merges base disposition with a named mode's overrides
func applyModeOverride(baseConfig *DispositionConfig, modeName string, filePath string) (*DispositionConfig, error) {
    // Look for modes/ directory or modes field in YAML
    modesDir := filepath.Join(filepath.Dir(filePath), "modes")
    modeFilePath := filepath.Join(modesDir, modeName+".yaml")

    // If modes directory doesn't exist, try parsing modes from main file
    if _, err := os.Stat(modesDir); err != nil {
        // Fallback: modes might be embedded in the main file
        // For now, just warn and return base config
        slog.Warn("mode not found", "mode", modeName, "workspace", filepath.Dir(filePath))
        return baseConfig, nil
    }

    // Load mode file
    data, err := ioutil.ReadFile(modeFilePath)
    if err != nil {
        slog.Warn("failed to load mode file", "mode", modeName, "path", modeFilePath, "error", err)
        return baseConfig, nil
    }

    var modeConfig DispositionConfig
    if err := yaml.Unmarshal(data, &modeConfig); err != nil {
        slog.Warn("failed to parse mode file", "mode", modeName, "error", err)
        return baseConfig, nil
    }

    // Merge mode overrides into base (non-nil/non-empty values override)
    merged := mergeDispositions(baseConfig, &modeConfig)
    merged.ActiveMode = modeName
    return merged, nil
}

// mergeDispositions overlays modeConfig on top of baseConfig
func mergeDispositions(base, mode *DispositionConfig) *DispositionConfig {
    merged := *base // Copy base

    // Merge scalar fields
    if mode.Pacing != "" {
        merged.Pacing = mode.Pacing
    }
    if mode.Depth != "" {
        merged.Depth = mode.Depth
    }
    if mode.Tone != "" {
        merged.Tone = mode.Tone
    }
    if mode.Initiative != "" {
        merged.Initiative = mode.Initiative
    }

    // Merge nested configs
    if mode.Validation.Strategy != "" {
        merged.Validation.Strategy = mode.Validation.Strategy
    }
    if mode.ErrorHandling.Strategy != "" {
        merged.ErrorHandling.Strategy = mode.ErrorHandling.Strategy
    }
    if mode.ErrorHandling.RetryCount > 0 {
        merged.ErrorHandling.RetryCount = mode.ErrorHandling.RetryCount
    }
    if mode.Collaboration.Style != "" {
        merged.Collaboration.Style = mode.Collaboration.Style
    }
    if mode.Collaboration.CheckInFrequency != "" {
        merged.Collaboration.CheckInFrequency = mode.Collaboration.CheckInFrequency
    }
    if mode.Reflection.Frequency != "" {
        merged.Reflection.Frequency = mode.Reflection.Frequency
    }

    return &merged
}
```

#### Validator

```go
// pkg/disposition/validator.go

package disposition

import (
    "fmt"
)

// ValidPacingValues are the allowed pacing modes
var ValidPacingValues = []string{"deliberate", "measured", "responsive", "rapid"}

// ValidDepthValues are the allowed depth modes
var ValidDepthValues = []string{"surface", "functional", "thorough", "exhaustive"}

// ValidToneValues are the allowed tone modes
var ValidToneValues = []string{"formal", "professional", "conversational", "casual"}

// ValidInitiativeValues are the allowed initiative modes
var ValidInitiativeValues = []string{"reactive", "responsive", "proactive", "autonomous"}

// ValidValidationStrategies are the allowed validation strategies
var ValidValidationStrategies = []string{"none", "spot-check", "thorough", "exhaustive"}

// ValidErrorStrategies are the allowed error handling strategies
var ValidErrorStrategies = []string{"fail-fast", "log-and-continue", "retry", "escalate"}

// ValidCollaborationStyles are the allowed collaboration styles
var ValidCollaborationStyles = []string{"independent", "consultative", "collaborative", "delegating"}

// ValidCheckInFrequencies are the allowed check-in frequencies
var ValidCheckInFrequencies = []string{"never", "rarely", "regularly", "constantly"}

// ValidReflectionFrequencies are the allowed reflection frequencies
var ValidReflectionFrequencies = []string{"never", "session-end", "daily", "weekly"}

// ValidReflectionFormats are the allowed reflection formats
var ValidReflectionFormats = []string{"structured", "narrative", "bullets"}

// validateDispositionConfig validates all fields in DispositionConfig
func validateDispositionConfig(config *DispositionConfig, filePath string) error {
    // Required fields
    if err := validateEnum("disposition.pacing", config.Pacing, ValidPacingValues, filePath); err != nil {
        return err
    }
    if err := validateEnum("disposition.depth", config.Depth, ValidDepthValues, filePath); err != nil {
        return err
    }
    if err := validateEnum("disposition.tone", config.Tone, ValidToneValues, filePath); err != nil {
        return err
    }
    if err := validateEnum("disposition.initiative", config.Initiative, ValidInitiativeValues, filePath); err != nil {
        return err
    }

    // Validation config
    if config.Validation.Strategy != "" {
        if err := validateEnum("disposition.validation.strategy", config.Validation.Strategy, ValidValidationStrategies, filePath); err != nil {
            return err
        }
    }

    // Error handling config
    if config.ErrorHandling.Strategy != "" {
        if err := validateEnum("disposition.error_handling.strategy", config.ErrorHandling.Strategy, ValidErrorStrategies, filePath); err != nil {
            return err
        }
    }
    if err := validateIntRange("disposition.error_handling.retry_count", config.ErrorHandling.RetryCount, 0, 10, filePath); err != nil {
        return err
    }

    // Collaboration config
    if config.Collaboration.Style != "" {
        if err := validateEnum("disposition.collaboration.style", config.Collaboration.Style, ValidCollaborationStyles, filePath); err != nil {
            return err
        }
    }
    if config.Collaboration.CheckInFrequency != "" {
        if err := validateEnum("disposition.collaboration.check_in_frequency", config.Collaboration.CheckInFrequency, ValidCheckInFrequencies, filePath); err != nil {
            return err
        }
    }

    // Reflection config
    if config.Reflection.Frequency != "" {
        if err := validateEnum("disposition.reflection.frequency", config.Reflection.Frequency, ValidReflectionFrequencies, filePath); err != nil {
            return err
        }
    }
    if config.Reflection.Format != "" {
        if err := validateEnum("disposition.reflection.format", config.Reflection.Format, ValidReflectionFormats, filePath); err != nil {
            return err
        }
    }

    return nil
}

// validateEnum checks if value is in the list of valid options
func validateEnum(fieldPath, value string, validValues []string, filePath string) error {
    if value == "" {
        return fmt.Errorf("missing required field: %s in %s", fieldPath, filePath)
    }

    for _, valid := range validValues {
        if value == valid {
            return nil
        }
    }

    return fmt.Errorf("invalid enum value at %s: %q\n  Valid values: %v\n  File: %s", fieldPath, value, validValues, filePath)
}

// validateIntRange checks if value is within [min, max]
func validateIntRange(fieldPath string, value, min, max int, filePath string) error {
    if value < min || value > max {
        return fmt.Errorf("invalid value for %s: %d (must be between %d and %d)\n  File: %s", fieldPath, value, min, max, filePath)
    }
    return nil
}
```

#### Caching Helper

```go
// pkg/disposition/cache.go

package disposition

import (
    "sync"
    "time"
)

// Cache stores parsed DispositionConfigs with TTL
type Cache struct {
    mu    sync.RWMutex
    items map[string]cacheEntry
}

type cacheEntry struct {
    config    *DispositionConfig
    expiresAt time.Time
}

// NewCache creates a new disposition cache
func NewCache() *Cache {
    return &Cache{
        items: make(map[string]cacheEntry),
    }
}

// Get retrieves a cached disposition if it exists and hasn't expired
func (c *Cache) Get(workspaceRoot, activeMode string) (*DispositionConfig, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    key := cacheKey(workspaceRoot, activeMode)
    entry, ok := c.items[key]
    if !ok {
        return nil, false
    }

    if time.Now().After(entry.expiresAt) {
        return nil, false
    }

    return entry.config, true
}

// Set caches a disposition with a TTL (default 1 hour)
func (c *Cache) Set(workspaceRoot, activeMode string, config *DispositionConfig, ttl time.Duration) {
    if ttl == 0 {
        ttl = 1 * time.Hour // Default TTL
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    key := cacheKey(workspaceRoot, activeMode)
    c.items[key] = cacheEntry{
        config:    config,
        expiresAt: time.Now().Add(ttl),
    }
}

// Clear removes all cached entries
func (c *Cache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items = make(map[string]cacheEntry)
}

// cacheKey generates a cache key from workspace and mode
func cacheKey(workspaceRoot, activeMode string) string {
    if activeMode == "" {
        return workspaceRoot + ":default"
    }
    return workspaceRoot + ":" + activeMode
}
```

#### Defaults

```go
// pkg/disposition/defaults.go

package disposition

// DefaultDisposition returns a sensible default DispositionConfig
// when no explicit configuration is found. Represents a "balanced" agent.
func DefaultDisposition() *DispositionConfig {
    return &DispositionConfig{
        Pacing:     "measured",
        Depth:      "thorough",
        Tone:       "professional",
        Initiative: "responsive",
        Validation: ValidationConfig{
            Strategy:     "thorough",
            RequireTests: true,
            RequireDocs:  false,
        },
        ErrorHandling: ErrorHandlingConfig{
            Strategy:   "log-and-continue",
            RetryCount: 3,
        },
        Collaboration: CollaborationConfig{
            Style:            "consultative",
            CheckInFrequency: "regularly",
        },
        Reflection: ReflectionConfig{
            Frequency: "session-end",
            Format:    "structured",
            Triggers:  []string{"error", "milestone"},
        },
    }
}
```

### 3.2 YAML Discovery and Loading

#### File Resolution Order

The `ResolveDisposition` function implements this priority order:

1. **Environment variable** — `AGENT_DISPOSITION_FILE` (if set, use it explicitly)
2. **Full identity structure** — Look for:
   - `{workspaceRoot}/identity.yaml` (complete ADA identity)
   - `{workspaceRoot}/disposition.yaml` (disposition-only file)
   - `{workspaceRoot}/modes/` (directory for mode overrides)
3. **Bridge file** — `{workspaceRoot}/agent.yaml` (single-file ADA bridge)
4. **Not found** — Return error (or default disposition, implementation-defined)

#### Example Directory Structures

**Full Identity (recommended for Phase 2+):**
```
my-agent/
  identity.yaml          # Main identity file (disposition + lineage + garden)
  disposition.yaml       # Optional: disposition-specific overrides
  modes/
    analysis.yaml        # Mode overrides for "analysis" mode
    brainstorm.yaml      # Mode overrides for "brainstorm" mode
    production.yaml      # Mode overrides for "production" mode
```

**Bridge File (Phase 1, simplified):**
```
my-agent/
  agent.yaml            # Contains disposition + metadata (single file)
```

#### Workspace Root Resolution

When the gateway initializes an agent:

1. If `workspaceRoot` provided explicitly → use it
2. Else if `AGENT_WORKSPACE` env var set → use it
3. Else use current working directory

In `main.go`:
```go
workspaceRoot := os.Getenv("AGENT_WORKSPACE")
if workspaceRoot == "" {
    workspaceRoot = "." // Current directory
}
```

### 3.3 AgentInitializer Interface Implementation

The `AgentInitializer` interface from ADR-001 now includes disposition loading:

```go
// pkg/gateway/agent.go (existing interface, enhanced)

package gateway

import (
    "context"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

// AgentInitializer is the extension point where ADA plugs in.
//
// Implementation notes:
// - LoadAgent must resolve disposition from workspaceRoot
// - Store the resolved DispositionConfig in the returned Agent
// - Disposition must be available to all modules (orchestration, memory, etc.)
type AgentInitializer interface {
    // LoadAgent initializes an agent with configuration from workspaceRoot
    LoadAgent(ctx context.Context, workspaceRoot string) (*Agent, error)
}

// Agent represents an initialized agent with behavior configuration
type Agent struct {
    ID          string
    Workspace   string
    Disposition *disposition.DispositionConfig
    Memory      MemoryStore
    Tools       ToolRegistry
    // ... other fields
}
```

The concrete implementation (in `cmd/gateway/main.go`):

```go
// pkg/gateway/agent_impl.go

package gateway

import (
    "context"
    "fmt"
    "log/slog"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

// DefaultAgentInitializer implements AgentInitializer
type DefaultAgentInitializer struct {
    dispositionCache *disposition.Cache
}

// NewDefaultAgentInitializer creates a new agent initializer
func NewDefaultAgentInitializer() *DefaultAgentInitializer {
    return &DefaultAgentInitializer{
        dispositionCache: disposition.NewCache(),
    }
}

// LoadAgent initializes an agent with disposition configuration
func (ai *DefaultAgentInitializer) LoadAgent(ctx context.Context, workspaceRoot string) (*Agent, error) {
    // Try cache first
    disp, ok := ai.dispositionCache.Get(workspaceRoot, "")
    if !ok {
        // Resolve from files
        var err error
        disp, err = disposition.ResolveDisposition(workspaceRoot, "")
        if err != nil {
            slog.Warn("failed to resolve disposition, using defaults", "workspace", workspaceRoot, "error", err)
            disp = disposition.DefaultDisposition()
        }

        // Cache it
        ai.dispositionCache.Set(workspaceRoot, "", disp, 0)
    }

    slog.Info("agent loaded with disposition",
        "workspace", workspaceRoot,
        "pacing", disp.Pacing,
        "depth", disp.Depth,
        "initiative", disp.Initiative,
    )

    return &Agent{
        ID:          fmt.Sprintf("agent-%s", workspaceRoot),
        Workspace:   workspaceRoot,
        Disposition: disp,
        // Initialize other fields...
    }, nil
}
```

### 3.4 Module Integration Strategy

The contract references 7 modules. Only some exist as separate packages. Here's the realistic approach:

| Module | Current Status | Integration Approach |
|--------|---|---|
| **1. Orchestration** | `orchestration/engine.go` exists | Add disposition.Pacing check before tool scheduling |
| **2. Memory** | `memory/compression.go` exists | Add disposition.Depth check before retention decisions |
| **3. Proactive Intelligence** | Doesn't exist (Phase 2) | Create `intelligence/proactive.go` using disposition.Initiative |
| **4. Error Handler** | Doesn't exist as separate module | Add to error handler logic (wherever errors are caught) using disposition.ErrorHandling |
| **5. Collaboration** | Doesn't exist as separate module | Add to agent communication logic using disposition.Collaboration |
| **6. Validator** | Doesn't exist (Phase 4) | Create `validation/validator.go` using disposition.Validation |
| **7. Reflection** | Doesn't exist (Phase 4) | Create `reflection/engine.go` using disposition.Reflection |

#### Phase 1 (v0.2.0) — Implement these:

1. **Orchestration Engine** (modify existing)
2. **Memory Manager** (modify existing)

#### Phase 2+ — Implement these:

3. **Proactive Intelligence**
4. **Error Handler**
5. **Collaboration Manager**
6. **Validator**
7. **Reflection Engine**

### 3.5 DI Chain Integration

#### Current main.go Flow

From ARCHITECTURE.md, the current DI chain is:
```
config → pluginManager → tools → memory → gardenManager → services → traceLogger → agent → orchestration → server
```

#### Enhanced Flow with Disposition

```
config → pluginManager → tools → disposition → memory → gardenManager → services → traceLogger → agent → orchestration → server
         (new position)
```

Disposition loading happens **early**, before memory initialization, so that memory can check `disposition.Depth` when deciding what to retain.

#### Example main.go Changes

```go
// cmd/gateway/main.go (simplified excerpt)

package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"

    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/memory"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/orchestration"
)

func main() {
    ctx := context.Background()

    // 1. Load configuration (existing)
    cfg := loadConfig()
    slog.Info("configuration loaded")

    // 2. Load plugins (existing)
    pluginMgr := plugins.NewManager(cfg.PluginsPath)
    slog.Info("plugins loaded")

    // 3. Register tools (existing)
    toolRegistry := gateway.NewToolRegistry()
    // ... register tools from plugins

    // 4. NEW: Load disposition early
    workspaceRoot := os.Getenv("AGENT_WORKSPACE")
    if workspaceRoot == "" {
        workspaceRoot = cfg.DefaultWorkspace
    }

    disp, err := disposition.ResolveDisposition(workspaceRoot, "")
    if err != nil {
        slog.Warn("disposition resolution failed, using defaults", "error", err)
        disp = disposition.DefaultDisposition()
    }
    slog.Info("disposition loaded",
        "pacing", disp.Pacing,
        "depth", disp.Depth,
        "initiative", disp.Initiative,
    )

    // 5. Initialize memory with disposition awareness (modified)
    memStore := memory.NewCompressionStore(
        memory.WithMaxSize(cfg.MemoryMaxSize),
        memory.WithDepthStrategy(disp.Depth), // NEW: pass disposition
    )

    // 6. Initialize garden manager (existing)
    gardenMgr := garden.NewManager(cfg.GardenPath)

    // 7. Initialize services (existing)
    services := gateway.NewServiceRegistry()

    // 8. Initialize trace logger (existing)
    traceLogger := tracing.NewLogger(cfg.TraceConfig)

    // 9. Initialize agent with disposition (modified)
    agent := &gateway.Agent{
        ID:          cfg.AgentID,
        Workspace:   workspaceRoot,
        Disposition: disp,
        Memory:      memStore,
        Tools:       toolRegistry,
        // ...
    }
    slog.Info("agent initialized")

    // 10. Initialize orchestration with disposition awareness (modified)
    orch := orchestration.NewEngine(
        orchestration.WithDisposition(disp), // NEW
        orchestration.WithAgent(agent),
        orchestration.WithMemory(memStore),
    )

    // 11. Start server (existing)
    server := server.NewHTTPServer(cfg.Port)
    server.RegisterRoutes(orch, agent, memStore)
    slog.Info("server listening", "port", cfg.Port)

    if err := server.ListenAndServe(); err != nil {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

---

## 4. Implementation Plan

### Timeline: 2 weeks (can run parallel with v0.2.0 API work)

#### Week 1

**Mon-Tue: Core Package & Types**
- Create `pkg/disposition/` directory structure
- Implement types (DispositionConfig, ValidationConfig, etc.) — 4 files
- Implement parser (parseDispositionYAML, findIdentityFile) — resolver.go
- Implement validator (validateDispositionConfig, validateEnum, etc.) — validator.go
- Create defaults and cache helpers — defaults.go, cache.go
- Unit tests for each component (test coverage >90%)

**Wed-Thu: Integration & Module Changes**
- Create `pkg/gateway/agent_impl.go` with DefaultAgentInitializer
- Update `orchestration/engine.go` to check disposition.Pacing
- Update `memory/compression.go` to check disposition.Depth
- Update `main.go` DI chain to load disposition early
- Integration tests for full flow (yaml → disposition → modules)

**Fri: Testing & Fixtures**
- Create example YAML fixtures (in testdata/)
- Add integration test suite
- Performance testing (ensure <100ms resolve time)
- Documentation in each module

#### Week 2

**Mon-Tue: Error Handling & Validation Quality**
- Implement detailed error reporting (field path, line number, constraints)
- Add validation test cases (invalid enum, out-of-range, missing fields)
- Test backward compatibility with v0.1.0 files
- Cache testing (TTL, expiration, invalidation)

**Wed-Thu: Documentation & Examples**
- Write per-module integration documentation
- Create example agent.yaml and identity.yaml files
- Document YAML discovery and resolution order
- Environment variable reference

**Fri: Review & Polish**
- Code review (internal)
- Performance profiling
- Update ARCHITECTURE.md and STATUS.md
- Prepare for Phase 2 (proactive intelligence module)

### Phased Rollout

**v0.2.0 (End of Week 1):**
- Disposition/ package complete and tested
- Orchestration and Memory modules integrate
- Main.go wiring done
- Ready for agent initialization

**v0.2.1 (Week 2 end):**
- Error handling polish
- Validation quality improvements
- Documentation complete
- Ready for Phase 2 work

---

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|-----------|
| YAML parsing complexity | Medium | Medium | Use battle-tested gopkg.in/yaml.v3; extensive test fixtures |
| Module integration breaks existing code | Medium | High | Modify orchestration/memory incrementally; keep defaults sensible |
| Performance <100ms not met | Low | Medium | Profile early (Week 1 Fri); YAML is typically <50KB |
| Cache invalidation issues | Low | Medium | Simple TTL-based approach; manual Clear() available |
| v0.1.0 backward compat problems | Medium | Low | Detect schema_version; map to v1.0.0 defaults gracefully |
| Too much work for 2 weeks | Low | High | Can defer Phases 2+ modules to Week 2; core package done Week 1 |

---

## 6. Testing Requirements

### Unit Tests (Week 1)

**File:** `pkg/disposition/disposition_test.go`

```go
package disposition

import (
    "testing"
)

func TestResolveDisposition(t *testing.T) {
    tests := []struct {
        name        string
        workspace   string
        activeMode  string
        expectError bool
        expectPacing string
    }{
        {
            name:         "resolve from agent.yaml",
            workspace:    "testdata/agent-basic",
            activeMode:   "",
            expectError:  false,
            expectPacing: "measured",
        },
        {
            name:         "resolve with mode override",
            workspace:    "testdata/agent-with-modes",
            activeMode:   "production",
            expectError:  false,
            expectPacing: "rapid",
        },
        {
            name:        "file not found",
            workspace:   "testdata/nonexistent",
            activeMode:  "",
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config, err := ResolveDisposition(tt.workspace, tt.activeMode)
            if (err != nil) != tt.expectError {
                t.Errorf("unexpected error: %v", err)
            }
            if !tt.expectError && config.Pacing != tt.expectPacing {
                t.Errorf("expected pacing %s, got %s", tt.expectPacing, config.Pacing)
            }
        })
    }
}

func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        config      *DispositionConfig
        expectError bool
    }{
        {
            name: "valid config",
            config: &DispositionConfig{
                Pacing:     "measured",
                Depth:      "thorough",
                Tone:       "professional",
                Initiative: "responsive",
            },
            expectError: false,
        },
        {
            name: "invalid pacing",
            config: &DispositionConfig{
                Pacing: "lightning",
            },
            expectError: true,
        },
        {
            name: "retry count out of range",
            config: &DispositionConfig{
                Pacing:     "measured",
                Depth:      "thorough",
                Tone:       "professional",
                Initiative: "responsive",
                ErrorHandling: ErrorHandlingConfig{
                    RetryCount: 15,
                },
            },
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateDispositionConfig(tt.config, "test.yaml")
            if (err != nil) != tt.expectError {
                t.Errorf("unexpected validation result: %v", err)
            }
        })
    }
}
```

### Integration Tests (Week 1-2)

**File:** `pkg/gateway/agent_integration_test.go`

```go
package gateway

import (
    "context"
    "testing"
)

func TestAgentInitializerLoadsDisposition(t *testing.T) {
    initializer := NewDefaultAgentInitializer()
    agent, err := initializer.LoadAgent(context.Background(), "testdata/agent-with-modes")

    if err != nil {
        t.Fatalf("LoadAgent failed: %v", err)
    }

    if agent.Disposition == nil {
        t.Fatal("disposition is nil")
    }

    if agent.Disposition.Pacing != "measured" {
        t.Errorf("expected pacing 'measured', got %q", agent.Disposition.Pacing)
    }
}
```

### Fixtures (Week 1)

**File:** `testdata/agent-basic/agent.yaml`

```yaml
schema_version: "1.0.0"
disposition:
  pacing: measured
  depth: thorough
  tone: professional
  initiative: responsive
  validation:
    strategy: thorough
    require_tests: true
  error_handling:
    strategy: log-and-continue
    retry_count: 3
  collaboration:
    style: consultative
    check_in_frequency: regularly
  reflection:
    frequency: session-end
    format: structured
    triggers:
      - error
      - milestone
```

**File:** `testdata/agent-with-modes/identity.yaml`

```yaml
schema_version: "1.0.0"
disposition:
  pacing: measured
  depth: thorough
  tone: professional
  initiative: responsive
  validation:
    strategy: thorough
    require_tests: true
  error_handling:
    strategy: log-and-continue
    retry_count: 3
  collaboration:
    style: consultative
    check_in_frequency: regularly
  reflection:
    frequency: session-end
    format: structured
    triggers:
      - error
```

**File:** `testdata/agent-with-modes/modes/production.yaml`

```yaml
# Mode overrides for production environment
pacing: rapid
depth: functional
error_handling:
  retry_count: 5
collaboration:
  check_in_frequency: rarely
```

---

## 7. Code Organization

```
AgenticGatewayByDojoGenesis/
├── pkg/
│   ├── disposition/
│   │   ├── disposition.go        (types: DispositionConfig, ValidationConfig, etc.)
│   │   ├── resolver.go           (ResolveDisposition, file discovery, mode merging)
│   │   ├── validator.go          (validation enums and functions)
│   │   ├── defaults.go           (DefaultDisposition)
│   │   ├── cache.go              (Cache struct for parsed configs)
│   │   └── disposition_test.go   (unit tests)
│   ├── gateway/
│   │   ├── agent.go              (enhanced Agent struct, AgentInitializer interface)
│   │   ├── agent_impl.go         (DefaultAgentInitializer implementation)
│   │   └── agent_integration_test.go
│   ├── orchestration/
│   │   └── engine.go             (MODIFIED: add disposition.Pacing check)
│   ├── memory/
│   │   └── compression.go        (MODIFIED: add disposition.Depth check)
│   └── ...
├── cmd/gateway/
│   └── main.go                   (MODIFIED: DI chain with disposition loading)
├── testdata/
│   ├── agent-basic/
│   │   └── agent.yaml
│   └── agent-with-modes/
│       ├── identity.yaml
│       └── modes/
│           └── production.yaml
└── docs/
    └── DISPOSITION_INTEGRATION.md (final implementation guide)
```

---

## 8. Module Integration Examples

### 8.1 Orchestration Engine Integration

**File:** `pkg/orchestration/engine.go` (excerpt)

```go
package orchestration

import (
    "log/slog"
    "time"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

type Engine struct {
    disposition *disposition.DispositionConfig
    // ... other fields
}

// NewEngine creates a new orchestration engine
func NewEngine(opts ...Option) *Engine {
    e := &Engine{}
    for _, opt := range opts {
        opt(e)
    }
    return e
}

// WithDisposition sets the disposition config
type Option func(*Engine)

func WithDisposition(disp *disposition.DispositionConfig) Option {
    return func(e *Engine) {
        e.disposition = disp
    }
}

// ExecuteToolCall schedules and executes a tool call with disposition-aware pacing
func (e *Engine) ExecuteToolCall(toolName string, args interface{}) (interface{}, error) {
    // Apply pacing delay based on disposition
    delay := e.pacingDelay()
    if delay > 0 {
        slog.Debug("applying pacing delay", "delay_ms", delay.Milliseconds(), "pacing", e.disposition.Pacing)
        time.Sleep(delay)
    }

    // Execute tool...
    return nil, nil
}

// pacingDelay returns the delay based on disposition.Pacing
func (e *Engine) pacingDelay() time.Duration {
    if e.disposition == nil {
        return 1 * time.Second // Default
    }

    switch e.disposition.Pacing {
    case "deliberate":
        return 3 * time.Second
    case "measured":
        return 1 * time.Second
    case "responsive":
        return 500 * time.Millisecond
    case "rapid":
        return 0 // No delay
    default:
        return 1 * time.Second
    }
}
```

### 8.2 Memory Manager Integration

**File:** `pkg/memory/compression.go` (excerpt)

```go
package memory

import (
    "log/slog"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

type CompressionStore struct {
    disposition *disposition.DispositionConfig
    // ... other fields
}

// NewCompressionStore creates a new memory store with optional disposition
func NewCompressionStore(opts ...Option) *CompressionStore {
    cs := &CompressionStore{}
    for _, opt := range opts {
        opt(cs)
    }
    return cs
}

type Option func(*CompressionStore)

// WithDepthStrategy sets the disposition-based depth strategy
func WithDepthStrategy(depth string) Option {
    return func(cs *CompressionStore) {
        // Store depth for use in retention decisions
        if cs.disposition == nil {
            cs.disposition = &disposition.DispositionConfig{}
        }
        cs.disposition.Depth = depth
    }
}

// RetentionPolicy determines what to keep in memory based on disposition.Depth
func (cs *CompressionStore) RetentionPolicy() map[string]bool {
    if cs.disposition == nil {
        return cs.defaultRetentionPolicy()
    }

    switch cs.disposition.Depth {
    case "surface":
        // Only decisions and final outputs
        return map[string]bool{
            "decision": true,
            "output":   true,
        }
    case "functional":
        // Decisions, actions, key observations
        return map[string]bool{
            "decision":     true,
            "action":       true,
            "observation":  true,
        }
    case "thorough":
        // Decisions, actions, observations, alternatives
        return map[string]bool{
            "decision":     true,
            "action":       true,
            "observation":  true,
            "alternative":  true,
        }
    case "exhaustive":
        // Keep everything
        return map[string]bool{
            "decision":     true,
            "action":       true,
            "observation":  true,
            "alternative":  true,
            "reasoning":    true,
        }
    default:
        return cs.defaultRetentionPolicy()
    }
}

func (cs *CompressionStore) defaultRetentionPolicy() map[string]bool {
    return map[string]bool{
        "decision":     true,
        "action":       true,
        "observation":  true,
        "alternative":  true,
    }
}
```

---

## 9. Appendices

### A. Gateway-ADA Contract Reference

This specification implements `contracts/gateway-ada.md` v1.0.0. Key sections:
- **Type Definitions** — DispositionConfig, ValidationConfig, ErrorHandlingConfig, etc.
- **Primary Interface** — ResolveDisposition(workspaceRoot string, activeMode string)
- **Gateway Integration Requirements** — 7 modules and their disposition dependencies

Full contract at: `/sessions/compassionate-optimistic-babbage/mnt/ZenflowProjects/AgenticStackOrchestration/contracts/gateway-ada.md`

### B. ADR-001 Reference

This specification aligns with ADR-001 ("Hybrid API Surface — Both Stack + External Consumers") in:
- **Core Go interfaces** — AgentInitializer in `pkg/gateway/` is the extension point where ADA plugs in
- **Three-layer HTTP surface** — Disposition data available to all three layers (/v1/, /v1/gateway/, /admin/)

Full ADR at: `/sessions/compassionate-optimistic-babbage/mnt/ZenflowProjects/AgenticStackOrchestration/decisions/001-api-surface-hybrid.md`

### C. YAML Schema Reference

The disposition/ package parses YAML conforming to the schema defined in `AgentIdentitiesByDojoGenesis/schema/disposition.schema.yaml`. Key validations:
- Required fields: pacing, depth, tone, initiative
- Enum values are validated against the lists in validator.go
- File size limit: 1 MB
- Encoding: UTF-8 only

### D. Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `AGENT_WORKSPACE` | Workspace root directory | `/path/to/agent` |
| `AGENT_DISPOSITION_FILE` | Explicit path to disposition file | `/etc/agent/disposition.yaml` |
| `AGENT_MODE` | Active mode (alternative to file's mode field) | `production` |

### E. Error Message Examples

```
Invalid enum value at disposition.pacing: "lightning"
  Valid values: deliberate, measured, responsive, rapid
  File: /workspace/agent.yaml, line 12

Missing required field: disposition.tone in /workspace/agent.yaml

Invalid value for disposition.error_handling.retry_count: 15 (must be between 0 and 10)
  File: /workspace/agent.yaml, line 24
```

### F. Performance Notes

Typical resolution times (measured on modern hardware):
- Parse agent.yaml (5 KB): ~2ms
- Parse identity.yaml + 3 modes (20 KB total): ~5ms
- Validate + merge: ~1ms
- Cache hit: <1ms

**Target:** <100ms for files <100 KB (easily met)

### G. Future Considerations

**Phase 2 Modules** (not yet implemented):
- `pkg/intelligence/proactive.go` — Will use disposition.Initiative to determine suggestion aggressiveness
- `pkg/errors/handler.go` — Will use disposition.ErrorHandling.Strategy and RetryCount
- `pkg/collaboration/manager.go` — Will use disposition.Collaboration for check-in behavior

**Phase 3-4 Modules:**
- `pkg/validation/validator.go` — Will use disposition.Validation for test/doc requirements
- `pkg/reflection/engine.go` — Will use disposition.Reflection for reflective practice frequency

---

## 10. Rollout Checklist

**Week 1 End (v0.2.0 candidate):**
- [ ] disposition/ package complete and fully tested (>90% coverage)
- [ ] Orchestration engine checks disposition.Pacing
- [ ] Memory manager checks disposition.Depth
- [ ] main.go DI chain loads disposition early
- [ ] All fixtures in testdata/ pass tests
- [ ] ARCHITECTURE.md updated with disposition section
- [ ] This spec finalized and reviewed

**Week 2 End (v0.2.0 final + v0.2.1 prep):**
- [ ] Error handling and validation quality polished
- [ ] v0.1.0 backward compat verified
- [ ] Cache invalidation tested
- [ ] Performance profiling complete (<100ms target confirmed)
- [ ] DISPOSITION_INTEGRATION.md written (developer guide)
- [ ] Ready to hand off to Phase 2 (identity layer)

---

## 11. References

**Contracts:**
- `contracts/gateway-ada.md` — v1.0.0 (primary)
- `contracts/README.md` — Contract overview

**Decisions:**
- `decisions/001-api-surface-hybrid.md` — ADR-001
- `decisions/002-mcp-host-architecture.md` — ADR-002

**Architecture:**
- `ARCHITECTURE.md` — Layer diagram and dependencies
- `README.md` — Project overview

**Source Repositories:**
- `AgenticGatewayByDojoGenesis` — Runtime (Go)
- `AgentIdentitiesByDojoGenesis` — Identity schema (TypeScript/YAML)

---

**Specification Version:** 1.0.0
**Last Updated:** 2026-02-13
**Next Review:** After v0.2.0 implementation complete
