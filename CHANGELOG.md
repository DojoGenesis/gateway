# Changelog

All notable changes to AgenticGateway will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-02-14

### Added
- Standalone `orchestration/` package — zero external dependencies, independently importable
- 5 adapter types bridging standalone orchestration to server concerns:
  - `ToolInvokerAdapter` — routes tool calls to server's tool registry
  - `TraceLoggerAdapter` — sends trace events to OTEL collector
  - `EventEmitterAdapter` — streams events to SSE clients
  - `BudgetTrackerAdapter` — tracks token usage per orchestration
  - `spanHandleAdapter` — wraps OTEL spans for the orchestration engine
- `pkg/skill/` package — SkillRegistry, SkillExecutor, SkillInvoker
- 44 skills ported across 7 plugin directories (28 Tier 1, 12 Tier 2, 4 Tier 3 stubs)
- `WebToolAdapter` for web search and fetch operations
- `ScriptExecutor` with security allowlist for bash/python execution
- Comprehensive skill smoke testing framework (40 scenarios)

### Changed
- Orchestration engine moved from `server/orchestration/` to standalone `orchestration/` package
- Server layer now acts as thin adapter over standalone orchestration interfaces
- `PlannerInterface`, `ToolInvokerInterface`, `TraceLoggerInterface`, `EventEmitterInterface`, `BudgetTrackerInterface` are now the canonical interfaces in `orchestration/`

### Fixed
- Race conditions in concurrent orchestration access (verified with `-race` flag)

### Metrics
- 170+ tests total (51 orchestration + 79 skill unit + 40 smoke)
- 100% test pass rate
- 86.4% code coverage on skill package
- Clean race detection

## [0.2.0] - 2026-02-13

### Added
- `pkg/gateway/` — 4 core interfaces (ToolRegistry, OrchestrationExecutor, MemoryStore, AgentInitializer)
- `pkg/disposition/` — Native Go ADA parser (YAML contract, no TypeScript runtime dependency)
- `mcp/` — MCP host module (MCPByDojoGenesis connected via stdio, 14 tools registered)
- 7 disposition-aware behavioral modules (orchestration pacing, memory depth, proactive intelligence, error handler, collaboration manager, validator, reflection engine)
- OTEL observability with Langfuse docker-compose example
- Docker Compose production deployment (gateway + MCP + OTEL + Langfuse + PostgreSQL)
- `/admin/mcp/status` health check endpoint
- `gateway-config.yaml` for production configuration
- Goreleaser configuration for binary distribution

### Changed
- Agent initialization now consumes ADA DispositionConfig at init time
- Tool registration uses namespace prefixes for MCP tools (e.g., `mcp_by_dojo:create_artifact`)

## [0.1.0] - 2026-02-12

### Added
- Initial AgenticGateway release
- 7-module Go workspace: shared, events, provider, tools, orchestration, memory, server
- Multi-provider LLM orchestration with context-based routing
- Tool registry and execution engine
- Conversation memory with semantic compression
- HTTP API server (OpenAI-compatible + agentic + admin layers)
- CI/CD pipeline with GitHub Actions
- Docker build support
