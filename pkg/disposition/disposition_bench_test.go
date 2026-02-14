package disposition

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkResolveDisposition benchmarks the file loading and parsing performance.
// Per the requirements, typical files (<100KB) must parse in <100ms.
func BenchmarkResolveDisposition(b *testing.B) {
	workspaceRoot := "testdata"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ResolveDisposition(workspaceRoot, "")
		if err != nil {
			b.Fatalf("ResolveDisposition failed: %v", err)
		}
	}
}

// BenchmarkResolveDispositionWithMode benchmarks file loading with mode merging.
func BenchmarkResolveDispositionWithMode(b *testing.B) {
	workspaceRoot := "testdata"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ResolveDisposition(workspaceRoot, "debug")
		if err != nil {
			b.Fatalf("ResolveDisposition with mode failed: %v", err)
		}
	}
}

// BenchmarkLoadDispositionFromFile benchmarks pure file reading and YAML parsing.
func BenchmarkLoadDispositionFromFile(b *testing.B) {
	filePath := filepath.Join("testdata", "agent-basic.yaml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loadDispositionFromFile(filePath)
		if err != nil {
			b.Fatalf("loadDispositionFromFile failed: %v", err)
		}
	}
}

// BenchmarkValidate benchmarks configuration validation.
func BenchmarkValidate(b *testing.B) {
	cfg := DefaultDisposition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Validate(cfg)
		if err != nil {
			b.Fatalf("Validate failed: %v", err)
		}
	}
}

// BenchmarkCacheGet benchmarks cache retrieval performance.
func BenchmarkCacheGet(b *testing.B) {
	cache := NewDispositionCache(0) // No expiration
	cfg := DefaultDisposition()
	key := makeCacheKey("/workspace", "prod")
	cache.Set(key, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, found := cache.Get(key)
		if !found {
			b.Fatal("Cache miss in benchmark")
		}
	}
}

// BenchmarkCacheSet benchmarks cache storage performance.
func BenchmarkCacheSet(b *testing.B) {
	cache := NewDispositionCache(0) // No expiration
	cfg := DefaultDisposition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := makeCacheKey("/workspace", string(rune(i%100)))
		cache.Set(key, cfg)
	}
}

// BenchmarkAgentInitializer benchmarks the full initialization flow with caching.
func BenchmarkAgentInitializer(b *testing.B) {
	initializer := NewAgentInitializer(0) // No expiration for consistent benchmarking
	ctx := context.Background()

	// Create a temporary workspace
	workspaceRoot := filepath.Join(os.TempDir(), "bench_workspace")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
  require_tests: true
  require_docs: false
error_handling:
  strategy: log-and-continue
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
  triggers:
    - error
    - milestone
`), 0644)
	if err != nil {
		b.Fatalf("Failed to create test agent.yaml: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := initializer.Initialize(ctx, workspaceRoot, "")
		if err != nil {
			b.Fatalf("Initialize failed: %v", err)
		}
	}
}

// BenchmarkAgentInitializerCacheMiss benchmarks initialization when cache is empty (disk I/O).
func BenchmarkAgentInitializerCacheMiss(b *testing.B) {
	ctx := context.Background()

	// Create a temporary workspace
	workspaceRoot := filepath.Join(os.TempDir(), "bench_workspace_miss")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: rapid
depth: functional
tone: conversational
initiative: proactive
validation:
  strategy: spot-check
  require_tests: false
  require_docs: false
error_handling:
  strategy: retry
  retry_count: 5
collaboration:
  style: collaborative
  check_in_frequency: rarely
reflection:
  frequency: weekly
  format: bullets
  triggers: []
`), 0644)
	if err != nil {
		b.Fatalf("Failed to create test agent.yaml: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create new initializer each time to ensure cache miss
		initializer := NewAgentInitializer(0)
		_, err := initializer.Initialize(ctx, workspaceRoot, "")
		if err != nil {
			b.Fatalf("Initialize failed: %v", err)
		}
	}
}
