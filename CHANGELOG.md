# Changelog

All notable changes to AgenticGateway will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> Entries for 2.5.0 through 3.3.10 were rebuilt on 2026-10-23 from the GitHub release notes at https://github.com/DojoGenesis/gateway/releases. Dates are UTC publish dates. DojoGenesis/gateway has GitHub releases only from v2.5.0, so nothing is rebuilt for versions between 1.1.0 and 2.5.0, or for tags that never got a release (v3.1.0, for example). Where a release body is an auto-generated commit list, the entry condenses it: bullets follow the commit subjects, and docs-only, test-only, and merge commits are folded together or left out.

## [3.3.10] - 2026-10-13

### Added
- `/dispatch` structured-output handler and kata memory handlers (DGS-149, DGS-150)
- Vendored `kata-ai/v1` contract; local `$id` refs are resolved and op loading is guarded (DGS-149)
- Kata first-party plugin: 13 `kata-ai/v1` prompts (DGS-149)

### Changed
- Tool invocation is restricted by role, and code execution is gated
- Every first-party skill name is now reserved, and the list is kept from drifting
- Deploy workflow runs build-bridge and deploy only on `DojoGenesis/gateway`
- Deploy docs point the version at the `provision.sh` pin; GHCR org corrected in a workflow comment
- `provision.sh` pinned to v3.3.10 (tool-invoke authorization)

### Fixed
- First-party skills can be packaged; the fake Publish Skill workflow is replaced by a real Skills Check

## [3.3.9] - 2026-09-30

### Added
- Chat says what the gateway injected into the prompt, and echoes it on request (DGS-141)

### Changed
- `provision.sh` pinned to v3.3.9

## [3.3.8] - 2026-09-30

### Added
- OpenAI `tools` passthrough on `/v1/chat/completions` (DGS-141)

### Changed
- `provision.sh` pinned to v3.3.8

### Fixed
- Ollama native tools now actually reach Ollama, and capabilities come from Ollama (DGS-141)

## [3.3.7] - 2026-09-30

### Changed
- `provision.sh` pinned to v3.3.7

### Security
- One answer to "is this production": the resolved config, everywhere

## [3.3.6] - 2026-09-30

### Changed
- One preparation path for `/v1/chat/completions`
- `REGISTRATION_ENABLED` is read through one helper
- CI green again: errcheck in netbind tests, G101 on an email fixture; email and SMS integration smoke tests authenticate like real senders; stale `VerifySignature` comment in `channel/webchat` fixed
- `provision.sh` pinned to v3.3.6

### Removed
- 8,437 lines of unwired handlers and tests in `server/handlers`
- Unimported `pkg/channel`, `pkg/workflow`, and `router_gaps_patch.go`

### Fixed
- SMS and email webhooks refuse when no secret is configured (DGS-115 follow-up)
- `deepseek-*` model routing

## [3.3.5] - 2026-09-30

### Changed
- Deploy CI asserts the container end state, not just that compose exited 0 (DGS-110)
- Test pins the post-DGS-100 workflow-path posture: anonymous gets 401
- `provision.sh` pinned to v3.3.5

### Fixed
- Gateway and bridge bind `127.0.0.1` by default, and widening is explicit (DGS-113, DGS-136); one bind policy for both, where loopback means `127.0.0.1` and `::1`
- Telegram and WhatsApp webhooks refuse when no secret is configured (DGS-115)
- WhatsApp verify `GET` is routed to its handler, and an empty token never matches (DGS-137)
- Webchat refuses when no token is configured, and the bridge registers it only with one (DGS-142)
- DGS-108 edge block removed from the deploy config; `/api/*` carries auth in the binary now

## [3.3.4] - 2026-08-05

### Changed
- E2E CI asks for the dev tokens explicitly; it was relying on the DGS-112 hole
- `provision.sh` pinned to v3.3.4

### Fixed
- Authentication is required on `/api/*` and `/events` (DGS-100)
- Development shim tokens are an explicit opt-in, not a default (DGS-112)

## [3.3.3] - 2026-08-04

### Changed
- Docs: route-disposition table for the `/api/*` auth design (DGS-100); registration comments corrected, since production is closed now (DGS-101); `POST /auth/register` documented as open by default (DGS-101)
- Unused `getEnvAsInt` removed
- `provision.sh` pinned to v3.3.3 so it cannot revert the DGS-108 fix

### Fixed
- Closed the unauthenticated RCE on `/api/workflows`: `run_command` is gated behind a capability flag, off in production, and `/api/*` is blocked at the edge (DGS-108)
- `POST /api/cas/gc` is admin-gated; it allowed unauthenticated content deletion (DGS-100)
- Caddyfile reconciled with production; it was missing 4 of 5 sites
- Prod compose reconciled with dojo-bridge; it was missing the MCP endpoint and Memory Hub
- Duplicate crash-looping gateway removed from dojo-bridge (DGS-104)

## [3.3.2] - 2026-07-29

### Added
- Service tokens for machine clients

### Changed
- `provision.sh` pinned to v3.3.2

### Fixed
- JWT secret config is honest, and fails closed in production
- Config file actually loads now, and no longer silently does nothing
- Build fix: `jwt_secret.go` and the service_token adaptations committed

## [3.3.1] - 2026-07-26

### Security
- Authentication is now required on all of `/v1`: every route needs a valid `Bearer` token. Before this release the only middleware reaching `/v1` was `OptionalAuthMiddleware`, which never rejects, so an anonymous caller could `POST /v1/gateway/agents` and `/v1/gateway/agents/:id/chat`, creating agents and billing real provider spend
- `/health`, `/metrics`, `/auth/*`, and `/.well-known/did.json` are outside the group and stay public

### Operators: confirm two host-side settings
Upgrading the binary alone is not sufficient.
- `JWT_SECRET` must be set. If it is unset, `server/middleware/auth.go` falls back to a default literal, and tokens can be forged against a known value
- `ENVIRONMENT` must equal `production`. Otherwise `isDevelopment()` is true and `validateToken` accepts `test-token`, `user-*`, and `admin-*`, and `validateAdminRole` grants admin for `admin-*`
- Issue one token per consumer rather than sharing one, so any consumer can be revoked independently

### Changed
- Regression test: unauthenticated `POST /v1/gateway/agents` must return 401
- `provision.sh` pinned to v3.3.1 so a re-provision cannot reinstate an older binary

## [3.3.0] - 2026-07-20

### Added
- Ollama Cloud provider (`ollama-cloud`)
- Dynamic `/models` per provider, never stale, plus OpenRouter
- Non-chat models filtered out of the OpenAI-format catalog; the chat-only filter is applied in the shared wrapper
- `claude-sonnet-5`

### Fixed
- `--version` flag implemented, so the `provision.sh` idempotency check works (#4)
- `temperature` omitted for all deprecated Anthropic models (allowlist)
- Dead `google` default
- `DEPLOYMENT.md` and deploy tooling corrected to production reality: port 7340, v3.2.2 (#3)

### Removed
- Retired `mcp-by-dojo` service dropped from the root `docker-compose.yaml` (#5)

## [3.2.2] - 2026-07-16

### Changed
- README LOC claim synced to the current actual, 148K to 152K
- Reconciled the `dojogenesis/gateway` mirror (28 commits, Apr-Jul gateway dev) with the local nexus `CLAUDE.md` addition
- Direct unit tests for `parseSkillFile` nested and flat metadata parsing (kata-orchestrator)

### Fixed
- Memory search is token-based and ranked, and list/search totals are honest

## [3.2.1] - 2026-07-04

### Fixed
- `temperature` omitted for fable-class models (Anthropic 400 deprecation)

## [3.2.0] - 2026-07-04

### Added
- `system_prompt` accepted from `/v1/chat` requests and passed to the LLM
- Specialist dispatch, DAG execution, agent split, and embeddings
- Semantic router: lazy init and threshold calibration
- Admin-gated MCP tool direct-invoke endpoint
- Opus 4.8/4.7 aliases and a newest-first model list; default is now `claude-sonnet-4-6` (ADR 028)
- Three-layer MCP tool routing fix (Routes E+A+D)
- Dev-first system prompt and absolute DB paths in the deploy service

### Changed
- CI: Node.js 24 opted in for all workflow actions (bridge) (#2); golangci-lint version pinned for reproducibility (#1)
- Unit tests for event channel subject helpers, `SkillDefinition.IsValid` branches and Tier/dependency predicates, and `extractSkillMDFromTar` (kata-orchestrator)
- STATUS test file count synced, 228 to 229, post ADR-024 Phase 1; handoff docs on gateway prod-deployment status corrected (SETTLED to DEFERRED, not deployed)

### Fixed
- Ollama probe is bounded: 3 attempts with backoff, then disabled
- Conversation persistence, GitHub OAuth allowlist, and redirect URI
- Absolute DB paths, to prevent seed data loss across restarts
- Two `AgentRuntime` and tool-registry races
- Background orchestration detached from the request context
- `claude-fable-5` alias registered, and the agent-create `WorkspaceRoot` contract relaxed
- Stale clone URL and gateway port references in docs
- Lint and nightly CI cleanup: gosec and staticcheck findings, embed anchors for chatui and workflowui dist dirs, remaining `WriteString(Sprintf)` converted to `Fprintf` in the reflection engine, backoff timing tolerance and migration SQLite lock, plugin `StartTimeout`, 30s context deadline on browser integration tests, and 21 code quality issues from a codebase audit

## [3.0.0] - 2026-04-12

### Added
- D1 CAS sync: schema, sync loop, delta/batch/status endpoints, D1Syncer wiring, and sync tests (Era 4 Track A)
- Edge Worker for webhook handling, with 8 platform adapters (Era 4 Track B)
- Federated Agent Mesh: DID identity, announce, delegate (Era 4 Track C)
- Integration gate and test coverage sweep (Era 4 Track D)

### Fixed
- MCP sidecar binary path in prod compose
- Root `Dockerfile` and `Dockerfile.bridge` synced with the actual `go.work` module layout
- CI unblocked: `dist/.gitkeep` added, GHCR namespace and goreleaser target fixed, and goreleaser `dist/` artifacts untracked because they blocked the Release workflow

## [2.5.0] - 2026-04-12

### Highlights
- 8 LLM providers: Anthropic, OpenAI, Google Gemini, Groq, Mistral, Kimi (K2.5), DeepSeek, Ollama
- 8 channel adapters: Slack, Discord, Telegram, Email, SMS, WhatsApp, Teams, WebChat
- 70+ HTTP routes: OpenAI-compatible API, agent management, CAS, MCP protocol, admin
- Telemetry worker: OTEL spans, orchestration events, D1 migration, dashboard panels

### Added (new since v2.4.0-era3)
- Telemetry worker endpoints and the Kimi (K2.5) provider
- D1 migration for CAS sync infrastructure

### Changed
- Release prep: 179MB of tracked binaries removed, goreleaser version injection fixed, CI Go version pins updated, e2e port in CI fixed

### Fixed
- Workspace root propagation for cross-process CWD resolution
- Port unification, 8080 to 7340, across 11 files
- Chat handler detached from the HTTP request context

Install: `go install github.com/DojoGenesis/gateway/cmd/dojo@v2.5.0`

## [1.1.0] - 2026-02-15

### Added (MCP Apps Bridge — The App Bridge)
- MCP Apps host infrastructure (beta)
- Resource serving for `ui://` URIs via `GET /v1/gateway/resources`
- App lifecycle management (launch, close, list) via `/v1/gateway/apps/*`
- Tool call proxying with authorization for MCP Apps
- Security enforcement: CSP headers, permission validation, sandbox headers
- `apps/` module with ResourceRegistry, AppRegistry, AuthorizationPolicy, ToolCallProxy, SecurityPolicy, AppManager
- `ReadResource` method on `MCPServerConnection` for fetching MCP resources
- `MCPResourceLoader` for loading `ui://` resources from MCP servers
- Feature flag: `MCP_APPS_ENABLED` (default: false)
- `MCPAppsConfig` in server config with YAML and environment variable support
- Integration tests covering full app lifecycle, concurrent tool calls, and security headers
- Performance benchmarks: resource serving ~15ns, proxy overhead ~400ns

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
