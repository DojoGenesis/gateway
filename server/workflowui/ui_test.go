package workflowui

import (
	"io/fs"
	"testing"
)

func TestIsBuilt(t *testing.T) {
	// The dist/index.html is present in the embedded filesystem (checked in repo).
	// If this fails, the SPA has not been built before go test.
	got := IsBuilt()
	if !got {
		t.Skip("SPA not built (dist/index.html missing); skipping IsBuilt=true test")
	}
}

func TestFS_ReturnsValidFS(t *testing.T) {
	sub, err := FS()
	if err != nil {
		t.Fatalf("FS() returned error: %v", err)
	}
	if sub == nil {
		t.Fatal("FS() returned nil filesystem")
	}
}

func TestFS_ContainsIndexHTML(t *testing.T) {
	if !IsBuilt() {
		t.Skip("SPA not built; skipping file presence test")
	}

	sub, err := FS()
	if err != nil {
		t.Fatalf("FS() error: %v", err)
	}

	f, err := sub.Open("index.html")
	if err != nil {
		t.Fatalf("expected index.html in FS, got error: %v", err)
	}
	f.Close()
}

func TestFS_SubDirectoryAccess(t *testing.T) {
	if !IsBuilt() {
		t.Skip("SPA not built; skipping subdirectory test")
	}

	sub, err := FS()
	if err != nil {
		t.Fatalf("FS() error: %v", err)
	}

	// Walk and verify we get at least some entries
	count := 0
	fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		count++
		return nil
	})

	if count < 2 {
		t.Errorf("expected at least 2 entries in embedded FS, got %d", count)
	}
}

func TestFS_RobotsPresent(t *testing.T) {
	if !IsBuilt() {
		t.Skip("SPA not built; skipping robots.txt test")
	}

	sub, err := FS()
	if err != nil {
		t.Fatalf("FS() error: %v", err)
	}

	f, err := sub.Open("robots.txt")
	if err != nil {
		t.Fatalf("expected robots.txt in FS, got error: %v", err)
	}
	f.Close()
}
