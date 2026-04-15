---
name: tldr-code-analysis
model: opus
description: "5-layer code analysis (AST, Call Graph, CFG, DFG, PDG) that yields 95% token savings over raw file reads. Use when: 'systematically understand an unfamiliar codebase', 'analyze code structure with minimal tokens', 'map call graphs and data flow', 'find dead code and architectural layers', 'trace variable definitions across files'."
category: continuous-learning
triggers:
  - "systematically understand an unfamiliar codebase"
  - "analyze code structure with minimal tokens"
  - "map call graphs and data flow"
  - "find dead code and architectural layers"
  - "trace variable definitions across files"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_path
    type: string
    description: Path to the repository or directory to analyze
    required: true
  - name: analysis_depth
    type: string
    description: "Depth of analysis: 'overview' (L1-L2 only), 'deep' (L1-L4), or 'full' (all 5 layers including PDG)"
    required: false
outputs:
  - name: analysis_report
    type: ref
    format: cas-ref
    description: Structured analysis report covering requested layers with token-efficient code summaries
---
# TLDR Code Analysis

**Purpose:** Systematically analyze unfamiliar codebases using a 5-layer analysis stack that reduces token consumption by 95% compared to raw file reads, while surfacing architecture, call graphs, control flow, data flow, and program dependencies.

---

## I. When to Use

- Onboarding to an unfamiliar codebase and need structural understanding fast
- Preparing for a refactor by mapping cross-file call graphs and data flow
- Hunting a bug that spans multiple files and need to trace variable definitions
- Auditing code quality: dead code detection, cyclomatic complexity, architectural layers
- Any codebase exploration where reading raw files would blow the context budget

---

## II. The 5-Layer Stack

```
Layer 1: AST         ~500 tokens   Function signatures, imports, class outlines
Layer 2: Call Graph  +440 tokens   What calls what (cross-file edges)
Layer 3: CFG         +110 tokens   Cyclomatic complexity, branches, loops
Layer 4: DFG         +130 tokens   Variable definitions, reads, modifications
Layer 5: PDG         +150 tokens   Program dependencies, slicing
---------------------------------------------------------------------------
Total:              ~1,200 tokens  vs 23,000 raw = 95% savings
```

### Depth Modes

| Mode | Layers | Token Budget | Best For |
|------|--------|-------------|----------|
| overview | L1-L2 | ~940 tokens | Quick orientation, "what does this repo do?" |
| deep | L1-L4 | ~1,180 tokens | Refactor prep, bug tracing |
| full | L1-L5 | ~1,330 tokens | Full program slicing, dependency analysis |

---

## III. Workflow

### Step 1: Structural Scan (Layer 1 - AST)

Extract file tree, function signatures, imports, and class outlines.

```bash
tldr tree [repo_path]
tldr structure [repo_path] --lang [detected_language]
tldr imports [key_file]
```

Produces a codemap: the skeleton of every file without reading bodies.

### Step 2: Call Graph Construction (Layer 2)

Build cross-file call graph to understand what calls what.

```bash
tldr calls [repo_path]
tldr impact [function_name] [repo_path]    # Reverse: who calls this?
tldr importers [module_name] [repo_path]   # Reverse import lookup
```

This layer reveals the actual dependency structure that import lists alone miss.

### Step 3: Control Flow Analysis (Layer 3 - CFG)

For functions of interest, analyze branching complexity.

```bash
tldr cfg [file] [function]
# Returns: cyclomatic complexity, block count, branch points, loop nesting
```

Skip this layer for overview depth. Use it when refactoring or assessing complexity.

### Step 4: Data Flow Analysis (Layer 4 - DFG)

Track where variables are defined, read, and modified.

```bash
tldr dfg [file] [function]
# Returns: definition sites, use sites, modification chains
```

Critical for bug tracing: "where does this value come from and where does it go?"

### Step 5: Program Dependency Graph (Layer 5 - PDG)

Full dependency analysis and program slicing.

```bash
tldr slice [file] [function] [line_number]
tldr slice [file] [function] [line] --direction forward
tldr slice [file] [function] [line] --var [variable_name]
```

Answers: "what code affects line X?" and "what does line X affect?" Use for surgical changes where you need to know the full blast radius.

### Step 6: Synthesis

Combine layer outputs into a structured analysis report:
1. Architecture overview (entry points, layers, circular deps)
2. Hot paths (most-called functions, highest complexity)
3. Risk areas (dead code, high cyclomatic complexity, deep nesting)
4. Actionable findings (specific files and functions to investigate)

```bash
tldr arch [repo_path]    # Architectural layer detection
tldr dead [repo_path]    # Dead/unreachable code
```

---

## IV. Language Support

| Language | AST | Call Graph | CFG | DFG | PDG |
|----------|-----|------------|-----|-----|-----|
| Python | Yes | Yes | Yes | Yes | Yes |
| TypeScript | Yes | Yes | Yes | Yes | Yes |
| JavaScript | Yes | Yes | Yes | Yes | Yes |
| Go | Yes | Yes | Yes | Yes | Yes |
| Rust | Yes | Yes | Yes | Yes | Yes |
| Java | Yes | Yes | - | - | - |
| C/C++ | Yes | Yes | - | - | - |

---

## V. Output

- A structured analysis report saved to the project directory
- Sections: Architecture Map, Call Graph Summary, Complexity Hotspots, Data Flow Findings, Dead Code Report, Actionable Recommendations
- Token budget: ~1,200 tokens for full 5-layer analysis vs ~23,000 for raw reads

---

## VI. Examples

**Scenario 1:** "Help me understand this Go microservice repo" with depth=overview --> AST scan (52 files, 340 functions), call graph (entry points: main->server->handlers, 3 architectural layers detected), report in ~940 tokens identifying 4 handler packages and the dependency flow between them.

**Scenario 2:** "Trace why user_id is None at line 87 in processor.py" with depth=full --> DFG traces variable from 3 definition sites, PDG slice reveals that one branch path skips assignment, call graph shows 2 callers that trigger the None path. Fix: add guard clause at line 71.

**Scenario 3:** "Audit this TypeScript monorepo before refactoring" with depth=deep --> AST finds 1,200 exports across 89 files, call graph reveals 47 are never imported (dead exports), CFG flags 3 functions with cyclomatic complexity >15, DFG finds 2 variables shadowed across module boundaries.

---

## VII. Edge Cases

- Repository uses multiple languages: run AST scan per language, then merge call graphs at the FFI/API boundary layer
- Very large monorepo (10,000+ files): use `--max` flag to cap file count per scan; analyze by package/directory rather than whole repo
- Language not in full support table (Java, C++): L1-L2 analysis only; note the limitation in the report and recommend manual inspection for flow analysis
- No entry point discoverable: use `tldr dead` with explicit entry point hints (`--entry main cli test_`)

---

## VIII. Anti-Patterns

- Reading raw files first and then running TLDR -- the whole point is to avoid reading raw files; navigate with TLDR, then read only the 2-4 files that matter
- Running full depth on every file -- use overview to find the interesting files, then deep/full on those specific files
- Treating TLDR output as the final answer for bug fixes -- TLDR navigates to the right code, but you still need to read the actual implementation to write the fix
- Skipping the call graph (L2) and jumping to CFG/DFG -- without knowing what calls what, flow analysis on individual functions misses cross-file interactions
