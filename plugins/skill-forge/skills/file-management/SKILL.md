---
name: file-management
description: A flexible guide for organizing files and directories in a way that is adaptable to different project environments, promoting clarity and good practice without being overly rigid.
triggers:
  - "organize these files and directories"
  - "clean up this project structure"
  - "suggest a file layout for this project"
metadata:
  version: "1.1"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
---

# File Management & Organization Skill

**Version:** 1.1
**Created:** 2026-02-04
**Last Updated:** 2026-04-06
**Author:** Manus AI  
**Purpose:** To provide a set of flexible principles and recommended patterns for file and directory organization, adaptable to diverse project environments.

---

## I. The Philosophy: A Place for Everything, and Everything in Its Place

Good file organization is not about rigid, universal rules. It is about creating a system where the location of a file is intuitive and predictable within the context of its own project. A well-organized project is a pleasure to work in; a chaotic one is a source of constant friction.

This skill provides a set of guiding principles, not a strict mandate. It is a flexible framework that can be adapted to the unique needs of any project, from a simple static website to a complex multi-service application. The goal is to create a sense of order and clarity that makes the project easier to understand, navigate, and maintain.

---

## II. When to Use This Skill

-   **When starting a new project:** Use these principles to establish a clean and logical directory structure from the outset.
-   **When a project feels disorganized:** Use this skill to guide a refactoring of the existing file structure.
-   **When onboarding a new team member or agent:** Use this as a guide to explain the project's organizational philosophy.

---

## III. Core Principles

1.  **Group by Feature or Domain:** Whenever possible, group files related to a single feature or domain together. This is often preferable to grouping by file type (e.g., all controllers in one directory, all models in another).

2.  **Separate Public from Private:** Keep the public interface of a module or service separate from its internal implementation details.

3.  **Keep the Root Directory Clean:** The root of your project should be as clean as possible, containing only essential configuration files, the main entry point, and a few key directories.

4.  **Use a Consistent Naming Convention:** Choose a naming convention (e.g., `kebab-case`, `snake_case`, `PascalCase`) for your files and directories and stick to it.

5.  **Document Your Structure:** A project's `README.md` or `ARCHITECTURE.md` should briefly explain the organizational philosophy of the project.

---

## IV. Recommended Patterns

These are flexible patterns that can be adapted to different environments.

### 1. The Generic Web Application

This is a good starting point for many web applications.

```
/
├── public/             # Static assets (images, fonts, etc.)
├── src/                # Source code
│   ├── api/            # Backend API handlers/controllers
│   ├── components/     # Reusable UI components
│   ├── lib/            # Shared libraries, utilities, and helpers
│   ├── pages/          # Page-level components (if using a framework like Next.js)
│   ├── services/       # Business logic and external API clients
│   └── styles/         # Global styles
├── tests/              # Tests
├── .env                # Environment variables
├── .gitignore
├── package.json
└── README.md
```

### 2. The AROMA-style Contemplative Repository

This pattern is optimized for knowledge bases and contemplative practice repositories.

```
/
├── seeds/              # Reusable patterns of thinking
├── thinking/           # Philosophical reflections and insights
├── conversations/      # Summaries of key discussions
├── docs/               # Formal documentation (specifications, retrospectives)
├── SKILLS/             # Reusable workflow skills
├── prompts/            # Prompts for other agents (e.g., implementation agents)
├── .gitignore
└── README.md
```

### 3. The Go Backend Service

A common structure for a Go backend service.

```
/
├── cmd/                # Main application entry points
│   └── api/            # The main API server
├── internal/           # Private application and library code
│   ├── handlers/       # HTTP request handlers
│   ├── models/         # Database models
│   └── store/          # Database access layer
├── pkg/                # Public library code (if any)
├── .gitignore
├── go.mod
└── README.md
```

---

## V. Best Practices

-   **Don't Over-Organize:** A flat structure is often better than a deeply nested one, especially in the early stages of a project.
-   **Be Pragmatic:** The best structure is the one that works for your team and your project. Don't be afraid to deviate from these patterns if you have a good reason.
-   **Refactor as You Go:** A project's file structure is not set in stone. As the project evolves, don't be afraid to refactor the file structure to better reflect the current state of the codebase.
-   **Consistency is Key:** Whatever structure you choose, be consistent. An inconsistent structure is often worse than no structure at all.

---

## VI. Example

**Problem:** The Dojo Genesis repository had grown organically from 5 skills to 47 skills, plus seeds, scouts, specs, and thinking artifacts. New files were being placed inconsistently -- some scouts landed in `docs/`, others in `scouts/`, and one was in the root directory. A new agent onboarding to the repository could not predict where to find anything.

**Process:**
1. Listed the root directory and identified 12 top-level entries, 6 of which were documentation files that belonged in subdirectories.
2. Applied Core Principle 1 (Group by Feature/Domain): reorganized into `skills/` (reusable workflows), `seeds/` (extracted patterns), `scouts/` (strategic explorations), `specs/` (release specifications), `docs/` (formal documentation), and `thinking/` (reflections).
3. Applied Core Principle 3 (Keep Root Clean): moved the 6 orphaned documentation files into their appropriate subdirectories based on content type.
4. Applied Core Principle 4 (Consistent Naming): standardized all skill directories to `kebab-case` and all date-prefixed files to `YYYY-MM-DD_description.md` format.
5. Updated README.md with a directory structure explanation following Core Principle 5 (Document Your Structure).

**Outcome:** The reorganized repository had a predictable structure where any agent could locate a file by its type: need a skill? Look in `skills/`. Need a strategic analysis? Look in `scouts/`. The onboarding time for new agents dropped from ~15 minutes of exploration to ~2 minutes of reading the README structure section.

**Key Insight:** The right time to reorganize is when a new type of artifact appears for the third time -- one instance is an exception, two is a coincidence, three is a pattern that deserves its own directory.

---

## VII. Common Pitfalls

1. **Creating directories preemptively.** Building an elaborate directory structure before there is content to fill it leads to empty directories that confuse rather than clarify.
   - *Solution:* Create directories only when you have at least 2-3 files that belong in them. Let the structure emerge from the content.

2. **Nesting too deeply.** Deeply nested structures (4+ levels) make navigation tedious and file paths unwieldy, especially in commit messages and documentation references.
   - *Solution:* Apply the "three-level rule" -- most projects should not need more than three levels of nesting (e.g., `skills/strategic-scout/references/`).

3. **Mixing concerns in a single directory.** Putting specs, retrospectives, audit logs, and architecture docs all in a single `docs/` folder creates a grab-bag that grows unmanageable.
   - *Solution:* Subdivide by artifact type when a directory exceeds ~15 files (e.g., `docs/specs/`, `docs/audits/`, `docs/retrospectives/`).

4. **Inconsistent naming conventions across directories.** Using `kebab-case` for skills but `PascalCase` for specs and `snake_case` for seeds creates cognitive overhead.
   - *Solution:* Choose one convention for directories and one for files at the project level. Document the choice in the README or CONTRIBUTING guide.

5. **Reorganizing without updating references.** Moving files to better locations but not updating the links, imports, or references that point to them creates broken references throughout the codebase.
   - *Solution:* After any reorganization, run a grep for the old paths and update all references. Use the documentation-auditor skill to verify no broken links remain.

---

## VIII. Related Skills

- **documentation-auditor** -- Use after a reorganization to verify that all internal links and references still resolve correctly.
- **repo-status** -- Generates the annotated directory tree that makes file organization visible. Run repo-status to assess the current structure before reorganizing.
- **pointer-directories** -- Handles the specific case of empty directories that serve as intentional references to external content, preventing accidental deletion during cleanup.
- **status-writer** -- The STATUS.md file this skill helps organize is itself a key artifact that benefits from consistent placement at the project root.
- **health-supervisor** -- Uses the file structure as input for its health assessment. A well-organized repository produces a more accurate and useful health audit.
