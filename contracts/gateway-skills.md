# Gateway-Skills Contract v1.0

**Version:** 1.0
**Last Updated:** 2026-02-13
**Status:** Phase 4a Implementation Complete

---

## Overview

This contract defines how skills are registered, discovered, invoked, and executed within the Agentic Gateway. Skills are portable, reusable workflow patterns that extend agent capabilities with specialized domain knowledge and tool integrations.

## Skill Lifecycle

### 1. Source
Skills are defined in `SKILL.md` files located at:
```
plugins/{plugin-name}/skills/{skill-name}/SKILL.md
```

### 2. Registration
The SkillRegistry discovers and registers skills via:
- **Filesystem Scan**: `LoadFromDirectory(ctx, dirPath, pluginName)`
  Recursively scans for SKILL.md files and parses YAML frontmatter
- **Validation**: Each skill is validated against the schema before registration
- **Deduplication**: Duplicate skill names are rejected

### 3. Tool Definition
Each registered skill becomes a tool accessible via the orchestration engine:
- **Tool Name**: `"invoke_skill"`
- **Parameters**: `{ "skill_name": string, "args": map[string]interface{} }`
- **Result**: `map[string]interface{}` (skill-specific output)

### 4. Invocation
Agents and other skills invoke skills through the orchestration engine:
```
ToolInvoker.InvokeTool(ctx, "invoke_skill", {
  "skill_name": "file-management",
  "args": { "operation": "list", "path": "/tmp" }
})
```

### 5. Execution
The SkillExecutor:
1. Looks up skill metadata in registry
2. Validates tool dependencies (Phase 4a: warnings only)
3. Starts OTEL tracing span (if tracer available)
4. Delegates to ToolInvoker with skill content + args
5. Returns result or error

### 6. Tracing
Each skill invocation creates a DAG node with:
- **Span Name**: `execute-skill:{skill-name}`
- **Metadata**: tier, plugin, triggers, portable flag
- **Status**: success/failure
- **Result**: captured for replay/debugging

---

## Skill File Format

### YAML Frontmatter (Required)

**Canonical Format (Nested Metadata Block):**

```yaml
---
name: {skill-name}                              # Required: slug format (kebab-case)
description: {1-sentence}                       # Required: concise purpose statement
triggers:                                       # Required: list of invocation phrases
  - "trigger phrase 1"
  - "trigger phrase 2"
metadata:                                       # Required: nested metadata block
  version: "1.0"                                # Required
  created: "2026-02-13"                         # Required
  author: "Manus AI"                            # Required
  tool_dependencies: [file_system, bash]        # Required: from allowlist
  portable: true                                # Required: boolean
  tier: 1                                       # Required: 1-4
  agents: [agent-name]                          # Required: which agents can invoke
  requires_version: "0.3.0"                     # Optional: minimum gateway version
  python_scripts: ["init_skill.py"]             # Optional: list of Python scripts
  shell_scripts: ["smart_clone.sh"]             # Optional: list of shell scripts
  hidden: false                                 # Optional: if true, only invokable by other skills
---
```

**Backward Compatibility:**
The parser also supports a flat format (without the `metadata:` block) for backward compatibility with initially ported skills. When `metadata:` is present, it takes precedence. Use the canonical nested format for all new skills.

### Valid Tool Dependencies (Allowlist)
- `file_system` - File operations (read, write, list, etc.)
- `bash` - Shell command execution
- `web_tools` - Web search and fetch (requires adapter in Phase 4a)
- `script_execution` - Allowed script execution (Phase 4a allowlist)
- `meta_skill` - Skill-to-skill invocation (Phase 4b, Tier 3+)

### Markdown Body (Required)

The content following the YAML frontmatter contains:
- Philosophy and purpose
- When to use this skill
- Workflow steps
- Best practices
- Quality checklist
- Common pitfalls
- Examples
- Related skills

**Format Guidelines:**
- Tool-agnostic language (no Claude-specific references)
- Concise and scannable (minimize token cost)
- Code examples where helpful
- Cross-references to related skills

---

## SkillRegistry Interface

```go
type SkillRegistry interface {
    // RegisterSkill adds a skill to the registry.
    // Returns error if skill name already exists or metadata is invalid.
    RegisterSkill(ctx context.Context, skill *SkillDefinition) error

    // GetSkill retrieves a skill by exact name.
    GetSkill(ctx context.Context, name string) (*SkillDefinition, error)

    // ListSkills returns all registered skills.
    ListSkills(ctx context.Context) ([]*SkillDefinition, error)

    // ListByPlugin returns skills for a specific plugin.
    ListByPlugin(ctx context.Context, pluginName string) ([]*SkillDefinition, error)

    // ListByTier returns skills filtered by portability tier.
    ListByTier(ctx context.Context, tier int) ([]*SkillDefinition, error)

    // ListByAgent returns skills available to a specific agent.
    // Hidden skills are excluded from agent queries.
    ListByAgent(ctx context.Context, agentName string) ([]*SkillDefinition, error)

    // LoadFromDirectory scans a directory for SKILL.md files and registers them.
    // Continues on error, returns aggregated errors at end.
    LoadFromDirectory(ctx context.Context, dirPath string, pluginName string) error

    // LoadFromManifest loads skills from a plugin manifest file.
    // Not implemented in Phase 4a (returns error).
    LoadFromManifest(ctx context.Context, manifestPath string) error
}
```

### SkillDefinition Type

```go
type SkillDefinition struct {
    ID               string    // UUID assigned at registration
    Name             string    // Slug: "file-management"
    Description      string    // 1-sentence purpose
    PluginName       string    // "skill-forge"
    FilePath         string    // Absolute path to SKILL.md
    Triggers         []string  // Trigger phrases
    ToolDependencies []string  // ["file_system", "bash"]
    Tier             int       // 1-4
    Portable         bool      // true if no Claude lock-in
    Agents           []string  // Agent slugs
    RequiredVersion  string    // "0.3.0"
    PythonScripts    []string  // Script names
    ShellScripts     []string  // Script names
    Content          string    // Raw markdown (no frontmatter)
    Hidden           bool      // If true, not invokable by agents
    ParsedAt         time.Time
    LoadedAt         time.Time
    Version          string
    Created          string
    Author           string
}
```

---

## SkillExecutor Interface

```go
type SkillExecutor interface {
    // Execute runs a skill with the given arguments.
    // If the skill invokes other skills, they become DAG subtask nodes.
    Execute(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error)

    // NOTE: Phase 4b methods (Tier 3 meta-skills):
    // - ExecuteAsSubtask: Run skill and register DAG node
    // - RegisterMetaSkillInvocation: Create subtask node for skill invocation
}
```

---

## Invocation Semantics (ADR-004: Skill Invocation = DAG Subtasks)

When a skill invokes another skill:

1. **PlanNode Creation**: A new `PlanNode` is created with:
   ```go
   ToolName: "invoke_skill"
   Parameters: { "skill_name": "target-skill", "args": {...} }
   ```

2. **DAG Integration**: Node is added to current plan's `Nodes` slice

3. **Engine Scheduling**: `executeNodesInParallel()` handles scheduling based on dependencies

4. **Automatic Tracing**: Trace logger wraps invocation as OTEL span

5. **Budget Tracking**: Budget tracker charges invocation separately

**Benefits:**
- Skills are first-class orchestration nodes
- Dependency tracking is automatic
- Tracing and observability built-in
- Budget enforcement per-skill

---

## Portability Tiers

### Tier 1: Zero-Change Port
- **Tool Dependencies**: `file_system`, `bash` only
- **Adapters Required**: None
- **Effort**: Direct copy with YAML metadata addition
- **Example**: file-management, project-exploration

### Tier 2: Adapter-Dependent
- **Tool Dependencies**: `web_tools`, `script_execution`
- **Adapters Required**: WebToolAdapter, ScriptExecutor
- **Effort**: Copy + verify adapter integration
- **Example**: web-research, code-review

### Tier 3: Meta-Skills (Phase 4b)
- **Tool Dependencies**: `meta_skill`
- **Adapters Required**: DAG subtask binding
- **Effort**: Refactor to use ExecuteAsSubtask
- **Example**: research-synthesis, orchestration workflows

### Tier 4: Complex Integration (Future)
- **Tool Dependencies**: Custom integrations
- **Adapters Required**: Significant refactoring
- **Effort**: Rewrite with gateway-native patterns
- **Example**: MCP server integrations

---

## Phase 4a Limitations

Phase 4a implements Tier 1 + Tier 2 skills only. The following are **not implemented** and will error:

### Not Implemented:
- `ExecuteAsSubtask()` - Phase 4b (Tier 3)
- `RegisterMetaSkillInvocation()` - Phase 4b (Tier 3)
- Max call depth enforcement (max=3) - Phase 4b
- Tool dependency hard validation - Phase 4a logs warnings only
- Plugin manifest loading - Phase 4a uses filesystem scan only

### Implemented:
- SkillRegistry with YAML parsing
- SkillExecutor with tracing
- SkillInvoker (orchestration integration)
- WebToolAdapter (Brave API + fallback)
- ScriptExecutor (allowlist security)
- 40 Tier 1 + Tier 2 skills ported and validated

---

## Security Model

### Script Execution Allowlist
Only these scripts can be executed:
- `init_skill.py`
- `suggest_seeds.py`
- `diff_tracker.py`
- `context_mapper.py`
- `smart_clone.sh`
- `apply_seed.py`
- `lychee`
- `validate_skill.py`

### Argument Validation
All script arguments are validated for shell metacharacters:
- Reject: `|`, `;`, `&`, `$`, `` ` ``, `\n`, `<`, `>`, `(`, `)`, `{`, `}`, `*`, `?`, `[`, `]`
- Allow: Letters, numbers, `-`, `_`, `/`, `.`, spaces

### Path Validation
All script paths must:
- Reside within configured base directory
- Not use path traversal (`../`)
- Be in the allowlist by name

### Timeout Enforcement
- Default: 30 seconds per script
- Configurable via ScriptExecutorConfig
- Hard timeout via context.WithTimeout

---

## Usage Examples

### Loading Skills on Startup

```go
// Create registry and load skills
registry := skill.NewInMemorySkillRegistry()
err := registry.LoadFromDirectory(ctx, "/plugins/skill-forge/skills", "skill-forge")

// List loaded skills
skills, _ := registry.ListSkills(ctx)
fmt.Printf("Loaded %d skills\n", len(skills))
```

### Invoking a Skill

```go
// Via SkillExecutor
executor := skill.NewSkillExecutor(registry, toolInvoker, traceLogger)
result, err := executor.Execute(ctx, "file-management", map[string]interface{}{
    "operation": "list",
    "path": "/tmp",
})

// Via SkillInvoker (orchestration layer)
invoker := orchestration.NewSkillInvoker(registry, executor, baseInvoker)
result, err := invoker.InvokeTool(ctx, "invoke_skill", map[string]interface{}{
    "skill_name": "file-management",
    "args": map[string]interface{}{"operation": "list", "path": "/tmp"},
})
```

### Filtering Skills by Agent

```go
// Get skills available to a specific agent
agentSkills, _ := registry.ListByAgent(ctx, "implementation-agent")
for _, skill := range agentSkills {
    fmt.Printf("- %s: %s\n", skill.Name, skill.Description)
}
```

---

## Migration Checklist

When porting a skill from CoworkPluginsByDojoGenesis:

1. **Copy SKILL.md** to gateway `plugins/{plugin}/skills/{skill}/SKILL.md`
2. **Add Required YAML Fields**:
   - `triggers` (list of invocation phrases)
   - `metadata.tool_dependencies` (from allowlist)
   - `metadata.tier` (1-4)
   - `metadata.portable` (true/false)
   - `metadata.agents` (list of agent slugs)
   - `metadata.version`, `metadata.created`, `metadata.author`
3. **Validate Tool Dependencies**: Ensure all deps are from allowlist
4. **Test Registration**: Verify skill loads without errors
5. **Test Invocation**: Execute with minimal args, verify no crash
6. **Document Changes**: Update MIGRATION.md with status

---

## Versioning and Evolution

### Contract Version
This contract is version 1.0 and applies to Phase 4a (Tier 1 + Tier 2 skills).

### Breaking Changes
Breaking changes will increment the major version and require:
- Migration guide for existing skills
- Deprecation period (1 release cycle minimum)
- Automated migration tooling where possible

### Additions
Non-breaking additions (new fields, optional parameters) increment the minor version.

---

## References

- **ADR-004**: Skill Invocation as DAG Subtasks
- **Phase 4a Scout**: `/scouts/2026-02-13_phase4_scout.md`
- **Orchestration Contract**: `/orchestration/README.md`
- **Tool Registry**: `/tools/README.md`

---

**End of Contract**
