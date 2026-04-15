---
name: orchestration-pattern-selector
description: Selects the correct multi-agent orchestration pattern for a task using an 11-signal selection matrix. Use when choosing which orchestration pattern fits a multi-agent task.
model: opus
category: agent-orchestration
version: 1.0.0
tags: [orchestration, patterns, selection, architecture, agents]
triggers:
  - "select orchestration pattern"
  - "which orchestration pattern fits"
  - "choose multi-agent pattern"
  - "orchestration pattern for this task"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: task
    type: string
    description: Description of the multi-agent task to be orchestrated
    required: true
  - name: constraints
    type: string
    description: Known constraints — agent count limits, latency requirements, failure tolerance
    required: false
outputs:
  - name: dispatch_plan
    type: text
    format: markdown
    description: Selected pattern with rationale, configured parameters, and a dispatch plan ready for parallel-dispatch or maestro-orchestration
---
# Orchestration Pattern Selector Skill

## Philosophy

Pattern selection is the architect's decision that happens before any agent is dispatched. Choosing the wrong pattern wastes more work than any individual agent failure — a Swarm dispatched for a Pipeline task produces divergent outputs; a Pipeline dispatched for a Map-Reduce task serializes work that could have parallelized. The cost of selecting the wrong pattern compounds with agent count.

This skill exists because most orchestration failures trace back to one of two mistakes: using Pipeline for everything (safe but slow and single-point-of-failure-prone), or dispatching agents without choosing a pattern at all (ad-hoc coordination that breaks at scale). Pattern selection is not overhead — it is the work that makes everything else deterministic.

## When to Use

- Before dispatching 2+ agents on any task with non-obvious topology
- When a task involves multiple output types, multiple domains, or multiple phases
- When the failure cost of wrong coordination is high (architectural changes, security reviews, large refactors)
- When asked to "plan how to orchestrate" a complex objective before starting execution

## The 12 Patterns

### Quick Reference

| # | Pattern | Topology | Best For |
|---|---------|----------|----------|
| 1 | Pipeline | Linear chain | Clear sequential phases where each stage feeds the next |
| 2 | Map-Reduce | Fan-out / fan-in | Independent subtasks that merge into one output |
| 3 | Hierarchical | Tree | Domain expertise at multiple levels, complex delegation |
| 4 | Swarm | Peer-to-peer | Exploratory tasks with unknown decomposition |
| 5 | Generator-Critic | Two-agent loop | Iterative refinement until quality threshold met |
| 6 | Adversarial | Red/blue + judge | Security auditing, assumption stress-testing |
| 7 | Jury | Parallel + aggregator | High-stakes decisions needing diverse perspectives |
| 8 | Blackboard | Shared workspace | Emergent solutions from accumulated contributions |
| 9 | Chain-of-Responsibility | Linear handoff | Tiered fallback, progressive specialization |
| 10 | Circuit-Breaker | Wrapper | External integration points with failure risk |
| 11 | Event-Driven | Reactive subscribers | Continuous monitoring, real-time pipelines |
| 12 | Meta-Selector | Decision agent | General orchestration where task type varies |

## Selection Matrix

Evaluate the task against these 11 signals. The first strong match determines the pattern. When multiple signals fire, use the tie-breaking column.

| Signal | Strength | Pattern | Tie-break note |
|--------|----------|---------|----------------|
| Task has clear sequential phases (phase A must complete before phase B starts) | High | Pipeline | Default safe choice if unsure |
| Task decomposes into N independent units of the same type | High | Map-Reduce | Requires a natural unit boundary (file, skill, record) |
| Task needs multiple domain experts who report upward | High | Hierarchical | Use when coordination overhead justifies the tree |
| Task decomposition is unknown — exploration required first | Medium | Swarm | Only when you genuinely cannot plan the subtasks |
| Output quality improves through revision cycles | High | Generator-Critic | Set max_iterations before dispatching |
| Correctness matters more than speed; diverse opinions reduce risk | High | Jury | Use odd agent count; diversity of disposition is load-bearing |
| Task benefits from adversarial pressure (red team / stress test) | High | Adversarial | Requires a neutral judge agent |
| Solution emerges from accumulated contributions with no master plan | Medium | Blackboard | Needs shared CAS or mutable shared state |
| Multiple handlers exist; only one should fire per request | High | Chain-of-Responsibility | Requires priority ordering of handlers |
| Task calls an external service that may fail or rate-limit | High | Circuit-Breaker | Wrap, do not replace, the underlying pattern |
| Task reacts to a stream of incoming events | High | Event-Driven | Requires backpressure design; not suitable for one-shot tasks |

**Default:** If no signal fires with high confidence, use Pipeline. It is the safest topology and fails gracefully.

## Workflow

### Step 1: Characterize the Task

Before consulting the matrix, answer these questions:

1. **Phases or units?** Does the task have sequential phases (A then B then C) or parallel units (analyze files 1 through N)? Phases → Pipeline or Hierarchical. Units → Map-Reduce or Swarm.
2. **Known or unknown decomposition?** Can you enumerate all subtasks now, or will subtasks only become visible during execution? Known → structured patterns. Unknown → Swarm or Blackboard.
3. **Output ordering?** Does the final output depend on the order subtasks complete? Yes → Pipeline or Chain-of-Responsibility. No → Map-Reduce or Jury.
4. **Failure tolerance?** Can partial results produce a usable output, or does every agent need to succeed? Partial ok → Map-Reduce (reducer handles gaps). All required → Pipeline with escalation.
5. **Quality vs. speed?** Is a single high-quality pass sufficient, or does refinement add material value? Single pass → Pipeline or Hierarchical. Refinement adds value → Generator-Critic.

### Step 2: Apply the Selection Matrix

Score the task against the 11 signals in the matrix. Use the first high-strength match. If two high-strength signals fire simultaneously (e.g., sequential phases AND quality refinement needed), stack the patterns: outer Pipeline with a Generator-Critic stage inside.

### Step 3: Document the Rationale

Write one paragraph explaining:
- Which signals fired
- Why the selected pattern matches those signals
- What the failure mode is and how you will guard against it

This is not optional overhead — it is the record that lets you debug the orchestration if it fails.

### Step 4: Configure Pattern Parameters

For the selected pattern, specify:

- **Agent count**: How many agents, what roles, what model (sonnet/opus)
- **File manifests** (for patterns that write code): which files each agent may touch
- **Timeout**: Maximum wall-clock time before the pattern is considered failed
- **Retry policy**: How many times to retry a failed agent before escalating
- **Success criteria**: The observable state that means the pattern completed correctly

### Step 5: Output the Dispatch Plan

Produce a dispatch plan in this format, ready for parallel-dispatch or maestro-orchestration to consume:

```
Pattern: [name]
Rationale: [one paragraph]
Failure mode: [what breaks and how you guard against it]

Agents:
  - Role: [name], Model: [sonnet|opus], Scope: [files or domain], Task: [one sentence]
  - ...

Coordination:
  - Communication: [direct|mailbox|blackboard|broadcast]
  - Termination: [when is the pattern done]
  - Max retries: [N per agent]
  - Wave structure: [if applicable]

Success criteria:
  - [observable check 1]
  - [observable check 2]
```

## Examples

**Example 1: "Wire the new authentication service across 3 microservices"**

Characterization: Three independent implementation targets (service A, B, C), no ordering dependency between them, all need to reach the same interface contract. Clear decomposition into 3 units.

Signal fired: "Task decomposes into N independent units of the same type" (Map-Reduce) — but these units are writing code, not analyzing data, and they need a coordination step (architect produces the interface contract first). This is Hierarchical, not Map-Reduce, because there is a root decision that the subtasks depend on.

Selected pattern: **Hierarchical**

```
Pattern: Hierarchical
Rationale: Three parallel implementation tracks (one per service) depend on a shared
interface contract that must be produced first. The architect produces the contract;
three kraken agents implement it independently; an arbiter validates cross-service
contract tests. This is not Map-Reduce because the units are not homogeneous — each
service has different internal structure.
Failure mode: Architect produces ambiguous contract → all three krakens diverge.
Guard: Arbiter reviews the contract before krakens are dispatched.

Agents:
  - Role: architect, Model: opus, Scope: auth/contract.go, Task: Define the shared AuthClient interface
  - Role: kraken-A, Model: sonnet, Scope: service-a/auth/, Task: Implement AuthClient for service A
  - Role: kraken-B, Model: sonnet, Scope: service-b/auth/, Task: Implement AuthClient for service B
  - Role: kraken-C, Model: sonnet, Scope: service-c/auth/, Task: Implement AuthClient for service C
  - Role: arbiter, Model: opus, Scope: all, Task: Run cross-service contract tests

Coordination:
  - Communication: direct (architect output feeds krakens; krakens feed arbiter)
  - Termination: arbiter reports all contract tests passing
  - Max retries: 2 per kraken before escalating
  - Wave structure: Wave 0 (architect alone) → Wave 1 (3 krakens parallel) → Wave 2 (arbiter)

Success criteria:
  - go build ./... passes for all three services
  - go test ./auth/... passes in all three services
  - arbiter contract test suite: 0 failures
```

**Example 2: "Review this architectural proposal — it's high stakes, we need multiple perspectives"**

Characterization: One input (the proposal), multiple independent evaluators, all evaluate the same artifact, output is a synthesis of perspectives. Quality through diversity.

Signal fired: "Correctness matters more than speed; diverse opinions reduce risk" → Jury.

Selected pattern: **Jury**

```
Pattern: Jury
Rationale: A single reviewer introduces bias and blind spots. Three jurors with
distinct dispositions (conservative/measured, progressive/bold, pragmatic/precise)
evaluate the same proposal independently, preventing groupthink. An aggregator
synthesizes into a ranked set of concerns with confidence levels.
Failure mode: Jurors with same model+disposition produce identical verdicts.
Guard: Assign different ADA dispositions; consider using different model sizes.

Agents:
  - Role: juror-conservative, Model: sonnet, Disposition: measured, Task: Evaluate proposal for risk, unknowns, and downside cases
  - Role: juror-progressive, Model: opus, Disposition: bold, Task: Evaluate proposal for missed opportunities and underambition
  - Role: juror-pragmatic, Model: sonnet, Disposition: precise, Task: Evaluate proposal for implementation feasibility and timeline accuracy
  - Role: aggregator, Model: opus, Scope: all juror outputs, Task: Synthesize verdicts into ranked concerns with quorum labels

Coordination:
  - Communication: mailbox (jurors do not see each other's verdicts before submitting)
  - Termination: aggregator completes synthesis
  - Quorum threshold: 2/3 agreement required to label a concern "consensus"
  - Max retries: 1 per juror (jurors rarely fail; failure usually means ambiguous input)

Success criteria:
  - All 3 jurors submitted independent verdicts
  - Aggregator produced ranked concern list with quorum labels
  - At least one concern reached 2/3 consensus (if none do, the proposal may be genuinely ambiguous — surface to user)
```

## Edge Cases

- **Two high-strength signals fire simultaneously**: Stack patterns. Example: Pipeline with a Generator-Critic stage inside the implementation phase. Document both patterns and their nesting.
- **Task appears to need all the patterns**: This is a signal that the task is too large and needs decomposition into sub-tasks first. Break it down before selecting a pattern.
- **Circuit-Breaker selected**: This is always a wrapper, not a standalone pattern. Identify what the underlying pattern is (usually Pipeline or Map-Reduce), then wrap the external-service stage with a circuit-breaker.
- **Swarm selected but agents diverge**: Swarm requires a convergence mechanism. If agents are not self-organizing toward a shared output, add a synthesis step or switch to Hierarchical with a coordinator.
- **Event-Driven selected but task is one-shot**: Event-Driven is not appropriate for tasks with a defined end state. Use Pipeline or Hierarchical instead. Event-Driven is for tasks that never "complete" — they run continuously.

## Anti-Patterns

- **Defaulting to Pipeline for everything**: Pipeline serializes work and has a single point of failure at each stage. It is the right default only when sequential dependency is real. Applying it to parallelizable tasks wastes time.
- **Jury with same-model, same-disposition jurors**: Three identical agents produce three identical outputs. The entire value of Jury is diversity. Explicitly vary dispositions and consider varying model sizes.
- **Generator-Critic without a max iteration limit**: Always set a maximum before dispatch. Without it, the loop runs indefinitely. The critic should have a "good enough" threshold, not just a "perfect" threshold.
- **Skipping rationale documentation**: Pattern selection without written rationale produces orchestrations that fail silently — you cannot debug what you did not reason about.
- **Using Swarm when decomposition is actually known**: Swarm's coordination overhead is only justified when you genuinely cannot enumerate the subtasks. If you can list the subtasks, use Map-Reduce or Hierarchical instead.
