package apps

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAppRegistry_LaunchAndGet(t *testing.T) {
	r := NewAppRegistry()

	inst, err := r.Launch("ui://test/app.html", "session-1")
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}

	if inst.ID == "" {
		t.Fatal("instance ID is empty")
	}
	if inst.ResourceURI != "ui://test/app.html" {
		t.Errorf("ResourceURI = %q, want %q", inst.ResourceURI, "ui://test/app.html")
	}
	if inst.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want %q", inst.SessionID, "session-1")
	}
	if inst.LaunchedAt.IsZero() {
		t.Error("LaunchedAt is zero")
	}
	if inst.Metadata == nil {
		t.Error("Metadata is nil")
	}

	got, err := r.Get(inst.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != inst.ID {
		t.Errorf("Get returned ID %q, want %q", got.ID, inst.ID)
	}
}

func TestAppRegistry_Close(t *testing.T) {
	r := NewAppRegistry()

	inst, err := r.Launch("ui://test/app.html", "session-1")
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}

	if err := r.Close(inst.ID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = r.Get(inst.ID)
	if err == nil {
		t.Fatal("expected error after Close, got nil")
	}

	if r.Count() != 0 {
		t.Errorf("Count = %d after close, want 0", r.Count())
	}
}

func TestAppRegistry_ListBySession(t *testing.T) {
	r := NewAppRegistry()

	// Launch 3 apps for session-1 and 2 for session-2
	for i := 0; i < 3; i++ {
		if _, err := r.Launch(fmt.Sprintf("ui://s1/app%d.html", i), "session-1"); err != nil {
			t.Fatalf("Launch s1 failed: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := r.Launch(fmt.Sprintf("ui://s2/app%d.html", i), "session-2"); err != nil {
			t.Fatalf("Launch s2 failed: %v", err)
		}
	}

	s1Apps := r.ListBySession("session-1")
	if len(s1Apps) != 3 {
		t.Errorf("ListBySession(session-1) = %d, want 3", len(s1Apps))
	}

	s2Apps := r.ListBySession("session-2")
	if len(s2Apps) != 2 {
		t.Errorf("ListBySession(session-2) = %d, want 2", len(s2Apps))
	}

	noApps := r.ListBySession("session-unknown")
	if len(noApps) != 0 {
		t.Errorf("ListBySession(unknown) = %d, want 0", len(noApps))
	}
}

func TestAppRegistry_UpdateActivity(t *testing.T) {
	r := NewAppRegistry()

	inst, err := r.Launch("ui://test/app.html", "session-1")
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}

	originalActive := inst.LastActive
	time.Sleep(2 * time.Millisecond)

	if err := r.UpdateActivity(inst.ID); err != nil {
		t.Fatalf("UpdateActivity failed: %v", err)
	}

	got, _ := r.Get(inst.ID)
	if !got.LastActive.After(originalActive) {
		t.Error("LastActive was not updated")
	}
}

func TestAppRegistry_CloseNotFound(t *testing.T) {
	r := NewAppRegistry()

	err := r.Close("nonexistent-id")
	if err == nil {
		t.Fatal("expected error closing non-existent instance, got nil")
	}
}

func TestAppRegistry_LaunchEmptyFields(t *testing.T) {
	r := NewAppRegistry()

	_, err := r.Launch("", "session-1")
	if err == nil {
		t.Fatal("expected error with empty resourceURI")
	}

	_, err = r.Launch("ui://test/app.html", "")
	if err == nil {
		t.Fatal("expected error with empty sessionID")
	}
}

func TestAppRegistry_Count(t *testing.T) {
	r := NewAppRegistry()

	if r.Count() != 0 {
		t.Errorf("Count = %d, want 0", r.Count())
	}

	inst1, _ := r.Launch("ui://a/1.html", "s1")
	inst2, _ := r.Launch("ui://a/2.html", "s1")

	if r.Count() != 2 {
		t.Errorf("Count = %d, want 2", r.Count())
	}

	_ = r.Close(inst1.ID)
	if r.Count() != 1 {
		t.Errorf("Count = %d after close, want 1", r.Count())
	}

	_ = r.Close(inst2.ID)
	if r.Count() != 0 {
		t.Errorf("Count = %d after close, want 0", r.Count())
	}
}

func TestAppRegistry_ConcurrentAccess(t *testing.T) {
	r := NewAppRegistry()

	var wg sync.WaitGroup
	const n = 100

	ids := make([]string, n)
	var mu sync.Mutex

	// Concurrent launches
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			inst, err := r.Launch(fmt.Sprintf("ui://concurrent/%d.html", i), "session-1")
			if err != nil {
				t.Errorf("concurrent Launch %d failed: %v", i, err)
				return
			}
			mu.Lock()
			ids[i] = inst.ID
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	if r.Count() != n {
		t.Errorf("Count = %d, want %d", r.Count(), n)
	}

	// Concurrent reads
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mu.Lock()
			id := ids[i]
			mu.Unlock()
			if id == "" {
				return
			}
			_, _ = r.Get(id)
		}(i)
	}
	wg.Wait()

	// Concurrent list
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.ListBySession("session-1")
		}()
	}
	wg.Wait()
}
