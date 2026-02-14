package orchestration

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NodeState represents the execution state of a plan node.
type NodeState string

const (
	NodeStatePending NodeState = "pending"
	NodeStateRunning NodeState = "running"
	NodeStateSuccess NodeState = "success"
	NodeStateFailed  NodeState = "failed"
	NodeStateSkipped NodeState = "skipped"
)

// Task represents a high-level user request to be orchestrated.
type Task struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PlanNode represents a single step in an execution plan DAG.
type PlanNode struct {
	ID           string                 `json:"id"`
	ToolName     string                 `json:"tool_name"`
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	State        NodeState              `json:"state"`
	Result       map[string]interface{} `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StartTime    *time.Time             `json:"start_time,omitempty"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	RetryCount   int                    `json:"retry_count"`
}

// Plan represents a DAG of PlanNodes to execute for a task.
type Plan struct {
	ID        string                 `json:"id"`
	TaskID    string                 `json:"task_id"`
	Nodes     []*PlanNode            `json:"nodes"`
	CreatedAt time.Time              `json:"created_at"`
	Version   int                    `json:"version"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewTask creates a new Task with a generated UUID.
func NewTask(userID, description string) *Task {
	return &Task{
		ID:          uuid.New().String(),
		UserID:      userID,
		Description: description,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

// NewPlan creates a new Plan for the given task ID.
func NewPlan(taskID string) *Plan {
	return &Plan{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Nodes:     make([]*PlanNode, 0),
		CreatedAt: time.Now(),
		Version:   1,
		Metadata:  make(map[string]interface{}),
	}
}

// NewPlanNode creates a new PlanNode with the given tool, parameters, and dependencies.
func NewPlanNode(toolName string, parameters map[string]interface{}, dependencies []string) *PlanNode {
	return &PlanNode{
		ID:           uuid.New().String(),
		ToolName:     toolName,
		Parameters:   parameters,
		Dependencies: dependencies,
		State:        NodeStatePending,
		RetryCount:   0,
	}
}

// IsReady returns true if all dependency nodes have completed successfully.
func (pn *PlanNode) IsReady(plan *Plan) bool {
	if len(pn.Dependencies) == 0 {
		return true
	}

	nodeMap := make(map[string]*PlanNode)
	for _, node := range plan.Nodes {
		nodeMap[node.ID] = node
	}

	for _, depID := range pn.Dependencies {
		depNode, exists := nodeMap[depID]
		if !exists {
			return false
		}
		if depNode.State != NodeStateSuccess {
			return false
		}
	}

	return true
}

// IsTerminal returns true if the node is in a final state.
func (pn *PlanNode) IsTerminal() bool {
	return pn.State == NodeStateSuccess || pn.State == NodeStateFailed || pn.State == NodeStateSkipped
}

// Duration returns the execution duration of the node.
func (pn *PlanNode) Duration() time.Duration {
	if pn.StartTime == nil {
		return 0
	}
	if pn.EndTime == nil {
		return time.Since(*pn.StartTime)
	}
	return pn.EndTime.Sub(*pn.StartTime)
}

// GetExecutableNodes returns all pending nodes whose dependencies are satisfied.
func (p *Plan) GetExecutableNodes() []*PlanNode {
	executableNodes := make([]*PlanNode, 0)

	for _, node := range p.Nodes {
		if node.State == NodeStatePending && node.IsReady(p) {
			executableNodes = append(executableNodes, node)
		}
	}

	return executableNodes
}

// GetNodeByID returns the node with the given ID, or nil if not found.
func (p *Plan) GetNodeByID(id string) *PlanNode {
	for _, node := range p.Nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}

// AllNodesCompleted returns true if all nodes are in a terminal state.
func (p *Plan) AllNodesCompleted() bool {
	if len(p.Nodes) == 0 {
		return false
	}

	for _, node := range p.Nodes {
		if node.State != NodeStateSuccess && node.State != NodeStateFailed && node.State != NodeStateSkipped {
			return false
		}
	}

	return true
}

// HasFailedNodes returns true if any node has failed.
func (p *Plan) HasFailedNodes() bool {
	for _, node := range p.Nodes {
		if node.State == NodeStateFailed {
			return true
		}
	}
	return false
}

// ValidateDAG checks that the plan forms a valid DAG (no cycles, valid dependencies).
func (p *Plan) ValidateDAG() error {
	nodeMap := make(map[string]*PlanNode)
	for _, node := range p.Nodes {
		nodeMap[node.ID] = node
	}

	for _, node := range p.Nodes {
		for _, depID := range node.Dependencies {
			if _, exists := nodeMap[depID]; !exists {
				return fmt.Errorf("node %s has invalid dependency %s", node.ID, depID)
			}
		}
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(nodeID string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node := nodeMap[nodeID]
		for _, depID := range node.Dependencies {
			if !visited[depID] {
				if hasCycle(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range nodeMap {
		if !visited[nodeID] {
			if hasCycle(nodeID) {
				return fmt.Errorf("plan contains cyclic dependencies")
			}
		}
	}

	return nil
}
