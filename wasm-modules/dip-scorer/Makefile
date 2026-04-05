.PHONY: build test clean

# Build the WASM module using TinyGo.
# Requires TinyGo installed: https://tinygo.org/getting-started/install/
build:
	tinygo build -target=wasi -o dip-scorer.wasm -no-debug .

# Run tests using standard Go (not TinyGo).
# The scoring logic in scorer.go has no build tags and compiles with regular Go.
test:
	go test ./...

# Run tests with verbose output.
test-verbose:
	go test -v ./...

# Clean build artifacts.
clean:
	rm -f dip-scorer.wasm
