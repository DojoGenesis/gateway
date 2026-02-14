# Contributing to Agentic Gateway

Thank you for your interest in contributing to the Agentic Gateway! This guide will help you get started with development, testing, and submitting contributions.

## Development Setup

### Prerequisites

- **Go 1.24+** (required)
- **Docker** (optional, for testing full stack)
- **golangci-lint** (optional, for linting)

### Getting Started

1. **Clone the repository:**
   ```bash
   git clone https://github.com/dojogenesis/agentic-gateway.git
   cd agentic-gateway
   ```

2. **Download dependencies:**
   ```bash
   go mod download
   ```

3. **Build the gateway:**
   ```bash
   make build
   ```
   This creates the binary at `bin/agentic-gateway`.

4. **Run the gateway locally:**
   ```bash
   ./bin/agentic-gateway
   ```
   The gateway will start on `http://localhost:8080` by default.

5. **Verify the health endpoint:**
   ```bash
   curl http://localhost:8080/health
   ```

## Testing

### Running Tests

Run all tests before submitting a PR:

```bash
make test
```

This runs tests with the race detector enabled to catch concurrency issues.

### Coverage Reports

Generate a coverage report:

```bash
make test-cover
```

This creates `coverage.html` which you can open in your browser.

### Coverage Goals

- Aim for **>80% coverage** on new code
- Critical paths (authentication, orchestration, tool execution) should have **>90% coverage**

### Writing Tests

- Use **table-driven tests** for complex logic
- Example:
  ```go
  func TestToolRegistry(t *testing.T) {
      tests := []struct {
          name    string
          input   ToolDefinition
          wantErr bool
      }{
          {"valid tool", validTool, false},
          {"invalid tool", invalidTool, true},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              // test logic here
          })
      }
  }
  ```

## Code Style

### Go Conventions

- **Format code with `gofmt`** before committing
- **Run `go vet`** to catch common mistakes:
  ```bash
  make vet
  ```
- **Run `golangci-lint`** for comprehensive linting:
  ```bash
  make lint
  ```

### Documentation

- **Godoc comments** on all exported symbols (functions, types, methods, constants)
- Start comments with the symbol name:
  ```go
  // ToolRegistry manages tool registration and lookup.
  type ToolRegistry interface { ... }
  ```
- Include examples where helpful for complex APIs

### Error Handling

- Use `fmt.Errorf("%w", err)` to wrap errors and preserve the error chain
- Example:
  ```go
  if err != nil {
      return fmt.Errorf("failed to register tool: %w", err)
  }
  ```

### Struct Tags

- All config structs must have both `json` and `yaml` tags
- Use `omitempty` for optional fields:
  ```go
  type Config struct {
      Required string `json:"required" yaml:"required"`
      Optional string `json:"optional,omitempty" yaml:"optional,omitempty"`
  }
  ```

## Docker

### Building the Docker Image

```bash
make docker
```

This builds the image as `agentic-gateway:latest`.

### Testing with Docker Compose

The repository includes a complete example stack with OTEL, Langfuse, and Postgres:

```bash
make docker-compose-up
```

Access:
- Gateway: `http://localhost:8080`
- Langfuse: `http://localhost:3000`
- OTEL Collector: `localhost:4317` (gRPC)

Stop the stack:

```bash
make docker-compose-down
```

## Git Workflow

### Branch Naming

- Feature branches: `feature/your-feature-name`
- Bug fixes: `fix/issue-description`
- Documentation: `docs/topic`

### Commit Messages

- Start with a verb: `Add`, `Fix`, `Refactor`, `Update`, `Remove`, etc.
- Keep the first line under 72 characters
- Use the body to explain *why*, not *what* (the code shows *what*)

Examples:
```
Add MCP tool bridge for external integrations

Implements the MCPToolBridge adapter that converts MCP tool calls
to the Gateway's ToolFunc signature. This enables integration with
external MCP servers like Composio.
```

```
Fix race condition in span storage

Adds proper mutex locking around activeSpans map access to prevent
concurrent map read/write panics during high-load scenarios.
```

### Pull Requests

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** and commit them

3. **Push your branch:**
   ```bash
   git push origin feature/my-feature
   ```

4. **Open a PR** with:
   - Clear description of changes
   - Reference any related issues (e.g., "Fixes #123")
   - Screenshots or examples if relevant

5. **Ensure CI passes:**
   - All tests pass
   - `go vet` passes
   - `golangci-lint` passes
   - Build succeeds

## Release Process

Releases are automated via Goreleaser and GitHub Actions.

### Creating a Release (Maintainers Only)

1. **Ensure all tests pass:**
   ```bash
   make test
   ```

2. **Update `CHANGELOG.md`** with the release date and any final entries.

3. **Create and push a git tag:**
   ```bash
   git tag -a v0.3.0 -m "Release v0.3.0"
   git push origin v0.3.0
   ```

4. **GitHub Actions takes over:**
   - Runs the full test suite (`go test -race ./...`)
   - Goreleaser builds binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
   - Builds and pushes the Docker image to `ghcr.io/dojogenesis/agentic-gateway`
   - Creates a GitHub Release with checksums and archives

5. **Verify the release** on the GitHub Releases page.

### Local Release Testing

To test the release pipeline locally without publishing:

```bash
goreleaser build --snapshot --clean
```

This produces binaries in `dist/` for your current platform.

### Versioning

We follow [Semantic Versioning](https://semver.org/):
- `MAJOR.MINOR.PATCH`
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes (backward compatible)

## Architecture Guidelines

### Module Structure

The project uses Go workspaces with separate modules:
- `shared/` — Shared types and utilities
- `events/` — Event streaming
- `provider/` — Provider plugin system
- `tools/` — Tool registry and execution
- `memory/` — Memory management
- `orchestration/` — DAG orchestration
- `server/` — HTTP server and handlers
- `mcp/` — MCP host integration (Phase 2A)
- `pkg/gateway/` — Core interfaces (Phase 1)
- `pkg/disposition/` — Agent configuration (Phase 2B)

### Adding New Features

1. **Define interfaces** in `pkg/gateway/` if extending the Gateway contract
2. **Implement** in the appropriate module
3. **Wire** in `main.go` during initialization
4. **Test** with both unit and integration tests
5. **Document** in Godoc and README if user-facing

### Dependency Management

- Use `go mod tidy` to clean up dependencies
- Avoid circular dependencies between modules
- Prefer stdlib over external libraries where reasonable

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Tag maintainers in PRs if you need review assistance

Thank you for contributing to the Agentic Gateway! 🚀
