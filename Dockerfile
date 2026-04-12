# Dockerfile — local development builds (builds from source, single-arch)
# For production multi-arch images, Goreleaser uses Dockerfile.goreleaser
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy workspace and root module files
COPY go.work go.work.sum* ./
COPY go.mod go.sum ./

# Copy each workspace module's go.mod/go.sum for dependency caching.
# This list must match the `use` block in go.work.
COPY apps/go.mod apps/go.sum* ./apps/
COPY cmd/dojo/go.mod cmd/dojo/go.sum* ./cmd/dojo/
COPY disposition/go.mod disposition/go.sum* ./disposition/
COPY integration/go.mod integration/go.sum* ./integration/
COPY mcp/go.mod mcp/go.sum* ./mcp/
COPY memory/go.mod memory/go.sum* ./memory/
COPY orchestration/go.mod orchestration/go.sum* ./orchestration/
COPY provider/go.mod provider/go.sum* ./provider/
COPY runtime/actor/go.mod runtime/actor/go.sum* ./runtime/actor/
COPY runtime/cas/go.mod runtime/cas/go.sum* ./runtime/cas/
COPY runtime/d1client/go.mod runtime/d1client/go.sum* ./runtime/d1client/
COPY runtime/event/go.mod runtime/event/go.sum* ./runtime/event/
COPY runtime/wasm/go.mod runtime/wasm/go.sum* ./runtime/wasm/
COPY server/go.mod server/go.sum* ./server/
COPY skill/go.mod skill/go.sum* ./skill/
COPY tools/go.mod tools/go.sum* ./tools/
COPY wasm-modules/dip-scorer/go.mod wasm-modules/dip-scorer/go.sum* ./wasm-modules/dip-scorer/
COPY workflow/go.mod workflow/go.sum* ./workflow/

# Download dependencies for all workspace modules
RUN for dir in apps cmd/dojo disposition integration mcp memory \
    orchestration provider runtime/actor runtime/cas runtime/d1client \
    runtime/event runtime/wasm server skill tools \
    wasm-modules/dip-scorer workflow; do \
      cd /app/$dir && go mod download; \
    done

# Copy source code
COPY . .

# Build with CGO disabled (pure-Go sqlite via modernc.org/sqlite)
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /agentic-gateway main.go

# Runtime stage — distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12

COPY --from=builder /agentic-gateway /agentic-gateway

EXPOSE 7340

# Health check: the binary supports --health-check flag (self-contained HTTP probe).
# Docker Compose uses: ["/agentic-gateway", "--health-check"]
# No curl/wget needed — works in distroless.

# Run as non-root user (nobody)
USER 65534:65534

ENTRYPOINT ["/agentic-gateway"]
