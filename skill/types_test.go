package skill

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// validSkill returns a fully-valid SkillDefinition suitable for use as a
// base fixture. Callers should override individual fields to exercise error
// branches.
func validSkill() *SkillDefinition {
	return &SkillDefinition{
		Name:             "my-skill",
		Description:      "does something useful",
		Triggers:         []string{"invoke my skill"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
	}
}

// ---------------------------------------------------------------------------
// IsValid
// ---------------------------------------------------------------------------

func TestIsValid_HappyPath(t *testing.T) {
	s := validSkill()
	assert.NoError(t, s.IsValid())
}

func TestIsValid_EmptyName(t *testing.T) {
	s := validSkill()
	s.Name = ""
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillName)
}

func TestIsValid_EmptyDescription(t *testing.T) {
	s := validSkill()
	s.Description = ""
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillDescription)
}

func TestIsValid_ZeroTriggers(t *testing.T) {
	s := validSkill()
	s.Triggers = nil
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillTriggers)
}

func TestIsValid_TierTooLow(t *testing.T) {
	s := validSkill()
	s.Tier = 0
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillTier)
}

func TestIsValid_TierTooHigh(t *testing.T) {
	s := validSkill()
	s.Tier = 5
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillTier)
}

func TestIsValid_NonHiddenZeroAgents(t *testing.T) {
	s := validSkill()
	s.Hidden = false
	s.Agents = nil
	assert.ErrorIs(t, s.IsValid(), ErrInvalidSkillAgents)
}

func TestIsValid_HiddenZeroAgents_IsValid(t *testing.T) {
	s := validSkill()
	s.Hidden = true
	s.Agents = nil
	// A hidden skill with no agents must still pass validation.
	assert.NoError(t, s.IsValid())
}

func TestIsValid_InvalidToolDependency(t *testing.T) {
	s := validSkill()
	s.ToolDependencies = []string{"file_system", "not_a_real_dep"}
	assert.ErrorIs(t, s.IsValid(), ErrInvalidToolDependency)
}

func TestIsValid_ValidToolDependencies_AllKnown(t *testing.T) {
	s := validSkill()
	s.ToolDependencies = []string{"file_system", "bash", "web_tools", "script_execution", "meta_skill"}
	s.Tier = 3 // meta_skill requires Tier >= 3 semantically, but IsValid only checks the set
	assert.NoError(t, s.IsValid())
}

// ---------------------------------------------------------------------------
// RequiresAdapter
// ---------------------------------------------------------------------------

func TestRequiresAdapter_Tier1_False(t *testing.T) {
	s := &SkillDefinition{Tier: 1}
	assert.False(t, s.RequiresAdapter())
}

func TestRequiresAdapter_Tier2_True(t *testing.T) {
	s := &SkillDefinition{Tier: 2}
	assert.True(t, s.RequiresAdapter())
}

func TestRequiresAdapter_Tier3_True(t *testing.T) {
	s := &SkillDefinition{Tier: 3}
	assert.True(t, s.RequiresAdapter())
}

func TestRequiresAdapter_Tier4_True(t *testing.T) {
	s := &SkillDefinition{Tier: 4}
	assert.True(t, s.RequiresAdapter())
}

// ---------------------------------------------------------------------------
// RequiresWebTools
// ---------------------------------------------------------------------------

func TestRequiresWebTools_Present_True(t *testing.T) {
	s := &SkillDefinition{ToolDependencies: []string{"file_system", "web_tools"}}
	assert.True(t, s.RequiresWebTools())
}

func TestRequiresWebTools_Absent_False(t *testing.T) {
	s := &SkillDefinition{ToolDependencies: []string{"file_system", "bash"}}
	assert.False(t, s.RequiresWebTools())
}

func TestRequiresWebTools_Empty_False(t *testing.T) {
	s := &SkillDefinition{}
	assert.False(t, s.RequiresWebTools())
}

// ---------------------------------------------------------------------------
// RequiresScriptExecution
// ---------------------------------------------------------------------------

func TestRequiresScriptExecution_Present_True(t *testing.T) {
	s := &SkillDefinition{ToolDependencies: []string{"bash", "script_execution"}}
	assert.True(t, s.RequiresScriptExecution())
}

func TestRequiresScriptExecution_Absent_False(t *testing.T) {
	s := &SkillDefinition{ToolDependencies: []string{"file_system", "web_tools"}}
	assert.False(t, s.RequiresScriptExecution())
}

func TestRequiresScriptExecution_Empty_False(t *testing.T) {
	s := &SkillDefinition{}
	assert.False(t, s.RequiresScriptExecution())
}

// ---------------------------------------------------------------------------
// IsMetaSkill
// ---------------------------------------------------------------------------

func TestIsMetaSkill_Tier1_False(t *testing.T) {
	s := &SkillDefinition{Tier: 1}
	assert.False(t, s.IsMetaSkill())
}

func TestIsMetaSkill_Tier2_False(t *testing.T) {
	s := &SkillDefinition{Tier: 2}
	assert.False(t, s.IsMetaSkill())
}

func TestIsMetaSkill_Tier3_True(t *testing.T) {
	s := &SkillDefinition{Tier: 3}
	assert.True(t, s.IsMetaSkill())
}

func TestIsMetaSkill_Tier4_True(t *testing.T) {
	s := &SkillDefinition{Tier: 4}
	assert.True(t, s.IsMetaSkill())
}
