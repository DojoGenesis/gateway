---
name: pdf-export
description: "Produces a typeset PDF artifact from any source file — code, Markdown, HTML, chat JSON, or structured Dojo files (specs, ADRs, skills). Use when: \"export this to PDF\", \"generate a PDF of this file\", \"make a PDF from this\"."
model: sonnet
category: publishing
triggers:
  - "export this to PDF"
  - "generate a PDF of this file"
  - "make a PDF from this"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: source_path
    type: string
    description: Path or glob pattern to the source file(s) to export
    required: true
  - name: output_path
    type: string
    description: Output path for the generated PDF file
    required: false
  - name: theme
    type: string
    description: Typography theme — light or dark
    required: false
outputs:
  - name: pdf_artifact
    type: ref
    format: cas-ref
    description: Typeset PDF file written to the specified output path
---
# PDF Export Skill

**Version:** 0.1.0
**Created:** 2026-04-08
**Author:** Dojo Genesis
**Purpose:** Convert any file into a beautifully typeset PDF using Pretext's Canvas-based typography engine for precise text measurement and layout.

---

## I. The Philosophy: Typography as Respect

A PDF is a permanent artifact. Unlike a terminal dump or a browser render, a PDF says "this was worth preserving." Pretext-powered export treats every character with the same care a typesetter would — measuring widths on a real Canvas, computing line breaks honestly, and placing glyphs with sub-pixel precision. The result: PDFs that are not just functional, but beautiful.

---

## II. When to Use This Skill

- **Code review artifacts:** Export source files with syntax highlighting for offline review or archival.
- **Documentation publishing:** Convert Markdown specs, ADRs, or READMEs into distributable PDFs.
- **Chat preservation:** Export DojoChat conversations with faithful typography and bubble layout.
- **Structured file bundles:** Batch-export SKILL.md files, handoff packages, or decision records.
- **HTML snapshots:** Render HTML pages to PDF preserving layout and styles.

---

## III. Steps

### Step 1: Identify Source Files

Determine the files to export and their types. Supported formats:

| Format | Renderer | Features |
|--------|----------|----------|
| `.ts`, `.js`, `.go`, `.py`, `.rs`, etc. | Code | Syntax highlighting via Shiki, line numbers, theme support |
| `.md` | Markdown | Heading hierarchy, code blocks, tables, links as footnotes |
| `.html` | HTML | DOM-faithful rendering, CSS preservation |
| Chat JSON | Chat | Bubble layout, Pretext shrink-wrap, disposition typography |
| `SKILL.md`, `ADR-*.md` | Structured | Section-aware layout, metadata headers, cross-references |

### Step 2: Configure Typography

Choose typography settings or accept defaults:

- **Font stack:** Inter (body), Recursive (code), Fraunces (display)
- **Body size:** 11pt (default), configurable 8-16pt
- **Line height:** 1.5 (default), configurable 1.2-2.0
- **Page size:** Letter (default), A4, Legal
- **Margins:** 72pt (1 inch) all sides (default)

### Step 3: Invoke the Export

Use the MCP tool `pretext-pdf.export` or the slash command:

```
/export-pdf path/to/file.md --output ./exports/
/export-pdf src/**/*.ts --output ./code-review.pdf --bundle
/export-pdf chat-export.json --theme dark
```

### Step 4: Verify Output

Check the generated PDF for:
- Text renders correctly (no tofu/missing glyphs)
- Code blocks preserve indentation and highlighting
- Page breaks fall at logical boundaries
- Headers and footers appear consistently

### Step 5: Deliver

The PDF is written to the specified output path. Share it, archive it, or upload it via a file storage connector.

---

## IV. Reflection Questions

- Is the source file format correctly detected, or does it need a format hint?
- Are the typography defaults appropriate for this content, or should they be tuned?
- Should multiple files be bundled into a single PDF or exported individually?
- Does the content contain sensitive information that should not be in a permanent artifact?

---

## V. Common Mistakes

- **Exporting without reading the file first.** Always read the source to understand its structure before choosing renderer settings.
- **Using default typography for code-heavy files.** Code benefits from a smaller font (9-10pt) and tighter line height (1.3) than prose.
- **Ignoring page breaks.** Long files need explicit section breaks. The Markdown renderer auto-breaks at `## ` headings; code breaks every 60 lines.
- **Bundling unrelated files.** A single PDF should have a coherent narrative. Don't bundle a README with unrelated test files.
- **Forgetting dark theme for code.** Syntax highlighting on white backgrounds can wash out. Use `--theme dark` for code-centric exports.

---

## VI. Variations

### Minimal Export
Single file, default settings, quick output:
```
/export-pdf README.md
```

### Code Review Bundle
Multiple source files with dark theme and line numbers:
```
/export-pdf src/lib/**/*.ts --bundle --theme dark --line-numbers
```

### Chat Archive
DojoChat conversation with Pretext typography:
```
/export-pdf conversation.json --renderer chat --disposition-typography
```

### Structured Ecosystem Export
All skills in a plugin as a reference manual:
```
/export-pdf plugins/agent-orchestration/skills/*/SKILL.md --bundle --toc
```

---

## VII. Example

**Scenario:** Export the handoff protocol skill as a distributable PDF for a new team member.

1. Read the source: `plugins/agent-orchestration/skills/handoff-protocol/SKILL.md`
2. Detect format: Structured Markdown (has YAML frontmatter + numbered sections)
3. Apply structured renderer with defaults: Inter 11pt, 1.5 line height, Letter
4. Generate table of contents from `## ` headings
5. Render code blocks in Recursive with Shiki highlighting
6. Output: `exports/handoff-protocol.pdf` (12 pages, 45KB)

---

## VIII. See Also

- `pdf-typography` — Fine-tune Pretext typography settings for specific content types
- `handoff-protocol` (agent-orchestration) — Export handoff packages as PDFs
- `specification-writer` (specification-driven-development) — Generate specs then export as PDF
- `status-writing` (system-health) — Export STATUS.md as formatted PDF reports

---

## Output

- A `.pdf` file written to the specified `--output` path (default: `./exports/<source-basename>.pdf`)
- When bundling multiple files: a single merged PDF with an auto-generated table of contents
- File size varies by content: prose ~30-80 KB/page, code ~15-40 KB/page

## Examples

**Scenario 1:** "Export this ADR as a PDF for the client" → reads `ADR-007.md`, applies structured renderer with Inter 11pt, generates `exports/ADR-007.pdf` with section headings and a ToC.

**Scenario 2:** "Make a PDF of all the TypeScript source files in src/lib" → bundles `src/lib/**/*.ts` with dark theme, syntax highlighting, line numbers; outputs `exports/src-lib-bundle.pdf`.

## Edge Cases

- Source file contains binary data or is not text-decodable: report the issue and skip that file; do not silently produce a corrupted PDF.
- Glob pattern matches zero files: stop and inform the user before attempting export.
- Output path directory does not exist: create intermediate directories rather than failing silently.
- File exceeds 10,000 lines: warn the user about page count before proceeding; offer to split by section.

## Anti-Patterns

- Exporting without reading the source first — choosing the wrong renderer (e.g., Markdown renderer for a raw HTML file) produces broken layout.
- Bundling files from different projects or contexts into a single PDF — the result lacks coherence and confuses readers.
- Using default 11pt body size for code-heavy files — code at 11pt overflows narrow margins; use 9-10pt for code-primary exports.
- Skipping the verify step (Step 4) — orphaned headings, overflowing code blocks, and tofu characters only surface at review time, not during rendering.
