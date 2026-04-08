# Prompt: Decouple orchestration/ from tools/ transitive dependency

## Context for the Implementing Agent

You are refactoring a Go workspace project. The codebase uses `go.work` with 9 modules. Your goal is to extract `pkg/disposition/` and `pkg/skill/` from the root module into standalone modules, so that `orchestration/` no longer transitively depends on `tools/` (which pulls in chromedp ~20MB and go-chart ~5MB).

**Root cause:** `orchestration/go.mod` requires the root module. The root module contains `pkg/gateway/interfaces.go`, which imports `tools.ToolDefinition`. But orchestration code NEVER uses tools — it only uses `pkg/disposition` and `pkg/skill` types. By extracting those into their own modules, orchestration can depend on them directly without pulling in the root module.

**IMPORTANT:** This is a large refactor touching ~25 files. Follow the phases sequentially. Run `go build ./...` after each phase as a gate.

---

## Phase 1: Create `disposition/` standalone module

### Step 1.1: Create the directory and go.mod

```bash
mkdir -p disposition
```

Create `disposition/go.mod`:
```
module github.com/DojoGenesis/gateway/disposition

go 1.24.0

require gopkg.in/yaml.v3 v3.0.1

require gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 // indirect
```

### Step 1.2: Copy files from pkg/disposition/ to disposition/

Copy these files (NOT `agent_initializer.go` or `agent_initializer_test.go` — those stay):

```bash
cp pkg/disposition/disposition.go disposition/
cp pkg/disposition/defaults.go disposition/
cp pkg/disposition/resolver.go disposition/
cp pkg/disposition/cache.go disposition/
cp pkg/disposition/validator.go disposition/
cp pkg/disposition/doc.go disposition/
cp pkg/disposition/defaults_test.go disposition/
cp pkg/disposition/resolver_test.go disposition/
cp pkg/disposition/cache_test.go disposition/
cp pkg/disposition/validator_test.go disposition/
cp pkg/disposition/disposition_bench_test.go disposition/
```

**DO NOT copy:**
- `pkg/disposition/agent_initializer.go` — stays in root module (imports pkg/gateway)
- `pkg/disposition/agent_initializer_test.go` — stays in root module
- `pkg/disposition/integration_test.go` — stays in root module (imports orchestration + memory)

### Step 1.3: Run go mod tidy in disposition/

```bash
cd disposition && go mod tidy && cd ..
```

### Step 1.4: Update go.work

Edit `go.work` to add `./disposition`:

```go
go 1.24.0

use (
	.
	./shared
	./events
	./provider
	./tools
	./memory
	./server
	./mcp
	./orchestration
	./disposition
)
```

### Step 1.5: Update pkg/disposition/agent_initializer.go import

The file currently imports:
```go
import (
	"context"
	"time"

	"github.com/DojoGenesis/gateway/pkg/gateway"
)
```

It also uses `DispositionConfig`, `DispositionCache`, `NewDispositionCache`, `ResolveDisposition`, and `convertToAgentConfig` — all of which now live in the `disposition/` module.

**Change the import to:**
```go
import (
	"context"
	"time"

	"github.com/DojoGenesis/gateway/disposition"
	"github.com/DojoGenesis/gateway/pkg/gateway"
)
```

**Then update ALL references in the file:**
- `*DispositionCache` → `*disposition.DispositionCache`
- `NewDispositionCache(cacheTTL)` → `disposition.NewDispositionCache(cacheTTL)`
- `ResolveDisposition(...)` → `disposition.ResolveDisposition(...)`
- `*DispositionConfig` → `*disposition.DispositionConfig`

The `convertToAgentConfig` function takes a `*DispositionConfig` and returns a `*gateway.AgentConfig`. Since the types are now in different modules, this function must stay in pkg/disposition/ (it bridges both modules). Update its parameter type:

```go
func convertToAgentConfig(d *disposition.DispositionConfig) *gateway.AgentConfig {
	return &gateway.AgentConfig{
		Pacing:     d.Pacing,
		Depth:      d.Depth,
		Tone:       d.Tone,
		Initiative: d.Initiative,
		Validation: gateway.ValidationConfig{
			Strategy:     d.Validation.Strategy,
			RequireTests: d.Validation.RequireTests,
			RequireDocs:  d.Validation.RequireDocs,
		},
		ErrorHandling: gateway.ErrorHandlingConfig{
			Strategy:   d.ErrorHandling.Strategy,
			RetryCount: d.ErrorHandling.RetryCount,
		},
		Collaboration: gateway.CollaborationConfig{
			Style:            d.Collaboration.Style,
			CheckInFrequency: d.Collaboration.CheckInFrequency,
		},
		Reflection: gateway.ReflectionConfig{
			Frequency: d.Reflection.Frequency,
			Format:    d.Reflection.Format,
			Triggers:  d.Reflection.Triggers,
		},
	}
}
```

### Step 1.6: Update the remaining pkg/disposition/ files

After copying, `pkg/disposition/` still has:
- `agent_initializer.go` (now imports `disposition/` module)
- `agent_initializer_test.go`
- `integration_test.go`

The original files (`disposition.go`, `defaults.go`, etc.) should be **deleted** from `pkg/disposition/` since they now live in `disposition/`:

```bash
rm pkg/disposition/disposition.go
rm pkg/disposition/defaults.go
rm pkg/disposition/resolver.go
rm pkg/disposition/cache.go
rm pkg/disposition/validator.go
rm pkg/disposition/doc.go
rm pkg/disposition/defaults_test.go
rm pkg/disposition/resolver_test.go
rm pkg/disposition/cache_test.go
rm pkg/disposition/validator_test.go
rm pkg/disposition/disposition_bench_test.go
```

### Step 1.7: Update all files importing pkg/disposition

Every file that imports `"github.com/DojoGenesis/gateway/pkg/disposition"` must change to `"github.com/DojoGenesis/gateway/disposition"`.

**Complete file list (19 files, excluding the ones we already handled):**

| File | Action |
|------|--------|
| `orchestration/engine.go` | Change import path |
| `orchestration/engine_test.go` | Change import path |
| `orchestration/pacing_test.go` | Change import path |
| `memory/depth_strategy.go` | Change import path |
| `memory/depth_strategy_test.go` | Change import path |
| `main.go` | Change import path |
| `integration_test.go` | Change import path |
| `phase2_integration_test.go` | Change import path |
| `pkg/collaboration/manager.go` | Change import path |
| `pkg/collaboration/manager_test.go` | Change import path |
| `pkg/intelligence/proactive.go` | Change import path |
| `pkg/intelligence/proactive_test.go` | Change import path |
| `pkg/errors/handler.go` | Change import path |
| `pkg/errors/handler_test.go` | Change import path |
| `pkg/validation/validator.go` | Change import path |
| `pkg/validation/validator_test.go` | Change import path |
| `pkg/reflection/engine.go` | Change import path |
| `pkg/reflection/engine_test.go` | Change import path |
| `pkg/disposition/integration_test.go` | Change import path |

**In each file, the change is a simple string replacement:**
```
OLD: "github.com/DojoGenesis/gateway/pkg/disposition"
NEW: "github.com/DojoGenesis/gateway/disposition"
```

No code changes needed — the package name is still `disposition`, so all usage like `disposition.DispositionConfig` remains the same.

### Step 1.8: Run go mod tidy in affected modules

```bash
cd disposition && go mod tidy && cd ..
cd orchestration && go mod tidy && cd ..
cd memory && go mod tidy && cd ..
go mod tidy
```

### Step 1.9: BUILD GATE

```bash
go build ./...
go vet ./...
```

If this fails, the most likely issues are:
- Missing `gopkg.in/yaml.v3` in `disposition/go.mod` (resolver.go needs it)
- `pkg/disposition/agent_initializer.go` still referencing types without `disposition.` prefix
- Root `go.mod` needs `disposition` module in its requires (workspace mode handles this, but `go mod tidy` may add it)

---

## Phase 2: Create `skill/` standalone module

### Step 2.1: Create the directory and go.mod

```bash
mkdir -p skill
```

Create `skill/go.mod`:
```
module github.com/DojoGenesis/gateway/skill

go 1.24.0

require (
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)
```

### Step 2.2: Copy ALL files from pkg/skill/ to skill/

```bash
cp pkg/skill/errors.go skill/
cp pkg/skill/interfaces.go skill/
cp pkg/skill/types.go skill/
cp pkg/skill/registry.go skill/
cp pkg/skill/executor.go skill/
cp pkg/skill/script_executor.go skill/
cp pkg/skill/registry_test.go skill/
cp pkg/skill/executor_test.go skill/
cp pkg/skill/script_executor_test.go skill/
mkdir -p skill/adapters
cp pkg/skill/adapters/web_tools.go skill/adapters/
cp pkg/skill/adapters/web_tools_test.go skill/adapters/
```

### Step 2.3: Delete the originals from pkg/skill/

```bash
rm pkg/skill/errors.go
rm pkg/skill/interfaces.go
rm pkg/skill/types.go
rm pkg/skill/registry.go
rm pkg/skill/executor.go
rm pkg/skill/script_executor.go
rm pkg/skill/registry_test.go
rm pkg/skill/executor_test.go
rm pkg/skill/script_executor_test.go
rm -rf pkg/skill/adapters
```

If pkg/skill/ is now empty, remove the directory:
```bash
rmdir pkg/skill 2>/dev/null || true
```

### Step 2.4: Update go.work

Edit `go.work` to add `./skill`:

```go
go 1.24.0

use (
	.
	./shared
	./events
	./provider
	./tools
	./memory
	./server
	./mcp
	./orchestration
	./disposition
	./skill
)
```

### Step 2.5: Update all files importing pkg/skill

| File | Action |
|------|--------|
| `orchestration/skill_invoker.go` | Change import path |
| `orchestration/skill_invoker_test.go` | Change import path |
| `tests/skills/smoke_test.go` | Change import path |

**In each file:**
```
OLD: "github.com/DojoGenesis/gateway/pkg/skill"
NEW: "github.com/DojoGenesis/gateway/skill"
```

### Step 2.6: Run go mod tidy in affected modules

```bash
cd skill && go mod tidy && cd ..
cd orchestration && go mod tidy && cd ..
go mod tidy
```

### Step 2.7: BUILD GATE

```bash
go build ./...
go vet ./...
```

---

## Phase 3: Update orchestration/go.mod to remove root module dependency

### Step 3.1: Edit orchestration/go.mod

**BEFORE:**
```go
module github.com/DojoGenesis/gateway/orchestration

go 1.24.0

require (
	github.com/DojoGenesis/gateway v0.0.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
)

replace github.com/DojoGenesis/gateway => ../
```

**AFTER:**
```go
module github.com/DojoGenesis/gateway/orchestration

go 1.24.0

require (
	github.com/DojoGenesis/gateway/disposition v0.0.0
	github.com/DojoGenesis/gateway/skill v0.0.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
)

replace (
	github.com/DojoGenesis/gateway/disposition => ../disposition
	github.com/DojoGenesis/gateway/skill => ../skill
)
```

### Step 3.2: Run go mod tidy

```bash
cd orchestration && go mod tidy && cd ..
```

### Step 3.3: VERIFY the dependency is gone

```bash
cd orchestration && grep -c chromedp go.sum; cd ..
```

**Expected output: `0`** (chromedp should no longer appear in orchestration's go.sum)

Also check:
```bash
cd orchestration && grep -c go-chart go.sum; cd ..
```

**Expected output: `0`**

### Step 3.4: FINAL BUILD GATE

```bash
go build ./...
go vet ./...
go test ./... -count=1
```

---

## Phase 4: Clean up root module

### Step 4.1: Run go mod tidy everywhere

```bash
for dir in . shared events provider tools memory server mcp orchestration disposition skill; do
  (cd "$dir" && go mod tidy)
done
```

### Step 4.2: Verify pkg/disposition/ is now thin adapter only

After the refactor, `pkg/disposition/` should contain ONLY:
- `agent_initializer.go` — imports both `disposition/` and `pkg/gateway/`
- `agent_initializer_test.go`
- `integration_test.go`

```bash
ls pkg/disposition/
```

Expected: `agent_initializer.go  agent_initializer_test.go  integration_test.go`

### Step 4.3: FINAL VERIFICATION

```bash
go build ./...
go vet ./...
go test ./... -count=1 -race
```

---

## Troubleshooting

### "package X is not in std"
The new module's `go.mod` is missing a dependency. Run `go mod tidy` in that module directory.

### "ambiguous import"
Two modules export the same package path. This happens if you forgot to delete the old files from `pkg/disposition/` or `pkg/skill/`. Verify the old files are deleted.

### "cannot find module"
The `replace` directive path is wrong, or `go.work` is missing the new module. Check both.

### Build fails in orchestration/ after Phase 3
If orchestration still tries to import the root module, grep for remaining imports:
```bash
grep -rn 'AgenticGatewayByDojoGenesis"' orchestration/ --include='*.go' | grep -v '/disposition\|/skill\|/orchestration"'
```
Any hits indicate code in orchestration/ that still imports the root module directly. Those imports need to be refactored to use the new standalone modules instead.

### integration_test.go fails
The integration test in `pkg/disposition/integration_test.go` may need its import path updated from `pkg/disposition` to `disposition`. It also imports `orchestration` and `memory` — verify those are still accessible through the workspace.

---

## File Manifest (complete list of changes)

### New files:
- `disposition/go.mod`
- `disposition/disposition.go` (moved from pkg/disposition/)
- `disposition/defaults.go` (moved)
- `disposition/resolver.go` (moved)
- `disposition/cache.go` (moved)
- `disposition/validator.go` (moved)
- `disposition/doc.go` (moved)
- `disposition/defaults_test.go` (moved)
- `disposition/resolver_test.go` (moved)
- `disposition/cache_test.go` (moved)
- `disposition/validator_test.go` (moved)
- `disposition/disposition_bench_test.go` (moved)
- `skill/go.mod`
- `skill/errors.go` (moved from pkg/skill/)
- `skill/interfaces.go` (moved)
- `skill/types.go` (moved)
- `skill/registry.go` (moved)
- `skill/executor.go` (moved)
- `skill/script_executor.go` (moved)
- `skill/registry_test.go` (moved)
- `skill/executor_test.go` (moved)
- `skill/script_executor_test.go` (moved)
- `skill/adapters/web_tools.go` (moved)
- `skill/adapters/web_tools_test.go` (moved)

### Modified files:
- `go.work` (add ./disposition, ./skill)
- `orchestration/go.mod` (replace root dep with disposition + skill)
- `orchestration/engine.go` (import path change)
- `orchestration/engine_test.go` (import path change)
- `orchestration/pacing_test.go` (import path change)
- `orchestration/skill_invoker.go` (import path change)
- `orchestration/skill_invoker_test.go` (import path change)
- `memory/depth_strategy.go` (import path change)
- `memory/depth_strategy_test.go` (import path change)
- `main.go` (import path change)
- `integration_test.go` (import path change)
- `phase2_integration_test.go` (import path change)
- `pkg/disposition/agent_initializer.go` (import disposition module + qualify types)
- `pkg/disposition/integration_test.go` (import path change)
- `pkg/collaboration/manager.go` (import path change)
- `pkg/collaboration/manager_test.go` (import path change)
- `pkg/intelligence/proactive.go` (import path change)
- `pkg/intelligence/proactive_test.go` (import path change)
- `pkg/errors/handler.go` (import path change)
- `pkg/errors/handler_test.go` (import path change)
- `pkg/validation/validator.go` (import path change)
- `pkg/validation/validator_test.go` (import path change)
- `pkg/reflection/engine.go` (import path change)
- `pkg/reflection/engine_test.go` (import path change)
- `tests/skills/smoke_test.go` (import path change)

### Deleted files:
- `pkg/disposition/disposition.go`
- `pkg/disposition/defaults.go`
- `pkg/disposition/resolver.go`
- `pkg/disposition/cache.go`
- `pkg/disposition/validator.go`
- `pkg/disposition/doc.go`
- `pkg/disposition/defaults_test.go`
- `pkg/disposition/resolver_test.go`
- `pkg/disposition/cache_test.go`
- `pkg/disposition/validator_test.go`
- `pkg/disposition/disposition_bench_test.go`
- `pkg/skill/errors.go`
- `pkg/skill/interfaces.go`
- `pkg/skill/types.go`
- `pkg/skill/registry.go`
- `pkg/skill/executor.go`
- `pkg/skill/script_executor.go`
- `pkg/skill/registry_test.go`
- `pkg/skill/executor_test.go`
- `pkg/skill/script_executor_test.go`
- `pkg/skill/adapters/web_tools.go`
- `pkg/skill/adapters/web_tools_test.go`
