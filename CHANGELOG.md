# Changelog

All notable changes to AgenticGateway will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-14

### Added (Provider Layer v1 — The Living Grid)
- 8 in-process provider adapters: Anthropic, OpenAI, Google (Gemini), Groq, Mistral, Kimi (K2.5), DeepSeek, Ollama
- `server/services/providers/` package with shared `BaseProvider` and `openaiCompatibleProvider` base
- Dynamic API key resolution: resolver → static → environment variable precedence
- `RegisterProviders()` auto-discovery at startup — cloud providers check for API keys, local providers check reachability
- `GET /admin/providers` endpoint — live provider health and model inventory
- Conformance test suite — standardized tests for all 8 providers with mock HTTP servers
- Integration test suite (gated behind `//go:build integration`) for real API validation
- Kimi (Moonshot AI K2.5) added to cloud provider routing preferences

### Added (MCP Transport — SSE & Streamable HTTP)
- `connectSSE()` method in `mcp/connection.go` — SSE transport via `client.NewSSEMCPClient` (mcp-go v0.43.2)
- `connectStreamableHTTP()` method in `mcp/connection.go` — streamable HTTP transport via `client.NewStreamableHttpClient`
- Custom header injection via `transport.WithHeaders()` for both SSE and streamable HTTP
- SSE server configuration example in `gateway-config.yaml`
- 4 new transport tests: SSE connect, streamable_http connect, SSE with headers, streamable_http with headers

### Added (Pre-v1 Hardening)
- Hard dependency validation in `skill/executor.go` — unmet deps return errors instead of warnings
- Execution tracking in `GatewayOrchestrationExecutor` — `Cancel(executionID)` and `Status(executionID)` methods
- User tier lookup via `db.GetSettings()` with fallback to `"free"` tier
- E2E smoke test job in CI — builds binary, starts gateway, validates `/health`, API endpoints, and `/admin/metrics/prometheus`
- Protobuf coverage exclusion — `provider/pb/` filtered from coverage reports for accurate metrics

### Added (Multi-Arch Docker)
- `Dockerfile.goreleaser` — 14-line distroless runtime image for release builds
- Per-architecture Docker images (amd64 + arm64) via Goreleaser `dockers` config
- Docker manifest lists for `:v{version}` and `:latest` tags
- OCI labels via `build_flag_templates` (created, title, revision, version, source)
- CI `docker` job with QEMU + Buildx for multi-arch verification (no push)
- `--health-check` flag in binary for distroless container health probes (no curl/wget needed)

### Changed (Provider Layer v1)
- `HandshakeValue` bumped from `"v0.0.15"` to `"v1.0.0"` in `provider/manager.go`
- `PluginManager` gained `ProviderCount()` and `ProviderStatuses()` methods
- `localProviderPreference` simplified to `["ollama"]` (removed `embedded-qwen3`)
- DeepSeek provider migrated from 554-line `inprocess_deepseek.go` to 35-line adapter using shared base
- Provider configs in `config.yaml` updated for v1.0.0 architecture

### Changed (Pre-v1 Hardening)
- `validateDependencies` in `skill/executor.go` upgraded from warning to hard error
- `Dockerfile` header comment added to distinguish local dev builds from Goreleaser release builds
- Cloud adapter error message updated to "intentionally deferred (v1 is local-first, SQLite-only)"

### Removed (Provider Layer v1)
- `provider/rpc.go` — 295 lines of dead NetRPC code (gRPC is the only protocol)
- `server/services/inprocess_deepseek.go` — replaced by `server/services/providers/deepseek.go`
- `embedded-qwen3` references removed from routing, config, cost tracking, and handlers

### Added (Post-v0.3.0)
- Meta-skill invocation with `ExecuteAsSubtask` method
- Call depth tracking with max depth enforcement (max=3)
- Budget tracking and propagation for meta-skill chains (`BudgetTracker`)
- Error sentinels: `ErrMaxDepthExceeded`, `ErrBudgetExhausted`
- OTEL span linking for meta-skill parent → child relationships
- SSE event catalog with type constants and schemas (`events/catalog.go`, 254 lines)
- Package-level documentation for `pkg/collaboration/`, `pkg/disposition/`, `pkg/intelligence/`, `pkg/reflection/`, `pkg/validation/`
- Integration path tests (`server/integration_paths_test.go`, 409 lines)
- SSE lifecycle tests (`server/sse_lifecycle_test.go`, 238 lines)
- Error response consistency tests (`server/error_response_test.go`, 381 lines)

### Changed (Post-v0.3.0)
- Gateway error handling improved with sentinel errors and context

### Added (Backend Integration Sweep)
- Full `handleGatewayAgentChat` implementation — agent lookup, orchestration plan creation, SSE streaming, error handling
- Full `handleGatewayOrchestrationDAG` implementation — DAG node/edge serialization with execution status
- Full `handleGatewayGetTrace` implementation — trace + span retrieval with duration calculation

### Not Implemented (by design)
- `POST /admin/config/reload` returns 501 — requires service restart (hot reload deferred to post-v1)
- Cloud adapter (Supabase) intentionally deferred — v1 is local-first, SQLite-only

## [0.3.0] - 2026-02-14

### Added
- Standalone `orchestration/` package — interface-driven, independently importable
- 5 adapter types bridging standalone orchestration to server concerns:
  - `ToolInvokerAdapter` — routes tool calls to server's tool registry
  - `TraceLoggerAdapter` — sends trace events to OTEL collector
  - `EventEmitterAdapter` — streams events to SSE clients
  - `BudgetTrackerAdapter` — tracks token usage per orchestration
  - `spanHandleAdapter` — wraps OTEL spans for the orchestration engine
- `skill/` package — SkillRegistry, SkillExecutor, ScriptExecutor
- `SkillInvoker` in `orchestration/` — wraps base ToolInvoker with skill-aware dispatch
- 44 skills ported across 7 plugin directories (28 Tier 1, 12 Tier 2, 4 Tier 3 stubs)
- `WebToolAdapter` for web search and fetch operations
- Comprehensive skill smoke testing framework (145 scenarios)

### Changed
- Orchestration engine moved from `server/orchestration/` to standalone `orchestration/` package
- Server layer now acts as thin adapter over standalone orchestration interfaces
- `PlannerInterface`, `ToolInvokerInterface`, `TraceLoggerInterface`, `EventEmitterInterface`, `BudgetTrackerInterface` are now the canonical interfaces in `orchestration/`

### Fixed
- Race conditions in concurrent orchestration access (verified with `-race` flag)

### Metrics
- 295 tests total (61 orchestration + 89 skill unit + 145 smoke)
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
