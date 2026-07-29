.PHONY: build build-spa build-chat-spa test test-server test-cover vet lint docker docker-compose-up docker-compose-down clean generate-openapi service-token

# Build the Workflow Builder SPA and embed it into the Go binary.
# Must be run before `make build` when the SPA has changed.
build-spa:
	@echo "Building Workflow Builder SPA..."
	@cd workflow-builder && npm run build
	@echo "Copying SPA output to server/workflowui/dist/ ..."
	@rm -rf server/workflowui/dist
	@cp -r workflow-builder/build server/workflowui/dist
	@echo "SPA embedded: server/workflowui/dist/ ($$(find server/workflowui/dist -type f | wc -l | tr -d ' ') files)"

# Build the Chat UI SPA and embed it into the Go binary.
# Must be run before `make build` when the chat SPA has changed.
build-chat-spa:
	@echo "Building Chat UI SPA..."
	@cd chat-ui && npm install && npm run build
	@echo "Copying SPA output to server/chatui/dist/ ..."
	@rm -rf server/chatui/dist
	@cp -r chat-ui/build server/chatui/dist
	@echo "Chat SPA embedded: server/chatui/dist/ ($$(find server/chatui/dist -type f | wc -l | tr -d ' ') files)"

# Build the binary (run `make build-spa` first if the SPA has changed)
build:
	@echo "Building agentic-gateway..."
	@mkdir -p bin
	@go build -o bin/agentic-gateway main.go
	@echo "Build complete: bin/agentic-gateway"

# Run all tests with race detector.
#
# `go test ./...` covers ONLY the root module. server/ is a separate module in
# go.work, so it is not matched by ./... from here — which meant the auth
# middleware was not covered by this gate at all. It is run explicitly below.
test: test-server
	@echo "Running tests..."
	@go test -race ./...

# Tests for the server module (auth middleware, router, handlers).
test-server:
	@echo "Running server module tests..."
	@cd server && go test -race ./...

# Run tests with coverage report
test-cover:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@# Exclude generated protobuf code from coverage report
	@grep -v 'provider/pb/' coverage.out > coverage_filtered.out
	@go tool cover -html=coverage_filtered.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run provider layer coverage check (targets: provider/ ≥80%, providers/ ≥85%)
test-cover-providers:
	@echo "Provider layer coverage..."
	@go test ./server/services/providers/... -coverprofile=providers.out -count=1
	@echo "---"
	@go test ./provider -coverprofile=provider.out -count=1
	@echo "--- server/services/providers/ ---"
	@go tool cover -func=providers.out | tail -1
	@echo "--- provider/ (excluding generated pb/) ---"
	@grep -v 'provider/pb/' provider.out > provider_filtered.out || true
	@go tool cover -func=provider_filtered.out | tail -1

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t agentic-gateway:latest .
	@echo "Docker image built: agentic-gateway:latest"

# Start example stack with docker-compose
docker-compose-up:
	@echo "Starting docker-compose stack..."
	@docker-compose -f docker-compose.yaml up -d
	@echo "Stack started. Gateway at http://localhost:7340, Langfuse at http://localhost:3000"

# Stop docker-compose stack
docker-compose-down:
	@echo "Stopping docker-compose stack..."
	@docker-compose -f docker-compose.yaml down
	@echo "Stack stopped."

# Mint a long-lived service token for a machine client (offline operator tool).
#
# Run this on the gateway host with the gateway's own environment loaded, so it
# signs with the SAME JWT_SECRET the running server verifies against. It refuses
# to mint if JWT_SECRET is unset (that would produce a token signed with the
# publicly known development fallback).
#
#   make service-token SERVICE=pdi
#   make service-token SERVICE=pdi TTL=720h
#
# The token is printed to stdout and nothing else; all diagnostics, including
# the jti you need in order to revoke it later, go to stderr. Pipe stdout
# straight into your secret store — never into a file in this repo.
service-token:
	@test -n "$(SERVICE)" || (echo "SERVICE is required, e.g. make service-token SERVICE=pdi" >&2 && exit 1)
	@cd server && go run ./cmd/servicetoken -service "$(SERVICE)" $(if $(TTL),-ttl "$(TTL)",)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin coverage.out coverage.html
	@echo "Clean complete."

# Generate OpenAPI spec (placeholder - requires swag or similar tool)
generate-openapi:
	@echo "Generating OpenAPI spec..."
	@which swag > /dev/null || (echo "swag not installed. Install with: go install github.com/swaggo/swag/cmd/swag@latest" && exit 1)
	@swag init -g main.go -o docs
	@echo "OpenAPI spec generated: docs/swagger.yaml"
