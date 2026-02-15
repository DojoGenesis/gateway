package apps

import (
	"fmt"
	"sync"
	"testing"
)

func TestResourceRegistry_RegisterAndGet(t *testing.T) {
	r := NewResourceRegistry()

	meta := &ResourceMeta{
		URI:      "ui://test/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>test</html>"),
		CacheKey: "test-key",
	}

	if err := r.Register(meta); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := r.Get("ui://test/app.html")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.URI != meta.URI {
		t.Errorf("URI = %q, want %q", got.URI, meta.URI)
	}
	if got.MimeType != meta.MimeType {
		t.Errorf("MimeType = %q, want %q", got.MimeType, meta.MimeType)
	}
	if string(got.Content) != string(meta.Content) {
		t.Errorf("Content = %q, want %q", got.Content, meta.Content)
	}
	if got.CacheKey != meta.CacheKey {
		t.Errorf("CacheKey = %q, want %q", got.CacheKey, meta.CacheKey)
	}
}

func TestResourceRegistry_DuplicateRegistration(t *testing.T) {
	r := NewResourceRegistry()

	meta := &ResourceMeta{URI: "ui://test/dup.html", MimeType: "text/html", Content: []byte("a")}
	if err := r.Register(meta); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := r.Register(meta)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
}

func TestResourceRegistry_GetNotFound(t *testing.T) {
	r := NewResourceRegistry()

	_, err := r.Get("ui://nonexistent/app.html")
	if err == nil {
		t.Fatal("expected error for non-existent resource, got nil")
	}
}

func TestResourceRegistry_List(t *testing.T) {
	r := NewResourceRegistry()

	uris := []string{"ui://a/1.html", "ui://b/2.html", "ui://c/3.html"}
	for _, uri := range uris {
		if err := r.Register(&ResourceMeta{URI: uri, Content: []byte("x")}); err != nil {
			t.Fatalf("Register %s failed: %v", uri, err)
		}
	}

	listed := r.List()
	if len(listed) != len(uris) {
		t.Fatalf("List returned %d items, want %d", len(listed), len(uris))
	}

	uriSet := make(map[string]bool)
	for _, u := range listed {
		uriSet[u] = true
	}
	for _, want := range uris {
		if !uriSet[want] {
			t.Errorf("List missing URI %q", want)
		}
	}
}

func TestResourceRegistry_Unregister(t *testing.T) {
	r := NewResourceRegistry()

	meta := &ResourceMeta{URI: "ui://test/remove.html", Content: []byte("bye")}
	if err := r.Register(meta); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := r.Unregister("ui://test/remove.html"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	_, err := r.Get("ui://test/remove.html")
	if err == nil {
		t.Fatal("expected error after Unregister, got nil")
	}

	// Unregister again should fail
	err = r.Unregister("ui://test/remove.html")
	if err == nil {
		t.Fatal("expected error on second Unregister, got nil")
	}
}

func TestResourceRegistry_RegisterNil(t *testing.T) {
	r := NewResourceRegistry()
	if err := r.Register(nil); err == nil {
		t.Fatal("expected error registering nil, got nil")
	}
}

func TestResourceRegistry_RegisterEmptyURI(t *testing.T) {
	r := NewResourceRegistry()
	if err := r.Register(&ResourceMeta{URI: "", Content: []byte("x")}); err == nil {
		t.Fatal("expected error registering empty URI, got nil")
	}
}

func TestResourceRegistry_ConcurrentAccess(t *testing.T) {
	r := NewResourceRegistry()

	var wg sync.WaitGroup
	const n = 100

	// Concurrent writes
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			uri := fmt.Sprintf("ui://concurrent/%d.html", i)
			_ = r.Register(&ResourceMeta{URI: uri, Content: []byte("data")})
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			uri := fmt.Sprintf("ui://concurrent/%d.html", i)
			_, _ = r.Get(uri)
		}(i)
	}
	wg.Wait()

	// Concurrent list
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.List()
		}()
	}
	wg.Wait()

	listed := r.List()
	if len(listed) != n {
		t.Errorf("after concurrent writes: got %d resources, want %d", len(listed), n)
	}
}
