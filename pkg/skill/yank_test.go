package skill

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

// installTestSkill installs a minimal skill into the store for testing.
func installTestSkill(t *testing.T, store *SkillStore, name, version string) {
	t.Helper()
	manifest := SkillManifest{
		Name:        name,
		Version:     version,
		Description: "test skill for " + name,
	}
	configBlob, _ := json.Marshal(manifest)
	contentTar := buildMinimalTar(t)
	if err := store.Install(context.Background(), manifest, configBlob, contentTar); err != nil {
		t.Fatalf("install test skill %s@%s: %v", name, version, err)
	}
}

// buildMinimalTar creates a minimal valid tar archive for testing.
func buildMinimalTar(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte("test\n")
	if err := tw.WriteHeader(&tar.Header{Name: "SKILL.md", Size: int64(len(content))}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return buf.Bytes()
}

func TestYank_SetsFields(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	installTestSkill(t, store, "yank-test", "1.0.0")

	// Yank the skill.
	if err := store.Yank(ctx, "yank-test", "1.0.0", "security vulnerability"); err != nil {
		t.Fatalf("Yank: %v", err)
	}

	// Verify the skill is yanked by reading it back as PluginManifest.
	m, err := store.Get(ctx, "yank-test", "1.0.0")
	if err != nil {
		t.Fatalf("Get after yank: %v", err)
	}

	// Re-unmarshal the raw blob as PluginManifest.
	ref, _ := casStore.Resolve(ctx, "skill/yank-test:config", "1.0.0")
	data, _, _ := casStore.Get(ctx, ref)
	var pm PluginManifest
	if err := json.Unmarshal(data, &pm); err != nil {
		t.Fatalf("unmarshal PluginManifest: %v", err)
	}

	if !pm.Yanked {
		t.Error("expected Yanked=true after yank")
	}
	if pm.YankReason != "security vulnerability" {
		t.Errorf("YankReason = %q, want %q", pm.YankReason, "security vulnerability")
	}
	if m.Name != "yank-test" {
		t.Errorf("Name = %q, want %q", m.Name, "yank-test")
	}
}

func TestUnYank_ClearsFields(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	installTestSkill(t, store, "unyank-test", "1.0.0")

	if err := store.Yank(ctx, "unyank-test", "1.0.0", "broken"); err != nil {
		t.Fatalf("Yank: %v", err)
	}

	if err := store.UnYank(ctx, "unyank-test", "1.0.0"); err != nil {
		t.Fatalf("UnYank: %v", err)
	}

	ref, _ := casStore.Resolve(ctx, "skill/unyank-test:config", "1.0.0")
	data, _, _ := casStore.Get(ctx, ref)
	var pm PluginManifest
	if err := json.Unmarshal(data, &pm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if pm.Yanked {
		t.Error("expected Yanked=false after unyank")
	}
	if pm.YankReason != "" {
		t.Errorf("YankReason = %q, want empty", pm.YankReason)
	}
}

func TestYank_NotFound(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	err := store.Yank(ctx, "nonexistent", "1.0.0", "reason")
	if err == nil {
		t.Fatal("expected error for nonexistent skill, got nil")
	}
}

func TestYank_EmptyReason(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	installTestSkill(t, store, "empty-reason-test", "1.0.0")

	err := store.Yank(ctx, "empty-reason-test", "1.0.0", "")
	if err == nil {
		t.Fatal("expected error for empty reason, got nil")
	}
}

func TestYank_AlreadyYanked(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	installTestSkill(t, store, "double-yank", "1.0.0")

	if err := store.Yank(ctx, "double-yank", "1.0.0", "first"); err != nil {
		t.Fatalf("first Yank: %v", err)
	}
	if err := store.Yank(ctx, "double-yank", "1.0.0", "second"); err == nil {
		t.Fatal("expected error for double yank, got nil")
	}
}

func TestUnYank_NotYanked(t *testing.T) {
	casStore := newTestStore(t)
	store := NewSkillStore(casStore)
	ctx := context.Background()

	installTestSkill(t, store, "not-yanked", "1.0.0")

	err := store.UnYank(ctx, "not-yanked", "1.0.0")
	if err == nil {
		t.Fatal("expected error for unyanking non-yanked skill, got nil")
	}
}
