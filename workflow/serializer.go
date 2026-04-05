package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Marshal validates a WorkflowDefinition then serializes it to JSON.
// Keys are sorted for stable CAS hashes and clean git diffs.
func Marshal(def *WorkflowDefinition) ([]byte, error) {
	if err := Validate(def); err != nil {
		return nil, err
	}
	return json.MarshalIndent(def, "", "  ")
}

// Unmarshal deserializes JSON into a WorkflowDefinition and validates it.
func Unmarshal(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("workflow: invalid JSON: %w", err)
	}
	if err := Validate(&def); err != nil {
		return nil, err
	}
	return &def, nil
}

// Validate checks a WorkflowDefinition for structural correctness:
//  1. Name is non-empty
//  2. At least one step
//  3. All step IDs are unique
//  4. All depends_on references exist
//  5. No cycles (Kahn's algorithm -- BFS topological sort)
//  6. All skill references are non-empty
func Validate(def *WorkflowDefinition) error {
	// 1. Name is non-empty.
	if strings.TrimSpace(def.Name) == "" {
		return fmt.Errorf("workflow: name must not be empty")
	}

	// 2. At least one step.
	if len(def.Steps) == 0 {
		return fmt.Errorf("workflow: must have at least one step")
	}

	// Build lookup of step IDs.
	stepIDs := make(map[string]struct{}, len(def.Steps))

	// 3. All step IDs are unique.
	for _, s := range def.Steps {
		if _, exists := stepIDs[s.ID]; exists {
			return fmt.Errorf("workflow: duplicate step ID %q", s.ID)
		}
		stepIDs[s.ID] = struct{}{}
	}

	// 4. All depends_on references exist.
	for _, s := range def.Steps {
		for _, dep := range s.DependsOn {
			if _, exists := stepIDs[dep]; !exists {
				return fmt.Errorf("workflow: step %q depends on non-existent step %q", s.ID, dep)
			}
		}
	}

	// 5. No cycles -- Kahn's algorithm (BFS topological sort).
	if err := detectCycles(def.Steps); err != nil {
		return err
	}

	// 6. All skill references are non-empty.
	for _, s := range def.Steps {
		if strings.TrimSpace(s.Skill) == "" {
			return fmt.Errorf("workflow: step %q has empty skill reference", s.ID)
		}
	}

	return nil
}

// detectCycles uses Kahn's algorithm (BFS topological sort) to detect cycles.
// If not all nodes can be visited, the remaining nodes form a cycle.
func detectCycles(steps []Step) error {
	// Build in-degree map and adjacency list.
	inDegree := make(map[string]int, len(steps))
	dependents := make(map[string][]string, len(steps))

	for _, s := range steps {
		if _, ok := inDegree[s.ID]; !ok {
			inDegree[s.ID] = 0
		}
		for _, dep := range s.DependsOn {
			// dep -> s.ID (s.ID depends on dep, so dep has an outgoing edge to s.ID)
			dependents[dep] = append(dependents[dep], s.ID)
			inDegree[s.ID]++
		}
	}

	// Seed the queue with nodes that have zero in-degree.
	var queue []string
	for _, s := range steps {
		if inDegree[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++

		for _, dep := range dependents[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if visited != len(steps) {
		// Collect the step IDs that are part of the cycle.
		var cycleIDs []string
		for id, deg := range inDegree {
			if deg > 0 {
				cycleIDs = append(cycleIDs, id)
			}
		}
		sort.Strings(cycleIDs)
		return fmt.Errorf("workflow: cycle detected involving steps: [%s]",
			strings.Join(cycleIDs, ", "))
	}

	return nil
}

// MarshalCanvas serializes a CanvasState to JSON.
func MarshalCanvas(state *CanvasState) ([]byte, error) {
	return json.MarshalIndent(state, "", "  ")
}

// UnmarshalCanvas deserializes JSON into a CanvasState.
func UnmarshalCanvas(data []byte) (*CanvasState, error) {
	var state CanvasState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("workflow: invalid canvas JSON: %w", err)
	}
	return &state, nil
}
