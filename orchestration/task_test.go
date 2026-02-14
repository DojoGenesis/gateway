package orchestration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	task := NewTask("user-1", "test task description")

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "user-1", task.UserID)
	assert.Equal(t, "test task description", task.Description)
	assert.NotZero(t, task.CreatedAt)
	assert.NotNil(t, task.Metadata)
}

func TestNewPlan(t *testing.T) {
	plan := NewPlan("task-1")

	assert.NotEmpty(t, plan.ID)
	assert.Equal(t, "task-1", plan.TaskID)
	assert.Equal(t, 1, plan.Version)
	assert.Empty(t, plan.Nodes)
	assert.NotNil(t, plan.Metadata)
}

func TestNewPlanNode(t *testing.T) {
	node := NewPlanNode("web_search", map[string]interface{}{"query": "test"}, []string{"dep-1"})

	assert.NotEmpty(t, node.ID)
	assert.Equal(t, "web_search", node.ToolName)
	assert.Equal(t, "test", node.Parameters["query"])
	assert.Equal(t, []string{"dep-1"}, node.Dependencies)
	assert.Equal(t, NodeStatePending, node.State)
	assert.Equal(t, 0, node.RetryCount)
}

func TestPlanNode_IsReady(t *testing.T) {
	plan := NewPlan("task-1")

	nodeA := NewPlanNode("tool_a", nil, nil)
	nodeA.ID = "node-a"

	nodeB := NewPlanNode("tool_b", nil, []string{"node-a"})
	nodeB.ID = "node-b"

	plan.Nodes = []*PlanNode{nodeA, nodeB}

	assert.True(t, nodeA.IsReady(plan))
	assert.False(t, nodeB.IsReady(plan))

	nodeA.State = NodeStateSuccess
	assert.True(t, nodeB.IsReady(plan))

	nodeA.State = NodeStateFailed
	assert.False(t, nodeB.IsReady(plan))
}

func TestPlanNode_IsReady_InvalidDependency(t *testing.T) {
	plan := NewPlan("task-1")
	node := NewPlanNode("tool_a", nil, []string{"nonexistent"})
	plan.Nodes = []*PlanNode{node}

	assert.False(t, node.IsReady(plan))
}

func TestPlanNode_IsTerminal(t *testing.T) {
	tests := []struct {
		state    NodeState
		terminal bool
	}{
		{NodeStatePending, false},
		{NodeStateRunning, false},
		{NodeStateSuccess, true},
		{NodeStateFailed, true},
		{NodeStateSkipped, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			node := &PlanNode{State: tt.state}
			assert.Equal(t, tt.terminal, node.IsTerminal())
		})
	}
}

func TestPlan_GetExecutableNodes(t *testing.T) {
	plan := NewPlan("task-1")

	nodeA := NewPlanNode("tool_a", nil, nil)
	nodeA.ID = "node-a"

	nodeB := NewPlanNode("tool_b", nil, []string{"node-a"})
	nodeB.ID = "node-b"

	nodeC := NewPlanNode("tool_c", nil, []string{"node-a"})
	nodeC.ID = "node-c"

	plan.Nodes = []*PlanNode{nodeA, nodeB, nodeC}

	exec := plan.GetExecutableNodes()
	require.Len(t, exec, 1)
	assert.Equal(t, "node-a", exec[0].ID)

	nodeA.State = NodeStateSuccess
	exec = plan.GetExecutableNodes()
	require.Len(t, exec, 2)
}

func TestPlan_GetNodeByID(t *testing.T) {
	plan := NewPlan("task-1")
	node := NewPlanNode("tool_a", nil, nil)
	node.ID = "node-x"
	plan.Nodes = []*PlanNode{node}

	found := plan.GetNodeByID("node-x")
	assert.NotNil(t, found)
	assert.Equal(t, "node-x", found.ID)

	notFound := plan.GetNodeByID("nonexistent")
	assert.Nil(t, notFound)
}

func TestPlan_AllNodesCompleted(t *testing.T) {
	plan := NewPlan("task-1")

	assert.False(t, plan.AllNodesCompleted())

	nodeA := NewPlanNode("tool_a", nil, nil)
	nodeB := NewPlanNode("tool_b", nil, nil)
	plan.Nodes = []*PlanNode{nodeA, nodeB}

	assert.False(t, plan.AllNodesCompleted())

	nodeA.State = NodeStateSuccess
	assert.False(t, plan.AllNodesCompleted())

	nodeB.State = NodeStateFailed
	assert.True(t, plan.AllNodesCompleted())
}

func TestPlan_HasFailedNodes(t *testing.T) {
	plan := NewPlan("task-1")
	nodeA := NewPlanNode("tool_a", nil, nil)
	plan.Nodes = []*PlanNode{nodeA}

	assert.False(t, plan.HasFailedNodes())

	nodeA.State = NodeStateFailed
	assert.True(t, plan.HasFailedNodes())
}

func TestPlan_ValidateDAG_Valid(t *testing.T) {
	plan := NewPlan("task-1")

	nodeA := NewPlanNode("tool_a", nil, nil)
	nodeA.ID = "a"

	nodeB := NewPlanNode("tool_b", nil, []string{"a"})
	nodeB.ID = "b"

	nodeC := NewPlanNode("tool_c", nil, []string{"a"})
	nodeC.ID = "c"

	plan.Nodes = []*PlanNode{nodeA, nodeB, nodeC}

	assert.NoError(t, plan.ValidateDAG())
}

func TestPlan_ValidateDAG_InvalidDependency(t *testing.T) {
	plan := NewPlan("task-1")
	node := NewPlanNode("tool_a", nil, []string{"nonexistent"})
	node.ID = "a"
	plan.Nodes = []*PlanNode{node}

	err := plan.ValidateDAG()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dependency")
}

func TestPlan_ValidateDAG_CyclicDependency(t *testing.T) {
	plan := NewPlan("task-1")

	nodeA := NewPlanNode("tool_a", nil, []string{"b"})
	nodeA.ID = "a"

	nodeB := NewPlanNode("tool_b", nil, []string{"a"})
	nodeB.ID = "b"

	plan.Nodes = []*PlanNode{nodeA, nodeB}

	err := plan.ValidateDAG()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic")
}

func TestPlanNode_Duration(t *testing.T) {
	node := &PlanNode{}

	assert.Equal(t, time.Duration(0), node.Duration())

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 0, 0, 5, 0, time.UTC)
	node.StartTime = &start
	node.EndTime = &end

	assert.Equal(t, 5*time.Second, node.Duration())
}
