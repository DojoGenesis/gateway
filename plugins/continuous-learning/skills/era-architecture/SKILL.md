---
name: era-architecture
model: opus
description: "Produces an Era Master Plan document defining shared vocabulary, architectural constraints, and a release roadmap across 3-7 releases. Use when: 'plan this era', 'define the architecture for multiple releases', 'create a multi-release vision', 'transition to a new product era', 'establish conceptual coherence across releases'."
category: continuous-learning
triggers:
  - "plan this era"
  - "define the architecture for multiple releases"
  - "create a multi-release vision"
  - "transition to a new product era"
  - "establish conceptual coherence across releases"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: era_description
    type: string
    description: High-level description of the product era and its goals
    required: true
  - name: release_count
    type: number
    description: Number of releases to plan across (3-7)
    required: false
outputs:
  - name: era_master_plan
    type: ref
    format: cas-ref
    description: Era Master Plan document defining shared vocabulary, architectural constraints, and release roadmap
---

# era-architecture

## I. The Philosophy

An era is not a list of releases. It's a conceptual architecture that coordinates multiple releases under shared constraints.

When you go from "we need to rebuild the frontend" to "we need a complete multi-release era plan," you're making a qualitative shift. Instead of asking "what should v0.2.1 do?" you're asking "what vocabulary, architectural patterns, and design language will span all releases in this era?"

This shift matters because:
- **Coherence**: Without a shared architecture, releases drift apart. Release A uses one data model, Release B contradicts it.
- **Dependency clarity**: You can't plan releases independently. A shared design language in Release 1 enables faster work in Release 2.
- **Decision efficiency**: The era's conceptual architecture is the constitution that downstream decisions must obey. This prevents constant re-litigation of foundational choices.

The key insight: scouts explore the facets (not releases). From those explorations, you extract the era's architecture (not the other way around).

## II. When to Use This Skill

**Use era-architecture when:**
- Planning a major product pivot that spans 3+ releases
- You need to define shared conceptual vocabulary before decomposing into releases
- Transitioning between product eras (e.g., engine-building → fresh shell → social layer)
- The scope is too large for one release but needs coherent vision across all releases
- You need cross-cutting architectural decisions (data model, design language, navigation patterns)

**Do NOT use era-architecture when:**
- Planning a single release → use `strategic-scout` + `parallel-tracks`
- Writing specs → use `specification-writer` or `frontend-from-backend`
- Converting specs to prompts → use `spec-constellation-to-prompt-suite` or `zenflow-prompt-writer`
- Exploring one strategic question → use `strategic-scout`

## III. The Workflow (7 Steps)

### Step 1: Name the Era and Its Predecessor

Give the era a name that captures its purpose (e.g., "The Fresh Shell", "The Engine", "The Social Layer").

Document what the previous era accomplished and what it leaves behind. Define the handoff: what assets/code/patterns carry forward, what gets archived.

This framing prevents the new era from being aimless. It grounds the work in what came before and what you're building toward.

### Step 2: Scout the Facets

Run 3-5 strategic scouts, each exploring a different facet of the era. Facets are not releases—they're architectural questions:
- Navigation/information architecture
- Data model and persistence
- Design language and UX philosophy
- Backend integration strategy
- Social/collaboration strategy (if applicable)

Each scout produces insights and open questions. Gather human decisions on those open questions before proceeding to Step 3. This is where the team shapes the architecture.

### Step 3: Define the Conceptual Architecture

From the scout insights and decisions, extract the era's conceptual vocabulary:
- **Core metaphors**: What's the guiding image? (e.g., "garden", "cognitive routes", "bonsai")
- **Architectural patterns**: What repeating structures apply everywhere? (e.g., "entity-as-lens", "content-edge transitions")
- **Design language**: Colors, typography, interaction patterns that all releases obey
- **Data model**: The types and relationships that span all releases

Write these as constraints: every release in the era must obey them. This is the era's constitution—it doesn't change per release.

### Step 4: Decompose into Releases

Define 3-7 releases that build on each other. Each release should have:
- A clear theme/name (e.g., "The Foundation", "The Shallow End", "The Deep End")
- Scope: what's built
- Dependencies: what must exist first
- Estimated effort

The first release is the thickest (it builds foundations). Later releases are thinner because the patterns are established. Some releases can run in parallel if they don't share dependencies.

### Step 5: Build the Dependency Graph

Map release-level dependencies (which releases block which). Map spec-level dependencies within releases (which specs must exist before others). Identify the critical path and parallel opportunities.

### Step 6: Write the Master Plan

Use the Master Plan Template (Section IV). The master plan is the single commissioning document for the entire era. Include vision, conceptual architecture summary, release roadmap, dependency graph, constraints, and parallel track allocation.

The master plan is a living document—decisions from later scouts update it. But the core architecture should remain stable.

### Step 7: Drop into Single-Release Work

For the first release, write detailed specs. Then convert specs to implementation prompts. Commission and execute the first release.

Before starting the next release, check: do the conceptual architecture constraints still hold? Do decisions from the first release affect later releases? Update the master plan if needed.

## IV. Master Plan Template

```markdown
# [Era Name] Master Plan
**Author:** [names]
**Status:** [Active/Complete]
**Era:** v[X].0 through v[X].N

## 1. Vision
[1-2 sentences: what this era accomplishes]

## 2. What Came Before
[Brief summary of previous era, what carries forward]

## 3. Conceptual Architecture
### Core Metaphors
### Architectural Patterns
### Design Language
### Data Model

## 4. Release Roadmap
| Version | Name | Focus | Dependencies | Est. Effort |

## 5. Dependency Graph
[ASCII or description showing release ordering]

## 6. Constraints
[Rules that all releases must obey]

## 7. Parallel Track Allocation
| Release | Tracks | Can Parallel With |

## 8. Commissioned Specifications
| Spec | Release | Status |

## 9. Next Steps
[Current focus and what follows]
```

## V. Best Practices

- **Eras need names**: Names create shared vocabulary for the team. "The Fresh Shell" is better than "v0.2 refactor".
- **Scout facets, not releases**: Architecture emerges from cross-cutting exploration, not from asking "what should v0.2.1 do?"
- **Conceptual architecture is the output**: It constrains everything downstream and prevents releases from drifting apart.
- **First release is heaviest**: Later releases feel lighter because the patterns are established.
- **Master plans are living documents**: Use decision protocols when decisions arrive that change scope.
- **Don't over-plan later releases**: Define theme and rough scope, but save detailed specs for when earlier releases are done and the architecture has been validated in practice.

## VI. Quality Checklist

- [ ] Era has a name and a clear relationship to its predecessor
- [ ] 3+ facets scouted with insights and decisions recorded
- [ ] Conceptual architecture defined (metaphors, patterns, design language, data model)
- [ ] 3-7 releases defined with themes, scope, and dependencies
- [ ] Dependency graph shows ordering and parallel opportunities
- [ ] Master plan written as single commissioning document
- [ ] First release has detailed specs ready for implementation
- [ ] Constraints are clear enough that any release can check compliance

---

## Output

- An Era Master Plan markdown document saved to the project's `docs/` or `AgenticStackOrchestration/phases/` directory
- Named: `[era-name]_master_plan.md`
- Contents: Vision, What Came Before, Conceptual Architecture, Release Roadmap table, Dependency Graph, Constraints, Parallel Track Allocation, Commissioned Specifications, Next Steps

---

## Examples

**Scenario 1:** "Plan the Fresh Shell era across 5 releases" → Era Master Plan with named conceptual architecture (metaphors, design language, data model), 5 releases defined with themes and dependencies, dependency graph showing Release 1 as blocking, Releases 3-5 as parallelizable

**Scenario 2:** "We need to transition from the engine-building era to the social layer era" → Era Master Plan capturing what the engine era accomplished, what carries forward, and the new era's conceptual architecture for social features across 3 releases

---

## Edge Cases

- **Only 1-2 releases planned:** This is a single-release scope, not an era — use `strategic-scout` + `parallel-tracks` instead
- **Team disagrees on conceptual vocabulary during Step 3:** Surface the disagreement as an open decision, do not proceed to Step 4 until it is resolved — unresolved vocabulary disagreements invalidate downstream release plans
- **First release's specs are not complete before commissioning begins:** Block commissioning until specs exist; attempting to build Release 1 without detailed specs turns the era plan into a wish list

---

## Anti-Patterns

- Defining releases first and then trying to extract a conceptual architecture — architecture must precede decomposition, not follow it
- Writing a master plan that changes with every release — the conceptual architecture should be stable; only the "Next Steps" and "Commissioned Specifications" sections evolve
- Using era-architecture for a single feature or a bug-fix sprint — the overhead is only justified when shared architectural decisions span 3+ releases
- Over-specifying later releases before earlier releases validate the architecture in practice — save detailed specs for when the patterns are proven
