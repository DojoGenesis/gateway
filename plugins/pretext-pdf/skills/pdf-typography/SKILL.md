---
name: pdf-typography
description: Configure Pretext typography settings for PDF export — font stacks, sizing, line height, page geometry, and disposition-driven typography mapping. Use when defaults need tuning for specific content types or audiences.
model: sonnet
category: pdf
triggers:
  - "configure pretext typography"
  - "tune PDF typography settings"
  - "adjust font stack for PDF export"
  - "set PDF page geometry"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: content_type
    type: string
    description: Content type to tune typography for — prose, code, chat, or structured
    required: true
  - name: typography_overrides
    type: object
    description: Typography settings to override (font_stack, body_size, line_height, page_size, margins)
    required: false
outputs:
  - name: typography_config
    type: object
    description: Resolved Pretext typography configuration object ready for use in pdf-export
---
# PDF Typography Skill

**Version:** 0.1.0
**Created:** 2026-04-08
**Author:** Dojo Genesis
**Purpose:** Fine-tune the Pretext typography engine for optimal PDF output across different content types, audiences, and aesthetic goals.

---

## I. The Philosophy: Measure Twice, Render Once

Pretext does one thing exceptionally well: it measures text on a real Canvas surface, then reports exact dimensions. This means every line break, every paragraph spacing, every margin is computed from actual glyph metrics — not CSS approximations. Typography configuration is the lever that controls how this precision manifests in the final PDF.

---

## II. When to Use This Skill

- **Content-specific tuning:** Code needs different settings than prose. Chat needs different settings than reports.
- **Audience adaptation:** A client-facing PDF needs different typography than an internal code review.
- **Brand alignment:** Match typography to organization design tokens or brand guidelines.
- **Disposition mapping:** Map ADA agent personality traits to typographic choices (DojoChat export).

---

## III. Steps

### Step 1: Identify Content Type

Each content type has an optimal typography profile:

| Content Type | Body Font | Code Font | Size | Line Height | Max Width |
|-------------|-----------|-----------|------|-------------|-----------|
| Prose (default) | Inter | Recursive | 11pt | 1.5 | 468pt (6.5in) |
| Code review | Inter | Recursive MONO=1 | 9pt | 1.3 | 504pt (7in) |
| Chat export | Inter Variable | Recursive | 13pt | 1.45 | 420pt (5.8in) |
| Presentation | Inter | Recursive | 14pt | 1.6 | 432pt (6in) |
| Reference manual | Inter | Recursive | 10pt | 1.4 | 468pt (6.5in) |

### Step 2: Configure Font Stack

The Pretext bridge supports three font families with variable axes:

**Inter** (body text):
- Axes: `opsz` (optical size), `wght` (weight)
- Range: 100-900 weight, 14-32 optical size
- Default: 400 weight, 16 optical size

**Recursive** (code):
- Axes: `MONO` (monospace), `CASL` (casual), `slnt` (slant), `wght` (weight)
- For code: `MONO=1, CASL=0, slnt=0, wght=400`
- For casual: `MONO=0, CASL=1, slnt=-15, wght=400`

**Fraunces** (display/headings):
- Axes: `SOFT` (softness), `WONK` (wonkiness), `opsz` (optical size), `wght` (weight)
- For headings: `SOFT=50, WONK=1, opsz=48, wght=700`

### Step 3: Set Page Geometry

```typescript
interface PageGeometry {
  size: 'letter' | 'a4' | 'legal';    // Default: 'letter'
  margins: {
    top: number;     // Default: 72pt (1in)
    right: number;   // Default: 72pt
    bottom: number;  // Default: 72pt
    left: number;    // Default: 72pt
  };
  columns: 1 | 2;                      // Default: 1
  headerHeight: number;                 // Default: 36pt
  footerHeight: number;                 // Default: 36pt
}
```

### Step 4: Configure Pretext Measurement

Pretext's `prepare()` and `layout()` functions take specific parameters:

```typescript
// One-time per text block
const prepared = pretext.prepare(text, fontDescriptor);

// Fast re-layout at any width
const layout = pretext.layout(prepared, maxWidth, lineHeight);
// Returns: { width, height, lines[] }
```

Key: the `fontDescriptor` must match the actual font loaded in the Canvas context. Mismatches cause measurement drift.

### Step 5: Apply and Validate

After configuration, export a test page and verify:
- Body text line length is 45-75 characters (optimal readability)
- Code blocks don't overflow margins
- Headings have appropriate visual weight
- Page breaks respect orphan/widow rules (min 2 lines)

---

## IV. Reflection Questions

- Is the content primarily prose, code, or mixed? Each has different optimal settings.
- Who is the audience? Technical readers tolerate denser typography; general audiences need more whitespace.
- Does the content have a brand or design system to align with?
- Are there accessibility requirements (minimum font size, contrast ratios)?

---

## V. Common Mistakes

- **Using system-ui in Canvas context.** Canvas and DOM resolve `system-ui` differently on macOS. Always specify exact font family names.
- **Ignoring optical sizing.** Inter and Fraunces have `opsz` axes — using body optical size for headings produces thin, fragile letterforms.
- **Setting line height too tight for code.** Code blocks with `1.0` line height make indentation guides invisible. Minimum `1.3` for code.
- **Mixing measurement contexts.** Pretext measures in CSS pixels. PDF uses points. The bridge handles conversion — don't double-convert.

---

## VI. Variations

### Disposition-Driven Typography (DojoChat)

Map ADA agent personality traits to typographic choices:

```typescript
function dispositionToTypography(disposition: Disposition): TypographyConfig {
  return {
    fontFamily: disposition.tone === 'formal' ? 'Inter' : 'Recursive',
    fontSize: mapVerbosity(disposition.verbosity), // verbose → smaller
    lineHeight: mapDepth(disposition.depth),        // deep → tighter
    maxBubbleWidth: mapBrevity(disposition.brevity), // brief → narrower
    fontVariationSettings: buildVariations(disposition),
  };
}
```

### Two-Column Reference

For dense reference material (API docs, skill catalogs):
```typescript
{
  columns: 2,
  fontSize: 9,
  lineHeight: 1.3,
  margins: { top: 54, right: 54, bottom: 54, left: 54 }
}
```

---

## VII. Example

**Scenario:** Configure typography for exporting a Go backend specification as a client-facing PDF.

1. Content type: Mixed prose + code → start with prose defaults
2. Audience: Client (non-technical stakeholders) → increase font to 12pt, line height 1.6
3. Code blocks: Go syntax → Recursive MONO=1, 10pt, 1.35 line height
4. Headings: Fraunces 700, SOFT=30, WONK=1 for professional warmth
5. Page: Letter, 1-inch margins, single column
6. Validate: Body line length = 58 characters (within 45-75 range)

---

## VIII. See Also

- `pdf-export` — Main export workflow that consumes typography configuration
- Pretext source: `@chenglou/pretext` — Canvas-based typography engine
- DojoChat disposition mapper: `src/lib/disposition/typography.ts`

---

## Output

A `TypographyConfig` object ready for consumption by the Pretext `prepare()`/`layout()` pipeline, plus page geometry settings for the PDF renderer. Concretely: validated font descriptors, line height values per content zone (body/code/headings), page margins in points, and column count. When used with `pdf-export`, the output becomes embedded in the rendered PDF — font choices and spacing are baked into the byte stream, not post-processable.

---

## Examples

**Scenario 1 — Client-facing Go spec PDF (mixed prose + code):**
Content type: prose primary, code secondary. Audience: non-technical client.
- Body: Inter 12pt, opsz=16, line height 1.6
- Code: Recursive MONO=1, 10pt, line height 1.35
- Headings: Fraunces 700, SOFT=30, WONK=1
- Page: Letter, 72pt margins, single column
- Validate: body line ~58 chars (within 45-75 target)

**Scenario 2 — Dense code review PDF:**
Content type: code-primary (diff output, function signatures). Audience: senior engineers.
- Body: Inter 9pt, opsz=14, line height 1.3
- Code: Recursive MONO=1, 9pt, line height 1.3 (same size as body — intentional)
- Headings: Inter 700, no Fraunces
- Page: Letter, 54pt margins, potentially 2-column
- Validate: code blocks fit within 504pt max width without overflow

---

## Edge Cases

- **Canvas vs. DOM font resolution on macOS:** `system-ui` resolves to different faces in Canvas and DOM contexts. Always pass explicit family names (`Inter`, `Recursive`, `Fraunces`) to `fontDescriptor`.
- **Optical sizing mismatches at large scales:** `Inter` at 48pt (heading) with `opsz=16` (body default) produces thin, fragile letterforms. Set `opsz` to match intended display size.
- **Mixed content with extremely long code lines:** If source code has 120+ char lines and the page is letter-width, either reduce font size or switch to landscape. The layout engine will not auto-truncate — overflow is clipped.
- **Recursive `CASL` axis in monospace mode:** Setting `MONO=1, CASL=1` simultaneously produces undefined intermediate rendering on some PDF engines. Use `CASL=0` when `MONO=1`.
- **Two-column layout with tall code blocks:** Page-break logic does not split code blocks across columns. A single tall block will overflow into the gutter.

---

## Anti-Patterns

- **Specifying `system-ui` in any Pretext context.** Breaks cross-machine reproducibility and causes Canvas/DOM measurement drift. Use explicit font family names.
- **Ignoring `opsz` at heading sizes.** Using body `opsz` (14-16) for large headings (24-48pt) produces optically incorrect letterforms. Set `opsz` to match render size.
- **Setting line height below 1.3 for code.** At `1.0-1.2`, indentation guides and leading dots between columns become invisible. Minimum `1.3` for monospace content.
- **Double-converting pt/px units.** Pretext measures in CSS pixels internally; the bridge converts to PDF points. Manually converting before passing to the API results in double-scaling (typically 72/96 = 0.75x error).
- **Applying a single TypographyConfig across all zones.** Body, code, captions, and headings each need separate configuration. A one-size config either makes code unreadable or body text too large.
