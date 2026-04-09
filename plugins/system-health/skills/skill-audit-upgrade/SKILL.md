---
name: skill-audit-upgrade
model: sonnet
description: "Produces a graded audit report and upgraded SKILL.md files by systematically assessing all skills against an A+ quality framework (8 sections, concrete examples, pitfalls, related skills) and patching gaps. Use when: 'audit all the skills', 'upgrade skills to A+ standard', 'after creating 5+ new skills', 'before a major release', 'quarterly skills maintenance'."
category: system-health
triggers:
  - "audit all the skills"
  - "upgrade skills to A+ standard"
  - "after creating new skills"
  - "before a major release"
  - "quarterly skills maintenance"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: skills_directory
    type: string
    description: Path to the skills directory to audit and upgrade
    required: true
outputs:
  - name: audit_report
    type: ref
    format: cas-ref
    description: Graded audit report and upgraded SKILL.md files assessed against the A+ quality framework
---

# Skill Audit and Upgrade

**Version:** 1.0  
**Created:** 2026-02-07  
**Author:** Manus AI  
**Purpose:** To provide a systematic, repeatable process for auditing all skills in a repository, identifying gaps, and upgrading incomplete skills to A+ standard.

---

## I. The Philosophy: Ecosystem Health Through Systematic Audits

A skills ecosystem, like any knowledge base, can degrade over time. New skills are created with high standards, but older skills may lack structure, examples, or ecosystem connections. Without regular audits, the quality becomes inconsistent, making the ecosystem harder to use and maintain.

This skill is about **proactive maintenance**. By systematically auditing all skills against clear quality criteria, we can identify gaps early, prioritize upgrades, and ensure the entire ecosystem remains healthy and production-ready.

**The core principle:** Maintenance is not just fixing what's broken—it's keeping what works working well.

---

## II. When to Use This Skill

- **Quarterly maintenance schedule** (every 3 months)
- **After creating 5+ new skills** (to check for consistency)
- **When a skill is reported as confusing or incomplete**
- **Before a major release** (to ensure all skills are production-ready)
- **When onboarding a new contributor** (to assess current state)
- **After a major refactor** (e.g., skill renames, terminology changes)

---

## III. The Workflow

This is a 5-step workflow for auditing and upgrading skills.

### Step 1: List All Skills and Create Audit Framework

**Goal:** Establish a systematic approach to auditing skills.

**Actions:**
1. List all skills in the repository:
   ```bash
   cd /path/to/repository/skills && ls -d */ | sed 's|/||' | sort
   ```
2. Create audit framework document defining:
   - **Quality criteria:** Required, recommended, and optional sections
   - **Assessment scale:** A+, A, B, C, D (with clear definitions)
   - **Audit process:** Step-by-step instructions
   - **Upgrade priority criteria:** High, medium, low

**Outputs:**
- List of all skills to audit
- `skills_audit_framework.md` document

**Quality Criteria Example:**

**Required Sections:**
- Philosophy (I.)
- When to Use (II.)
- Workflow (III.)
- Best Practices (IV.)
- Quality Checklist (V.)

**Recommended Sections:**
- Example (VI.)
- Common Pitfalls (VII.)
- Related Skills (VIII.)

**Optional Sections:**
- Skill Metadata (IX.)

**Assessment Scale:**
- **A+:** All required + all recommended sections
- **A:** All required sections, missing some recommended
- **B:** Missing 1-2 required sections
- **C:** Missing 3+ required sections
- **D:** Incomplete or unusable

---

### Step 2: Read and Assess Each Skill Against Quality Criteria

**Goal:** Systematically evaluate each skill's completeness.

**Actions:**
1. Check line counts for all skills to identify obviously incomplete ones:
   ```bash
   for dir in */; do echo "$dir: $(wc -l < "$dir/SKILL.md") lines"; done
   ```
2. Use grep to extract section headers from each skill:
   ```bash
   for skill in */SKILL.md; do echo "=== $skill ==="; grep -E "^## [IVX]+\." "$skill"; done
   ```
3. For each skill, assess against quality criteria:
   - Check for required sections
   - Check for recommended sections
   - Check for proper structure (section numbering)
4. Assign grades (A+, A, B, C, D) based on completeness

**Outputs:**
- Assessment for each skill
- Gaps identified for each skill
- Grades assigned

**Tips:**
- Use grep with regex to extract only numbered sections: `grep -E "^## [IVX]+\."`
- Check line counts first to identify obviously incomplete skills
- If a skill has <50 lines, it's likely incomplete

---

### Step 3: Identify Skills Needing Upgrade and Prioritize

**Goal:** Create a prioritized list of skills to upgrade.

**Actions:**
1. Categorize skills by grade:
   - Count A+ skills
   - Count A skills
   - Count B/C/D skills
2. Define upgrade priorities:
   - **High Priority:** D or C skills (critically incomplete)
   - **Medium Priority:** B skills (structural fixes needed)
   - **Low Priority:** A skills (enhancement, not blocking)
3. Estimate time for each upgrade:
   - Structural fixes: ~30-45 minutes per skill
   - Adding examples/pitfalls: ~15-30 minutes per skill
4. Create upgrade plan with prioritized list

**Outputs:**
- `skills_audit_report.md` with complete assessment
- Prioritized upgrade list
- Time estimates for upgrades

**Upgrade Priority Criteria:**
- **High:** Skill is blocking workflows or unusable
- **Medium:** Skill is functional but lacks structure/consistency
- **Low:** Skill is functional, enhancement would be nice-to-have

---

### Step 4: Upgrade Incomplete Skills

**Goal:** Bring incomplete skills to A+ standard.

**Actions:**
1. For each skill to upgrade:
   - Read current content
   - Add missing required sections
   - Add section numbering (I., II., III., etc.)
   - Add Example section (use real project examples)
   - Add Common Pitfalls section (5 pitfalls with solutions)
   - Add Related Skills section (5 complementary skills)
   - Add Skill Metadata section (token savings, maintenance schedule)
   - Version bump (e.g., v1.0 → v1.1)

2. For structural fixes:
   - Ensure consistent section numbering
   - Ensure all required sections present
   - Ensure proper frontmatter (name, description)

3. For content enhancements:
   - Add concrete examples from real projects
   - Structure examples as: Problem → Process → Outcome
   - Make pitfalls specific with before/after examples
   - Link to complementary skills in the ecosystem

**Outputs:**
- Upgraded skills at A or A+ standard
- Version bumps documented

**Best Practices:**
- Use real project examples for credibility
- Structure examples consistently
- Make pitfalls actionable (not just warnings)
- Create ecosystem connections through related skills

---

### Step 5: Commit All Upgrades and Deliver Final Audit Report

**Goal:** Finalize the work and document the results.

**Actions:**
1. Copy audit documents to appropriate directory (e.g., `thinking/`)
2. Stage all changes: `git add skills/ thinking/`
3. Commit with comprehensive message documenting:
   - Number of skills upgraded
   - Changes made (section numbering, examples, pitfalls, etc.)
   - Grade improvements (e.g., B+ → A+)
   - Audit results (before/after percentages)
4. Push to remote: `git push origin main`
5. Create final summary document

**Outputs:**
- Committed and pushed changes
- Final audit summary document
- Updated skills ecosystem

**Commit Message Template:**
```
Upgrade X skills to A+ standard and complete skills audit

SKILLS UPGRADED (vX.Y → vX.Z):

1. skill-name-1
   - Added [changes]
   - Grade: B+ → A+

2. skill-name-2
   - Added [changes]
   - Grade: A → A+

AUDIT RESULTS:

- A+ Skills: X → Y (Z%)
- A Skills: X → Y (Z%)
- B+ Skills: X → Y (Z%)

[Summary of ecosystem health]
```

---

## IV. Best Practices

### 1. Define Clear Quality Criteria Upfront

**Why:** Makes the audit objective and actionable.

**How:** Create a framework document before starting the audit.

---

### 2. Use Automation for Initial Assessment

**Why:** Manually reading 10+ skills is time-consuming.

**How:** Use shell commands (grep, wc) to quickly identify gaps.

---

### 3. Focus on Structural Issues First

**Why:** Structure is easier to fix than content, and has high impact.

**How:** Prioritize adding section numbering and missing required sections.

---

### 4. Use Real Project Examples

**Why:** Concrete examples make skills immediately actionable.

**How:** Reference actual project work (e.g., "Dojo Genesis v0.0.31").

---

### 5. Create Ecosystem Connections

**Why:** Skills are not isolated—they're part of an ecosystem.

**How:** Add Related Skills sections to every skill.

---

### 6. Document the Audit Process

**Why:** Creates a historical record and enables future audits.

**How:** Create audit framework and audit report documents.

---

### 7. Celebrate the Work

**Why:** Maintenance is often invisible work. Make it visible.

**How:** Create a comprehensive summary document highlighting improvements.

---

## V. Quality Checklist

Before completing the audit, ensure you can answer "yes" to all of the following questions:

- [ ] Have you listed all skills in the repository?
- [ ] Have you created an audit framework with clear quality criteria?
- [ ] Have you assessed each skill against the quality criteria?
- [ ] Have you assigned grades (A+, A, B, C, D) to all skills?
- [ ] Have you identified and prioritized skills needing upgrade?
- [ ] Have you upgraded all high-priority skills?
- [ ] Have you committed all changes with comprehensive messages?
- [ ] Have you created a final audit report documenting the results?
- [ ] Are all skills now at A or A+ standard (or have a plan to get there)?

---

## VI. Example: Dojo Genesis Skills Audit (February 7, 2026)

**The Problem:** After creating 5 new skills in one day, we needed to audit the entire skills ecosystem to ensure consistency and completeness.

**The Process:**

1. **Listed all skills:** 11 skills in dojo-genesis/skills directory
2. **Created audit framework:** Defined quality criteria and assessment scale
3. **Assessed each skill:**
   - A+ Skills: 4/11 (36%)
   - A Skills: 5/11 (45%)
   - B+ Skills: 2/11 (18%)
4. **Identified gaps:**
   - 2 skills lacking section numbering (I., II., III.)
   - 2 skills missing Example, Common Pitfalls, Related Skills sections
5. **Upgraded 2 skills:**
   - multi-surface-strategy (v1.0 → v1.1)
   - product-positioning (v1.0 → v1.1)
   - Added section numbering, examples, pitfalls, related skills
6. **Committed and documented:**
   - Commit `eeca836` pushed to dojo-genesis
   - Final audit report created

**The Outcome:**
- A+ Skills: 4 → 6 (55%)
- A Skills: 5 (45%)
- B+ Skills: 2 → 0 (0%)
- 100% of skills now at A or A+ standard

**Time Investment:** ~1.5 hours  
**Return:** Healthy, production-ready skills ecosystem

**Key Insight:** Most skills were already functional (A-grade). Only 2 needed structural fixes. Regular audits prevent degradation.

---

## VII. Common Pitfalls to Avoid

### Pitfall 1: No Clear Quality Criteria

**Problem:** Without clear criteria, the audit becomes subjective and inconsistent.

**Solution:** Create an audit framework document before starting the audit.

---

### Pitfall 2: Trying to Read Every Skill Manually

**Problem:** Manually reading 10+ skills is time-consuming and error-prone.

**Solution:** Use shell commands (grep, wc) to quickly identify gaps.

---

### Pitfall 3: Upgrading Everything at Once

**Problem:** Trying to upgrade all skills to A+ standard is overwhelming.

**Solution:** Prioritize. Focus on high-priority (blocking) skills first.

---

### Pitfall 4: Skipping the Audit Report

**Problem:** Without documentation, the audit work is invisible and not repeatable.

**Solution:** Create audit framework and audit report documents.

---

### Pitfall 5: Not Using Real Examples

**Problem:** Generic examples don't help users understand how to apply the skill.

**Solution:** Use concrete examples from real projects (e.g., "Dojo Genesis v0.0.31").

---

## VIII. Related Skills

- **`skill-maintenance`** - Use this for ongoing maintenance of individual skills
- **`process-extraction`** - Use this to create new skills from processes
- **`compression-ritual`** - Use this to preserve learnings before/after audits
- **`strategic-to-tactical-workflow`** - The complete workflow that these skills support
- **`release-specification`** - Example of an A+ skill with all recommended sections

---

## IX. Skill Metadata

**Token Savings:** ~10,000-15,000 tokens per audit (by using systematic process)  
**Quality Impact:** Ensures consistent, high-quality skills ecosystem  
**Maintenance:** Update when quality criteria evolve or new section types emerge

**When to Update This Skill:**
- After completing 2-3 audits (to incorporate new patterns)
- When quality criteria change (e.g., new required sections)
- When new assessment scales or tools become available

**Recommended Audit Frequency:**
- Quarterly (every 3 months)
- After creating 5+ new skills
- Before major releases

---

**Last Updated:** 2026-02-07  
**Maintained By:** Manus AI  
**Status:** Active

## Output
- `skills_audit_framework.md` defining quality criteria and assessment scale for this audit run.
- `skills_audit_report.md` with per-skill grades (A+, A, B, C, D), gaps identified, and prioritized upgrade list with time estimates.
- Upgraded SKILL.md files for all high-priority (C/D) and medium-priority (B) skills, committed with comprehensive messages documenting grade improvements.

## Examples
**Scenario 1:** "Audit all skills after a sprint that added 6 new skills" → 6 new skills assessed: 4 graded A (missing Related Skills and Pitfalls), 2 graded B (missing examples and quality checklist). All 6 upgraded to A+. Audit report committed.
**Scenario 2:** "Are the system-health skills production-ready?" → 17 skills assessed, grades assigned. 3 skills graded C (missing workflow and quality checklist sections). All 3 upgraded with structural fixes, version bumped, and ecosystem connections added.

## Edge Cases
- If a skill is intentionally minimal (e.g., a stub or pointer), document this in the audit report rather than upgrading it — not all skills need all 8 sections.
- If a skill's content is correct but line count exceeds 500, prioritize trimming over adding sections — length is a quality signal, not just a constraint.

## Anti-Patterns
- Trying to upgrade all skills to A+ in a single pass without grading first — grade before upgrading so effort is directed to the highest-impact gaps.
- Using generic examples in upgraded skills ("Example: User asks about X") — examples must reference real project work to be credible and useful.
