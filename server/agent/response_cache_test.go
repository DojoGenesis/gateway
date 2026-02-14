package agent

import (
	"testing"
	"time"
)

func TestResponseCache_SetGet(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 100)
	defer cache.Clear()

	query := "hello"
	response := "Hello! How can I help you today?"

	cache.Set(query, response)

	got, found := cache.Get(query)
	if !found {
		t.Fatal("Expected to find cached response")
	}

	if got != response {
		t.Errorf("Got %q, want %q", got, response)
	}
}

func TestResponseCache_Miss(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 100)
	defer cache.Clear()

	_, found := cache.Get("nonexistent query")
	if found {
		t.Error("Expected cache miss")
	}
}

func TestResponseCache_TTL(t *testing.T) {
	cache := NewResponseCache(100*time.Millisecond, 100)
	defer cache.Clear()

	cache.Set("test", "response")

	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get("test")
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestResponseCache_MaxSize(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 5)
	defer cache.Clear()

	for i := 0; i < 10; i++ {
		cache.Set(string(rune('a'+i)), "response")
	}

	size := cache.Size()
	if size > 5 {
		t.Errorf("Cache size %d exceeds max size 5", size)
	}
}

func TestResponseCache_Stats(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 100)
	defer cache.Clear()

	cache.Set("hello", "response1")
	cache.Set("help", "response2")

	cache.Get("hello")
	cache.Get("hello")
	cache.Get("help")
	cache.Get("nonexistent")

	hits, misses, hitRate := cache.Stats()

	if hits != 3 {
		t.Errorf("Expected 3 hits, got %d", hits)
	}

	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}

	expectedRate := 0.75
	if hitRate != expectedRate {
		t.Errorf("Expected hit rate %.2f, got %.2f", expectedRate, hitRate)
	}
}

func TestResponseCache_Disable(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 100)
	defer cache.Clear()

	cache.Set("hello", "response")
	cache.Disable()

	_, found := cache.Get("hello")
	if found {
		t.Error("Expected cache to be disabled")
	}

	cache.Enable()
	_, found = cache.Get("hello")
	if !found {
		t.Error("Expected cache to be re-enabled")
	}
}

func TestResponseCache_Clear(t *testing.T) {
	cache := NewResponseCache(1*time.Hour, 100)

	cache.Set("hello", "response1")
	cache.Set("help", "response2")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache to be empty, got size %d", cache.Size())
	}

	hits, misses, _ := cache.Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("Expected stats to be reset, got hits=%d misses=%d", hits, misses)
	}
}

func BenchmarkResponseCache_Get(b *testing.B) {
	cache := NewResponseCache(1*time.Hour, 1000)
	defer cache.Clear()

	cache.Set("hello", "Hello! How can I help you?")
	cache.Set("help", "I can assist with various tasks")
	cache.Set("goodbye", "Goodbye! Have a great day!")

	queries := []string{"hello", "help", "goodbye", "nonexistent"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		cache.Get(query)
	}
}

func BenchmarkResponseCache_Set(b *testing.B) {
	cache := NewResponseCache(1*time.Hour, 10000)
	defer cache.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("query", "response")
	}
}

func BenchmarkResponseCache_SetGet(b *testing.B) {
	cache := NewResponseCache(1*time.Hour, 1000)
	defer cache.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("query", "response")
		cache.Get("query")
	}
}
