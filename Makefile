.PHONY: build test test-cover vet lint docker docker-compose-up docker-compose-down clean generate-openapi

# Build the binary
build:
	@echo "Building agentic-gateway..."
	@mkdir -p bin
	@go build -o bin/agentic-gateway main.go
	@echo "Build complete: bin/agentic-gateway"

# Run all tests with race detector
test:
	@echo "Running tests..."
	@go test -race ./...

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
	@echo "Stack started. Gateway at http://localhost:8080, Langfuse at http://localhost:3000"

# Stop docker-compose stack
docker-compose-down:
	@echo "Stopping docker-compose stack..."
	@docker-compose -f docker-compose.yaml down
	@echo "Stack stopped."

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
