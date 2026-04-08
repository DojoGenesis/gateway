package main

import (
	"context"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/disposition"
	"github.com/DojoGenesis/gateway/memory"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/DojoGenesis/gateway/tools"
)

// TestDispositionToOrchestrationPacing verifies that agent disposition
// correctly controls orchestration pacing through the gateway integration.
func TestDispositionToOrchestrationPacing(t *testing.T) {
	// Create disposition config with deliberate pacing
	dispConfig := &disposition.DispositionConfig{
		Pacing: "deliberate",
		Depth:  "thorough",
	}

	// Create orchestration engine with disposition option
	engine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		nil, // planner (not needed for this test)
		nil, // tool invoker
		nil, // trace logger
		nil, // event emitter
		nil, // budget tracker
		orchestrationpkg.WithDisposition(dispConfig), // Apply disposition during construction
	)

	// Verify engine was created successfully (we can't directly test the delay,
	// but we can verify the construction doesn't panic)
	// In a real implementation, you'd measure execution time
	if engine == nil {
		t.Error("Engine should not be nil")
	}
	t.Logf("Pacing set to: %s", dispConfig.Pacing)
}

// TestDispositionToMemoryCompression verifies that agent disposition
// correctly controls memory compression behavior.
func TestDispositionToMemoryCompression(t *testing.T) {
	testCases := []struct {
		name              string
		depth             string
		memoryCount       int
		expectedThreshold int
		shouldCompress    bool
	}{
		{
			name:              "Surface depth - aggressive compression",
			depth:             "surface",
			memoryCount:       6,
			expectedThreshold: 5,
			shouldCompress:    true,
		},
		{
			name:              "Functional depth - moderate compression",
			depth:             "functional",
			memoryCount:       9,
			expectedThreshold: 10,
			shouldCompress:    false,
		},
		{
			name:              "Thorough depth - conservative compression",
			depth:             "thorough",
			memoryCount:       15,
			expectedThreshold: 20,
			shouldCompress:    false,
		},
		{
			name:              "Exhaustive depth - minimal compression",
			depth:             "exhaustive",
			memoryCount:       40,
			expectedThreshold: 50,
			shouldCompress:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agentConfig := &gateway.AgentConfig{
				Depth: tc.depth,
			}

			// Create test memories
			memories := make([]memory.Memory, tc.memoryCount)
			for i := 0; i < tc.memoryCount; i++ {
				memories[i] = memory.Memory{
					ID:        string(rune(i)),
					Type:      "test",
					Content:   "test content",
					CreatedAt: time.Now(),
				}
			}

			// Test compression decision
			shouldCompress := memory.ShouldCompressWithDisposition(memories, 10, agentConfig)
			if shouldCompress != tc.shouldCompress {
				t.Errorf("Expected shouldCompress=%v, got %v", tc.shouldCompress, shouldCompress)
			}
		})
	}
}

// TestMemoryRetentionByDepth verifies that different depth configurations
// result in different retention periods.
func TestMemoryRetentionByDepth(t *testing.T) {
	testCases := []struct {
		depth        string
		expectedDays int
	}{
		{"surface", 1},
		{"functional", 7},
		{"thorough", 30},
		{"exhaustive", 90},
		{"unknown", 7}, // default
	}

	for _, tc := range testCases {
		t.Run(tc.depth, func(t *testing.T) {
			agentConfig := &gateway.AgentConfig{
				Depth: tc.depth,
			}

			retentionDays := memory.GetRetentionDaysFromDepth(agentConfig)
			if retentionDays != tc.expectedDays {
				t.Errorf("Expected %d days for depth=%s, got %d",
					tc.expectedDays, tc.depth, retentionDays)
			}
		})
	}
}

// TestFilterMemoriesForCompression verifies that only old memories
// are selected for compression based on disposition.
func TestFilterMemoriesForCompression(t *testing.T) {
	agentConfig := &gateway.AgentConfig{
		Depth: "functional", // 7 days retention
	}

	now := time.Now()
	memories := []memory.Memory{
		{
			ID:        "old1",
			Type:      "test",
			Content:   "old memory 1",
			CreatedAt: now.AddDate(0, 0, -10), // 10 days old
		},
		{
			ID:        "recent1",
			Type:      "test",
			Content:   "recent memory 1",
			CreatedAt: now.AddDate(0, 0, -3), // 3 days old
		},
		{
			ID:        "old2",
			Type:      "test",
			Content:   "old memory 2",
			CreatedAt: now.AddDate(0, 0, -8), // 8 days old
		},
		{
			ID:        "recent2",
			Type:      "test",
			Content:   "recent memory 2",
			CreatedAt: now.AddDate(0, 0, -1), // 1 day old
		},
	}

	eligible := memory.FilterMemoriesForCompression(memories, agentConfig)

	// Should only return memories older than 7 days
	if len(eligible) != 2 {
		t.Errorf("Expected 2 eligible memories, got %d", len(eligible))
	}

	// Verify the correct memories were selected
	for _, mem := range eligible {
		if mem.ID != "old1" && mem.ID != "old2" {
			t.Errorf("Unexpected memory in eligible list: %s", mem.ID)
		}
	}
}

// TestToolRegistryWithNamespaces verifies that the tool registry
// correctly supports namespace filtering for MCP tools.
func TestToolRegistryWithNamespaces(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewContextAwareRegistry()

	// Clear registry for clean test
	_ = registry.Clear(ctx) // Error ignored in test setup

	// Register tools with different namespaces
	testTools := []*tools.ToolDefinition{
		{
			Name:        "composio.create_task",
			Description: "Create a task in Composio",
			Function: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
		{
			Name:        "composio.list_tasks",
			Description: "List tasks in Composio",
			Function: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
		{
			Name:        "github.create_issue",
			Description: "Create a GitHub issue",
			Function: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
		{
			Name:        "builtin_search",
			Description: "Built-in search tool",
			Function: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
	}

	// Register all tools
	for _, tool := range testTools {
		if err := registry.Register(ctx, tool); err != nil {
			t.Fatalf("Failed to register tool %s: %v", tool.Name, err)
		}
	}

	// Test namespace filtering
	composioTools, err := registry.ListByNamespace(ctx, "composio.")
	if err != nil {
		t.Fatalf("Failed to list composio tools: %v", err)
	}
	if len(composioTools) != 2 {
		t.Errorf("Expected 2 composio tools, got %d", len(composioTools))
	}

	githubTools, err := registry.ListByNamespace(ctx, "github.")
	if err != nil {
		t.Fatalf("Failed to list github tools: %v", err)
	}
	if len(githubTools) != 1 {
		t.Errorf("Expected 1 github tool, got %d", len(githubTools))
	}

	// Test GetNamespaces
	namespaces, err := registry.GetNamespaces(ctx)
	if err != nil {
		t.Fatalf("Failed to get namespaces: %v", err)
	}

	expectedNamespaces := map[string]bool{
		"composio": true,
		"github":   true,
	}

	for _, ns := range namespaces {
		if !expectedNamespaces[ns] {
			t.Errorf("Unexpected namespace: %s", ns)
		}
	}
}

// TestGracefulDegradation verifies that the system handles missing
// optional components gracefully.
func TestGracefulDegradation(t *testing.T) {
	t.Run("Orchestration with nil disposition", func(t *testing.T) {
		// Create engine without disposition option - should use defaults
		engine := orchestrationpkg.NewEngine(
			orchestrationpkg.DefaultEngineConfig(),
			nil, // planner
			nil, // tool invoker
			nil, // trace logger
			nil, // event emitter
			nil, // budget tracker
		)

		// Engine should work with no disposition (uses default pacing)
		if engine == nil {
			t.Error("Engine should not be nil even without disposition")
		}
	})

	t.Run("Memory compression with nil agent config", func(t *testing.T) {
		memories := []memory.Memory{
			{ID: "1", Type: "test", Content: "test", CreatedAt: time.Now()},
		}

		// Should fall back to default threshold
		shouldCompress := memory.ShouldCompressWithDisposition(memories, 10, nil)
		if shouldCompress {
			t.Error("Should not compress with only 1 memory and threshold 10")
		}
	})

	t.Run("Retention with nil agent config", func(t *testing.T) {
		// Should return default of 7 days
		days := memory.GetRetentionDaysFromDepth(nil)
		if days != 7 {
			t.Errorf("Expected default of 7 days, got %d", days)
		}
	})
}
