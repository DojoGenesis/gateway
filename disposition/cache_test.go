package disposition

import (
	"testing"
	"time"
)

func TestDispositionCache_SetAndGet(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	cfg := DefaultDisposition()
	key := makeCacheKey("/workspace", "prod")

	cache.Set(key, cfg)

	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached config")
	}

	if retrieved.Pacing != cfg.Pacing {
		t.Errorf("Expected pacing '%s', got '%s'", cfg.Pacing, retrieved.Pacing)
	}
}

func TestDispositionCache_NotFound(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	_, found := cache.Get("nonexistent_key")
	if found {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestDispositionCache_TTLExpiration(t *testing.T) {
	cache := NewDispositionCache(100 * time.Millisecond)

	cfg := DefaultDisposition()
	key := makeCacheKey("/workspace", "debug")

	cache.Set(key, cfg)

	// Should be found immediately
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find cached config immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get(key)
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestDispositionCache_Clear(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	cfg1 := DefaultDisposition()
	cfg2 := DefaultDisposition()
	cfg2.Pacing = "rapid" // Make it different from cfg1

	cache.Set("key1", cfg1)
	cache.Set("key2", cfg2)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected not to find key1 after clear")
	}
}

func TestDispositionCache_Delete(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	cfg := DefaultDisposition()
	key := makeCacheKey("/workspace", "test")

	cache.Set(key, cfg)

	deleted := cache.Delete(key)
	if !deleted {
		t.Error("Expected Delete to return true")
	}

	_, found := cache.Get(key)
	if found {
		t.Error("Expected not to find deleted key")
	}

	// Try deleting again
	deleted = cache.Delete(key)
	if deleted {
		t.Error("Expected Delete to return false for nonexistent key")
	}
}

func TestDispositionCache_Size(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", DefaultDisposition())
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", DefaultDisposition())
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}

	cache.Delete("key1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1 after delete, got %d", cache.Size())
	}
}

func TestDispositionCache_MultipleKeys(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	cfg1 := DefaultDisposition()
	cfg1.Pacing = "deliberate"
	cfg2 := DefaultDisposition()
	cfg2.Pacing = "rapid"

	cache.Set("/workspace:base", cfg1)
	cache.Set("/workspace:prod", cfg2)

	retrieved1, found := cache.Get("/workspace:base")
	if !found || retrieved1.Pacing != "deliberate" {
		t.Error("Failed to retrieve first cached config")
	}

	retrieved2, found := cache.Get("/workspace:prod")
	if !found || retrieved2.Pacing != "rapid" {
		t.Error("Failed to retrieve second cached config")
	}
}

func TestMakeCacheKey(t *testing.T) {
	tests := []struct {
		workspace   string
		mode        string
		expectedKey string
	}{
		{"/path/to/workspace", "prod", "/path/to/workspace:prod"},
		{"/path/to/workspace", "", "/path/to/workspace:base"},
		{"/another/workspace", "debug", "/another/workspace:debug"},
	}

	for _, tt := range tests {
		key := makeCacheKey(tt.workspace, tt.mode)
		if key != tt.expectedKey {
			t.Errorf("makeCacheKey(%q, %q) = %q, expected %q", tt.workspace, tt.mode, key, tt.expectedKey)
		}
	}
}

func TestDispositionCache_ConcurrentAccess(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			cfg := DefaultDisposition()
			cache.Set(makeCacheKey("/workspace", string(rune(id))), cfg)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			cache.Get(makeCacheKey("/workspace", string(rune(id))))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestDispositionCache_ZeroTTL(t *testing.T) {
	cache := NewDispositionCache(0) // Never expire

	cfg := DefaultDisposition()
	key := "test_key"

	cache.Set(key, cfg)

	time.Sleep(100 * time.Millisecond)

	// Should still be found (no expiration)
	_, found := cache.Get(key)
	if !found {
		t.Error("Expected to find entry with zero TTL")
	}
}

func TestDispositionCache_SetOverwrite(t *testing.T) {
	cache := NewDispositionCache(5 * time.Minute)

	cfg1 := DefaultDisposition()
	cfg1.Pacing = "deliberate"

	cfg2 := DefaultDisposition()
	cfg2.Pacing = "rapid"

	key := "overwrite_key"

	cache.Set(key, cfg1)
	cache.Set(key, cfg2) // Overwrite

	retrieved, found := cache.Get(key)
	if !found {
		t.Fatal("Expected to find overwritten key")
	}

	if retrieved.Pacing != "rapid" {
		t.Errorf("Expected pacing 'rapid', got '%s'", retrieved.Pacing)
	}
}

func TestMakeCacheKey_Consistency(t *testing.T) {
	key1 := makeCacheKey("/workspace", "prod")
	key2 := makeCacheKey("/workspace", "prod")

	if key1 != key2 {
		t.Errorf("Expected consistent cache keys, got %s and %s", key1, key2)
	}

	key3 := makeCacheKey("/workspace", "debug")
	if key1 == key3 {
		t.Error("Expected different keys for different modes")
	}
}
