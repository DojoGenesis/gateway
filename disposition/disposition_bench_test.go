package disposition

import (
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
