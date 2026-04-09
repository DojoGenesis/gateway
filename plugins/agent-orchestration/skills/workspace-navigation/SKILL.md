---
name: workspace-navigation
model: sonnet
description: "Produces correct file placement, frontmatter, and navigation paths for shared agent workspaces. Use when: 'organize this in the workspace', 'where does this belong in the structure', 'find the decision that answers this', 'set up a collaboration repo'."
triggers:
  - "organize this in the workspace"
  - "where does this belong in the structure"
  - "navigate to the relevant research"
  - "find the decision that answers this"
  - "set up a collaboration repo"
metadata:
  version: "1.1"
  created: "2026-02-02"
  author: "Manus"
  tool_dependencies: ["file_system"]
  portable: true
  tier: 2
  agents: ["agent-orchestration"]
category: agent-orchestration

inputs:
  - name: artifact
    type: string
    description: The file, decision, or content item to place or locate in the workspace
    required: true
  - name: workspace_root
    type: string
    description: Root path of the shared agent workspace
    required: false
outputs:
  - name: placement_decision
    type: string
    description: Correct file placement, frontmatter, and navigation paths for the workspace artifact
---

# Agent Workspace Navigator Skill

## I. Philosophy

A shared workspace is not a file dump — it is a thinking room. Structure enables clarity; clarity enables collaboration. This skill encodes best practices for reading, writing, and organizing content in shared agent repositories so that any agent can orient in under 1,000 tokens and contribute without creating disorder.

## II. When to Use

- Working in a shared private repository with other agents
- Contributing to or reading from a collaborative discussion space
- Organizing thoughts, specifications, or research in a structured way
- Coordinating work across multiple agents or sessions
- Building shared context without unstructured file chaos

## III. Workspace Structure

### Standard Directory Layout

```
workspace/
├── README.md                    # Workspace overview and navigation guide
├── 00_Active/                   # Current work in progress
│   ├── discussions/             # Active discussions and threads
│   ├── drafts/                  # Work in progress (not final)
│   └── decisions/               # Decisions made (with rationale)
├── 01_Specifications/           # Finalized specs and designs
│   ├── architecture/
│   ├── features/
│   └── protocols/
├── 02_Research/                 # Research findings and synthesis
│   ├── deep-dives/
│   ├── landscape-scans/
│   └── references/
├── 03_Memory/                   # Shared memory and knowledge base
│   ├── seeds/
│   ├── learnings/
│   └── context/
├── 04_Artifacts/                # Generated artifacts and outputs
│   ├── code/
│   ├── diagrams/
│   └── documents/
└── 05_Archive/                  # Completed or deprecated work
    ├── by-date/
    └── by-topic/
```

### File Naming Convention

Format: `YYYY-MM-DD_topic-name_author.md`

Examples: `2026-02-02_memory-garden-design_manus.md`, `2026-02-02_collaboration-protocol_manus-cipher.md`

Date prefix enables sorting; topic name provides context; author attribution enables tracking.

## IV. Reading Workflow

**Step 1:** Read `README.md` first to understand workspace purpose, active discussions, and how to contribute.

**Step 2:** Navigate by purpose, not time. Identify your goal, go to the relevant directory, read the most recent relevant file.

**Step 3:** Use Grep for targeted search:
```bash
grep -r "memory compression" workspace/ --include="*.md"
```

**Step 4:** Read frontmatter before body. Check status, date, and tags to decide if the file is worth reading in full.

**Step 5:** Extract key insights (1-3 sentences per section). Link to the source document rather than copying it. See `references/workspace-templates.md` for the Insights Extraction Template.

## V. Writing Workflow

### Choose the Right Location

```
Is this finalized?
├─ No → 00_Active/
│   ├─ Is this a discussion? → discussions/
│   ├─ Is this a draft spec? → drafts/
│   └─ Is this a decision? → decisions/
└─ Yes → Where does it belong?
    ├─ Specification → 01_Specifications/
    ├─ Research → 02_Research/
    ├─ Knowledge/Pattern → 03_Memory/
    ├─ Artifact/Output → 04_Artifacts/
    └─ Completed/Deprecated → 05_Archive/
```

### Templates

Full templates for Discussion, Decision, Threaded Discussion, and Insights Extraction are in `references/workspace-templates.md`. Key points:

- Every document requires frontmatter: `title`, `author`, `date`, `status`, `tags`, `related`
- Decisions: use `decision-propagation` skill for the full Decision Template
- Specifications and research: see `specification-writer` and `research-modes` skills

### Write for Scannability

- Use descriptive headings ("Memory Compression Strategy" not "Strategy")
- Put key insights at the top
- Use lists, tables, and blockquotes — not prose paragraphs
- Use specific language ("Use 3-month rule" not "Consider time-based compression")

### After Writing

Always add a Related Content block (see `references/workspace-templates.md`) and update `README.md` if the content is significant.

## VI. Collaboration Patterns

**Pattern 1 — Threaded Discussions:** Agent A creates discussion → Agent B adds perspective → Agent C synthesizes → move to `decisions/` when consensus reached.

**Pattern 2 — Specification Review:** Agent A creates draft → Agent B reviews with inline comments → Agent A addresses comments → move to `01_Specifications/` when finalized.

**Pattern 3 — Parallel Research:** Agents divide scope → each writes in `02_Research/` → one agent synthesizes → synthesized doc moves to `03_Memory/`.

**Pattern 4 — Seed Sharing:** Agent extracts seed using `seed-extraction` skill → documents in `03_Memory/seeds/` → updates `seeds/README.md`.

## VII. Token Efficiency

- **Surgical reading:** README (~500-1000 tokens) + 2-3 relevant files (~2000-5000 tokens each). Total ~5-10K tokens vs. 50K+ for reading everything.
- **Incremental context:** Start minimal (README + task), add files as needed, compress after milestones.
- **Reference, don't copy:** Link to source documents; extract 1-3 key sentences; never duplicate content.
- **Metadata filtering:** Check tags, status, and date before reading a file.

## VIII. Maintenance

**Weekly (30 min):** Move finalized drafts, archive completed discussions, update decision log, refresh README priorities.

**Monthly (1-2 hrs):** Review specs for staleness, audit seeds for unused patterns, archive deprecated content, refactor structure as needed.

## IX. Quality Checklist

- [ ] File is in the correct directory with proper naming convention
- [ ] Frontmatter (title, author, date, status, tags) is complete
- [ ] Purpose is clear and key insights are at the top
- [ ] Content is scannable (headings, lists, tables)
- [ ] Links to related content are included
- [ ] Workspace README updated if content is significant
- [ ] No unnecessary prose

## Output

- File placed in the correct workspace directory with proper `YYYY-MM-DD_topic-name_author.md` naming
- Frontmatter block with title, author, date, status, tags, and related fields
- Related Content block linking to connected documents
- Updated workspace `README.md` when the new file is significant
- Grep results or directory listing when navigating to find existing content

## Examples

**Scenario 1:** "Where does the new context compression proposal go?" → Placed at `00_Active/drafts/2026-04-08_context-compression-proposal_manus.md` with complete frontmatter; README updated with a pointer.

**Scenario 2:** "Find the decision about which model router to use." → Grep across `00_Active/decisions/` and `01_Specifications/` for "model router"; returns file path and key excerpt from the decision's Rationale section.

## Edge Cases

- Workspace has no `README.md`: create a minimal one before placing any new file; navigation depends on it
- Two agents writing to the same discussion file simultaneously: use threaded comment format with agent name + date prefix on each new block; never overwrite another agent's contribution
- A document spans multiple categories (e.g., a spec that also contains a decision): place in the primary category and link to it from the secondary directory's README

## Anti-Patterns

- Placing files in the root of the workspace instead of the correct subdirectory — makes the repo unsearchable within three sessions
- Writing a document without frontmatter — breaks metadata filtering and forces full reads to determine relevance
- Copying full document text into another document instead of linking — doubles token cost every time both are loaded
- Creating `thoughts.md` or `notes.md` with no date or author — untraceable and unarchivable
- Skipping the README update after adding a significant file — the next agent starts blind
