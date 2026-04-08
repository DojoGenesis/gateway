#!/usr/bin/env bash
# validate-skills.sh — Validate YAML frontmatter in all SKILL.md files
# Checks that name:, description:, and triggers: fields are present.
# Exit 0 if all pass, exit 1 with list of failures.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_PATTERN="plugins/*/skills/*/SKILL.md"

failures=()
checked=0

for skill_file in $REPO_ROOT/$SKILL_PATTERN; do
  [ -f "$skill_file" ] || continue
  checked=$((checked + 1))

  # Extract YAML frontmatter (between first two --- delimiters)
  frontmatter=""
  in_frontmatter=false
  while IFS= read -r line; do
    if [ "$line" = "---" ]; then
      if $in_frontmatter; then
        break
      else
        in_frontmatter=true
        continue
      fi
    fi
    if $in_frontmatter; then
      frontmatter="${frontmatter}${line}"$'\n'
    fi
  done < "$skill_file"

  if [ -z "$frontmatter" ]; then
    failures+=("$skill_file — missing YAML frontmatter")
    continue
  fi

  missing=""
  echo "$frontmatter" | grep -q '^name:' || missing="${missing} name"
  echo "$frontmatter" | grep -q '^description:' || missing="${missing} description"
  echo "$frontmatter" | grep -q '^triggers:' || missing="${missing} triggers"

  if [ -n "$missing" ]; then
    rel_path="${skill_file#$REPO_ROOT/}"
    failures+=("$rel_path — missing:${missing}")
  fi
done

if [ "$checked" -eq 0 ]; then
  echo "No SKILL.md files found matching $SKILL_PATTERN"
  exit 0
fi

if [ ${#failures[@]} -gt 0 ]; then
  echo "=== SKILL.md Validation FAILED ==="
  echo "Checked: $checked files"
  echo "Failures: ${#failures[@]}"
  echo ""
  for f in "${failures[@]}"; do
    echo "  FAIL: $f"
  done
  exit 1
fi

echo "=== SKILL.md Validation PASSED ==="
echo "Checked: $checked files — all have name, description, triggers"
exit 0
