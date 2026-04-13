# Agent Handoff Package

**From:** Claude Opus 4.6 (session 1d4839e9)
**To:** Opus-level orchestrator
**Date:** 2026-04-13
**Subject:** Dojo Gateway Wave 2 — Private Agentic Workspace for CWD Consulting

---

## 1. Objective

Stand up a working private agentic workspace at `gateway.trespies.dev` that CWD staff can use for daily consulting work — chat, document Q&A, Atlas pipeline triggers — by completing 10 discrete work items across the Gateway, chat SPA, and infrastructure layers.

---

## 2. Required Context

### Project Memory (read these first)
- `/Users/alfonsomorales/.claude/projects/-Users-alfonsomorales-ZenflowProjects/memory/MEMORY.md` — full project index
- `/Users/alfonsomorales/.claude/projects/-Users-alfonsomorales-ZenflowProjects/memory/compression_log_2026-04-13.md` — what shipped this session
- `/Users/alfonsomorales/.claude/projects/-Users-alfonsomorales-ZenflowProjects/memory/project_dojo_platform_era.md` — Gateway era history
- `/Users/alfonsomorales/.claude/projects/-Users-alfonsomorales-ZenflowProjects/memory/entity_tres_pies_ecosystem.md` — TresPies entity structure, CWD relationship

### Repository State (as of Apr 13, 2026)
- **Gateway HEAD:** `a46953d` — `TresPies-source/AgenticGatewayByDojoGenesis` (main)
- **CWD HEAD:** `815ac37` — `TresPies-source/CWD` (master)
- Both repos are clean (0 uncommitted files)

### Key Gateway Files
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/router.go` — all route registrations
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/handle_conversations.go` — Wave 1 conversation + message handlers (NEW, unprotected — see item 5 below)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/handle_oauth.go` — GitHub OAuth2 flow (NEW)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/handle_workflow_builder.go` — the embed/serve pattern for SPAs (reference for item 1)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/workflowui/ui.go` — Go embed boilerplate for SPA (reference for item 1)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/database/adapter.go` — DatabaseAdapter interface (Conversation + Message models)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/server/database/local_adapter.go` — SQLite implementations including messages table

### Chat SPA (Wave 1 Track C output)
- `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/chat-ui/` — SvelteKit 2 + Svelte 5 SPA
- `chat-ui/src/lib/api.ts` — API client (login, streamChat, listConversations, createMessage, etc.)
- `chat-ui/src/lib/stores.svelte.ts` — $state runes for auth, messages, conversations
- `chat-ui/src/routes/+page.svelte` — main chat page (sidebar + thread + input)
- `chat-ui/src/routes/login/+page.svelte` — login + GitHub OAuth button
- **Build:** `npm run build` from `chat-ui/` → produces `chat-ui/build/`
- **Status:** SPA scaffolded and building, but NOT yet embedded in or served by the Gateway binary

### Atlas Pipeline (Wave 1 Track E output)
- `/Users/alfonsomorales/ZenflowProjects/CWD/atlas/pipeline.json` — 6-stage DAG, 7 variables
- `/Users/alfonsomorales/ZenflowProjects/CWD/atlas/Makefile` — parallel stage execution
- `/Users/alfonsomorales/ZenflowProjects/CWD/atlas/lib/` — stats, io_utils, geo, census modules

### Deferred Feature Specs
- Full specs for DF-1 through DF-6 (image gen, voice, i18n, horizontal scaling, marketplace, SAML) are in the session transcript. Not in memory files — capture to `handoffs/deferred_features_df1-df6.md` if needed for future handoffs.

---

## 3. Task Definition — 10 Prioritized Work Items

Work items are ordered: dependencies first, highest CWD value first within a tier.

---

### Item 1 — Serve chat SPA from Gateway binary [BLOCKS deployment]
**Type:** Go + build system
**Effort:** S (2-4 hrs)
**Files to create:**
- `server/chatui/ui.go` — Go embed package, same pattern as `server/workflowui/ui.go`
- `server/handle_chat_builder.go` — handler, same pattern as `server/handle_workflow_builder.go`

**Pattern (exact):** Copy `workflowui/ui.go`, change package name to `chatui`, change embed path to `all:dist`. Copy `handle_workflow_builder.go`, rename to `handle_chat_builder.go`, change prefix from `/workflow` to `/chat`, change import from `workflowui` to `chatui`.

**Router change** (`server/router.go`): Add below the `/workflow` block:
```go
chatHandler := s.chatUIHandler()
s.router.GET("/chat", chatHandler)
s.router.GET("/chat/*filepath", chatHandler)
```

**Makefile change:** Add `build-chat-spa` target that runs `npm run build` in `chat-ui/` and copies `chat-ui/build/` to `server/chatui/dist/`.

**Definition of done:**
- `go build ./...` passes with the new chatui package
- `curl http://localhost:7340/chat` returns 200 with HTML content
- If SPA not built, returns 503 with instructions (same pattern as workflow builder)

---

### Item 2 — Auth middleware on conversation routes [SECURITY]
**Type:** Go — 1 line change
**Effort:** XS (15 min)
**File:** `server/router.go`

**Problem:** The `/v1/conversations` group was added in Wave 1 but has no auth middleware. Any unauthenticated caller can read/write conversations.

**Fix:** Add `middleware.AuthMiddleware()` to the conversations group:
```go
convGroup := v1.Group("/conversations")
convGroup.Use(middleware.AuthMiddleware())   // ← ADD THIS LINE
{
    convGroup.GET("", s.handleListConversations)
    // ... rest of routes unchanged
}
```

The conversation handlers already extract `user_id` from gin context (set by AuthMiddleware) — this just enforces that the middleware actually runs.

**Definition of done:**
- `curl -X GET http://localhost:7340/v1/conversations` without a token returns 401
- `curl -X GET ... -H "Authorization: Bearer <valid_token>"` returns 200

---

### Item 3 — Conversation persistence wiring in chat SPA [CLOSES WAVE 1 LOOP]
**Type:** Svelte 5 TypeScript
**Effort:** S (3-5 hrs)
**Files:** `chat-ui/src/routes/+page.svelte`, `chat-ui/src/lib/api.ts`, `chat-ui/src/lib/stores.svelte.ts`

**Problem:** The message handlers exist (Wave 1 Track A), and the API client has stubs, but the chat SPA does not persist messages to the backend. Conversations live only in memory and are lost on reload.

**What to wire:**
1. On first message send: `POST /v1/conversations` to create conversation, store `conversationId` in state
2. After each user send: `POST /v1/conversations/:id/messages` with `{role: "user", content}`
3. After each assistant response completes (stream done): `POST /v1/conversations/:id/messages` with `{role: "assistant", content, model, provider}`
4. On sidebar load: `GET /v1/conversations` to populate the conversation list
5. On sidebar item click: `GET /v1/conversations/:id/messages` to restore the thread

**Definition of done:**
- Send a message, close browser, reopen — conversation is in the sidebar and thread restores
- Conversation title auto-set to first N chars of first user message (can be done client-side or via a future auto-title endpoint)

---

### Item 4 — Hetzner VPS provisioning [DEPLOYS THE PRODUCT]
**Type:** Infrastructure
**Effort:** M (4-6 hrs total, mostly waiting for DNS)
**Depends on:** Items 1, 2, 3 ideally done first (so the deployed version is functional)

**Spec already written:** Hetzner CPX21, Ubuntu 24.04 LTS, Ashburn VA, ~$9/mo + ~$1.50/mo backup.

**Steps:**
1. Create server via Hetzner Cloud console (or API)
2. Point `gateway.trespies.dev` A record to server IP (trespies.dev registered, DNS provider TBD)
3. SSH in, run `provision.sh` (to be written):
   ```bash
   # provision.sh shape:
   apt install -y caddy sqlite3
   useradd -r -s /sbin/nologin dojo
   # Download gateway binary from goreleaser / GitHub Release
   # Write /etc/dojo/gateway-config.yaml
   # Write /etc/caddy/Caddyfile (reverse proxy gateway.trespies.dev → localhost:7340)
   # Write /etc/systemd/system/dojo-gateway.service
   # systemctl enable + start dojo-gateway caddy
   # curl https://gateway.trespies.dev/health → 200
   ```
4. Set environment variables on the server:
   - `GITHUB_OAUTH_CLIENT_ID` + `GITHUB_OAUTH_CLIENT_SECRET` (create GitHub OAuth App at github.com/settings/applications/new, callback URL: `https://gateway.trespies.dev/auth/github/callback`)
   - `GITHUB_OAUTH_REDIRECT_URI=https://gateway.trespies.dev/auth/github/callback`
   - `GITHUB_OAUTH_ENABLED=true`
   - `REGISTRATION_ENABLED=false` (closed — admin creates users manually)
   - `DOJO_JWT_SECRET=<random 64-char hex>`
   - Provider API keys (ANTHROPIC_API_KEY, OPENAI_API_KEY)
5. Create first admin user via CLI or direct DB insert

**Definition of done:**
- `https://gateway.trespies.dev/health` returns `{"status":"ok"}`
- `https://gateway.trespies.dev/chat` serves the SPA
- Can log in with GitHub account
- Can send a chat message and receive a streaming response

---

### Item 5 — RAG document Q&A pipeline [BIGGEST CWD VALUE UNLOCK]
**Type:** Go backend + Svelte frontend
**Effort:** L (1-2 sessions)
**This is Wave 2-A from the LibreChat parity roadmap.**

**What CWD actually needs:** Ask questions over grant documents, Atlas analysis PDFs, policy briefs. "What does the MCF LOI say about timeline?" "Summarize the key findings from the Blinder-Oaxaca analysis."

**Architecture (SQLite-only, no vector DB):**
1. File upload endpoint: `POST /v1/files` (multipart) → parse PDF/txt/md → chunk (512 tokens, 50 overlap) → embed each chunk via existing `memory/embeddings.go` → store in new `document_chunks` SQLite table with `embedding BLOB`
2. On chat with a document: retrieve top-K chunks via cosine similarity (SQLite FTS or manual cosine over stored embeddings) → inject into system prompt: `"Context from [filename]:\n{chunk1}\n{chunk2}\n..."`
3. Chat SPA: file attachment button in input bar, file list in sidebar, "Ask about this document" mode

**Key existing asset:** `memory/embeddings.go` already has embedding generation. The RAG pipeline reuses it for document chunks rather than memory entries.

**New database objects:**
```sql
CREATE TABLE documents (id TEXT PRIMARY KEY, user_id TEXT, filename TEXT, content_type TEXT, size_bytes INT, created_at DATETIME);
CREATE TABLE document_chunks (id TEXT PRIMARY KEY, document_id TEXT, chunk_index INT, content TEXT, embedding BLOB, created_at DATETIME);
```

**Definition of done:**
- Upload a PDF, ask a question, receive an answer that cites content from the document
- Works with: Atlas data-context.md, MCF LOI draft, Arnold Ventures proposal

---

### Item 6 — Atlas pipeline as Gateway commission [BRIDGES THE TWO SYSTEMS]
**Type:** Go backend + commission definition
**Effort:** M (1 session)
**Depends on:** Items 1-4 (deployed VPS)

**Goal:** CWD staff (including Justice) can trigger an Atlas data refresh from the chat UI: "Refresh the Atlas with 2025 ACS data." The Gateway runs the pipeline, reports progress, delivers the built HTML artifact.

**How:**
1. Register `CWD/atlas/pipeline.json` as a Gateway DAG template (copy to `gateway/templates/orchestration/atlas-pipeline.yaml` or adapt pipeline.json to Gateway DAG format)
2. Create a commission handler: `POST /v1/commissions/atlas-refresh` — reads pipeline variables from request, executes `make -C $CWD_ATLAS_PATH ACS_YEAR=<year> all` as a subprocess in WASM sandbox or direct subprocess
3. Stream progress via existing SSE orchestration events (`GET /v1/orchestrate/:id/events`)
4. On completion, attach the built `atlas/dist/Madison_Equity_Atlas.html` as a CAS artifact, return download URL
5. Chat SPA: commission button in sidebar or slash command `/atlas refresh`

**Definition of done:**
- From chat: type `/atlas refresh` → Gateway runs `make all` in CWD/atlas/ → streams stage-by-stage progress → posts link to rebuilt Atlas HTML

---

### Item 7 — Admin dashboard UI [OPERATIONAL]
**Type:** Svelte 5
**Effort:** M (1 session)
**Depends on:** Item 1 (SPA serving)

**APIs already exist** (no new backend work needed):
- `GET /admin/users` — list users (Wave 1 Track A)
- `POST /admin/users/:id/deactivate` + `/activate` (Wave 1 Track A)
- `GET /admin/providers` — provider health
- `GET /admin/costs` — token spend aggregation
- `GET /admin/mcp/status` — MCP server status

**Add to chat SPA:** An `/admin` route (protected, admin JWT role only) with tabs: Users, Providers, Costs. Wire existing admin endpoints into the UI. No new Go code required.

**Definition of done:**
- Admin user can navigate to `/chat/admin`, see user list, deactivate a user, see provider health
- Non-admin JWT returns 403 on /admin

---

### Item 8 — Prompt template library [CWD ADOPTION]
**Type:** Go backend + Svelte frontend
**Effort:** S (1 session)
**This is Wave 2-D from the LibreChat parity roadmap.**

**CWD-specific templates to pre-populate:**
- "Grant writing assistant" — system prompt for grant narrative drafting, MCF/Arnold context
- "Atlas data analyst" — system prompt with NARI methodology, Blinder-Oaxaca interpretation guidance
- "Policy brief drafter" — CWD format, audience: Madison City Council + county board
- "Meeting summarizer" — structured output: decisions, action items, owners

**Backend:**
```sql
CREATE TABLE prompt_templates (id TEXT PRIMARY KEY, user_id TEXT, title TEXT, system_prompt TEXT, is_public BOOL, created_at DATETIME);
```
Routes: `GET/POST /v1/templates`, `PUT/DELETE /v1/templates/:id`

**Frontend:** Template picker in the input bar "new conversation" flow. Select a template → system prompt pre-loaded. Edit before confirming.

**Definition of done:**
- Create a template from the UI, select it when starting a conversation, confirm the system prompt is injected into the first API call

---

### Item 9 — D1Syncer → Server injection [CLOSES OPEN GATEWAY ITEM]
**Type:** Go — dependency injection
**Effort:** S (1-2 hrs)
**Files:** `server/server.go`, `main.go`

**Problem (from MEMORY.md open items):** The D1Syncer runs in a `main.go` goroutine but `Server.d1Syncer` field is never populated. `GET /api/cas/status` always returns nil.

**Fix:** In `main.go`, after creating the D1Syncer, pass it to the Server constructor or inject it via `server.SetD1Syncer(syncer)`. Check `server/server.go` for the struct field name (`d1Syncer`).

**Required env vars for this to actually work:**
- `DOJO_D1_DATABASE_ID` — Cloudflare D1 database ID
- `DOJO_D1_ACCOUNT_ID` — Cloudflare account ID
- `DOJO_D1_API_TOKEN` — Cloudflare API token with D1 write permissions

**Definition of done:**
- `curl https://gateway.trespies.dev/api/cas/status` returns non-nil JSON (not null or empty response)

---

### Item 10 — Provision script + GitHub OAuth App [DEPLOYMENT PREREQUISITE]
**Type:** Bash + GitHub App creation
**Effort:** S (2-3 hrs)
**Depends on:** DNS for trespies.dev propagated (may need to do before Item 4)

**Write `deploy/provision.sh`** — idempotent, dry-run capable (per `seed_dry_run_as_founding_gate.md`). First run must be `--dry-run`. Structure:
```bash
#!/bin/bash
set -euo pipefail
DRY_RUN="${1:-}"
DOMAIN="gateway.trespies.dev"
GATEWAY_VERSION="v3.0.0"   # pin to current tag

# Phase 1: System packages (apt)
# Phase 2: dojo user + directories
# Phase 3: gateway binary from goreleaser release
# Phase 4: gateway-config.yaml
# Phase 5: Caddyfile
# Phase 6: systemd unit files
# Phase 7: enable + start
# Phase 8: health check

if [[ $DRY_RUN == "--dry-run" ]]; then
    echo "[DRY RUN] Would execute: ..."
fi
```

**GitHub OAuth App:**
- Create at `https://github.com/settings/applications/new`
- Homepage URL: `https://trespies.dev`
- Callback URL: `https://gateway.trespies.dev/auth/github/callback`
- Copy Client ID + Client Secret → VPS environment variables

**Definition of done:**
- `./provision.sh --dry-run` prints all steps without executing
- `./provision.sh` (live) results in a running Gateway at `gateway.trespies.dev/health`

---

## 4. Definition of Done — Full Wave 2

- [ ] Chat SPA served at `https://gateway.trespies.dev/chat`
- [ ] Login with GitHub account, sessions persist (JWT)
- [ ] Send a message, close browser, reopen — conversation restored
- [ ] Conversation routes require valid JWT (no anonymous reads)
- [ ] Upload a PDF, ask a question, receive a grounded answer
- [ ] Admin user can manage users and see provider health via `/chat/admin`
- [ ] Prompt templates selectable when starting a new conversation
- [ ] Atlas pipeline triggerable from chat (`/atlas refresh`)
- [ ] D1Syncer injected — `/api/cas/status` returns non-nil
- [ ] VPS live at `gateway.trespies.dev`, auto-TLS, GitHub OAuth working

---

## 5. Constraints & Boundaries

- **DO NOT** use CGO-dependent packages (gateway uses `modernc.org/sqlite`, pure Go)
- **DO NOT** add PostgreSQL, Redis, or any new infrastructure to the VPS — SQLite WAL is sufficient for CWD's single-tenant scale
- **DO NOT** commit `server/chatui/dist/` to git — it is produced by `make build-chat-spa` at deploy time (same as workflowui)
- **DO NOT** store LLM API keys in code or git — env vars only
- **DO NOT** enable `REGISTRATION_ENABLED=true` on the VPS — registration must be closed; admin creates users
- **MUST** follow Svelte 5 conventions: `onclick={handler}` not `on:click`, `$state` runes not stores
- **MUST** run `go build ./... && go test ./server/ -timeout 120s` after any Go changes
- **MUST** keep all Go changes in the `server/` package tree — no changes to `workflow/`, `memory/`, `orchestration/`, or other modules unless item specifically requires it
- **Repo auth:** Push Gateway to `TresPies-source/AgenticGatewayByDojoGenesis` as TresPies-source. Push to `DojoGenesis/gateway` as DojoGenesis. Run `gh auth switch --user TresPies-source` before pushing to TresPies-source remotes.

---

## 6. Dispatch Strategy

These 10 items decompose into 3 waves of parallel dispatch:

**Wave A (unblock everything, 2-3 Sonnet agents in parallel):**
- Item 1: Chat SPA serving (Go embed + handler)
- Item 2: Auth middleware on conversation routes (15 min, trivial)
- Item 10: Provision script + GitHub OAuth App setup

**Wave B (runs after Wave A, 2-3 Sonnet agents in parallel):**
- Item 3: Conversation persistence wiring in chat SPA
- Item 7: Admin dashboard UI
- Item 8: Prompt template library
- Item 9: D1Syncer injection

**Wave C (Opus orchestrator synthesizes):**
- Item 4: VPS provisioning (execute provision.sh after Wave A/B ships)
- Item 5: RAG pipeline (standalone, can run in parallel with VPS)
- Item 6: Atlas pipeline as Gateway commission (deploy + test end-to-end)

Track D after all waves: End-to-end test from Justice's browser. Send a message, upload a grant doc, ask a question, trigger Atlas refresh.

---

## 7. Next Steps After Completion

Upon Wave 2 completion, hand off to an Opus-level orchestrator for:

1. **Wave 3 work** (quality-of-life): Agent builder UI, skill browser, message search, conversation export, artifacts rendering
2. **CWD onboarding**: Walk Justice through the deployed workspace, create CWD-specific prompt templates, load Atlas data context as the first RAG document
3. **Grant proposal integration**: Use the workspace to actively draft the MCF LOI (due Jun 3 2026) and Arnold Ventures follow-up

---

## Handoff Checklist (Self-Assessment)

- [x] Objective is a single, clear sentence
- [x] All Required Context files are valid paths in currently committed repos
- [x] Task Definition is unambiguous — each item has specific files, patterns, and code sketches
- [x] Definition of Done is a list of binary, testable criteria
- [x] Constraints are explicit (no CGO, no new infra, no open registration)
- [x] Next steps after completion are clearly defined
- [x] Dispatch strategy provided (Wave A → B → C ordering with agent model assignments)
