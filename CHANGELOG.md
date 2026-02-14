<!-- WARNING: Release dates below are PLACEHOLDERS (2026-02-XX). -->
<!-- Finalize dates immediately before tagging each release.      -->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] — 2026-02-XX

### Added
- Standalone `orchestration/` module extraction from `server/`
- Phase 4a — 44 behavioral skills ported (86.4% coverage)
- 5 orchestration adapters (anthropic, openai, gateway, mcp, memory)

### Changed
- DAG engine moved from `server/` to `orchestration/`

### Metrics
- 9 Go files extracted, 51 tests passing, 5 adapters

## [0.2.0] — 2026-02-XX

### Added
- ADA integration — `pkg/disposition/` with YAML parser (ADR-007)
- MCP host capability — `mcp/` module with unified tool namespace (ADR-002)
- Composio bridge specification (ADR-003)
- OTEL span export with Langfuse docker-compose example (ADR-005)

### Changed
- API surface formalized — `/v1/`, `/v1/gateway/`, `/admin/` (ADR-001)

## [0.1.0] — 2026-02-XX

### Added
- Initial 7-module Go workspace (shared, events, provider, tools, orchestration, memory, server)
- HTTP API server with agent logic
- Provider plugin system via gRPC
- Conversation memory with semantic compression
