---
name: semantic-clusters
description: >
  Map any software system's capabilities using action-verb clusters -- grouping
  components by what they DO rather than where they LIVE. Produces a behavioral
  architecture map that reveals capabilities, gaps, cross-cutting concerns, and
  architectural confusion that directory trees hide. Use this skill whenever you
  need to understand what a codebase does, explain a system to someone, plan a
  refactor, audit feature coverage, or answer "what does this app actually do?"
  Even for simple questions like "walk me through the features" or "what are the
  main capabilities", this skill produces a structured answer grounded in the
  actual code, not just marketing copy.
triggers:
  - "map what this system does using semantic clusters"
  - "group capabilities by behavior, not file location"
  - "what does this app actually do, show the action verbs"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - health-agent
---

# Semantic Clusters Skill

**Version:** 1.0
**Created:** 2026-02-11
**Purpose:** Map software systems by behavioral capabilities using action-verb clusters, revealing what a system does rather than where its files live.

---

## I. The Philosophy: Behavior Over Location

Every codebase has two architectures:

1. **The filesystem architecture** -- where files live. Directories, packages, modules. This is what `ls` shows you.
2. **The behavioral architecture** -- what the system *does*. Capabilities that cross-cut directories, features that span layers.

Most people only see architecture #1. They think in terms of `frontend/`, `backend/`, `utils/`. But understanding comes from architecture #2 -- the verbs.

A "chat" feature isn't in one directory. It's a component in the frontend, a handler in the backend, a state engine in a context, a streaming service, an SSE connection, and a set of tools. These parts live in 6 different directories. But they all serve one verb: **CONVERSE**.

Semantic clusters make the behavioral architecture visible. Each cluster is named with an action verb, and every significant component in the codebase maps to one (sometimes two) clusters. The result is a map of *what the system can do* -- not just where its files happen to be.

This distinction matters because:
- **Refactoring within a cluster is safer** than refactoring across clusters. Components in the same cluster share a purpose; changing them together makes sense.
- **Gaps become visible.** If you have PERSIST but no PROTECT, you're storing data without security boundaries.
- **Coupling is explicit.** Cross-cluster components are integration points where changes propagate. You can see them in the map instead of discovering them during a production incident.
- **Onboarding improves.** "Let me explain what this system can DO" is a better orientation than "let me show you the directory tree."

---

## II. When to Use This Skill

- **Exploring a new codebase:** Before diving into files, map the behavioral capabilities to build your mental model.
- **Explaining a system:** Clusters make better explanations than directory trees because they answer "what does it do?" not "where are the files?"
- **Planning a refactor:** Clusters reveal which components serve the same capability. Refactoring within a cluster is safer than across clusters.
- **Auditing feature coverage:** Clusters expose gaps -- capabilities the system lacks or has only partially implemented.
- **Identifying architectural confusion:** If a component maps to 3+ clusters, it's probably doing too much. If a directory has components in 5 different clusters, it may need restructuring.
- **Writing status documents:** The behavioral architecture section of a STATUS.md uses clusters.
- **Sprint planning:** Assign work by cluster. "This sprint we're focused on OBSERVE and PRESENT."
- **Code review:** Ask "which clusters does this PR touch?" A PR that modifies 4+ clusters deserves extra scrutiny.
- **Technical debt tracking:** Rate each cluster's health independently. Focus debt reduction on unhealthy clusters.

**When NOT to use:** For a quick file lookup (just use the filesystem). For understanding a single function (just read it). For projects with fewer than 10 files (the overhead isn't worth it).

---

## III. The Starter Verbs

These 13 action verbs cover most software systems. They are a starting point, not a straitjacket.

| Verb | Emoji | What It Means | Example Systems |
|------|-------|--------------|-----------------|
| **CONVERSE** | 🗣️ | Real-time communication with users | Chat apps, messaging, support tools |
| **REASON** | 🧠 | Thinking, planning, deciding | AI agents, rule engines, recommendation systems |
| **REMEMBER** | 💾 | Storing and recalling knowledge | Knowledge bases, caching, memory systems |
| **OBSERVE** | 👁️ | Watching and reporting | Monitoring, analytics, logging, tracing |
| **LEARN** | 📚 | Adapting based on feedback | Calibration, A/B testing, preference learning |
| **ACT** | 🔧 | Executing side effects | Tool systems, API calls, file operations, cron |
| **PROTECT** | 🛡️ | Enforcing boundaries | Auth, encryption, rate limiting, validation |
| **CONNECT** | 🔌 | Integrating externally | Plugins, APIs, webhooks, bots, OAuth |
| **PRESENT** | 🎨 | Rendering UI | Shells, layouts, component libraries, themes |
| **PERSIST** | 💿 | Storing data durably | Databases, migrations, ORMs, caches |
| **BUILD** | 🏗️ | Building, testing, shipping | CI/CD, Docker, scripts, test suites |
| **THINK** | 💭 | Meta-cognition about itself | Skills, prompts, documentation, retrospectives |
| **ORCHESTRATE** | 🎼 | Coordinating multi-step work | DAG engines, task queues, workflows, sagas |

Not every project needs all 13. A simple CRUD app might only have CONVERSE, PERSIST, PRESENT, PROTECT, and BUILD. A complex AI platform might use all 13 plus custom ones.

**Key distinctions between similar verbs:**
- **REMEMBER vs PERSIST:** PERSIST stores data (rows in a database). REMEMBER stores *knowledge* (data with semantic meaning, retrieval by relevance, influence on behavior).
- **CONNECT vs ACT:** ACT executes a side effect (sending an email). CONNECT is the *infrastructure* for integration (the plugin system, the OAuth flow, the webhook receiver).
- **REASON vs ORCHESTRATE:** REASON decides *what* to do. ORCHESTRATE manages *how* to execute it across multiple steps.
- **LEARN vs REMEMBER:** REMEMBER stores knowledge for retrieval. LEARN changes behavior based on feedback.
- **THINK vs BUILD:** BUILD is the infrastructure for shipping (CI, tests, Docker). THINK is meta-cognition -- the system's knowledge about how to work on itself (skills, prompts, retrospectives).

---

## IV. The Clustering Workflow

### Step 1: Inventory First

You can't cluster what you haven't seen. Before clustering, you need a component inventory -- a list of every significant component with its location, approximate LOC, and current status.

Walk the filesystem:
- Get the top-level shape of the project
- Recursively explore significant directories
- Count LOC per directory or module
- Note the approximate scale of each component

### Step 2: Assign Verbs

For each significant component in your inventory, ask: **"What verb describes what this does?"**

Rules of thumb:
- Most components map to **one** verb. If you can't decide, pick the one that best describes the component's *primary purpose*.
- Some components legitimately serve **two** verbs (cross-cluster). This is fine -- note both.
- If a component maps to **three or more** verbs, it's probably doing too much. Flag it as an architectural concern.
- If a component doesn't fit any verb, it might be dead code, or you might need a **new verb** (see Section V).
- Let the *behavior* dictate the cluster, not the directory. A dashboard showing traces in `frontend/src/components/` belongs in OBSERVE, not PRESENT. A form managing API keys belongs in PROTECT, not PRESENT.

### Step 3: Build Cluster Tables

For each verb that has components, create a subsection:

```markdown
### [emoji] VERB -- [Short Description]
> [One sentence explaining what this capability means.]

| Component | Location | Status | LOC |
|-----------|----------|--------|-----|
| [Name] | [path/] | [emoji] | [~number] |

**Health:** [emoji] [one-line assessment]
**Audit Notes:** [technical details, constraints, risks -- 1-2 lines]
```

### Step 4: Identify Cross-Cluster Components

Some components serve multiple clusters. List these explicitly:

```markdown
### Cross-Cluster Components
| Component | Directory | Primary Cluster | Secondary | Notes |
|-----------|-----------|----------------|-----------|-------|
| [Name] | [path/] | [VERB] | [VERB] | [Why] |
```

This table is gold for understanding coupling. Components that are cross-cluster are integration points -- they're where changes in one capability can break another.

### Step 5: Identify Orphans

Walk the directory tree and check: **is every significant directory represented in at least one cluster?**

Orphan directories -- significant code that doesn't map to any cluster -- signal one of three things:
1. **Dead code** that should be removed.
2. **An emerging capability** that deserves its own verb.
3. **A gap in your analysis** that needs a second look.

Document orphans explicitly. Don't sweep them under the rug.

### Step 6: Write Health Assessments

For each cluster, write a 2-line health assessment:
- **Health line:** Overall emoji + one-sentence verdict.
- **Audit Notes line:** Key constraints, risks, or technical details.

Be honest. A cluster with 85% test coverage and active development is GREEN. A cluster with no tests and a known security gap is YELLOW or RED. A cluster that's completely broken is RED.

---

## V. Creating New Verbs

The 13 starter verbs are a starting point. Your project may need verbs not on the list.

**Good custom verbs** are specific and immediately communicative:

| Verb | Emoji | Use When |
|------|-------|----------|
| TRANSLATE | 🌐 | i18n-heavy apps, multi-language support |
| SIMULATE | 🧪 | Apps with simulation engines, digital twins |
| COMPOSE | ✏️ | Content creation tools, editors, IDEs |
| GOVERN | ⚖️ | Apps with complex compliance, policy, approval workflows |
| DISCOVER | 🔍 | Search-heavy apps, recommendation engines, explorers |
| TRANSFORM | 🔄 | Data pipeline apps, ETL systems, media converters |
| SCHEDULE | 📅 | Calendar-heavy apps, booking systems, cron managers |
| NOTIFY | 🔔 | Apps where notification delivery is a core capability |
| RENDER | 🖼️ | 3D rendering, video processing, image generation |
| MEASURE | 📊 | Measurement/sensor systems beyond basic monitoring |

**Bad custom verbs** are vague and don't tell you anything:

| Bad Verb | Why | Better Alternative |
|----------|-----|-------------------|
| MANAGE | Too vague -- manage *what*? | Use the specific behavior verb |
| PROCESS | Too vague -- process *what*? | TRANSFORM, ANALYZE, CONVERT |
| HANDLE | Too vague -- handle *what*? | Name the specific handling |
| DO | Everything "does" something | Not a useful cluster name |
| RUN | Same as DO | EXECUTE, ORCHESTRATE, or ACT |

**The test:** Can someone read just the verb name and guess what kinds of components belong in that cluster? If yes, it's a good verb.

---

## VI. Common Pitfalls

### Pitfall 1: Over-Clustering

**Problem:** Creating 20 clusters for a 30-file project. If a cluster has only 1-2 components, it's probably not a real capability -- it's noise.

**Solution:** A good rule: 4-8 clusters for small projects, 8-15 for large ones. If a cluster has fewer than 2 components, merge it with a related cluster.

### Pitfall 2: Under-Clustering

**Problem:** Dumping everything into PRESENT and PERSIST. If a cluster has 25+ components, it's hiding internal structure.

**Solution:** Split large clusters. PRESENT might split into PRESENT (layout) and COMPOSE (content editing). PERSIST might split into PERSIST (database) and REMEMBER (knowledge).

### Pitfall 3: Confusing Location with Behavior

**Problem:** A component in `frontend/src/components/` is automatically assigned to PRESENT.

**Solution:** Let the *behavior* dictate the cluster, not the directory. A dashboard that displays traces belongs in OBSERVE. A form that manages API keys belongs in PROTECT. A settings page that configures preferences belongs in LEARN.

### Pitfall 4: Ignoring Tests

**Problem:** Assigning test files to the cluster of the code they test.

**Solution:** Tests belong in BUILD, not in the cluster of the code they test. They're infrastructure, not capabilities.

### Pitfall 5: Forgetting Infrastructure

**Problem:** CI/CD, Docker, and deployment configs are left out of the map.

**Solution:** BUILD is a real cluster, not an afterthought. Infrastructure components deserve the same analysis as application code.

### Pitfall 6: Treating Clusters as Hierarchical

**Problem:** Arranging clusters in a hierarchy where REASON is "above" ACT.

**Solution:** Clusters are *flat*. They're peers -- different capabilities of the same system. The component tables within each cluster may have internal hierarchy, but the clusters themselves don't.

---

## VII. Using Clusters Beyond Architecture Maps

Semantic clusters have applications beyond standalone maps:

- **Architecture Decision Records (ADRs):** Frame decisions by which cluster they affect. "This ADR impacts REASON and ORCHESTRATE."
- **Sprint planning:** Assign work by cluster. "This sprint we're focused on OBSERVE and PRESENT."
- **Code review:** Ask "which clusters does this PR touch?" A PR that modifies 4+ clusters deserves extra scrutiny.
- **Onboarding:** Walk new team members through clusters, not directories. "Let me explain what this system can DO."
- **Technical debt tracking:** Rate each cluster's health independently. Focus debt reduction on unhealthy clusters.
- **Status reports:** The behavioral architecture section of a STATUS.md summarizes clusters.
- **Health audits:** Cluster health assessments feed directly into the health audit's sustainability dimensions.

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
- [ ] 4-8 clusters for small projects, 8-15 for large ones
