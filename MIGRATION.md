# Phase 4a Skill Migration Tracker

**Total Skills:** 44
**Target for Phase 4a:** 40 (28 Tier 1 + 12 Tier 2)
**Phase 4b (Tier 3):** 4 skills deferred

---

## Migration Status Summary

| Status | Count | Description |
|--------|-------|-------------|
| ✅ Ported | 40 | Successfully migrated with spec-compliant YAML |
| 🔄 In Progress | 0 | Currently being ported |
| ⏳ Pending | 0 | Awaiting migration |
| ⏭️ Phase 4b | 4 | Deferred to Tier 3 implementation |

---

## Tier 1 Skills (28 total) - Zero-Change Port

**Tool Dependencies:** `file_system`, `bash` only
**Adapters Required:** None
**Effort:** Direct copy with YAML metadata addition

| Plugin | Skill | Status |
|--------|-------|--------|
| skill-forge | file-management | ✅ Ported |
| skill-forge | skill-creation | ✅ Ported |
| skill-forge | skill-maintenance | ✅ Ported |
| skill-forge | process-extraction | ✅ Ported |
| continuous-learning | project-exploration | ✅ Ported |
| continuous-learning | debugging | ✅ Ported |
| continuous-learning | retrospective | ✅ Ported |
| continuous-learning | patient-learning-protocol | ✅ Ported |
| continuous-learning | era-architecture | ✅ Ported |
| specification-driven-development | specification-writer | ✅ Ported |
| specification-driven-development | release-specification | ✅ Ported |
| specification-driven-development | implementation-prompt | ✅ Ported |
| specification-driven-development | planning-with-files | ✅ Ported |
| specification-driven-development | pre-implementation-checklist | ✅ Ported |
| specification-driven-development | parallel-tracks | ✅ Ported |
| specification-driven-development | context-ingestion | ✅ Ported |
| specification-driven-development | zenflow-prompt-writer | ✅ Ported |
| system-health | health-audit | ✅ Ported |
| system-health | documentation-audit | ✅ Ported |
| system-health | repo-status | ✅ Ported |
| system-health | status-template | ✅ Ported |
| system-health | status-writing | ✅ Ported |
| system-health | semantic-clusters | ✅ Ported |
| strategic-thinking | strategic-scout | ✅ Ported |
| strategic-thinking | product-positioning | ✅ Ported |
| strategic-thinking | iterative-scouting | ✅ Ported |
| strategic-thinking | strategic-to-tactical-workflow | ✅ Ported |
| agent-orchestration | workspace-navigation | ✅ Ported |

---

## Tier 2 Skills (12 total) - Adapter-Dependent

**Tool Dependencies:** `web_tools`, `script_execution`
**Adapters Required:** WebToolAdapter, ScriptExecutor
**Effort:** Copy + verify adapter integration

| Plugin | Skill | Status | Adapters |
|--------|-------|--------|----------|
| continuous-learning | web-research | ✅ Ported | web_tools |
| continuous-learning | research-synthesis | ✅ Ported | web_tools |
| continuous-learning | research-modes | ✅ Ported | web_tools |
| specification-driven-development | frontend-from-backend | ✅ Ported | web_tools |
| specification-driven-development | spec-constellation-to-prompt-suite | ✅ Ported | script_execution |
| system-health | repo-context-sync | ✅ Ported | script_execution |
| system-health | skill-audit-upgrade | ✅ Ported | script_execution |
| strategic-thinking | multi-surface-strategy | ✅ Ported | web_tools |
| wisdom-garden | seed-extraction | ✅ Ported | script_execution |
| wisdom-garden | seed-library | ✅ Ported | script_execution |
| wisdom-garden | compression-ritual | ✅ Ported | script_execution |
| wisdom-garden | memory-garden | ✅ Ported | script_execution |

---

## Tier 3 Skills (4 total) - Meta-Skills (Phase 4b)

**Tool Dependencies:** `meta_skill`
**Adapters Required:** DAG subtask binding
**Effort:** Refactor to use ExecuteAsSubtask

| Plugin | Skill | Status |
|--------|-------|--------|
| agent-orchestration | handoff-protocol | ⏭️ Phase 4b |
| agent-orchestration | agent-teaching | ⏭️ Phase 4b |
| agent-orchestration | decision-propagation | ⏭️ Phase 4b |
| wisdom-garden | seed-to-skill-converter | ⏭️ Phase 4b |

---

## Validation Checklist

Before marking a skill as ✅ Ported:

- [ ] YAML frontmatter is valid and complete
- [ ] All required fields present (name, description, triggers, version, tier, agents, tool_dependencies)
- [ ] Tool dependencies are from allowlist
- [ ] Tier matches specification (28 Tier 1, 12 Tier 2)
- [ ] File copied to correct location in gateway
- [ ] Skill registers without errors
- [ ] Skill invokes without crashing (smoke test)
- [ ] Changes documented in this file
