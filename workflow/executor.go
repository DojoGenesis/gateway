package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// SkillRunner executes a named skill with the given inputs and returns the
// output string. Implementations bridge the workflow executor to the skill
// runtime (e.g. skill.Executor). The workflow package defines this interface
// to avoid importing the skill package directly.
type SkillRunner interface {
	RunSkill(ctx context.Context, skillName string, input map[string]string) (string, error)
}

// WorkflowExecutor manages workflow execution lifecycle.
// It resolves workflow definitions from CAS, validates them, executes
// steps in dependency order (parallel where possible), and publishes
// lifecycle events via the eventFn callback.
type WorkflowExecutor struct {
	cas         cas.Store
	eventFn     func(workflowID, stepID, status string)
	skillRunner SkillRunner
}

// NewWorkflowExecutor returns a WorkflowExecutor backed by the given CAS store.
// eventFn is called with (workflowID, stepID, status) for each step lifecycle
// event (e.g. "running", "completed", "failed", "skipped").
// If eventFn is nil, events are silently dropped.
// runner is an optional SkillRunner for dispatching steps to the skill runtime;
// if nil, steps fall back to a simulated execution (useful for tests).
func NewWorkflowExecutor(store cas.Store, eventFn func(string, string, string), runner ...SkillRunner) *WorkflowExecutor {
	if eventFn == nil {
		eventFn = func(string, string, string) {}
	}
	var sr SkillRunner
	if len(runner) > 0 && runner[0] != nil {
		sr = runner[0]
	}
	return &WorkflowExecutor{
		cas:         store,
		eventFn:     eventFn,
		skillRunner: sr,
	}
}

// ExecutionResult captures the outcome of a workflow run.
type ExecutionResult struct {
	// WorkflowName is the workflow identifier.
	WorkflowName string

	// Status is the overall outcome: "completed", "failed", or "cancelled".
	Status string

	// StartedAt is when execution began.
	StartedAt time.Time

	// CompletedAt is when execution finished (all steps resolved).
	CompletedAt time.Time

	// StepResults holds per-step outcomes keyed by step ID.
	StepResults map[string]StepResult
}

// StepResult captures the outcome of a single step execution.
type StepResult struct {
	// StepID is the step identifier within the workflow.
	StepID string

	// Skill is the skill invoked by this step.
	Skill string

	// Status is the step outcome: "completed", "failed", or "skipped".
	Status string

	// Output is the simulated step output.
	Output string

	// Error holds the error message if Status == "failed".
	Error string

	// Duration is how long the step took to execute.
	Duration time.Duration
}

// Execute runs the workflow identified by name. It resolves the latest version
// from CAS, validates the definition, then executes steps in topological order.
// Steps with no outstanding dependencies run in parallel.
//
// If a step fails, all steps that transitively depend on it are marked "skipped".
// Context cancellation is respected between step waves; if cancelled, remaining
// steps are skipped and Status is set to "cancelled".
func (e *WorkflowExecutor) Execute(ctx context.Context, name string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		WorkflowName: name,
		StartedAt:    time.Now().UTC(),
		StepResults:  make(map[string]StepResult),
	}

	// 1. Resolve workflow definition from CAS.
	def, err := e.resolveWorkflow(ctx, name)
	if err != nil {
		return nil, err
	}

	// 2. Validate (cycle check + structural).
	if err := Validate(def); err != nil {
		return nil, fmt.Errorf("executor: invalid workflow %q: %w", name, err)
	}

	// 3. Build dependency graph.
	//    dependents[X] = list of step IDs that directly depend on X.
	//    inDegree[X]   = number of unresolved dependencies for X.
	dependents := make(map[string][]string, len(def.Steps))
	inDegree := make(map[string]int, len(def.Steps))
	stepByID := make(map[string]Step, len(def.Steps))

	for _, s := range def.Steps {
		stepByID[s.ID] = s
		if _, ok := inDegree[s.ID]; !ok {
			inDegree[s.ID] = 0
		}
		for _, dep := range s.DependsOn {
			dependents[dep] = append(dependents[dep], s.ID)
			inDegree[s.ID]++
		}
	}

	// 4. Execute steps in topological waves.
	//    Track which steps have failed so dependents can be skipped.
	failedSteps := make(map[string]bool)
	var mu sync.Mutex // guards failedSteps and result.StepResults

	// ready holds step IDs whose in-degree has reached zero.
	ready := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			ready = append(ready, id)
		}
	}

	// Process waves until no more steps are ready.
	for len(ready) > 0 {
		// Check for context cancellation before starting each wave.
		if ctx.Err() != nil {
			// Mark all remaining steps (ready + pending) as skipped.
			e.markRemainingSkipped(ctx, def, result, inDegree, stepByID)
			result.Status = "cancelled"
			result.CompletedAt = time.Now().UTC()
			return result, nil
		}

		// Partition ready steps: those that can run vs those whose dependencies failed.
		canRun := make([]string, 0, len(ready))
		for _, id := range ready {
			if hasFailed(id, def.Steps, failedSteps) {
				// All deps failed — mark as skipped immediately.
				sr := StepResult{
					StepID: id,
					Skill:  stepByID[id].Skill,
					Status: "skipped",
				}
				mu.Lock()
				result.StepResults[id] = sr
				failedSteps[id] = true // propagate failure to dependents
				mu.Unlock()
				// Unlock any dependents of this now-skipped step.
				for _, dep := range dependents[id] {
					inDegree[dep]--
					if inDegree[dep] == 0 {
						ready = append(ready, dep)
					}
				}
			} else {
				canRun = append(canRun, id)
			}
		}

		// Steps in canRun are truly ready. Run them in parallel.
		var wg sync.WaitGroup
		newReady := make([]string, 0)
		var newReadyMu sync.Mutex

		for _, id := range canRun {
			wg.Add(1)
			go func(stepID string) {
				defer wg.Done()

				step := stepByID[stepID]
				e.eventFn(name, stepID, "running")

				start := time.Now()
				output, execErr := e.executeStep(ctx, step)
				duration := time.Since(start)

				sr := StepResult{
					StepID:   stepID,
					Skill:    step.Skill,
					Duration: duration,
					Output:   output,
				}

				if execErr != nil {
					sr.Status = "failed"
					sr.Error = execErr.Error()
					e.eventFn(name, stepID, "failed")

					mu.Lock()
					failedSteps[stepID] = true
					mu.Unlock()
				} else {
					sr.Status = "completed"
					e.eventFn(name, stepID, "completed")
				}

				mu.Lock()
				result.StepResults[stepID] = sr
				mu.Unlock()

				// Unlock dependents regardless of success/failure.
				mu.Lock()
				for _, dep := range dependents[stepID] {
					inDegree[dep]--
					if inDegree[dep] == 0 {
						newReadyMu.Lock()
						newReady = append(newReady, dep)
						newReadyMu.Unlock()
					}
				}
				mu.Unlock()
			}(id)
		}

		wg.Wait()

		// Replace ready with newly unblocked steps.
		ready = newReady
	}

	// 5. Determine overall status.
	mu.Lock()
	defer mu.Unlock()

	result.CompletedAt = time.Now().UTC()

	anyFailed := false
	for _, sr := range result.StepResults {
		if sr.Status == "failed" {
			anyFailed = true
			break
		}
	}

	if anyFailed {
		result.Status = "failed"
	} else {
		result.Status = "completed"
	}

	return result, nil
}

// resolveWorkflow fetches the latest version of a named workflow from CAS.
func (e *WorkflowExecutor) resolveWorkflow(ctx context.Context, name string) (*WorkflowDefinition, error) {
	ref, err := e.cas.Resolve(ctx, "workflow/"+name, "latest")
	if err != nil {
		if errors.Is(err, cas.ErrNotFound) {
			return nil, fmt.Errorf("executor: workflow %q not found", name)
		}
		return nil, fmt.Errorf("executor: resolve %q: %w", name, err)
	}

	data, _, err := e.cas.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("executor: get %q: %w", name, err)
	}

	def, err := Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("executor: unmarshal %q: %w", name, err)
	}

	return def, nil
}

// executeStep dispatches to the registered SkillRunner if available,
// otherwise falls back to a simulated execution for backward compatibility.
func (e *WorkflowExecutor) executeStep(ctx context.Context, step Step) (string, error) {
	// Check for cancellation before executing.
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Dispatch to the real skill runtime when a runner is configured.
	if e.skillRunner != nil {
		return e.skillRunner.RunSkill(ctx, step.Skill, step.Inputs)
	}

	// Fallback: simulate execution with a brief pause to allow concurrency
	// to be observable. This path is used when no SkillRunner is wired (tests).
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(1 * time.Millisecond):
	}

	return fmt.Sprintf("output of skill %q (step %q)", step.Skill, step.ID), nil
}

// hasFailed returns true if any dependency of the given step ID has failed.
func hasFailed(stepID string, steps []Step, failedSteps map[string]bool) bool {
	for _, s := range steps {
		if s.ID == stepID {
			for _, dep := range s.DependsOn {
				if failedSteps[dep] {
					return true
				}
			}
			return false
		}
	}
	return false
}

// markRemainingSkipped marks all steps that have not yet completed as "skipped".
// Called when context cancellation is detected.
func (e *WorkflowExecutor) markRemainingSkipped(
	ctx context.Context,
	def *WorkflowDefinition,
	result *ExecutionResult,
	inDegree map[string]int,
	stepByID map[string]Step,
) {
	for _, s := range def.Steps {
		if _, done := result.StepResults[s.ID]; !done {
			result.StepResults[s.ID] = StepResult{
				StepID: s.ID,
				Skill:  s.Skill,
				Status: "skipped",
			}
		}
	}
}
