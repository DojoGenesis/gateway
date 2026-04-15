---
name: codebase-cartography
model: sonnet
description: "Produces a structured codebase map — directory roles, entry points, dependency graph, architectural patterns, and dragon locations — for any unfamiliar repository. Use when: 'map this codebase for me', 'I need to understand this repo before I start', 'give me a reading order for this project', 'where are the dragons in this code'."
license: proprietary
category: continuous-learning
triggers:
  - "map this codebase for me"
  - "I need to understand this repo before I start"
  - "give me a reading order for this project"
  - "where are the dragons in this code"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_path
    type: string
    description: Path to the repository root to map
    required: true
outputs:
  - name: codebase_map
    type: string
    description: Structured codebase map with directory roles, entry points, dependency graph, architectural patterns, and dragon locations
---
# Codebase Cartography

## I. Philosophy

Understanding a codebase is not reading files -- it is building a mental model
that lets you predict where things are before you look. Cartography moves from
the coarsest grain (directory structure) to the finest (behavioral patterns)
in deliberate layers, each grounded in the previous.

The difference between exploration and cartography: exploration discovers what
exists; cartography produces a reusable map that accelerates future navigation
for yourself and others. A good codebase map answers three questions: Where do
I start? What depends on what? Where are the dragons?

This skill complements `project-exploration` (which assesses collaboration
potential) and `semantic-clusters` (which groups by action verbs). Cartography
produces the structural map; those skills interpret it.

## II. When to Use

- Onboarding to a new codebase (first day on a project).
- Preparing to write a specification that must integrate with existing code.
- Before a major refactoring to understand what will be affected.
- When debugging and you cannot find where a behavior originates.
- When handing off a project and need to produce orientation documentation.
- After a dependency upgrade to verify the impact surface.

Do not use this skill for assessing whether to collaborate with a project (use
`project-exploration`). Do not use it for action-verb clustering (use
`semantic-clusters`).

## III. Workflow

**Step 1 -- Structural survey.**

Map the directory tree at depth 3. Classify directories by role:

| Role | Examples |
|------|----------|
| Source code | `src/`, `lib/`, `pkg/`, `internal/` |
| Configuration | `.claude/`, `config/`, `.github/` |
| Tests | `tests/`, `__tests__/`, `*_test.go` |
| Documentation | `docs/`, `*.md` |
| Build/Deploy | `Dockerfile`, `Makefile`, `package.json`, `go.mod` |
| Generated | `dist/`, `build/`, `node_modules/` |

Count files per directory. Identify the heaviest directories (most code).

**Step 2 -- Entry point identification.**

Find the application's entry points:
- `main.go`, `main.py`, `index.ts`, `app.py` -- application entry
- `Makefile`, `package.json` scripts -- build/run commands
- `cmd/` directory -- CLI subcommands
- Test entry points -- what the test suite exercises

**Step 3 -- Dependency mapping.**

Trace import/require/use statements to build a dependency graph:
- Internal dependencies (which packages import which)
- External dependencies (third-party libraries)
- Circular dependencies (dragons)
- Interface boundaries (where abstractions live)

**Step 4 -- Pattern recognition.**

Identify architectural patterns in use:
- Layer pattern (handler -> service -> repository)
- Plugin/hook pattern (extension points)
- Event-driven (pub/sub, channels, SSE)
- Actor model (goroutines, workers, supervisors)
- Configuration: env vars vs config files vs flags

**Step 5 -- Produce the map.**

Output a structured codebase map containing:
- Directory role classification
- Entry point catalog with descriptions
- Dependency graph (mermaid or text)
- Architectural pattern summary
- Dragon locations (complex, fragile, or undocumented areas)
- Recommended reading order for a new contributor

## IV. Best Practices

- Start from the build system (package.json, go.mod, Cargo.toml) -- it reveals
  the dependency tree before you read any source code.
- Read tests before implementation. Tests reveal intended behavior.
- Follow the data flow, not the call stack. Where does data enter? Where is it
  persisted? Where does it exit?
- Note what is absent. Missing tests, missing docs, missing error handling are
  all map features.
- Use `semantic-clusters` after cartography to group by action verbs.

## V. Quality Checklist

- [ ] Directory tree mapped at depth 3 with role classification
- [ ] All entry points identified and described
- [ ] Internal dependency graph produced
- [ ] External dependencies cataloged with versions
- [ ] Architectural patterns identified and named
- [ ] Dragon locations documented with risk assessment
- [ ] Recommended reading order provided

## VI. Common Pitfalls

- **Reading every file.** Cartography is breadth-first. Depth comes later.
- **Skipping the build system.** The dependency manifest is the most information-dense file.
- **Mapping generated code.** `dist/`, `node_modules/`, `vendor/` are noise.
- **Missing the config layer.** Environment variables and config files shape
  behavior as much as source code.
- **Producing a map nobody reads.** Keep the output under 200 lines. Link to
  details rather than inlining them.

## VII. Related Skills

- `project-exploration` -- Assess collaboration potential with a new project
- `semantic-clusters` -- Group components by action verbs (what they DO)
- `repo-status` -- Generate living status documents for repositories
- `health-audit` -- Comprehensive repository health assessment
- `repo-context-sync` -- Sync and extract context from GitHub repositories

## Output

- Structured codebase map (markdown, under 200 lines) with: directory role table, entry point catalog, dependency graph (mermaid or ASCII), architectural pattern summary, dragon registry, recommended reading order

## Examples

**Scenario 1:** "Map this Go microservice repo before I add a new endpoint" -> Directory tree classified by role, entry point at `cmd/server/main.go` identified, handler->service->repository pattern named, circular dependency in `pkg/auth` flagged as dragon, reading order: go.mod -> cmd -> internal/service -> internal/handler

**Scenario 2:** "I need to hand off this TypeScript monorepo to another developer" -> Map produced with workspace package roles, shared dependency graph across packages, test coverage gaps noted, recommended onboarding order from foundational packages outward

## Edge Cases

- **Generated directories are large (node_modules, dist, vendor):** List them in the map as "Generated — skip" rather than mapping their contents; their presence is a map feature, their contents are not
- **Repo has no README or docs:** Note absence explicitly in the dragon registry; document what the build system reveals as a substitute

## Anti-Patterns

- Reading every source file before producing the map — cartography is breadth-first; depth comes after the map is complete
- Including generated code content in the dependency graph — it inflates the graph and obscures real internal dependencies
- Producing a map longer than 200 lines — link to details rather than inlining them; a map nobody reads has failed
