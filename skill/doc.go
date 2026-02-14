// Package skill provides the skill registry, parser, and executor for the
// Agentic Gateway's skill system.
//
// Skills are reusable, composable units of agent capability defined by SKILL.md
// files with YAML frontmatter. This package handles:
//
//   - Registry: skill registration, lookup, and trigger-phrase matching
//   - Parser: SKILL.md file parsing with YAML frontmatter extraction
//   - Executor: skill execution with adapter resolution and script sandboxing
//   - Adapters: tool and service integrations (MCP, web, file system)
//
// Sentinel errors in errors.go cover registry, parsing, execution, and version
// failures — use errors.Is() to check specific failure modes.
package skill
