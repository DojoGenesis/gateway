package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func BenchmarkMemoryOperations(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_memory.db")

	mgr, err := NewMemoryManager(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mem := &Memory{
				ID:      fmt.Sprintf("mem-%d", i),
				Type:    "conversation",
				Content: "This is a test memory for benchmarking",
				Metadata: map[string]interface{}{
					"user_id": "user1",
					"session": "session1",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := mgr.StoreMemory(ctx, mem); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Store some memories for retrieval benchmarks
	for i := 0; i < 1000; i++ {
		mem := &Memory{
			ID:      fmt.Sprintf("test-%d", i),
			Type:    "conversation",
			Content: fmt.Sprintf("This is test memory number %d", i),
			Metadata: map[string]interface{}{
				"user_id": "user1",
				"index":   i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mgr.StoreMemory(ctx, mem)
	}

	b.Run("Retrieve", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("test-%d", i%1000)
			_, err := mgr.GetMemory(ctx, id)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Search", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mgr.SearchMemories(ctx, "test memory", 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mem := &Memory{
				ID:        fmt.Sprintf("delete-%d", i),
				Type:      "temporary",
				Content:   "To be deleted",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			mgr.StoreMemory(ctx, mem)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("delete-%d", i)
			if err := mgr.DeleteMemory(ctx, id); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SearchByType", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mgr.SearchMemories(ctx, "conversation", 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConcurrentMemoryOperations(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_concurrent_memory.db")

	mgr, err := NewMemoryManager(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		mem := &Memory{
			ID:      fmt.Sprintf("concurrent-%d", i),
			Type:    "conversation",
			Content: fmt.Sprintf("Concurrent test memory %d", i),
			Metadata: map[string]interface{}{
				"index": i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mgr.StoreMemory(ctx, mem)
	}

	benchmarks := []struct {
		name       string
		concurrent int
	}{
		{"Concurrent_10", 10},
		{"Concurrent_100", 100},
		{"Concurrent_1000", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("Read_%s", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(bm.concurrent)
				for j := 0; j < bm.concurrent; j++ {
					go func(idx int) {
						defer wg.Done()
						id := fmt.Sprintf("concurrent-%d", idx%1000)
						mgr.GetMemory(ctx, id)
					}(j)
				}
				wg.Wait()
			}
		})

		b.Run(fmt.Sprintf("Write_%s", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(bm.concurrent)
				for j := 0; j < bm.concurrent; j++ {
					go func(idx int) {
						defer wg.Done()
						mem := &Memory{
							ID:        fmt.Sprintf("write-%d-%d", i, idx),
							Type:      "test",
							Content:   "Concurrent write test",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}
						mgr.StoreMemory(ctx, mem)
					}(j)
				}
				wg.Wait()
			}
		})

		b.Run(fmt.Sprintf("Mixed_%s", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(bm.concurrent)
				for j := 0; j < bm.concurrent; j++ {
					go func(idx int) {
						defer wg.Done()
						if idx%2 == 0 {
							id := fmt.Sprintf("concurrent-%d", idx%1000)
							mgr.GetMemory(ctx, id)
						} else {
							mem := &Memory{
								ID:        fmt.Sprintf("mixed-%d-%d", i, idx),
								Type:      "test",
								Content:   "Mixed operation test",
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
							}
							mgr.StoreMemory(ctx, mem)
						}
					}(j)
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkMemorySearch(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_search_memory.db")

	mgr, err := NewMemoryManager(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()

	topics := []string{"golang", "python", "javascript", "database", "api", "testing"}
	for i := 0; i < 10000; i++ {
		topic := topics[i%len(topics)]
		mem := &Memory{
			ID:      fmt.Sprintf("search-%d", i),
			Type:    "conversation",
			Content: fmt.Sprintf("This is a memory about %s with index %d", topic, i),
			Metadata: map[string]interface{}{
				"topic": topic,
				"index": i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mgr.StoreMemory(ctx, mem)
	}

	b.Run("SearchSmallResult", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mgr.SearchMemories(ctx, "golang", 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SearchMediumResult", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mgr.SearchMemories(ctx, "golang", 100)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SearchLargeResult", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mgr.SearchMemories(ctx, "golang", 1000)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Ensure os is used (for compatibility)
var _ = os.Stderr
