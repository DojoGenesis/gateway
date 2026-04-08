package skills_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/skill"
)

// mockToolInvoker for testing skill execution
type mockToolInvoker struct {
	invokeFn func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockToolInvoker) InvokeTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, toolName, params)
	}
	// Default: return success
	return map[string]interface{}{
		"status": "success",
		"output": "Skill executed successfully",
	}, nil
}

// setupRegistry loads all skills from the plugins directory
func setupRegistry(t *testing.T) *skill.InMemorySkillRegistry {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Load all plugin skills
	plugins := []string{
		"skill-forge",
		"continuous-learning",
		"system-health",
		"strategic-thinking",
		"specification-driven-development",
		"agent-orchestration",
		"wisdom-garden",
	}

	for _, plugin := range plugins {
		dir := "../../plugins/" + plugin + "/skills"
		err := registry.LoadFromDirectory(ctx, dir, plugin)
		// Don't fail if plugin directory doesn't exist yet
		if err != nil && !assert.Contains(t, err.Error(), "no such file") {
			t.Logf("Warning loading %s: %v", plugin, err)
		}
	}

	return registry
}

// setupExecutor creates a skill executor with mock tool invoker
func setupExecutor(registry *skill.InMemorySkillRegistry) *skill.DefaultSkillExecutor {
	invoker := &mockToolInvoker{}
	return skill.NewSkillExecutor(registry, invoker, nil)
}

// TestSmokeAllSkills verifies that all registered skills can be invoked without crashing
func TestSmokeAllSkills(t *testing.T) {
	registry := setupRegistry(t)
	executor := setupExecutor(registry)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	skills, err := registry.ListSkills(ctx)
	require.NoError(t, err)

	if len(skills) == 0 {
		t.Skip("No skills loaded - this is expected during initial porting")
	}

	t.Logf("Testing %d skills for basic invocation", len(skills))

	passCount := 0
	failCount := 0

	for _, sk := range skills {
		t.Run(sk.Name, func(t *testing.T) {
			// Invoke with empty args (skill should handle gracefully)
			result, err := executor.Execute(ctx, sk.Name, map[string]interface{}{})

			// Tier 1 skills should invoke without error
			if sk.Tier == 1 {
				if assert.NoError(t, err, "Tier 1 skill %s should not error", sk.Name) {
					passCount++
				} else {
					failCount++
					t.Logf("Tier 1 skill %s failed: %v", sk.Name, err)
				}
			}

			// Tier 2 skills might require adapters; log but don't fail
			if sk.Tier == 2 {
				if err != nil {
					t.Logf("Tier 2 skill %s requires adapter: %v", sk.Name, err)
					passCount++ // Still counts as pass (expected behavior)
				} else {
					passCount++
				}
			}

			// Tier 3+ skills are Phase 4b (should not be in Phase 4a batch)
			if sk.Tier >= 3 {
				t.Skipf("Tier %d skill %s deferred to Phase 4b", sk.Tier, sk.Name)
			}

			// Result should be a map
			if result != nil {
				assert.IsType(t, make(map[string]interface{}), result)
			}
		})
	}

	t.Logf("Smoke Test Results: %d passed, %d failed out of %d skills", passCount, failCount, len(skills))
}

// TestSkillsByTier verifies skill tier distribution
func TestSkillsByTier(t *testing.T) {
	registry := setupRegistry(t)
	ctx := context.Background()

	for tier := 1; tier <= 4; tier++ {
		t.Run(fmt.Sprintf("Tier%d", tier), func(t *testing.T) {
			skills, err := registry.ListByTier(ctx, tier)
			require.NoError(t, err)
			t.Logf("Tier %d: %d skills", tier, len(skills))

			for _, sk := range skills {
				t.Logf("  - %s (%s)", sk.Name, sk.PluginName)
			}
		})
	}
}

// TestSkillMetadataCompleteness verifies all skills have required metadata
func TestSkillMetadataCompleteness(t *testing.T) {
	registry := setupRegistry(t)
	ctx := context.Background()

	skills, err := registry.ListSkills(ctx)
	require.NoError(t, err)

	if len(skills) == 0 {
		t.Skip("No skills loaded")
	}

	for _, sk := range skills {
		t.Run(sk.Name, func(t *testing.T) {
			// Required fields
			assert.NotEmpty(t, sk.Name, "Name required")
			assert.NotEmpty(t, sk.Description, "Description required")
			assert.NotEmpty(t, sk.Triggers, "Triggers required")
			assert.NotZero(t, sk.Tier, "Tier required")
			assert.NotEmpty(t, sk.ToolDependencies, "ToolDependencies required")

			// Non-hidden skills must have agents
			if !sk.Hidden {
				assert.NotEmpty(t, sk.Agents, "Non-hidden skills must specify agents")
			}

			// Tier must be 1-4
			assert.GreaterOrEqual(t, sk.Tier, 1)
			assert.LessOrEqual(t, sk.Tier, 4)

			// Tool dependencies must be valid
			for _, dep := range sk.ToolDependencies {
				assert.Contains(t, skill.ValidToolDependencies, dep, "Invalid tool dependency: %s", dep)
			}

			// Content should not be empty
			assert.NotEmpty(t, sk.Content, "Skill content should not be empty")
		})
	}
}

// TestSkillExecutionPerformance verifies skills execute within reasonable time
func TestSkillExecutionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	registry := setupRegistry(t)
	executor := setupExecutor(registry)
	ctx := context.Background()

	skills, err := registry.ListSkills(ctx)
	require.NoError(t, err)

	if len(skills) == 0 {
		t.Skip("No skills loaded")
	}

	maxDuration := 5 * time.Second

	for _, sk := range skills {
		t.Run(sk.Name, func(t *testing.T) {
			start := time.Now()
			_, _ = executor.Execute(ctx, sk.Name, map[string]interface{}{})
			duration := time.Since(start)

			assert.Less(t, duration, maxDuration, "Skill %s took too long: %v", sk.Name, duration)
			t.Logf("Skill %s executed in %v", sk.Name, duration)
		})
	}
}

// TestSkillsByAgent verifies agent-skill assignments
func TestSkillsByAgent(t *testing.T) {
	registry := setupRegistry(t)
	ctx := context.Background()

	agents := []string{
		"implementation-agent",
		"research-agent",
		"strategic-agent",
		"health-agent",
	}

	for _, agent := range agents {
		t.Run(agent, func(t *testing.T) {
			skills, err := registry.ListByAgent(ctx, agent)
			require.NoError(t, err)

			t.Logf("%s has access to %d skills", agent, len(skills))

			// Hidden skills should not appear in agent queries
			for _, sk := range skills {
				assert.False(t, sk.Hidden, "Hidden skill %s should not appear for agent", sk.Name)
			}
		})
	}
}
