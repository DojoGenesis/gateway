package skill

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// newTestStore creates a temporary SQLite-backed CAS store for testing.
func newTestStore(t *testing.T) cas.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test-cas.db")
	store, err := cas.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// createSkillDir creates a temporary directory with SKILL.md for testing.
func createSkillDir(t *testing.T, frontmatter, body string) string {
	t.Helper()
	dir := t.TempDir()

	content := "---\n" + frontmatter + "\n---\n\n" + body
	err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Add a supplementary file to verify tar includes it.
	err = os.WriteFile(filepath.Join(dir, "helpers.md"), []byte("# Helpers\n\nSome helper content."), 0644)
	if err != nil {
		t.Fatalf("failed to write helpers.md: %v", err)
	}

	return dir
}

// --- PackSkill tests ---

func TestPackSkill_BasicManifest(t *testing.T) {
	frontmatter := `name: strategic-scout
version: 2.1.0
description: Explore strategic tensions
license: MIT`

	dir := createSkillDir(t, frontmatter, "# Strategic Scout\n\nA skill for scouting.")

	manifest, configBlob, contentTar, err := PackSkill(dir)
	if err != nil {
		t.Fatalf("PackSkill failed: %v", err)
	}

	// Verify manifest fields.
	if manifest.Name != "strategic-scout" {
		t.Errorf("expected name %q, got %q", "strategic-scout", manifest.Name)
	}
	if manifest.Version != "2.1.0" {
		t.Errorf("expected version %q, got %q", "2.1.0", manifest.Version)
	}
	if manifest.Description != "Explore strategic tensions" {
		t.Errorf("expected description %q, got %q", "Explore strategic tensions", manifest.Description)
	}
	if manifest.License != "MIT" {
		t.Errorf("expected license %q, got %q", "MIT", manifest.License)
	}

	// Verify configBlob is valid JSON that round-trips.
	var decoded SkillManifest
	if err := json.Unmarshal(configBlob, &decoded); err != nil {
		t.Fatalf("configBlob is not valid JSON: %v", err)
	}
	if decoded.Name != manifest.Name {
		t.Errorf("JSON round-trip name mismatch: %q vs %q", decoded.Name, manifest.Name)
	}

	// Verify contentTar is a valid tar archive containing our files.
	tarFiles := listTarFiles(t, contentTar)
	if !containsFile(tarFiles, "SKILL.md") {
		t.Error("tar archive missing SKILL.md")
	}
	if !containsFile(tarFiles, "helpers.md") {
		t.Error("tar archive missing helpers.md")
	}
}

func TestPackSkill_DefaultVersion(t *testing.T) {
	frontmatter := `name: my-skill
description: A test skill`

	dir := createSkillDir(t, frontmatter, "# Test")

	manifest, _, _, err := PackSkill(dir)
	if err != nil {
		t.Fatalf("PackSkill failed: %v", err)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("expected default version %q, got %q", "1.0.0", manifest.Version)
	}
}

func TestPackSkill_WithTriggers(t *testing.T) {
	frontmatter := `name: debug-skill
version: 1.0.0
description: Debug things
triggers:
- "something is broken"
- "debug this"
- "troubleshoot"`

	dir := createSkillDir(t, frontmatter, "# Debug\n\nDebugging skill.")

	manifest, _, _, err := PackSkill(dir)
	if err != nil {
		t.Fatalf("PackSkill failed: %v", err)
	}

	if len(manifest.Triggers) != 3 {
		t.Fatalf("expected 3 triggers, got %d: %v", len(manifest.Triggers), manifest.Triggers)
	}
	expected := []string{"something is broken", "debug this", "troubleshoot"}
	for i, want := range expected {
		if manifest.Triggers[i] != want {
			t.Errorf("trigger[%d] = %q, want %q", i, manifest.Triggers[i], want)
		}
	}
}

func TestPackSkill_WithDependencies(t *testing.T) {
	frontmatter := `name: advanced-skill
version: 1.0.0
description: Needs other skills
dependencies:
- strategic-scout
- debug-skill`

	dir := createSkillDir(t, frontmatter, "# Advanced")

	manifest, _, _, err := PackSkill(dir)
	if err != nil {
		t.Fatalf("PackSkill failed: %v", err)
	}

	if len(manifest.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(manifest.Dependencies))
	}
	if manifest.Dependencies[0] != "strategic-scout" {
		t.Errorf("dep[0] = %q, want %q", manifest.Dependencies[0], "strategic-scout")
	}
}

func TestPackSkill_MissingDir(t *testing.T) {
	_, _, _, err := PackSkill("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing directory")
	}
}

func TestPackSkill_MissingSkillMd(t *testing.T) {
	dir := t.TempDir()
	_, _, _, err := PackSkill(dir)
	if err == nil {
		t.Error("expected error for missing SKILL.md")
	}
}

func TestPackSkill_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# No frontmatter here"), 0644)
	_, _, _, err := PackSkill(dir)
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

// --- Resolver tests ---

func TestResolve_SkillScheme(t *testing.T) {
	tests := []struct {
		ref      string
		wantReg  string
		wantPath string
		wantTag  string
	}{
		{"skill://strategic-scout@1.0.0", "ghcr.io", "dojo-skills/strategic-scout", "1.0.0"},
		{"skill://my-skill@latest", "ghcr.io", "dojo-skills/my-skill", "latest"},
		{"skill://simple-skill", "ghcr.io", "dojo-skills/simple-skill", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			r, err := Resolve(tt.ref)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.ref, err)
			}
			if r.Scheme != "skill" {
				t.Errorf("scheme = %q, want %q", r.Scheme, "skill")
			}
			if r.Registry != tt.wantReg {
				t.Errorf("registry = %q, want %q", r.Registry, tt.wantReg)
			}
			if r.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", r.Path, tt.wantPath)
			}
			if r.Tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", r.Tag, tt.wantTag)
			}
		})
	}
}

func TestResolve_OCIScheme(t *testing.T) {
	tests := []struct {
		ref      string
		wantReg  string
		wantPath string
		wantTag  string
	}{
		{"oci://ghcr.io/dojo-skills/strategic-scout:1.0.0", "ghcr.io", "dojo-skills/strategic-scout", "1.0.0"},
		{"oci://registry.example.com/org/skill:latest", "registry.example.com", "org/skill", "latest"},
		{"oci://docker.io/library/skill", "docker.io", "library/skill", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			r, err := Resolve(tt.ref)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.ref, err)
			}
			if r.Scheme != "oci" {
				t.Errorf("scheme = %q, want %q", r.Scheme, "oci")
			}
			if r.Registry != tt.wantReg {
				t.Errorf("registry = %q, want %q", r.Registry, tt.wantReg)
			}
			if r.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", r.Path, tt.wantPath)
			}
			if r.Tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", r.Tag, tt.wantTag)
			}
		})
	}
}

func TestResolve_GitHubScheme(t *testing.T) {
	tests := []struct {
		ref      string
		wantReg  string
		wantPath string
		wantTag  string
	}{
		{"github:dojo-org/skills//strategic-scout", "github.com", "dojo-org/skills/strategic-scout", "main"},
		{"github:org/repo//path/to/skill@v2.0.0", "github.com", "org/repo/path/to/skill", "v2.0.0"},
		{"github:org/repo", "github.com", "org/repo", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			r, err := Resolve(tt.ref)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.ref, err)
			}
			if r.Scheme != "github" {
				t.Errorf("scheme = %q, want %q", r.Scheme, "github")
			}
			if r.Registry != tt.wantReg {
				t.Errorf("registry = %q, want %q", r.Registry, tt.wantReg)
			}
			if r.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", r.Path, tt.wantPath)
			}
			if r.Tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", r.Tag, tt.wantTag)
			}
		})
	}
}

func TestResolve_BareReference(t *testing.T) {
	// Bare references (no scheme) should be treated as skill://.
	r, err := Resolve("my-skill@2.0.0")
	if err != nil {
		t.Fatalf("Resolve bare ref error: %v", err)
	}
	if r.Scheme != "skill" {
		t.Errorf("scheme = %q, want %q", r.Scheme, "skill")
	}
	if r.Path != "dojo-skills/my-skill" {
		t.Errorf("path = %q, want %q", r.Path, "dojo-skills/my-skill")
	}
	if r.Tag != "2.0.0" {
		t.Errorf("tag = %q, want %q", r.Tag, "2.0.0")
	}
}

func TestResolve_Errors(t *testing.T) {
	badRefs := []string{
		"",
		"oci://",
		"oci://registry-only",
		"github:",
		"github:no-slash",
	}
	for _, ref := range badRefs {
		t.Run(ref, func(t *testing.T) {
			_, err := Resolve(ref)
			if err == nil {
				t.Errorf("Resolve(%q) should have returned an error", ref)
			}
		})
	}
}

// --- SkillStore tests ---

func TestSkillStore_InstallListGetRoundtrip(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	manifest := SkillManifest{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "A test skill for unit testing",
		Triggers:    []string{"test this"},
		License:     "MIT",
		Authors:     []string{"tester"},
	}
	configBlob, _ := json.Marshal(manifest)
	contentTar := makeTar(t, map[string]string{"SKILL.md": "# Test Skill"})

	// Install.
	err := ss.Install(ctx, manifest, configBlob, contentTar)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// List.
	manifests, err := ss.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(manifests))
	}
	if manifests[0].Name != "test-skill" {
		t.Errorf("listed name = %q, want %q", manifests[0].Name, "test-skill")
	}
	if manifests[0].Version != "1.0.0" {
		t.Errorf("listed version = %q, want %q", manifests[0].Version, "1.0.0")
	}

	// Get.
	got, err := ss.Get(ctx, "test-skill", "1.0.0")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Description != "A test skill for unit testing" {
		t.Errorf("description = %q, want %q", got.Description, "A test skill for unit testing")
	}
	if len(got.Triggers) != 1 || got.Triggers[0] != "test this" {
		t.Errorf("triggers = %v, want [test this]", got.Triggers)
	}

	// GetContent.
	content, err := ss.GetContent(ctx, "test-skill", "1.0.0")
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}
	if !bytes.Equal(content, contentTar) {
		t.Error("content tar mismatch")
	}
}

func TestSkillStore_MultipleVersions(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	for _, ver := range []string{"1.0.0", "2.0.0"} {
		m := SkillManifest{
			Name:        "versioned-skill",
			Version:     ver,
			Description: "Version " + ver,
		}
		blob, _ := json.Marshal(m)
		tarData := makeTar(t, map[string]string{"SKILL.md": "# Version " + ver})
		if err := ss.Install(ctx, m, blob, tarData); err != nil {
			t.Fatalf("Install %s failed: %v", ver, err)
		}
	}

	// Get each version.
	m1, err := ss.Get(ctx, "versioned-skill", "1.0.0")
	if err != nil {
		t.Fatalf("Get 1.0.0 failed: %v", err)
	}
	if m1.Description != "Version 1.0.0" {
		t.Errorf("1.0.0 description = %q", m1.Description)
	}

	m2, err := ss.Get(ctx, "versioned-skill", "2.0.0")
	if err != nil {
		t.Fatalf("Get 2.0.0 failed: %v", err)
	}
	if m2.Description != "Version 2.0.0" {
		t.Errorf("2.0.0 description = %q", m2.Description)
	}
}

func TestSkillStore_GetNotFound(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	_, err := ss.Get(ctx, "nonexistent", "1.0.0")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestSkillStore_InstallValidation(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	validTar := makeTar(t, map[string]string{"SKILL.md": "# Test"})

	// Missing name.
	err := ss.Install(ctx, SkillManifest{Version: "1.0.0"}, []byte(`{"version":"1.0.0"}`), validTar)
	if err == nil {
		t.Error("expected error for missing name")
	}

	// Missing version.
	err = ss.Install(ctx, SkillManifest{Name: "test"}, []byte(`{"name":"test"}`), validTar)
	if err == nil {
		t.Error("expected error for missing version")
	}

	// E30: Invalid tar archive.
	m := SkillManifest{Name: "test", Version: "1.0.0"}
	blob, _ := json.Marshal(m)
	err = ss.Install(ctx, m, blob, []byte("not-a-tar-archive"))
	if err == nil {
		t.Error("expected error for invalid tar archive")
	}
	if !strings.Contains(err.Error(), "invalid tar archive") {
		t.Errorf("expected 'invalid tar archive' in error, got: %v", err)
	}

	// E31: Config blob name/version mismatch.
	wrongBlob, _ := json.Marshal(SkillManifest{Name: "wrong-name", Version: "9.9.9"})
	err = ss.Install(ctx, m, wrongBlob, validTar)
	if err == nil {
		t.Error("expected error for config blob mismatch")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("expected 'mismatch' in error, got: %v", err)
	}

	// E31: Config blob is not valid JSON.
	err = ss.Install(ctx, m, []byte("not json"), validTar)
	if err == nil {
		t.Error("expected error for invalid config blob JSON")
	}
	if !strings.Contains(err.Error(), "invalid config blob") {
		t.Errorf("expected 'invalid config blob' in error, got: %v", err)
	}
}

// --- CASStorage (ORAS bridge) tests ---

func TestCASStorage_PushExistsFetchRoundtrip(t *testing.T) {
	casStore := newTestStore(t)
	ociStore := NewCASStorage(casStore)
	ctx := context.Background()

	data := []byte("hello, OCI world")
	mediaType := ConfigMediaType

	// Push.
	digest, err := ociStore.Push(ctx, data, mediaType)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify digest format.
	if len(digest) < 7 || digest[:7] != "sha256:" {
		t.Fatalf("unexpected digest format: %s", digest)
	}

	// Verify digest matches expected hash.
	expectedDigest := Digest(data)
	if digest != expectedDigest {
		t.Errorf("digest = %s, want %s", digest, expectedDigest)
	}

	// Exists.
	exists, err := ociStore.Exists(ctx, digest)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected content to exist after push")
	}

	// Fetch.
	fetched, err := ociStore.Fetch(ctx, digest)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if !bytes.Equal(fetched, data) {
		t.Error("fetched data does not match pushed data")
	}
}

func TestCASStorage_ExistsNotFound(t *testing.T) {
	casStore := newTestStore(t)
	ociStore := NewCASStorage(casStore)
	ctx := context.Background()

	exists, err := ociStore.Exists(ctx, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected content to not exist")
	}
}

func TestCASStorage_FetchNotFound(t *testing.T) {
	casStore := newTestStore(t)
	ociStore := NewCASStorage(casStore)
	ctx := context.Background()

	_, err := ociStore.Fetch(ctx, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestCASStorage_InvalidDigest(t *testing.T) {
	casStore := newTestStore(t)
	ociStore := NewCASStorage(casStore)
	ctx := context.Background()

	// Wrong algorithm.
	_, err := ociStore.Exists(ctx, "md5:abc123")
	if err == nil {
		t.Error("expected error for non-sha256 digest")
	}

	// Too short.
	_, err = ociStore.Exists(ctx, "sha256:short")
	if err == nil {
		t.Error("expected error for short digest")
	}
}

func TestCASStorage_TagAndResolve(t *testing.T) {
	casStore := newTestStore(t)
	ociStore := NewCASStorage(casStore)
	ctx := context.Background()

	data := []byte("tagged content")
	digest, err := ociStore.Push(ctx, data, "application/octet-stream")
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Tag.
	err = ociStore.Tag(ctx, "test/artifact", "1.0.0", digest)
	if err != nil {
		t.Fatalf("Tag failed: %v", err)
	}

	// Resolve.
	resolved, err := ociStore.Resolve(ctx, "test/artifact", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved != digest {
		t.Errorf("resolved digest = %s, want %s", resolved, digest)
	}
}

// --- Manifest serialization tests ---

func TestManifest_JSONRoundtrip(t *testing.T) {
	original := SkillManifest{
		Name:         "roundtrip-test",
		Version:      "3.2.1",
		Description:  "Testing JSON serialization",
		Triggers:     []string{"trigger one", "trigger two"},
		Dependencies: []string{"dep-a", "dep-b"},
		License:      "Apache-2.0",
		Authors:      []string{"Alice", "Bob"},
		Platform:     map[string]string{"os": "linux", "arch": "amd64"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SkillManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all fields survive the round-trip.
	if decoded.Name != original.Name {
		t.Errorf("name mismatch: %q vs %q", decoded.Name, original.Name)
	}
	if decoded.Version != original.Version {
		t.Errorf("version mismatch: %q vs %q", decoded.Version, original.Version)
	}
	if decoded.Description != original.Description {
		t.Errorf("description mismatch")
	}
	if len(decoded.Triggers) != len(original.Triggers) {
		t.Errorf("triggers length mismatch: %d vs %d", len(decoded.Triggers), len(original.Triggers))
	}
	if len(decoded.Dependencies) != len(original.Dependencies) {
		t.Errorf("dependencies length mismatch: %d vs %d", len(decoded.Dependencies), len(original.Dependencies))
	}
	if decoded.License != original.License {
		t.Errorf("license mismatch: %q vs %q", decoded.License, original.License)
	}
	if len(decoded.Authors) != len(original.Authors) {
		t.Errorf("authors length mismatch: %d vs %d", len(decoded.Authors), len(original.Authors))
	}
	if decoded.Platform["os"] != "linux" || decoded.Platform["arch"] != "amd64" {
		t.Errorf("platform mismatch: %v", decoded.Platform)
	}
}

func TestManifest_OmitEmpty(t *testing.T) {
	minimal := SkillManifest{
		Name:    "minimal",
		Version: "1.0.0",
	}

	data, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify omitempty fields are absent.
	var raw map[string]interface{}
	_ = json.Unmarshal(data, &raw)

	for _, key := range []string{"triggers", "dependencies", "license", "authors", "platform"} {
		if _, found := raw[key]; found {
			t.Errorf("expected field %q to be omitted for zero value", key)
		}
	}
}

// --- Media type constants tests ---

func TestMediaTypeConstants(t *testing.T) {
	if ArtifactType != "application/vnd.dojo.skill.v1" {
		t.Errorf("ArtifactType = %q", ArtifactType)
	}
	if ConfigMediaType != "application/vnd.dojo.skill.config.v1+json" {
		t.Errorf("ConfigMediaType = %q", ConfigMediaType)
	}
	if ContentMediaType != "application/vnd.dojo.skill.content.v1+tar" {
		t.Errorf("ContentMediaType = %q", ContentMediaType)
	}
}

// --- End-to-end: PackSkill -> Install -> Get ---

func TestEndToEnd_PackInstallGet(t *testing.T) {
	frontmatter := `name: e2e-skill
version: 1.5.0
description: End-to-end test skill
triggers:
- "run e2e"
- "test everything"
authors:
- "E2E Tester"
license: MIT`

	dir := createSkillDir(t, frontmatter, "# E2E Skill\n\nFull lifecycle test.")

	// Pack.
	manifest, configBlob, contentTar, err := PackSkill(dir)
	if err != nil {
		t.Fatalf("PackSkill failed: %v", err)
	}

	// Install into CAS.
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	err = ss.Install(ctx, manifest, configBlob, contentTar)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Retrieve and verify.
	got, err := ss.Get(ctx, "e2e-skill", "1.5.0")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != "e2e-skill" {
		t.Errorf("name = %q, want %q", got.Name, "e2e-skill")
	}
	if got.Version != "1.5.0" {
		t.Errorf("version = %q, want %q", got.Version, "1.5.0")
	}
	if len(got.Triggers) != 2 {
		t.Errorf("triggers = %d, want 2", len(got.Triggers))
	}
	if got.License != "MIT" {
		t.Errorf("license = %q, want %q", got.License, "MIT")
	}

	// Verify content tar is retrievable and contains expected files.
	content, err := ss.GetContent(ctx, "e2e-skill", "1.5.0")
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}
	tarFiles := listTarFiles(t, content)
	if !containsFile(tarFiles, "SKILL.md") {
		t.Error("content tar missing SKILL.md")
	}
	if !containsFile(tarFiles, "helpers.md") {
		t.Error("content tar missing helpers.md")
	}
}

// --- Digest function test ---

func TestDigest(t *testing.T) {
	data := []byte("test data")
	d := Digest(data)

	if d[:7] != "sha256:" {
		t.Errorf("digest missing sha256: prefix: %s", d)
	}
	if len(d) != 71 { // "sha256:" (7) + 64 hex chars
		t.Errorf("unexpected digest length: %d", len(d))
	}

	// Same data should produce same digest.
	d2 := Digest(data)
	if d != d2 {
		t.Error("digest not deterministic")
	}

	// Different data should produce different digest.
	d3 := Digest([]byte("different data"))
	if d == d3 {
		t.Error("different data produced same digest")
	}
}

// --- CLI function tests ---

func TestListSkills_PrintsTable(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	// Install a skill so there is something to list.
	manifest := SkillManifest{
		Name:        "list-test-skill",
		Version:     "1.0.0",
		Description: "A skill for testing list output",
	}
	configBlob, _ := json.Marshal(manifest)
	contentTar := makeTar(t, map[string]string{"SKILL.md": "# List Test"})

	if err := ss.Install(ctx, manifest, configBlob, contentTar); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := ListSkills(ctx, ss); err != nil {
		_ = w.Close()
		os.Stdout = old
		t.Fatalf("ListSkills failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "list-test-skill") {
		t.Errorf("expected output to contain skill name, got: %s", output)
	}
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version, got: %s", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("expected output to contain header NAME, got: %s", output)
	}
}

func TestListSkills_EmptyStore(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := ListSkills(ctx, ss); err != nil {
		_ = w.Close()
		os.Stdout = old
		t.Fatalf("ListSkills failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No skills installed") {
		t.Errorf("expected 'No skills installed' message, got: %s", output)
	}
}

func TestPublishSkill_EndToEnd(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	// Create a temp directory with SKILL.md.
	dir := createSkillDir(t, `name: publish-test
version: 2.0.0
description: Test publish end-to-end`, "# Publish Test Skill")

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := PublishSkill(ctx, ss, dir); err != nil {
		_ = w.Close()
		os.Stdout = old
		t.Fatalf("PublishSkill failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "publish-test@2.0.0") {
		t.Errorf("expected output to contain skill ref, got: %s", output)
	}

	// Verify it is actually installed.
	got, err := ss.Get(ctx, "publish-test", "2.0.0")
	if err != nil {
		t.Fatalf("Get after publish failed: %v", err)
	}
	if got.Description != "Test publish end-to-end" {
		t.Errorf("description = %q, want %q", got.Description, "Test publish end-to-end")
	}
}

func TestSkillInfo_PrintsMetadata(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	manifest := SkillManifest{
		Name:        "info-test-skill",
		Version:     "3.0.0",
		Description: "Testing skill info display",
		License:     "Apache-2.0",
		Authors:     []string{"Alice", "Bob"},
		Triggers:    []string{"run info test"},
	}
	configBlob, _ := json.Marshal(manifest)
	contentTar := makeTar(t, map[string]string{"SKILL.md": "# Info Test"})

	if err := ss.Install(ctx, manifest, configBlob, contentTar); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	got, err := SkillInfo(ctx, ss, "info-test-skill", "3.0.0")
	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("SkillInfo failed: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify printed output.
	for _, want := range []string{"info-test-skill", "3.0.0", "Testing skill info display", "Apache-2.0", "Alice", "Bob", "run info test"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got: %s", want, output)
		}
	}

	// Verify returned manifest.
	if got.Name != "info-test-skill" {
		t.Errorf("returned name = %q, want %q", got.Name, "info-test-skill")
	}
}

func TestInstallSkill_LocalDirectory(t *testing.T) {
	casStore := newTestStore(t)
	ss := NewSkillStore(casStore)
	ctx := context.Background()

	dir := createSkillDir(t, `name: local-install-test
version: 1.2.3
description: Test local directory install`, "# Local Install")

	// Capture stdout (InstallSkill prints to stdout).
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := InstallSkill(ctx, ss, dir, false, false); err != nil {
		_ = w.Close()
		os.Stdout = old
		t.Fatalf("InstallSkill failed: %v", err)
	}
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify the skill was installed.
	got, err := ss.Get(ctx, "local-install-test", "1.2.3")
	if err != nil {
		t.Fatalf("Get after install failed: %v", err)
	}
	if got.Description != "Test local directory install" {
		t.Errorf("description = %q, want %q", got.Description, "Test local directory install")
	}
}

// --- Helpers ---

// makeTar creates a valid tar archive from a map of filename -> content.
func makeTar(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("makeTar: write header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("makeTar: write content %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("makeTar: close: %v", err)
	}
	return buf.Bytes()
}

// listTarFiles returns the names of all files in a tar archive.
func listTarFiles(t *testing.T, tarData []byte) []string {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(tarData))
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading tar: %v", err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

// containsFile checks if a filename exists in a list.
func containsFile(files []string, name string) bool {
	for _, f := range files {
		if f == name {
			return true
		}
	}
	return false
}

// --- Era 3: PortDefinition and PluginManifest tests ---

func TestPortDefinition_JSONRoundtrip(t *testing.T) {
	port := PortDefinition{
		Name:        "sources",
		Type:        "string[]",
		Description: "Array of research source texts",
		Required:    true,
	}

	data, err := json.Marshal(port)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded PortDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Name != port.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, port.Name)
	}
	if decoded.Type != port.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, port.Type)
	}
	if decoded.Required != port.Required {
		t.Errorf("Required: got %v, want %v", decoded.Required, port.Required)
	}
}

func TestPortDefinition_WithEnum(t *testing.T) {
	port := PortDefinition{
		Name:    "format",
		Type:    "string",
		Enum:    []string{"brief", "detailed", "executive"},
		Default: "brief",
	}

	data, err := json.Marshal(port)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded PortDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Enum) != 3 {
		t.Fatalf("Enum count: got %d, want 3", len(decoded.Enum))
	}
	if decoded.Enum[0] != "brief" {
		t.Errorf("Enum[0]: got %q, want %q", decoded.Enum[0], "brief")
	}
}

func TestSkillManifest_WithPorts(t *testing.T) {
	m := SkillManifest{
		Name:        "research-synthesis",
		Version:     "2.0.0",
		Description: "Synthesize research findings",
		Inputs: []PortDefinition{
			{Name: "sources", Type: "string[]", Required: true},
			{Name: "format", Type: "string", Enum: []string{"brief", "detailed"}, Default: "brief"},
		},
		Outputs: []PortDefinition{
			{Name: "synthesis", Type: "string"},
			{Name: "themes", Type: "string[]"},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SkillManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Inputs) != 2 {
		t.Fatalf("Inputs count: got %d, want 2", len(decoded.Inputs))
	}
	if len(decoded.Outputs) != 2 {
		t.Fatalf("Outputs count: got %d, want 2", len(decoded.Outputs))
	}
	if decoded.Inputs[0].Name != "sources" {
		t.Errorf("Inputs[0].Name: got %q, want %q", decoded.Inputs[0].Name, "sources")
	}
	if decoded.Outputs[1].Type != "string[]" {
		t.Errorf("Outputs[1].Type: got %q, want %q", decoded.Outputs[1].Type, "string[]")
	}
}

func TestPluginManifest_JSONRoundtrip(t *testing.T) {
	p := PluginManifest{
		SkillManifest: SkillManifest{
			Name:        "design-intelligence",
			Version:     "1.0.0",
			Description: "Design system intelligence suite",
			License:     "Apache-2.0",
			Authors:     []string{"Tres Pies Design"},
		},
		PluginType: "bundle",
		TrustTier:  1,
		Contents: []ContentEntry{
			{Type: "skill", Name: "strategic-scout", Path: "skills/strategic-scout/"},
			{Type: "skill", Name: "research-synthesis", Path: "skills/research-synthesis/"},
			{Type: "wasm-module", Name: "dip-scorer", Path: "wasm/dip-scorer.wasm"},
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded PluginManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.PluginType != "bundle" {
		t.Errorf("PluginType: got %q, want %q", decoded.PluginType, "bundle")
	}
	if decoded.TrustTier != 1 {
		t.Errorf("TrustTier: got %d, want 1", decoded.TrustTier)
	}
	if len(decoded.Contents) != 3 {
		t.Fatalf("Contents count: got %d, want 3", len(decoded.Contents))
	}
	if decoded.Contents[2].Type != "wasm-module" {
		t.Errorf("Contents[2].Type: got %q, want %q", decoded.Contents[2].Type, "wasm-module")
	}
	// Verify SkillManifest fields are preserved.
	if decoded.Name != "design-intelligence" {
		t.Errorf("Name: got %q, want %q", decoded.Name, "design-intelligence")
	}
}

func TestPluginManifest_YankFields(t *testing.T) {
	p := PluginManifest{
		SkillManifest: SkillManifest{
			Name:    "vulnerable-skill",
			Version: "1.0.0",
		},
		Yanked:     true,
		YankReason: "security: CVE-2026-1234",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded PluginManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !decoded.Yanked {
		t.Error("Yanked: got false, want true")
	}
	if decoded.YankReason != "security: CVE-2026-1234" {
		t.Errorf("YankReason: got %q, want %q", decoded.YankReason, "security: CVE-2026-1234")
	}
}

func TestPluginManifest_OmitsEmptyEra3Fields(t *testing.T) {
	// A basic SkillManifest with no Era 3 fields should omit them from JSON.
	m := SkillManifest{
		Name:    "simple",
		Version: "1.0.0",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	s := string(data)
	if strings.Contains(s, "inputs") {
		t.Error("JSON should omit empty inputs field")
	}
	if strings.Contains(s, "outputs") {
		t.Error("JSON should omit empty outputs field")
	}
	if strings.Contains(s, "plugin_type") {
		t.Error("JSON should omit empty plugin_type field")
	}
}
