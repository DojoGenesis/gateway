package agent

import (
	"context"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/orchestration"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// createTestAgentWithOrchestration creates a test agent with mock orchestration components
func createTestAgentWithOrchestration(t *testing.T) *PrimaryAgent {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}
	
	agent := NewPrimaryAgent(pm)
	
	eventChan := make(chan events.StreamEvent, 10)
	costTracker := services.NewCostTracker()
	budgetTracker := services.NewBudgetTracker(1000, 5000, 10000)
	
	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestration.Task) (*orchestration.Plan, error) {
			plan := orchestration.NewPlan(task.ID)
			plan.Nodes = []*orchestration.PlanNode{
				{
					ID:           uuid.New().String(),
					ToolName:     "test_tool",
					Parameters:   map[string]interface{}{"param1": "value1"},
					Dependencies: []string{},
					State:        orchestration.NodeStatePending,
				},
			}
			return plan, nil
		},
	}
	
	engine := orchestration.NewEngine(
		orchestration.DefaultEngineConfig(),
		mockPlanner,
		nil,
		eventChan,
		costTracker,
		budgetTracker,
	)
	
	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)
	agent.EnableOrchestration(true)
	
	t.Cleanup(func() {
		close(eventChan)
	})
	
	return agent
}

func TestShouldUseOrchestration(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		intent      Intent
		confidence  float64
		userTier    string
		setupAgent  func(*PrimaryAgent)
		expected    bool
		description string
	}{
		{
			name:        "multi-step with 'and then'",
			query:       "Research local-first software and then create a summary",
			intent:      IntentThink,
			confidence:  0.85,
			userTier:    "premium",
			expected:    true,
			description: "Should route multi-step query with 'and then'",
		},
		{
			name:        "multi-step with 'then'",
			query:       "Find three articles, then summarize them",
			intent:      IntentSearch,
			confidence:  0.9,
			userTier:    "premium",
			expected:    true,
			description: "Should route multi-step query with 'then'",
		},
		{
			name:        "numbered steps",
			query:       "1. Search for articles 2. Summarize findings 3. Create report",
			intent:      IntentBuild,
			confidence:  0.8,
			userTier:    "premium",
			expected:    true,
			description: "Should route query with numbered steps",
		},
		{
			name:        "research and creation",
			query:       "Research the latest trends and create a markdown report",
			intent:      IntentBuild,
			confidence:  0.85,
			userTier:    "premium",
			expected:    true,
			description: "Should route research+creation workflow",
		},
		{
			name:        "multiple actions",
			query:       "Analyze the data, extract key points, and summarize the findings",
			intent:      IntentThink,
			confidence:  0.9,
			userTier:    "premium",
			expected:    true,
			description: "Should route query with multiple actions",
		},
		{
			name:        "complex intent high confidence long query",
			query:       "I need you to help me understand the key differences between various local-first architectures and their trade-offs",
			intent:      IntentThink,
			confidence:  0.85,
			userTier:    "premium",
			expected:    true,
			description: "Should route complex intent with high confidence and long query",
		},
		{
			name:        "simple query",
			query:       "What is 2+2?",
			intent:      IntentGeneral,
			confidence:  0.95,
			userTier:    "premium",
			expected:    false,
			description: "Should not route simple query",
		},
		{
			name:        "single action",
			query:       "Summarize this article",
			intent:      IntentThink,
			confidence:  0.9,
			userTier:    "premium",
			expected:    false,
			description: "Should not route single action query",
		},
		{
			name:        "no orchestration components",
			query:       "Research and create a report",
			intent:      IntentBuild,
			confidence:  0.9,
			userTier:    "premium",
			setupAgent:  func(pa *PrimaryAgent) {
				pa.orchestrationPlanner = nil
				pa.orchestrationEngine = nil
			},
			expected:    false,
			description: "Should not route if orchestration components not initialized",
		},
		{
			name:        "complex but low confidence",
			query:       "Research trends and create report",
			intent:      IntentBuild,
			confidence:  0.6,
			userTier:    "premium",
			expected:    true,
			description: "Should still route if has clear multi-step pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal agent with existing test mocks
			agent := createTestAgentWithOrchestration(t)
			
			// Override setup if specified
			if tt.setupAgent != nil {
				tt.setupAgent(agent)
			}

			result := agent.shouldUseOrchestration(tt.intent, tt.confidence, tt.query, tt.userTier)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestGetOrchestrationReason(t *testing.T) {
	agent := createTestAgentWithOrchestration(t)

	tests := []struct {
		query    string
		intent   Intent
		expected string
	}{
		{
			query:    "Do this and then do that",
			intent:   IntentThink,
			expected: "multi_step_sequence_detected",
		},
		{
			query:    "Research trends and create a report",
			intent:   IntentBuild,
			expected: "research_and_creation_workflow",
		},
		{
			query:    "1. First step 2. Second step",
			intent:   IntentBuild,
			expected: "numbered_steps_detected",
		},
		{
			query:    "Analyze data and summarize findings",
			intent:   IntentThink,
			expected: "multiple_actions_detected",
		},
		{
			query:    "This is a very complex query that requires deep understanding",
			intent:   IntentThink,
			expected: "complex_intent_high_confidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			reason := agent.getOrchestrationReason(tt.intent, 0.85, tt.query)
			assert.Equal(t, tt.expected, reason)
		})
	}
}
