# Agent Skill Mapping

Quick reference for which skills are assigned to which agents.

## implementation-agent (23 skills)

Handles code implementation, file management, and specification work.

### skill-forge (4 skills)
- file-management
- process-extraction
- skill-creation
- skill-maintenance

### specification-driven-development (10 skills)
- context-ingestion
- frontend-from-backend
- implementation-prompt
- parallel-tracks
- planning-with-files
- pre-implementation-checklist
- release-specification
- spec-constellation-to-prompt-suite
- specification-writer
- zenflow-prompt-writer

### continuous-learning (1 skill)
- debugging

### agent-orchestration (1 skill)
- workspace-navigation

### system-health (7 skills - but assigned to health-agent)
*See health-agent section below*

## research-agent (8 skills)

Handles investigation, analysis, and knowledge work.

### continuous-learning (7 skills)
- era-architecture
- patient-learning-protocol
- project-exploration
- research-modes (Tier 2)
- research-synthesis
- retrospective
- web-research (Tier 2)

### wisdom-garden (4 skills)
- compression-ritual
- memory-garden
- seed-extraction
- seed-library

### strategic-thinking (3 skills - dual assignment with strategic-agent)
- iterative-scouting (Tier 2)
- product-positioning (Tier 2)
- strategic-scout (Tier 2)

## strategic-agent (5 skills)

Handles planning, positioning, and strategic work.

### strategic-thinking (5 skills)
- iterative-scouting (Tier 2) *with research-agent*
- multi-surface-strategy
- product-positioning (Tier 2) *with research-agent*
- strategic-scout (Tier 2) *with research-agent*
- strategic-to-tactical-workflow

## health-agent (8 skills)

Handles system health, audits, and status reporting.

### system-health (8 skills)
- documentation-audit
- health-audit
- repo-context-sync
- repo-status
- semantic-clusters
- skill-audit-upgrade
- status-template
- status-writing

## Tier Distribution

**Tier 1 (35 skills):** file_system + bash only
- implementation-agent: 16 skills
- research-agent: 8 skills
- strategic-agent: 2 skills
- health-agent: 8 skills
- agent-orchestration: 1 skill

**Tier 2 (5 skills):** includes web_tools
- research-agent: 2 skills (research-modes, web-research)
- strategic-agent + research-agent: 3 skills (iterative-scouting, product-positioning, strategic-scout)

---

**Note:** Some strategic skills have dual agent assignments (strategic-agent + research-agent) because they require both strategic thinking and web research capabilities.
