#!/usr/bin/env bash
# Usage: ./scripts/rename-module.sh <old-path> <new-path>
# Example: ./scripts/rename-module.sh github.com/TresPies-source/AgenticGatewayByDojoGenesis github.com/DojoGenesis/gateway
#
# Renames the Go module path across all go.mod, go.sum, and *.go files.
# SKIPS *.pb.go files (protobuf binary descriptors contain baked-in paths
# that corrupt when string length changes). Regenerate .pb.go separately.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

OLD_PATH="${1:-}"
NEW_PATH="${2:-}"
AUTO_YES=false

# Parse --yes flag from any position
for arg in "$@"; do
  if [[ "$arg" == "--yes" ]]; then
    AUTO_YES=true
  fi
done

# Strip --yes from positional args
args=()
for arg in "$@"; do
  if [[ "$arg" != "--yes" ]]; then
    args+=("$arg")
  fi
done
OLD_PATH="${args[0]:-}"
NEW_PATH="${args[1]:-}"

# --- Validation ---
if [[ -z "$OLD_PATH" || -z "$NEW_PATH" ]]; then
  echo -e "${RED}Error: both <old-path> and <new-path> are required.${NC}"
  echo "Usage: $0 <old-path> <new-path> [--yes]"
  echo "Example: $0 github.com/TresPies-source/AgenticGatewayByDojoGenesis github.com/DojoGenesis/gateway"
  exit 1
fi

if [[ "$OLD_PATH" == "$NEW_PATH" ]]; then
  echo -e "${RED}Error: old-path and new-path are identical.${NC}"
  exit 1
fi

# --- Dry Run: count affected files ---
echo "=== Module Rename: Dry Run ==="
echo "  Old: $OLD_PATH"
echo "  New: $NEW_PATH"
echo ""

# Find affected .go files (excluding .pb.go)
GO_FILES=$(grep -rl --include='*.go' "$OLD_PATH" . | grep -v '\.pb\.go$' | grep -v '/vendor/' || true)
GO_COUNT=$(echo "$GO_FILES" | grep -c '.' || echo 0)

# Find affected go.mod files
GOMOD_FILES=$(grep -rl --include='go.mod' "$OLD_PATH" . | grep -v '/vendor/' || true)
GOMOD_COUNT=$(echo "$GOMOD_FILES" | grep -c '.' || echo 0)

# Find affected go.sum files
GOSUM_FILES=$(grep -rl --include='go.sum' "$OLD_PATH" . | grep -v '/vendor/' || true)
GOSUM_COUNT=$(echo "$GOSUM_FILES" | grep -c '.' || echo 0)

# Find affected YAML files
YML_FILES=$(grep -rl --include='*.yml' --include='*.yaml' "$OLD_PATH" . | grep -v '/vendor/' || true)
YML_COUNT=$(echo "$YML_FILES" | grep -c '.' || echo 0)

# Find affected Markdown files
MD_FILES=$(grep -rl --include='*.md' "$OLD_PATH" . | grep -v '/vendor/' || true)
MD_COUNT=$(echo "$MD_FILES" | grep -c '.' || echo 0)

# Find .pb.go files that would be skipped
PBGO_FILES=$(grep -rl --include='*.pb.go' "$OLD_PATH" . | grep -v '/vendor/' || true)
PBGO_COUNT=$(echo "$PBGO_FILES" | grep -c '.' || echo 0)

TOTAL=$((GO_COUNT + GOMOD_COUNT + GOSUM_COUNT + YML_COUNT + MD_COUNT))

echo "Files to rename:"
echo "  *.go (non-pb):  $GO_COUNT"
echo "  go.mod:         $GOMOD_COUNT"
echo "  go.sum:         $GOSUM_COUNT"
echo "  *.yml / *.yaml: $YML_COUNT"
echo "  *.md:           $MD_COUNT"
echo "  ──────────────────────"
echo "  Total:          $TOTAL"
echo ""

if [[ $PBGO_COUNT -gt 0 ]]; then
  echo -e "${YELLOW}Will SKIP $PBGO_COUNT .pb.go file(s) (regenerate protobuf separately).${NC}"
  echo ""
fi

if [[ $TOTAL -eq 0 ]]; then
  echo -e "${YELLOW}No files contain the old path. Nothing to do.${NC}"
  exit 0
fi

# --- Confirmation ---
if [[ "$AUTO_YES" != true ]]; then
  read -rp "Proceed with rename? [y/N] " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 0
  fi
fi

# --- Execute sed replacements ---
echo ""
echo "=== Renaming... ==="

# Portable sed -i: macOS requires '' after -i, GNU does not
SED_INPLACE=(-i)
if [[ "$(uname)" == "Darwin" ]]; then
  SED_INPLACE=(-i '')
fi

rename_files() {
  local files="$1"
  local label="$2"
  if [[ -n "$files" ]]; then
    echo "$files" | while IFS= read -r f; do
      sed "${SED_INPLACE[@]}" "s|${OLD_PATH}|${NEW_PATH}|g" "$f"
    done
    echo -e "  ${GREEN}$label: done${NC}"
  fi
}

rename_files "$GO_FILES" "*.go files"
rename_files "$GOMOD_FILES" "go.mod files"
rename_files "$GOSUM_FILES" "go.sum files"
rename_files "$YML_FILES" "YAML files"
rename_files "$MD_FILES" "Markdown files"

# --- Report skipped .pb.go files ---
if [[ $PBGO_COUNT -gt 0 ]]; then
  echo ""
  echo -e "${YELLOW}=== SKIPPED .pb.go files (regenerate with protoc) ===${NC}"
  echo "$PBGO_FILES" | while IFS= read -r f; do
    echo "  $f"
  done
fi

# --- Verification build ---
echo ""
echo "=== Verifying: go build ./... ==="
if go build ./... 2>&1; then
  BUILD_STATUS="PASS"
  BUILD_COLOR="$GREEN"
else
  BUILD_STATUS="FAIL"
  BUILD_COLOR="$RED"
fi

# --- Summary ---
echo ""
echo "=== Summary ==="
echo -e "  Renamed: ${GREEN}$TOTAL${NC} files"
echo -e "  Skipped: ${YELLOW}$PBGO_COUNT${NC} .pb.go files"
echo -e "  Build:   ${BUILD_COLOR}${BUILD_STATUS}${NC}"

if [[ "$BUILD_STATUS" == "FAIL" ]]; then
  exit 1
fi
