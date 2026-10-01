package skill

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// The reserved-names list (ADR-020) keeps THIRD parties from squatting
// DojoGenesis's first-party skill names. The publisher had no notion of the
// owner, so DojoGenesis could not package its own reserved skills: 45 of the
// 89 under plugins/ were refused, and the tag-time workflow failed on every
// release.

func TestPublish_DefaultStillRefusesAReservedName(t *testing.T) {
	ss := NewSkillStore(newTestStore(t))
	dir := createSkillDir(t, "name: debugging\nversion: 1.0.0\ndescription: d", "# d")
	err := PublishSkill(context.Background(), ss, dir)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("a third-party publish of a reserved name must be refused, got %v", err)
	}
}

func TestPublish_DefaultStillRefusesANearReservedName(t *testing.T) {
	ss := NewSkillStore(newTestStore(t))
	dir := createSkillDir(t, "name: mcp-builder\nversion: 1.0.0\ndescription: d", "# d")
	if err := PublishSkill(context.Background(), ss, dir); err == nil {
		t.Fatal("a third-party name one edit-distance step from a reserved name must still be refused")
	}
}

func TestPublish_FirstPartyMayPublishItsOwnReservedNames(t *testing.T) {
	ss := NewSkillStore(newTestStore(t))
	for _, name := range []string{"debugging", "mcp-builder"} {
		dir := createSkillDir(t, "name: "+name+"\nversion: 1.0.0\ndescription: d", "# d")
		if _, err := PublishSkillWithOptions(context.Background(), ss, dir, PublishOptions{FirstParty: true}); err != nil {
			t.Errorf("first-party publish of %q: %v", name, err)
		}
	}
}

func TestPublish_FirstPartyReturnsTheManifestName(t *testing.T) {
	ss := NewSkillStore(newTestStore(t))
	dir := createSkillDir(t, "name: retrospective\nversion: 1.0.0\ndescription: d", "# d")
	name, err := PublishSkillWithOptions(context.Background(), ss, dir, PublishOptions{FirstParty: true})
	if err != nil || name != "retrospective" {
		t.Fatalf("got name %q err %v", name, err)
	}
}

// Every first-party skill in this repo must package as DojoGenesis would
// publish it. This runs with the root module's tests, so a broken SKILL.md
// fails its own pull request instead of the release-tag workflow.
func TestPublish_EveryFirstPartySkillInThisRepoPackages(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", "plugins", "*", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) < 80 {
		t.Fatalf("found only %d first-party SKILL.md files; the glob is wrong or skills went missing", len(dirs))
	}
	ss := NewSkillStore(newTestStore(t))
	seen := map[string]string{}
	for _, f := range dirs {
		dir := filepath.Dir(f)
		name, err := PublishSkillWithOptions(context.Background(), ss, dir, PublishOptions{FirstParty: true})
		if err != nil {
			t.Errorf("%s: %v", dir, err)
			continue
		}
		if prev, dup := seen[name]; dup {
			t.Errorf("duplicate first-party skill name %q: %s and %s", name, prev, dir)
		}
		seen[name] = dir
	}
}
