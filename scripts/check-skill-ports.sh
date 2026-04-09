#!/usr/bin/env bash
# check-skill-ports.sh — Scan CAS skills directory for missing inputs/outputs port declarations.
#
# Usage:
#   ./scripts/check-skill-ports.sh [SKILLS_DIR]
#
# SKILLS_DIR defaults to the CoworkPluginsByDojoGenesis first-party plugins
# directory adjacent to the gateway repo (../..).
#
# Exit codes:
#   0 — all SKILL.md files found have inputs and outputs blocks
#   1 — one or more SKILL.md files are missing port declarations
#
# Designed for CI: add to your pipeline after skill annotation passes.
# The check is idempotent: running it multiple times produces the same result.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default: first-party plugins only (no community-skills)
DEFAULT_SKILLS_ROOT="$(cd "${GATEWAY_ROOT}/../../CoworkPluginsByDojoGenesis/plugins" 2>/dev/null || echo "")"

SKILLS_DIR="${1:-${DEFAULT_SKILLS_ROOT}}"

if [[ -z "${SKILLS_DIR}" || ! -d "${SKILLS_DIR}" ]]; then
  echo "ERROR: skills directory not found at '${SKILLS_DIR}'" >&2
  echo "Usage: $0 [SKILLS_DIR]" >&2
  exit 1
fi

echo "Scanning SKILL.md files in: ${SKILLS_DIR}"
echo ""

MISSING=()
CHECKED=0

# Walk the directory tree — skip community-skills (third-party, not annotated by this pass).
while IFS= read -r -d '' skill_file; do
  # Skip community-skills subtree.
  if [[ "${skill_file}" == *"/community-skills/"* ]]; then
    continue
  fi

  CHECKED=$((CHECKED + 1))

  has_inputs=false
  has_outputs=false

  # Check for inputs block in YAML frontmatter.
  # Accept both "inputs: []" (empty array inline) and "inputs:" followed by a list item.
  if grep -qE '^\s*inputs\s*:' "${skill_file}"; then
    has_inputs=true
  fi

  # Check for outputs block in YAML frontmatter.
  if grep -qE '^\s*outputs\s*:' "${skill_file}"; then
    has_outputs=true
  fi

  if [[ "${has_inputs}" == false || "${has_outputs}" == false ]]; then
    MISSING+=("${skill_file}")
    missing_fields=""
    [[ "${has_inputs}" == false ]]  && missing_fields+=" inputs"
    [[ "${has_outputs}" == false ]] && missing_fields+=" outputs"
    echo "MISSING [${missing_fields# }]: ${skill_file}"
  fi
done < <(find "${SKILLS_DIR}" -name "SKILL.md" -print0 | sort -z)

echo ""
echo "Checked : ${CHECKED} SKILL.md files"
echo "Missing : ${#MISSING[@]} file(s) without complete port declarations"

if [[ ${#MISSING[@]} -gt 0 ]]; then
  echo ""
  echo "FAIL — Add inputs: and outputs: blocks to the files listed above."
  exit 1
fi

echo "PASS — All skills have port declarations."
exit 0
