// Package orchestration provides DAG-based task planning and execution for the
// Agentic Gateway.
//
// It models work as a Plan (a directed acyclic graph of PlanNodes), where each
// node represents a tool invocation with typed parameters and explicit
// dependencies. The Engine executes these plans with:
//
//   - Parallel node execution (respecting dependency edges)
//   - Disposition-driven pacing (deliberate → rapid)
//   - Automatic retry with exponential backoff and jitter
//   - Circuit breaker per tool for persistent failures
//   - Auto-replanning when transient errors occur
//   - Budget-aware execution with token cost estimation
//
// Key types: Task, Plan, PlanNode, Engine, EngineConfig.
package orchestration
