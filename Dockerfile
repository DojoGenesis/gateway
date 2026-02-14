FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy workspace and module files
COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum* ./shared/
COPY events/go.mod events/go.sum* ./events/
COPY provider/go.mod provider/go.sum* ./provider/
COPY tools/go.mod tools/go.sum* ./tools/
COPY memory/go.mod memory/go.sum* ./memory/
COPY server/go.mod server/go.sum* ./server/
COPY mcp/go.mod mcp/go.sum* ./mcp/

# Download dependencies for all modules
RUN cd shared && go mod download && \
    cd ../events && go mod download && \
    cd ../provider && go mod download && \
    cd ../tools && go mod download && \
    cd ../memory && go mod download && \
    cd ../server && go mod download && \
    cd ../mcp && go mod download

# Copy source code
COPY . .

# Build with CGO disabled (pure-Go sqlite via modernc.org/sqlite)
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /agentic-gateway main.go

# Runtime stage — distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12

COPY --from=builder /agentic-gateway /agentic-gateway

EXPOSE 8080

# Run as non-root user (nobody)
USER 65534:65534

ENTRYPOINT ["/agentic-gateway"]
