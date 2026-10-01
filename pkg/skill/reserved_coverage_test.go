package skill

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// updateReserved regenerates the Official section of both reserved-names
// copies from the first-party skills under plugins/:
//
//	go test ./pkg/skill -run TestReservedNames -update-reserved
var updateReserved = flag.Bool("update-reserved", false, "regenerate the Official section of reserved-names.txt from plugins/")

// The two copies: the repo-root file people read, and the one go:embed
// compiles into the binary. They must never drift.
var reservedCopies = []string{
	filepath.Join("..", "..", "reserved-names.txt"),
	"reserved-names.txt",
}

const officialHeaderPrefix = "# === Official Skills ("

// firstPartySkillNames is every first-party skill name, resolved exactly as
// the publisher resolves it (manifest name, directory fallback), lowercased.
func firstPartySkillNames(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "plugins", "*", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 80 {
		t.Fatalf("found only %d first-party SKILL.md files; the glob is wrong or skills went missing", len(files))
	}
	set := map[string]bool{}
	for _, f := range files {
		m, _, _, err := PackSkill(filepath.Dir(f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		set[strings.ToLower(m.Name)] = true
	}
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// TestReservedNames_CoverEveryFirstPartySkill: the reserved list exists so
// third parties can't claim DojoGenesis's skill names (ADR-020). It was
// generated once (2026-04-05, 44 names) and drifted: by 2026-09-30, 45 of
// the 89 first-party skills were unreserved. This keeps it from drifting.
func TestReservedNames_CoverEveryFirstPartySkill(t *testing.T) {
	names := firstPartySkillNames(t)

	if *updateReserved {
		for _, p := range reservedCopies {
			if err := rewriteOfficialSection(p, names); err != nil {
				t.Fatal(err)
			}
		}
	}

	data, err := os.ReadFile(reservedCopies[0])
	if err != nil {
		t.Fatal(err)
	}
	reserved := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			reserved[strings.ToLower(line)] = true
		}
	}
	var missing []string
	for _, n := range names {
		if !reserved[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d first-party skill names are not reserved (a third party could claim them): %s\n"+
			"regenerate: go test ./pkg/skill -run TestReservedNames -update-reserved",
			len(missing), strings.Join(missing, ", "))
	}
}

func TestReservedNames_TheTwoCopiesAreIdentical(t *testing.T) {
	a, err := os.ReadFile(reservedCopies[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(reservedCopies[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Errorf("%s and %s differ; the embedded copy is what the binary enforces", reservedCopies[0], reservedCopies[1])
	}
}

// rewriteOfficialSection replaces the Official section with names, keeping
// every reservation already there (a name is never un-reserved by
// regeneration), and leaves every other section untouched.
func rewriteOfficialSection(path string, names []string) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is one of the fixed reservedCopies

	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, officialHeaderPrefix) {
			start = i
			break
		}
	}
	if start < 0 {
		return fmt.Errorf("%s: no %q section", path, officialHeaderPrefix)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# === ") {
			end = i
			break
		}
	}
	set := map[string]bool{}
	for _, l := range lines[start+1 : end] {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			set[strings.ToLower(l)] = true
		}
	}
	for _, n := range names {
		set[n] = true
	}
	merged := make([]string, 0, len(set))
	for n := range set {
		merged = append(merged, n)
	}
	sort.Strings(merged)

	section := append([]string{fmt.Sprintf("%s%d) ===", officialHeaderPrefix, len(merged))}, merged...)
	section = append(section, "")
	out := append(append(append([]string{}, lines[:start]...), section...), lines[end:]...)
	text := strings.Join(out, "\n")
	text = replaceLinePrefix(text, "# Generated: ", "# Generated: "+time.Now().UTC().Format("2006-01-02")+" (Official section regenerated from plugins/)")
	text = replaceLinePrefix(text, "# [official] - ", fmt.Sprintf("# [official] - Current Dojo Platform skills (%d)", len(merged)))
	return os.WriteFile(path, []byte(text), 0o644) //nolint:gosec // G306: a checked-in data file is world-readable by design
}

func replaceLinePrefix(text, prefix, replacement string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			lines[i] = replacement
			return strings.Join(lines, "\n")
		}
	}
	return text
}
