#!/usr/bin/env bash
# check-drift.sh — Detect drift between documentation claims and codebase reality
# Compares README.md / STATUS.md claims against actual counts.
# Exit 0 if no drift, exit 1 if any drift detected.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

drift_found=false

echo "=== Drift Check ==="

# --- Go LOC ---
if command -v cloc &>/dev/null; then
  go_loc=$(cloc --json . 2>/dev/null | grep -o '"Go"[^}]*"code":[0-9]*' | grep -o '[0-9]*$' || echo "0")
else
  # Fallback: count non-blank, non-comment lines in .go files
  go_loc=$(find . -name "*.go" -not -path "./.git/*" -not -path "./vendor/*" -exec cat {} + 2>/dev/null | grep -cv '^\s*$\|^\s*//' || echo "0")
fi

# Extract README LOC claim (looks for patterns like "83,000+ lines" or "83000 lines")
readme_loc_claim=""
if [ -f README.md ]; then
  readme_loc_claim=$(grep -oE '[0-9,]+\+?\s*lines\s*(of\s*Go)?' README.md | head -1 | grep -oE '[0-9,]+' | tr -d ',' || true)
fi

go_loc_fmt=$(printf "%'d" "$go_loc" 2>/dev/null || echo "$go_loc")
if [ -n "$readme_loc_claim" ]; then
  readme_loc_fmt=$(printf "%'d" "$readme_loc_claim" 2>/dev/null || echo "$readme_loc_claim")
  # Allow 20% tolerance for LOC claims
  threshold=$(( readme_loc_claim * 80 / 100 ))
  if [ "$go_loc" -lt "$threshold" ] || [ "$go_loc" -gt $(( readme_loc_claim * 150 / 100 )) ]; then
    echo "Go LOC: $go_loc_fmt (README claims: $readme_loc_fmt) — DRIFT"
    drift_found=true
  else
    echo "Go LOC: $go_loc_fmt (README claims: $readme_loc_fmt) — OK"
  fi
else
  echo "Go LOC: $go_loc_fmt (no README claim found)"
fi

# --- Test files ---
test_files=$(find . -name "*_test.go" -not -path "./.git/*" | wc -l | tr -d ' ')

# Extract STATUS.md test file claim
status_test_claim=""
if [ -f STATUS.md ]; then
  status_test_claim=$(grep -oE '[0-9]+ test files' STATUS.md | head -1 | grep -oE '[0-9]+' || true)
fi

if [ -n "$status_test_claim" ]; then
  if [ "$test_files" -ne "$status_test_claim" ]; then
    echo "Test files: $test_files (STATUS claims: $status_test_claim) — DRIFT"
    drift_found=true
  else
    echo "Test files: $test_files (STATUS claims: $status_test_claim) — OK"
  fi
else
  echo "Test files: $test_files (no STATUS claim found)"
fi

# --- Packages ---
pkg_count=0
if command -v go &>/dev/null && [ -f go.work ] || [ -f go.mod ]; then
  pkg_count=$(go list ./... 2>/dev/null | wc -l | tr -d ' ')
fi
echo "Packages: $pkg_count"

# --- Modules ---
mod_count=$(find . -name "go.mod" -not -path "./.git/*" | wc -l | tr -d ' ')

# Extract README module claim
readme_mod_claim=""
if [ -f README.md ]; then
  readme_mod_claim=$(grep -oiE '[0-9]+\s*independently.versioned\s*modules' README.md | head -1 | grep -oE '[0-9]+' || true)
fi

if [ -n "$readme_mod_claim" ]; then
  if [ "$mod_count" -ne "$readme_mod_claim" ]; then
    echo "Modules: $mod_count (README claims: $readme_mod_claim) — DRIFT"
    drift_found=true
  else
    echo "Modules: $mod_count (README claims: $readme_mod_claim) — OK"
  fi
else
  echo "Modules: $mod_count (no README claim found)"
fi

# --- Skills count ---
skill_count=$(find plugins -name "SKILL.md" 2>/dev/null | wc -l | tr -d ' ')

readme_skill_claim=""
if [ -f README.md ]; then
  readme_skill_claim=$(grep -oE '[0-9]+ built-in skills' README.md | head -1 | grep -oE '[0-9]+' || true)
  [ -z "$readme_skill_claim" ] && readme_skill_claim=$(grep -oE '[0-9]+ skills' README.md | head -1 | grep -oE '[0-9]+' || true)
fi

if [ -n "$readme_skill_claim" ]; then
  if [ "$skill_count" -ne "$readme_skill_claim" ]; then
    echo "Skills: $skill_count (README claims: $readme_skill_claim) — DRIFT"
    drift_found=true
  else
    echo "Skills: $skill_count (README claims: $readme_skill_claim) — OK"
  fi
else
  echo "Skills: $skill_count (no README claim found)"
fi

echo ""
if $drift_found; then
  echo "RESULT: Drift detected — documentation needs updating"
  exit 1
else
  echo "RESULT: No drift detected"
  exit 0
fi
