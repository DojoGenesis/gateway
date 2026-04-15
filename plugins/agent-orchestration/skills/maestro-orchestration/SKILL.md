---
name: maestro-orchestration
model: opus
description: "Conductor agent pattern that decomposes complex tasks, dispatches specialist sub-agents, manages dependencies, and synthesizes results into a unified deliverable. Use when: 'coordinate multiple specialist agents', 'orchestrate a complex multi-phase task', 'run a conductor pattern across agents', 'this task needs live agent coordination', 'decompose and dispatch to sub-agents'."
category: agent-orchestration
triggers:
  - "coordinate multiple specialist agents"
  - "orchestrate a complex multi-phase task"
  - "run a conductor pattern across agents"
  - "decompose and dispatch to sub-agents"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: objective
    type: string
    description: The complex task or objective that requires multi-agent coordination
    required: true
  - name: available_agents
    type: string[]
    description: List of specialist agent types available for dispatch (e.g., scout, architect, kraken, arbiter)
    required: false
outputs:
  - name: synthesis
    type: string
    description: Unified orchestration report with task decomposition, agent outputs, conflict resolutions, and integrated deliverables
---
# Maestro Orchestration

**Purpose:** Act as a conductor agent that decomposes complex objectives into subtasks, dispatches specialist sub-agents with appropriate isolation and model routing, manages inter-agent dependencies, resolves conflicts between agent outputs, and synthesizes results into a coherent deliverable.

---

## I. When to Use

- A task requires multiple specialist capabilities (research + implementation + validation)
- Work has both independent tracks (parallelizable) and dependent phases (sequential)
- The objective is too complex for a single agent pass -- it needs decomposition and coordination
- You need live conflict resolution when parallel agents produce contradictory outputs
- Multi-repo or multi-package work where different agents own different scopes

---

## II. Orchestration Patterns

### Pattern A: Hierarchical (Default for Implementation)

```
Maestro
  +-- architect  (plan)
  +-- kraken     (implement)
  +-- arbiter    (validate)
```

Best when the task has a clear plan-build-verify shape.

### Pattern B: Pipeline (Linear Dependency)

```
scout --> architect --> kraken --> arbiter --> herald
```

Best when each phase depends strictly on the prior phase's output.

### Pattern C: Swarm (Parallel Research)

```
Maestro
  +-- scout   (internal codebase)
  +-- oracle  (external research)
  +-- scout   (pattern analysis)
  --> synthesize all results
```

Best for broad information gathering before a decision.

### Pattern D: Generator-Critic (Iterative Refinement)

```
architect --> critic --> architect --> critic --> final
```

Best for high-quality outputs that benefit from review cycles.

### Pattern E: Jury (High-Stakes Decisions)

```
critic_1 --+
critic_2 --+--> majority vote --> decision
critic_3 --+
```

Best for architecture decisions or security reviews where multiple perspectives reduce risk.

---

## III. Workflow

### Step 1: Task Decomposition

Before dispatching any agent, decompose the objective:

1. **Parse the objective** into discrete subtasks
2. **Map dependencies** between subtasks (which must complete before others can start)
3. **Identify parallelism** -- subtasks with no dependencies between them can run concurrently
4. **Select pattern** from Section II based on the dependency graph shape

### Step 2: Agent-to-Task Assignment

For each subtask:

1. **Select the specialist agent** best suited (scout for exploration, architect for planning, kraken for implementation, arbiter for validation)
2. **Assign model routing** -- Opus for judgment/synthesis tasks, Sonnet for mechanical/execution tasks
3. **Define isolation** -- worktree if same repo different dirs, background if different repos, foreground if blocking
4. **Write the agent prompt** with: goal (first line), exact file paths, verification command, files NOT to touch

### Step 3: Dispatch and Monitor

Execute the orchestration plan:

1. **Phase blockers first** -- foreground agents that must complete before parallel work begins
2. **Launch parallel tracks** -- background agents for independent subtasks
3. **Collect results** -- read agent output files as they complete
4. **Resolve conflicts** -- when two agents produce contradictory changes, make the judgment call in the conductor thread

### Step 4: Synthesis

After all agents complete:

1. **Integrate outputs** -- merge code changes, combine reports, unify recommendations
2. **Verify combined result** -- run build/test gates on the integrated output
3. **Produce orchestration report** -- document what was done, by whom, what conflicts arose, and what the final state is
4. **Surface lessons learned** -- what worked, what failed, what to do differently next time

---

## IV. Agent Reference

| Agent | Purpose | Model | Best For |
|-------|---------|-------|----------|
| scout | Codebase exploration | sonnet | Finding patterns, mapping structure |
| oracle | External research | opus | Web/docs, best practices |
| architect | Feature planning | opus | Design, specification |
| kraken | TDD implementation | opus | Building features |
| arbiter | Validation/testing | opus | Unit/integration tests |
| critic | Code review | sonnet | Quality assessment |
| herald | Release preparation | sonnet | Deployment, changelog |
| phoenix | Refactor planning | opus | Technical debt |

If `available_agents` is not specified, select from the full roster based on the task.

---

## V. Output

- Orchestration Report containing: task decomposition table, pattern selected with rationale, execution log per phase, agent output summaries, conflict resolutions, integrated deliverables list, validation status, and lessons learned
- Saved to: project's orchestration output directory or returned inline for lightweight tasks
- Format: structured markdown with tables for subtask tracking

---

## VI. Examples

**Scenario 1:** "Wire the new authentication service across 3 microservices" --> Hierarchical pattern; architect produces integration plan; 3 parallel kraken agents (one per service, worktree isolation); arbiter validates cross-service contract tests; maestro resolves one shared-schema conflict between services A and C.

**Scenario 2:** "Research MCP architecture patterns and produce an ADR" --> Swarm pattern; scout analyzes internal codebase patterns, oracle researches external MCP ecosystem, second scout catalogs existing ADRs; maestro synthesizes into a single ADR with 3 options ranked by trade-offs.

**Scenario 3:** "Refactor the event system -- this is high-stakes, get multiple reviews" --> Generator-Critic into Jury; architect proposes refactor plan, critic_1 reviews for correctness, critic_2 reviews for performance, critic_3 reviews for backward compatibility; majority vote selects approach B; kraken implements; arbiter validates with regression suite.

---

## VII. Edge Cases

- No specialist agents available: maestro falls back to single-thread execution, performing each subtask sequentially in its own context
- Two agents modify the same file: do NOT dispatch both in parallel; run one foreground, apply its changes, then dispatch the second with the updated file state
- Agent reports "complete" but verification fails: re-dispatch with the failure output included in the prompt; cap retries at 2 before escalating to the user
- Objective is too vague to decompose: ask the user for clarification before dispatching; do not guess at subtask boundaries

---

## VIII. Anti-Patterns

- Dispatching agents before completing decomposition -- leads to redundant work and conflicts
- Using Opus for mechanical fix agents that have exact instructions -- waste of reasoning budget; route to Sonnet
- Launching more than 4 concurrent agents -- diminishing returns from coordination overhead; 3 is the sweet spot
- Passing synthesis responsibility to a sub-agent -- the conductor (maestro) must own synthesis; sub-agents produce parts, maestro produces the whole
- Skipping the verification gate after integration -- combined outputs can have conflicts that individual agent verification missed
