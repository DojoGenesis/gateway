package skill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSkillFile is a small helper that writes content to a SKILL.md-style
// file inside t.TempDir() and returns the absolute path. Hermetic: no
// network, no ~/.dojo, no shared state.
func writeSkillFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestParseSkillFile_FileReadError(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "does-not-exist.md")

	skill, err := parseSkillFile(missingPath, "some-plugin")

	require.Error(t, err)
	assert.Nil(t, skill)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestParseSkillFile_MissingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "# No Frontmatter\n\nJust a body, no --- markers."
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "some-plugin")

	require.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, ErrMissingFrontmatter, "extractFrontmatter's error must propagate unwrapped")
}

func TestParseSkillFile_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	// Unbalanced flow sequence ("[1, 2" never closed) is a YAML syntax
	// error regardless of target field types, so this reliably exercises
	// the yaml.Unmarshal error path rather than a type-conversion path.
	content := `---
name: malformed-skill
description: a skill with broken frontmatter
tier: [1, 2
---

Body content.`
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "some-plugin")

	require.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, ErrInvalidYAMLFrontmatter)
}

// wantMetadataFields captures the metadata-derived fields that should be
// identical whether they came from the nested `metadata:` block (spec
// format) or the flat top-level format (backward compat) — the two
// formats are expected to coalesce into the same SkillDefinition shape.
type wantMetadataFields struct {
	tier             int
	agents           []string
	toolDependencies []string
	portable         bool
	requiredVersion  string
	pythonScripts    []string
	shellScripts     []string
	hidden           bool
	version          string
	created          string
	author           string
}

func assertMetadataFields(t *testing.T, skill *SkillDefinition, want wantMetadataFields) {
	t.Helper()
	assert.Equal(t, want.tier, skill.Tier)
	assert.Equal(t, want.agents, skill.Agents)
	assert.Equal(t, want.toolDependencies, skill.ToolDependencies)
	assert.Equal(t, want.portable, skill.Portable)
	assert.Equal(t, want.requiredVersion, skill.RequiredVersion)
	assert.Equal(t, want.pythonScripts, skill.PythonScripts)
	assert.Equal(t, want.shellScripts, skill.ShellScripts)
	assert.Equal(t, want.hidden, skill.Hidden)
	assert.Equal(t, want.version, skill.Version)
	assert.Equal(t, want.created, skill.Created)
	assert.Equal(t, want.author, skill.Author)
}

func TestParseSkillFile_NestedMetadataFormat(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: nested-skill
description: A spec-compliant nested-metadata skill
triggers:
  - trigger one
metadata:
  tier: 3
  agents:
    - agent-x
    - agent-y
  tool_dependencies:
    - bash
    - web_tools
  portable: true
  requires_version: ">=3.0.0"
  python_scripts:
    - setup.py
  shell_scripts:
    - run.sh
  hidden: true
  version: "2.5.1"
  created: "2026-03-01"
  author: "Ada Lovelace"
---

# Nested Skill

Nested body content.`
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "nested-plugin")

	require.NoError(t, err)
	require.NotNil(t, skill)
	assertMetadataFields(t, skill, wantMetadataFields{
		tier:             3,
		agents:           []string{"agent-x", "agent-y"},
		toolDependencies: []string{"bash", "web_tools"},
		portable:         true,
		requiredVersion:  ">=3.0.0",
		pythonScripts:    []string{"setup.py"},
		shellScripts:     []string{"run.sh"},
		hidden:           true,
		version:          "2.5.1",
		created:          "2026-03-01",
		author:           "Ada Lovelace",
	})
}

func TestParseSkillFile_FlatMetadataFormat(t *testing.T) {
	dir := t.TempDir()
	// Same values as the nested-format fixture above, but at the top
	// level with the flat-format key name (required_version, not
	// requires_version) — backward-compat path, no `metadata:` block.
	content := `---
name: flat-skill
description: A backward-compat flat-metadata skill
triggers:
  - trigger one
tier: 3
agents:
  - agent-x
  - agent-y
tool_dependencies:
  - bash
  - web_tools
portable: true
required_version: ">=3.0.0"
python_scripts:
  - setup.py
shell_scripts:
  - run.sh
hidden: true
version: "2.5.1"
created: "2026-03-01"
author: "Ada Lovelace"
---

# Flat Skill

Flat body content.`
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "flat-plugin")

	require.NoError(t, err)
	require.NotNil(t, skill)
	assertMetadataFields(t, skill, wantMetadataFields{
		tier:             3,
		agents:           []string{"agent-x", "agent-y"},
		toolDependencies: []string{"bash", "web_tools"},
		portable:         true,
		requiredVersion:  ">=3.0.0",
		pythonScripts:    []string{"setup.py"},
		shellScripts:     []string{"run.sh"},
		hidden:           true,
		version:          "2.5.1",
		created:          "2026-03-01",
		author:           "Ada Lovelace",
	})
}

func TestParseSkillFile_CommonMappingsAndContent(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: common-skill
description: A skill for verifying common field mappings
triggers:
  - alpha trigger
  - beta trigger
tier: 1
agents:
  - agent1
tool_dependencies:
  - file_system
portable: true
---

# Common Skill

This is the markdown body that should land in Content.`
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "common-plugin")

	require.NoError(t, err)
	require.NotNil(t, skill)

	assert.Equal(t, "common-skill", skill.Name)
	assert.Equal(t, "A skill for verifying common field mappings", skill.Description)
	assert.Equal(t, []string{"alpha trigger", "beta trigger"}, skill.Triggers)
	assert.Equal(t, "common-plugin", skill.PluginName, "PluginName must be the passed pluginName arg")
	assert.Equal(t, path, skill.FilePath, "FilePath must be the passed path arg")
	assert.True(t, strings.HasPrefix(skill.Content, "# Common Skill"))
	assert.Contains(t, skill.Content, "This is the markdown body that should land in Content.")
}

func TestParseSkillFile_EraThreePortsCoalesceToEmptySlices(t *testing.T) {
	dir := t.TempDir()
	// No `inputs:` / `outputs:` keys at all — Era 3 backward-compat path.
	content := `---
name: no-ports-skill
description: A skill declaring no Era 3 port contracts
triggers:
  - trigger
tier: 1
agents:
  - agent1
tool_dependencies:
  - file_system
portable: true
---

# No Ports Skill`
	path := writeSkillFile(t, dir, "SKILL.md", content)

	skill, err := parseSkillFile(path, "ports-plugin")

	require.NoError(t, err)
	require.NotNil(t, skill)

	// NOTE: asserting the actual coalescing behavior — nil-from-YAML must
	// become a non-nil empty slice, not merely an "empty" (possibly nil)
	// one. assert.Empty alone would also pass for a nil slice, so we
	// additionally assert NotNil to lock in the real conversion.
	assert.NotNil(t, skill.Inputs, "Inputs must be coalesced to a non-nil empty slice")
	assert.Equal(t, []PortDef{}, skill.Inputs)
	assert.NotNil(t, skill.Outputs, "Outputs must be coalesced to a non-nil empty slice")
	assert.Equal(t, []PortDef{}, skill.Outputs)
}

func TestParseSkillFile_MalformedYAML_ErrorsIsAlsoTrue(t *testing.T) {
	// Separate, minimal case double-checking errors.Is semantics (as
	// opposed to string matching) against a plain error value, per the
	// task's requirement to assert the sentinel via errors.Is.
	dir := t.TempDir()
	content := "---\nname: [unterminated\n---\nbody"
	path := writeSkillFile(t, dir, "SKILL.md", content)

	_, err := parseSkillFile(path, "some-plugin")

	require.Error(t, err)
	var target error = ErrInvalidYAMLFrontmatter
	assert.True(t, errors.Is(err, target))
}
