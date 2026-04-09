---
name: semantic-clusters
model: opus
description: Produces a behavioral architecture map grouping components by action-verb clusters (CONVERSE, REASON, REMEMBER, etc.) that reveals capabilities, gaps, and cross-cutting concerns invisible to directory trees. Use when: "what does this app actually do", "walk me through the features", "plan a refactor", "audit feature coverage", "explain this system to someone new".
category: system-health

inputs:
  - name: repo_path
    type: string
    description: Path to the repository or codebase to analyze
    required: true
outputs:
  - name: behavioral_map
    type: string
    description: Behavioral architecture map grouping components by action-verb clusters revealing capabilities, gaps, and cross-cutting concerns
---

# Semantic Clusters Skill

**Version:** 1.0
**Created:** 2026-02-08
**Author:** Cruz + Manus (Cowork)
**Origin:** Elevated from the `repo-status` skill's semantic clusters reference, which was codified from a live strategic-scout audit of Dojo Genesis (13 clusters, 333 Go files, 416 frontend files mapped).
**Lineage:** strategic-scout (exploration) → repo-status (formalization) → this standalone skill.

---

## I. The Philosophy: Behavior Over Location

Every codebase has two architectures:

1. **The filesystem architecture** — where files live. Directories, packages, modules. This is what `ls` shows you.
2. **The behavioral architecture** — what the system *does*. Capabilities that cross-cut directories, features that span layers.

Most people only see architecture #1. They think in terms of `frontend/`, `backend/`, `utils/`. But understanding comes from architecture #2 — the verbs.

A "chat" feature isn't in one directory. It's a component in the frontend, a handler in the backend, a state engine in a context, a streaming service, an SSE connection, and a set of tools. These parts live in 6 different directories. But they all serve one verb: **CONVERSE**.

Semantic clusters make the behavioral architecture visible. Each cluster is named with an action verb, and every significant component in the codebase maps to one (sometimes two) clusters. The result is a map of *what the system can do* — not just where its files happen to be.

---

## II. When to Use This Skill

- **Exploring a new codebase:** Before diving into files, map the behavioral capabilities to build your mental model.
- **Explaining a system:** Clusters make better explanations than directory trees because they answer "what does it do?" not "where are the files?"
- **Planning a refactor:** Clusters reveal which components serve the same capability. Refactoring within a cluster is safer than across clusters.
- **Auditing feature coverage:** Clusters expose gaps — capabilities the system lacks or has only partially implemented.
- **Identifying architectural confusion:** If a component maps to 3+ clusters, it's probably doing too much. If a directory has components in 5 different clusters, it may need restructuring.
- **Writing status documents:** Section 4 of the `status-template` skill uses clusters. This skill provides the methodology.

---

## III. The Starter Verbs

These 13 action verbs cover most software systems. Read `references/verb-catalog.md` for detailed definitions, component examples, and signals for each.

| Verb | What It Means | Example Systems That Need It |
|------|--------------|---------------------------|
| **CONVERSE** | Real-time communication with users | Chat apps, messaging, support tools |
| **REASON** | Thinking, planning, deciding | AI agents, rule engines, recommendation systems |
| **REMEMBER** | Storing and recalling knowledge | Knowledge bases, caching, memory systems |
| **OBSERVE** | Watching and reporting | Monitoring, analytics, logging, tracing |
| **LEARN** | Adapting based on feedback | Calibration, A/B testing, preference learning |
| **ACT** | Executing side effects | Tool systems, API calls, file operations, cron |
| **PROTECT** | Enforcing boundaries | Auth, encryption, rate limiting, validation |
| **CONNECT** | Integrating externally | Plugins, APIs, webhooks, bots, OAuth |
| **PRESENT** | Rendering UI | Shells, layouts, component libraries, themes |
| **PERSIST** | Storing data durably | Databases, migrations, ORMs, caches |
| **BUILD** | Building, testing, shipping | CI/CD, Docker, scripts, test suites |
| **THINK** | Meta-cognition about itself | Skills, prompts, documentation, retrospectives |
| **ORCHESTRATE** | Coordinating multi-step work | DAG engines, task queues, workflows, sagas |

Not every project needs all 13. A simple CRUD app might only have CONVERSE, PERSIST, PRESENT, PROTECT, and BUILD. A complex AI platform might use all 13 plus custom ones.

---

## IV. The Clustering Workflow

### Step 1: Inventory First

You can't cluster what you haven't seen. Before clustering, you need a component inventory — a list of every significant component with its location, approximate LOC, and current status.

If you've already run the `repo-status` skill (Phase 2), use that inventory. Otherwise, walk the filesystem yourself:

```bash
# Get top-level shape
ls -la project-root/

# Recursively explore significant directories
find project-root/src -type f | head -50
find project-root/backend -type d

# Count LOC per directory
find project-root/backend -name "*.go" | xargs wc -l | tail -1
```

### Step 2: Assign Verbs

For each significant component in your inventory, ask: **"What verb describes what this does?"**

Rules of thumb:
- Most components map to **one** verb. If you can't decide, pick the one that best describes the component's *primary purpose*.
- Some components legitimately serve **two** verbs (cross-cluster). This is fine — note both.
- If a component maps to **three or more** verbs, it's probably doing too much. Flag it as an architectural concern.
- If a component doesn't fit any verb, it might be dead code, or you might need a **new verb** (see Section V).

### Step 3: Build Cluster Tables

For each verb that has components, create a subsection:

```markdown
### [emoji] VERB — [Short Description]
> [One sentence explaining what this capability means.]

| Component | Location | Status | LOC |
|-----------|----------|--------|-----|
| [Name] | [path/] | [emoji] | [~number] |

**Health:** [emoji] [one-line assessment]
**Audit Notes:** [technical details, constraints, risks — 1-2 lines]
```

**Choosing emojis for verbs:**
| Verb | Suggested Emoji |
|------|----------------|
| CONVERSE | 🗣️ |
| REASON | 🧠 |
| REMEMBER | 💾 |
| OBSERVE | 👁️ |
| LEARN | 📚 |
| ACT | 🔧 |
| PROTECT | 🛡️ |
| CONNECT | 🔌 |
| PRESENT | 🎨 |
| PERSIST | 💿 |
| BUILD | 🏗️ |
| THINK | 💭 |
| ORCHESTRATE | 🎼 |

### Step 4: Identify Cross-Cluster Components

Some components serve multiple clusters. List these explicitly:

```markdown
### Cross-Cluster Components
| Component | Directory | Primary Cluster | Secondary | Notes |
|-----------|-----------|----------------|-----------|-------|
| [Name] | [path/] | [VERB] | [VERB] | [Why] |
```

This table is gold for understanding coupling. Components that are cross-cluster are integration points — they're where changes in one capability can break another.

### Step 5: Identify Orphans

Walk the directory tree and check: **is every significant directory represented in at least one cluster?**

Orphan directories — significant code that doesn't map to any cluster — signal one of three things:
1. **Dead code** that should be removed.
2. **An emerging capability** that deserves its own verb.
3. **A gap in your analysis** that needs a second look.

Document orphans explicitly. Don't sweep them under the rug.

### Step 6: Write Health Assessments

For each cluster, write a 2-line health assessment:
- **Health line:** Overall emoji + one-sentence verdict.
- **Audit Notes line:** Key constraints, risks, or technical details.

Be honest. A cluster with 85% test coverage and active development is ✅. A cluster with no tests and a known security gap is ⚠️. A cluster that's completely broken is ❌.

---

## V. Creating New Verbs

The 13 starter verbs are a starting point. Your project may need verbs not on the list.

**Good custom verbs** are specific and immediately communicative:
| Verb | Use When |
|------|----------|
| TRANSLATE | i18n-heavy apps, multi-language support |
| SIMULATE | Apps with simulation engines, digital twins |
| COMPOSE | Content creation tools, editors, IDEs |
| GOVERN | Apps with complex compliance, policy, approval workflows |
| DISCOVER | Search-heavy apps, recommendation engines, explorers |
| TRANSFORM | Data pipeline apps, ETL systems, media converters |
| SCHEDULE | Calendar-heavy apps, booking systems, cron managers |
| NOTIFY | Apps where notification delivery is a core capability |

**Bad custom verbs** are vague and don't tell you anything:
- MANAGE (manage *what*?)
- PROCESS (process *what*?)
- HANDLE (handle *what*?)
- DO (everything "does" something)

The test: can someone read just the verb name and guess what kinds of components belong in that cluster? If yes, it's a good verb.

---

## VI. Common Pitfalls

**1. Over-clustering.** Don't create 20 clusters for a 30-file project. If a cluster has only 1-2 components, merge it with a related cluster. A good rule: 4-8 clusters for small projects, 8-15 for large ones.

**2. Under-clustering.** Don't dump everything into PRESENT and PERSIST. If a cluster has 25+ components, consider splitting it. PRESENT might split into PRESENT (layout) and COMPOSE (content editing).

**3. Confusing location with behavior.** A component in `frontend/src/components/` isn't automatically PRESENT. A dashboard that displays traces belongs in OBSERVE. A form that manages API keys belongs in PROTECT. Let the *behavior* dictate the cluster, not the directory.

**4. Ignoring tests.** Tests belong in BUILD, not in the cluster of the code they test. They're infrastructure, not capabilities.

**5. Forgetting infrastructure.** CI/CD, Docker, and deployment configs are real capabilities. BUILD is a real cluster, not an afterthought.

**6. Treating clusters as hierarchical.** Clusters are *flat*. REASON is not "above" ACT. They're peers — different capabilities of the same system. The component tables within each cluster may have internal hierarchy, but the clusters themselves don't.

---

## VII. Using Clusters Beyond Status Documents

Semantic clusters have applications beyond the status-template:

- **Architecture Decision Records (ADRs):** Frame decisions by which cluster they affect. "This ADR impacts REASON and ORCHESTRATE."
- **Sprint planning:** Assign work by cluster. "This sprint we're focused on OBSERVE and PRESENT."
- **Code review:** Ask "which cluster does this PR touch?" A PR that modifies 4+ clusters deserves extra scrutiny.
- **Onboarding:** Walk new team members through clusters, not directories. "Let me explain what this system can DO."
- **Technical debt tracking:** Rate each cluster's health independently. Focus debt reduction on ⚠️ and ❌ clusters.

---

## VIII. Quality Checklist

Before delivering your semantic cluster map, confirm:

- [ ] Every significant component maps to at least one cluster
- [ ] No cluster has fewer than 2 components (merge if so)
- [ ] No cluster has more than 20 components (split if so)
- [ ] Cross-cluster components are explicitly listed
- [ ] Orphan directories are documented and explained
- [ ] Each cluster has a health assessment with emoji + notes
- [ ] Custom verbs (if any) pass the "can you guess the contents?" test
- [ ] The cluster map covers both frontend and backend (if applicable)
- [ ] LOC estimates are approximate but not fictional
- [ ] The map tells a coherent story about what the system does

## Output
- A semantic cluster map with one subsection per verb, each containing: a component table (Component, Location, Status, LOC), a one-line health assessment, and audit notes.
- A cross-cluster components table listing integration points and their dual-cluster membership.
- An orphans section documenting any directories that didn't map to any cluster, with an explanation.

## Examples
**Scenario 1:** "Walk me through what the Dojo Gateway actually does" → 9 clusters identified (CONVERSE, REASON, PROTECT, OBSERVE, PERSIST, CONNECT, BUILD, ORCHESTRATE, ACT). Cross-cluster table shows TraceLogger serves both OBSERVE and BUILD. One orphan directory found: `/legacy/` — documented as dead code candidate.
**Scenario 2:** "Plan the refactor of the frontend" → PRESENT cluster has 25 components — flagged for splitting into PRESENT (layout) and COMPOSE (content editing). 3 components mapping to 4+ clusters flagged as architectural concerns requiring decomposition.

## Edge Cases
- If a component maps to three or more clusters, do not force it into one — flag it explicitly as an architectural concern and note which capabilities are bleeding into each other.
- Tests always belong in BUILD, never in the cluster of the code they test — even if they live adjacent to the tested components.

## Anti-Patterns
- Assigning components to clusters based on directory location rather than behavior — a dashboard in `frontend/components/` that displays traces belongs in OBSERVE, not PRESENT.
- Creating clusters with vague verbs like MANAGE, PROCESS, or HANDLE — if someone cannot guess the contents from the verb name alone, it's not a good cluster verb.
