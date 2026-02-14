# Agentic Gateway by Dojo Genesis — Strategic Scout

**Author:** Cruz + Claude
**Date:** 2026-02-12
**Status:** Decision Made

---

## The Tension

"Should we extract the agentic infrastructure from Dojo Genesis into a standalone open-source agentic API framework, or continue building within the monolith?"

## Routes Scouted

| Route | Approach | Speed | Architecture | Positioning |
|-------|----------|-------|--------------|-------------|
| 1. Clean Fork & Strip | Fork, remove Dojo code | Fast (2-3 wk) | Carries debt | Working product |
| 2. Extract & Rebuild | Reference impl, rewrite | Slow (6-10 wk) | Cleanest | Ideal but late |
| 3. Library Only | Go modules, no server | Fastest (1-2 wk) | Composable | Undersells value |
| 4. Modular Monorepo | Restructure in-place | Medium (4-6 wk) | Forced discipline | Ambiguous identity |
| 5. OpenAI-Compatible Gateway | Standard API surface | Medium (3-5 wk) | Constrained by spec | Strongest adoption |

## Decision: Hybrid Route

**"The Go Agentic Gateway by Dojo Genesis"**

Combines:
- **Route 1's speed** — start from working, tested code (52k lines, 178 test files)
- **Route 3's composability** — independent Go modules with own `go.mod`
- **Route 4's discipline** — extraction forces clean separation; Dojo Genesis eventually consumes the framework
- **Route 5's positioning** — OpenAI-compatible API surface as adoption lever

### What This Ships

Five independent Go modules:

| Module | Source Package | Responsibility |
|--------|---------------|----------------|
| `provider` | `plugin/` | ModelProvider interface, PluginManager, gRPC transport, routing |
| `tools` | `tools/` | Tool registry, definition types, execution with timeouts |
| `orchestration` | `orchestration/` | DAG planner interface, engine, auto-replanning, circuit breakers |
| `memory` | `memory/` | Conversation memory, garden compression, seed extraction |
| `server` | `handlers/` + `streaming/` + `events/` | Gin HTTP server, OpenAI-compatible + agentic API, SSE streaming |

### What This Rules Out

- Full rewrite (too slow)
- Staying in monolith (no open-source story)
- Library-only without server (hides the differentiators)

### Key Decisions

1. **Name:** Agentic Gateway by Dojo Genesis
2. **v0.1.0 Scope:** Full — includes orchestration engine (the main differentiator)
3. **Market Position:** First serious agentic framework in Go. OpenAI API compatible.
4. **Approach:** Fork + modularize + clean interfaces at boundaries

### Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Phase 0: Fork + Foundation | 1-2 weeks | Go workspace, module scaffolding, shared types |
| Phase 1: Core Modules | 2-3 weeks | provider + tools + orchestration + memory modules |
| Phase 2: Server + API | 1-2 weeks | OpenAI-compatible + agentic endpoints, SSE |
| Phase 3: Docs + Examples | 1 week | README, quickstart, example providers/tools |

**Total: 5-8 weeks to v0.1.0**
