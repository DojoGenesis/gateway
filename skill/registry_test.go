package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemorySkillRegistry(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.skills)
}

func TestRegisterSkill_HappyPath(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	err := registry.RegisterSkill(ctx, skill)
	assert.NoError(t, err)
	assert.NotEmpty(t, skill.ID, "Should assign UUID")
	assert.False(t, skill.LoadedAt.IsZero(), "Should set LoadedAt timestamp")
}

func TestRegisterSkill_NilSkill(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	err := registry.RegisterSkill(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRegisterSkill_DuplicateName(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill1 := &SkillDefinition{
		Name:             "duplicate-skill",
		Description:      "First skill",
		Triggers:         []string{"trigger1"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	skill2 := &SkillDefinition{
		Name:             "duplicate-skill",
		Description:      "Second skill",
		Triggers:         []string{"trigger2"},
		ToolDependencies: []string{"bash"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent2"},
	}

	err := registry.RegisterSkill(ctx, skill1)
	assert.NoError(t, err)

	err = registry.RegisterSkill(ctx, skill2)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSkillAlreadyExists)
}

func TestRegisterSkill_InvalidTier(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	tests := []struct {
		name string
		tier int
	}{
		{"tier 0", 0},
		{"tier 5", 5},
		{"tier -1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &SkillDefinition{
				Name:             "test-skill",
				Description:      "A test skill",
				Triggers:         []string{"trigger"},
				ToolDependencies: []string{"file_system"},
				Tier:             tt.tier,
				Portable:         true,
				Agents:           []string{"agent1"},
			}

			err := registry.RegisterSkill(ctx, skill)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidSkillTier)
		})
	}
}

func TestRegisterSkill_InvalidToolDependency(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		Triggers:         []string{"trigger"},
		ToolDependencies: []string{"invalid_tool", "file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	err := registry.RegisterSkill(ctx, skill)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToolDependency)
}

func TestRegisterSkill_EmptyAgents(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Non-hidden skill with empty agents should fail
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		Triggers:         []string{"trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{},
		Hidden:           false,
	}

	err := registry.RegisterSkill(ctx, skill)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSkillAgents)

	// Hidden skill with empty agents should succeed
	skill.Hidden = true
	err = registry.RegisterSkill(ctx, skill)
	assert.NoError(t, err)
}

func TestGetSkill_HappyPath(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		Triggers:         []string{"trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	err := registry.RegisterSkill(ctx, skill)
	require.NoError(t, err)

	retrieved, err := registry.GetSkill(ctx, "test-skill")
	assert.NoError(t, err)
	assert.Equal(t, "test-skill", retrieved.Name)
	assert.Equal(t, "A test skill", retrieved.Description)
}

func TestGetSkill_NotFound(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	_, err := registry.GetSkill(ctx, "nonexistent-skill")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestListSkills(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Empty registry
	skills, err := registry.ListSkills(ctx)
	assert.NoError(t, err)
	assert.Empty(t, skills)

	// Add skills
	skill1 := &SkillDefinition{
		Name:             "skill1",
		Description:      "First skill",
		Triggers:         []string{"trigger1"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	skill2 := &SkillDefinition{
		Name:             "skill2",
		Description:      "Second skill",
		Triggers:         []string{"trigger2"},
		ToolDependencies: []string{"bash"},
		Tier:             2,
		Portable:         true,
		Agents:           []string{"agent2"},
	}

	err = registry.RegisterSkill(ctx, skill1)
	require.NoError(t, err)
	err = registry.RegisterSkill(ctx, skill2)
	require.NoError(t, err)

	skills, err = registry.ListSkills(ctx)
	assert.NoError(t, err)
	assert.Len(t, skills, 2)
}

func TestListByPlugin(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill1 := &SkillDefinition{
		Name:             "skill1",
		Description:      "First skill",
		PluginName:       "plugin-a",
		Triggers:         []string{"trigger1"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	skill2 := &SkillDefinition{
		Name:             "skill2",
		Description:      "Second skill",
		PluginName:       "plugin-b",
		Triggers:         []string{"trigger2"},
		ToolDependencies: []string{"bash"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	skill3 := &SkillDefinition{
		Name:             "skill3",
		Description:      "Third skill",
		PluginName:       "plugin-a",
		Triggers:         []string{"trigger3"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}

	require.NoError(t, registry.RegisterSkill(ctx, skill1))
	require.NoError(t, registry.RegisterSkill(ctx, skill2))
	require.NoError(t, registry.RegisterSkill(ctx, skill3))

	// List plugin-a skills
	skills, err := registry.ListByPlugin(ctx, "plugin-a")
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	// List plugin-b skills
	skills, err = registry.ListByPlugin(ctx, "plugin-b")
	assert.NoError(t, err)
	assert.Len(t, skills, 1)

	// List nonexistent plugin
	skills, err = registry.ListByPlugin(ctx, "plugin-c")
	assert.NoError(t, err)
	assert.Empty(t, skills)
}

func TestListByTier(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Add skills with different tiers
	for tier := 1; tier <= 4; tier++ {
		for i := 0; i < tier; i++ {
			skill := &SkillDefinition{
				Name:             fmt.Sprintf("tier%d-skill%d", tier, i),
				Description:      fmt.Sprintf("Tier %d skill %d", tier, i),
				Triggers:         []string{"trigger"},
				ToolDependencies: []string{"file_system"},
				Tier:             tier,
				Portable:         true,
				Agents:           []string{"agent1"},
			}
			require.NoError(t, registry.RegisterSkill(ctx, skill))
		}
	}

	// Test each tier
	for tier := 1; tier <= 4; tier++ {
		skills, err := registry.ListByTier(ctx, tier)
		assert.NoError(t, err)
		assert.Len(t, skills, tier, "Tier %d should have %d skills", tier, tier)
	}

	// Invalid tier
	_, err := registry.ListByTier(ctx, 5)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSkillTier)
}

func TestListByAgent(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	skill1 := &SkillDefinition{
		Name:             "skill1",
		Description:      "First skill",
		Triggers:         []string{"trigger1"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1", "agent2"},
	}

	skill2 := &SkillDefinition{
		Name:             "skill2",
		Description:      "Second skill",
		Triggers:         []string{"trigger2"},
		ToolDependencies: []string{"bash"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent2"},
	}

	skill3 := &SkillDefinition{
		Name:             "skill3",
		Description:      "Hidden skill",
		Triggers:         []string{"trigger3"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Hidden:           true,
	}

	require.NoError(t, registry.RegisterSkill(ctx, skill1))
	require.NoError(t, registry.RegisterSkill(ctx, skill2))
	require.NoError(t, registry.RegisterSkill(ctx, skill3))

	// Agent1 should see skill1 but not skill3 (hidden)
	skills, err := registry.ListByAgent(ctx, "agent1")
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "skill1", skills[0].Name)

	// Agent2 should see skill1 and skill2
	skills, err = registry.ListByAgent(ctx, "agent2")
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	// Nonexistent agent
	skills, err = registry.ListByAgent(ctx, "agent3")
	assert.NoError(t, err)
	assert.Empty(t, skills)
}

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFront string
		wantBody  string
		wantErr   bool
	}{
		{
			name: "valid frontmatter",
			content: `---
name: test-skill
description: A test skill
---

# Skill Content

This is the body.`,
			wantFront: "name: test-skill\ndescription: A test skill",
			wantBody:  "# Skill Content\n\nThis is the body.",
			wantErr:   false,
		},
		{
			name:    "missing frontmatter",
			content: "# Skill Content\n\nNo frontmatter here.",
			wantErr: true,
		},
		{
			name: "incomplete frontmatter",
			content: `---
name: test-skill
description: A test skill

# Missing closing ---`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body, err := extractFrontmatter(tt.content)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantFront, front)
				assert.Equal(t, tt.wantBody, body)
			}
		})
	}
}

func TestLoadFromDirectory(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Create temp directory with test SKILL.md files
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin", "skills")

	// Create skill directories
	skill1Dir := filepath.Join(pluginDir, "skill1")
	skill2Dir := filepath.Join(pluginDir, "skill2")
	require.NoError(t, os.MkdirAll(skill1Dir, 0755))
	require.NoError(t, os.MkdirAll(skill2Dir, 0755))

	// Create valid SKILL.md for skill1
	skill1Content := `---
name: skill1
description: First test skill
triggers:
  - trigger1
  - trigger2
tool_dependencies:
  - file_system
tier: 1
portable: true
agents:
  - agent1
---

# Skill 1

This is the first test skill.`

	err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skill1Content), 0644)
	require.NoError(t, err)

	// Create valid SKILL.md for skill2
	skill2Content := `---
name: skill2
description: Second test skill
triggers:
  - trigger3
tool_dependencies:
  - bash
  - web_tools
tier: 2
portable: true
agents:
  - agent1
  - agent2
---

# Skill 2

This is the second test skill.`

	err = os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(skill2Content), 0644)
	require.NoError(t, err)

	// Load from directory
	err = registry.LoadFromDirectory(ctx, pluginDir, "test-plugin")
	assert.NoError(t, err)

	// Verify skills were loaded
	skills, err := registry.ListSkills(ctx)
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	// Verify skill1
	skill1, err := registry.GetSkill(ctx, "skill1")
	assert.NoError(t, err)
	assert.Equal(t, "skill1", skill1.Name)
	assert.Equal(t, "First test skill", skill1.Description)
	assert.Equal(t, 1, skill1.Tier)
	assert.Contains(t, skill1.Triggers, "trigger1")
	assert.Contains(t, skill1.Content, "This is the first test skill")

	// Verify skill2
	skill2, err := registry.GetSkill(ctx, "skill2")
	assert.NoError(t, err)
	assert.Equal(t, "skill2", skill2.Name)
	assert.Equal(t, 2, skill2.Tier)
	assert.Contains(t, skill2.Agents, "agent2")
}

func TestLoadFromDirectory_WithInvalidSkill(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Create temp directory
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin", "skills")

	skill1Dir := filepath.Join(pluginDir, "skill1")
	skill2Dir := filepath.Join(pluginDir, "skill2")
	require.NoError(t, os.MkdirAll(skill1Dir, 0755))
	require.NoError(t, os.MkdirAll(skill2Dir, 0755))

	// Valid skill
	validContent := `---
name: valid-skill
description: A valid skill
triggers:
  - trigger1
tool_dependencies:
  - file_system
tier: 1
portable: true
agents:
  - agent1
---

# Valid Skill`

	err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(validContent), 0644)
	require.NoError(t, err)

	// Invalid skill (missing required fields)
	invalidContent := `---
name: invalid-skill
---

# Invalid Skill`

	err = os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Load should report errors but continue
	err = registry.LoadFromDirectory(ctx, pluginDir, "test-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loaded directory with")

	// Valid skill should still be loaded
	skill, err := registry.GetSkill(ctx, "valid-skill")
	assert.NoError(t, err)
	assert.Equal(t, "valid-skill", skill.Name)

	// Invalid skill should not be loaded
	_, err = registry.GetSkill(ctx, "invalid-skill")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Test concurrent RegisterSkill and GetSkill
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			skill := &SkillDefinition{
				Name:             fmt.Sprintf("skill%d", idx),
				Description:      "Concurrent test skill",
				Triggers:         []string{"trigger"},
				ToolDependencies: []string{"file_system"},
				Tier:             1,
				Portable:         true,
				Agents:           []string{"agent1"},
			}
			_ = registry.RegisterSkill(ctx, skill)
			_, _ = registry.GetSkill(ctx, fmt.Sprintf("skill%d", idx))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all skills were registered
	skills, err := registry.ListSkills(ctx)
	assert.NoError(t, err)
	assert.Len(t, skills, 10)
}
