package server

// Vendored-contract integrity check (DGS-149).
//
// plugins/kata/contracts/kata-ai/v1 is vendored from DojoGenesis/kata
// (see SOURCE there). MANIFEST.sha256 lists `shasum -a 256` of every file
// except MANIFEST.sha256 and SOURCE. A hand edit here, or a partial
// re-vendor, fails this test: change the contract in Kata, re-vendor, and
// regenerate the manifest.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKataContractManifest(t *testing.T) {
	dir := filepath.Join("..", "plugins", "kata", "contracts", "kata-ai", "v1")
	if v := os.Getenv("KATA_PLUGIN_DIR"); v != "" {
		dir = filepath.Join(v, "contracts", "kata-ai", "v1")
	}
	mf, err := os.Open(filepath.Join(dir, "MANIFEST.sha256"))
	if os.IsNotExist(err) {
		t.Skipf("kata-ai/v1 contract not vendored at %s", dir)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer mf.Close()

	listed := map[string]string{}
	sc := bufio.NewScanner(mf)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		sum, name, ok := strings.Cut(line, "  ")
		if !ok {
			t.Fatalf("malformed manifest line: %q", line)
		}
		listed[filepath.Clean(name)] = sum
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("manifest is empty")
	}

	seen := map[string]bool{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "MANIFEST.sha256" || rel == "SOURCE" {
			return nil
		}
		seen[rel] = true
		want, ok := listed[rel]
		if !ok {
			t.Errorf("%s is not in MANIFEST.sha256 (added without re-vendoring?)", rel)
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		got := sha256.Sum256(b)
		if hex.EncodeToString(got[:]) != want {
			t.Errorf("%s hash differs from MANIFEST.sha256 (edited in place? change it in Kata and re-vendor)", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name := range listed {
		if !seen[name] {
			t.Errorf("%s is in MANIFEST.sha256 but missing on disk", name)
		}
	}
}

// TestKataPluginLoadsAllOps guards the startup path: a malformed SKILL.md or
// schema is logged and skipped at boot (by design, so one bad op never takes
// the gateway down), which means nothing else would notice an op silently
// missing from the registry.
func TestKataPluginLoadsAllOps(t *testing.T) {
	pluginDir := filepath.Join("..", "plugins", "kata")
	if v := os.Getenv("KATA_PLUGIN_DIR"); v != "" {
		pluginDir = v
	}
	entries, err := os.ReadDir(filepath.Join(pluginDir, "skills"))
	if os.IsNotExist(err) {
		t.Skipf("kata plugin not present at %s", pluginDir)
	}
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, e := range entries {
		if e.IsDir() {
			want++
		}
	}
	r := loadDispatchRegistry(pluginDir, "test-model", 0)
	if len(r.structured) != want {
		var got []string
		for id := range r.structured {
			got = append(got, id)
		}
		t.Fatalf("loaded %d structured ops, want %d (one per skills/ dir); loaded: %v", len(r.structured), want, got)
	}
	for id, op := range r.structured {
		if op.ID != id {
			t.Errorf("op %q registered under %q", op.ID, id)
		}
	}
}

// TestCompileSchemaFile_LocalRefsOnly: an input schema may $ref a shared
// schema by absolute $id when that schema sits on disk under the plugin root;
// an $id that is not on disk must still fail (nothing is fetched).
func TestCompileSchemaFile_LocalRefsOnly(t *testing.T) {
	root := t.TempDir()
	v1 := filepath.Join(root, "contracts", "x", "v1")
	op := filepath.Join(v1, "op")
	if err := os.MkdirAll(op, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(p, s string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(v1, "_defs.schema.json"), `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://example.test/x/v1/_defs.schema.json","$defs":{"Level":{"enum":["low","high"]}}}`)
	write(filepath.Join(op, "ok.schema.json"), `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://example.test/x/v1/op/ok.schema.json","type":"object","properties":{"level":{"$ref":"https://example.test/x/v1/_defs.schema.json#/$defs/Level"}}}`)
	write(filepath.Join(op, "remote.schema.json"), `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"level":{"$ref":"https://example.test/elsewhere.schema.json"}}}`)

	sch, _, err := compileSchemaFile(root, "contracts/x/v1/op/ok.schema.json")
	if err != nil {
		t.Fatalf("local $ref should resolve: %v", err)
	}
	if err := sch.Validate(map[string]any{"level": "low"}); err != nil {
		t.Errorf("valid instance rejected: %v", err)
	}
	if err := sch.Validate(map[string]any{"level": "medium"}); err == nil {
		t.Error("enum from the shared $defs was not enforced")
	}
	if _, _, err := compileSchemaFile(root, "contracts/x/v1/op/remote.schema.json"); err == nil {
		t.Error("a $ref to a schema not on disk must fail, not be fetched")
	}
}
