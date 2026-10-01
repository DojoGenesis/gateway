package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/pkg/skill"
	"github.com/DojoGenesis/gateway/runtime/cas"
)

func writeSkill(t *testing.T, root, rel, name string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\nversion: 1.0.0\ndescription: d\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testSkillStore(t *testing.T) *skill.SkillStore {
	t.Helper()
	c, err := cas.NewSQLiteStore(filepath.Join(t.TempDir(), "cas.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return skill.NewSkillStore(c)
}

// First-party mode skips the third-party name checks, so it must still catch
// two skills claiming one name: they would silently replace each other.
func TestPackageAll_FirstPartyRefusesDuplicateNames(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "a/skills/one", "debugging")
	writeSkill(t, root, "b/skills/two", "debugging")
	err := packageAll(context.Background(), testSkillStore(t), root, true)
	if err == nil {
		t.Fatal("two first-party skills named \"debugging\" must fail the batch")
	}
}

func TestPackageAll_FirstPartyAcceptsReservedNamesWithoutTheFlagItRefuses(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "a/skills/one", "debugging") // reserved
	if err := packageAll(context.Background(), testSkillStore(t), root, true); err != nil {
		t.Fatalf("first-party: %v", err)
	}
	err := packageAll(context.Background(), testSkillStore(t), root, false)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("without --first-party a reserved name must still fail, got %v", err)
	}
}

func TestFirstPartyArgs(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		path     string
		fp, fail bool
	}{
		{[]string{"plugins"}, "plugins", false, false},
		{[]string{"--first-party", "plugins"}, "plugins", true, false},
		{[]string{"plugins", "--first-party"}, "plugins", true, false},
		{[]string{"--bogus", "plugins"}, "", false, true},
		{[]string{}, "", false, true},
	} {
		p, fp, err := firstPartyArgs(tc.args, "usage")
		if (err != nil) != tc.fail || p != tc.path || fp != tc.fp {
			t.Errorf("%v: got (%q, %v, %v)", tc.args, p, fp, err)
		}
	}
}
