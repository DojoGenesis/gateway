#!/bin/bash
set -e

echo "=== Validating Disposition/Skill Module Refactoring ==="
echo

# Test 1: Modules build
echo "✓ Testing builds..."
go build ./disposition > /dev/null 2>&1
go build ./skill > /dev/null 2>&1
go build ./orchestration > /dev/null 2>&1
go build ./pkg/disposition > /dev/null 2>&1
echo "  ✅ All refactored modules build successfully"

# Test 2: Vet passes
echo "✓ Testing go vet..."
go vet ./disposition/... > /dev/null 2>&1
go vet ./skill/... > /dev/null 2>&1
go vet ./orchestration/... > /dev/null 2>&1
go vet ./pkg/disposition/... > /dev/null 2>&1
echo "  ✅ All modules pass go vet"

# Test 3: Tests pass
echo "✓ Running tests..."
go test ./disposition/... -count=1 > /dev/null 2>&1
go test ./skill/... -count=1 > /dev/null 2>&1
go test ./orchestration/... -count=1 > /dev/null 2>&1
go test ./pkg/disposition/... -count=1 > /dev/null 2>&1
echo "  ✅ All tests pass"

# Test 4: Heavy dependencies removed
echo "✓ Checking dependencies..."
if grep -q chromedp orchestration/go.sum 2>/dev/null; then
    echo "  ❌ chromedp still present!"
    exit 1
fi
if grep -q go-chart orchestration/go.sum 2>/dev/null; then
    echo "  ❌ go-chart still present!"
    exit 1
fi
echo "  ✅ chromedp and go-chart removed from orchestration"

# Test 5: Module structure
echo "✓ Checking module structure..."
[ -d disposition ] || { echo "  ❌ disposition/ missing"; exit 1; }
[ -d skill ] || { echo "  ❌ skill/ missing"; exit 1; }
[ -d pkg/disposition ] || { echo "  ❌ pkg/disposition/ missing"; exit 1; }
[ ! -d pkg/skill ] || { echo "  ❌ pkg/skill/ should not exist"; exit 1; }
echo "  ✅ Module structure correct"

# Test 6: Required files exist
echo "✓ Checking required files..."
[ -f disposition/go.mod ] || { echo "  ❌ disposition/go.mod missing"; exit 1; }
[ -f skill/go.mod ] || { echo "  ❌ skill/go.mod missing"; exit 1; }
[ -f orchestration/go.mod ] || { echo "  ❌ orchestration/go.mod missing"; exit 1; }
echo "  ✅ All required files present"

# Test 7: Orchestration module dependencies
echo "✓ Checking orchestration dependencies..."
if grep -q "github.com/TresPies-source/AgenticGatewayByDojoGenesis v0.0.0" orchestration/go.mod; then
    echo "  ❌ orchestration still depends on root module!"
    exit 1
fi
if ! grep -q "github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition v0.0.0" orchestration/go.mod; then
    echo "  ❌ orchestration missing disposition dependency!"
    exit 1
fi
if ! grep -q "github.com/TresPies-source/AgenticGatewayByDojoGenesis/skill v0.0.0" orchestration/go.mod; then
    echo "  ❌ orchestration missing skill dependency!"
    exit 1
fi
echo "  ✅ Orchestration dependencies correct"

echo
echo "=== ALL VALIDATION CHECKS PASSED ==="
echo "The refactoring is complete and verified."
