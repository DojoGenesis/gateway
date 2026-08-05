# `/api/*` route disposition

**Status:** `/api/*` requires authentication in code. Not yet deployed.
**Issues:** DGS-108 (RCE — closed, deployed v3.3.3), DGS-100 (auth — this change), DGS-101 (registration — closed on the host), DGS-112 (dev-token fail-open — closed).
**Last verified against the code and the live host:** 2026-08-05.

This file started as the input to the DGS-100 auth design. It now records what
was decided and why, including a premise it originally got wrong.

---

## The correction that unblocked this

DGS-100 sat open from 2026-07-24 on one belief:

> The SPAs call `/api/workflows*` and `/api/cas/*` **from the browser**.
> Attaching `AuthMiddleware` blind would break the workflow builder.

**That consumer does not exist in any deployed gateway.** Three independent checks:

| Check | Result |
|---|---|
| `git ls-files server/workflowui/dist server/chatui/dist` | only `.gitkeep` — the SPA build is a separate `make build-spa` step |
| `.goreleaser.yml` `before.hooks` | `go mod download` only. No `npm`, no `make build-spa` |
| `GET https://gateway.trespies.dev/workflow` and `/chat` | **503** — the deployed binary has no SPA embedded |

No released binary has ever contained the workflow builder. The blocker was
protecting a browser client that has never shipped, while the routes it was
protecting served the whole internet.

So the SPA's auth story is a **prerequisite for shipping the workflow builder**,
not a reason to leave the API open. It is a feature question now, not a
security one.

---

## The two defects that shared this prefix

| | Defect | Layer | Status |
|---|---|---|---|
| **A** | A workflow step executed shell commands as the gateway process | capability | **Closed and deployed** — `WORKFLOW_RUN_COMMAND_ENABLED`, off in production (v3.3.3, verified on the host) |
| **B** | `/api/*` had no authentication | authn | **Closed in code**, not yet deployed |

They are independent controls and neither replaces the other. A capability gate
holds when authentication is bypassed; authentication holds when a capability is
deliberately enabled.

---

## The two-handler-path trap

`/api/*` is served by two registration paths that share no middleware. Any fix
must cover both; a grep of one proves nothing about the other.

| Path | Where | How it is gated |
|---|---|---|
| 1 — Gin | `server/router.go` `setupRoutes()`, on the root `s.router` | `middleware.AuthMiddleware()` at each registration |
| 2 — ServeMux | `workflow/api/handler.go` `RegisterRoutes`, mounted via `gin.WrapH` | the bare mux cannot take gin middleware — the **gin route that wraps it** carries it instead |

The global `OptionalAuthMiddleware` never rejects; it assigns a guest UUID and
calls `c.Next()`. Blocking happens only where a route or group adds
`AuthMiddleware()` explicitly.

---

## Disposition

| Route | Live callers | Disposition |
|---|---|---|
| `POST /api/cas/gc` | none | **Admin** (`AdminAuthMiddleware`, on top of group auth) |
| `/api/cas/*` — tags, content, refs, import, export, batch, status, delta | `dojo` CLI | **Auth required** — see migration below |
| `GET /api/workflows`, `/api/skills`, `POST /api/workflows`, `PUT /:name/canvas`, `POST /:name/validate` | `dojo` CLI (skills); SPA never shipped | **Auth required** |
| `POST /api/workflows/:name/execute` | `dojo` CLI | **Auth required** + capability-gated (DGS-108) |
| `GET /api/workflows/:name/execution` | `dojo` CLI (manual SSE `GET`, carries a bearer) | **Auth required** |
| `GET /api/ws/workflow` | none | **Auth required** |
| `POST /api/ada/validate` | none | **Auth required** |
| `GET /events` | none | **Auth required** |
| `GET /health`, `/metrics` | deploy liveness probe | **Public, deliberately** — `deploy/provision.sh` hits `/health` unauthenticated |
| `/auth/login`, `/auth/register` | portal | **Public** — register is closed by config (DGS-101), not by auth |
| `/mesh/*`, `/.well-known/did.json` | federation | **Out of scope** — peers cannot hold a gateway JWT; needs DID-signature auth |
| `/api/telemetry/*` | a separate Cloudflare Worker | **Not this router** |

---

## Migration: the `dojo` CLI

The one accepted breakage. `cli/internal/client/client.go` attaches
`Authorization: Bearer <token>` only when `c.token` is non-empty, and the
default from `cli/internal/config` is empty. A CLI that has never been
configured gets **401** on `/api/cas/*`, `/api/skills` and the workflow routes.

The credential already exists — `svc:dojo-cli`, minted during the DGS-88
service-token work. Configure it **before this reaches a host the CLI talks to**:

```bash
export DOJO_GATEWAY_TOKEN="$(cat ~/.claude/.trespies-secrets/gateway-token-dojo-cli.token)"
```

or set `gateway.token` in `~/.dojo/settings.json`. Verify with any read:

```bash
dojo skill list
```

---

## When the workflow builder ships

`workflow-builder/src/lib/api.ts:92` subscribes to execution over `EventSource`:

```ts
const es = new EventSource(`${BASE}/api/workflows/${runId}/execution`);
```

A browser `EventSource` **cannot** set an `Authorization` header — not "does
not", cannot. The same is true of a browser `WebSocket`, which
`/api/ws/workflow` would need for the Phase 2 bridge. So a bearer token cannot
cover the SPA, and a `Secure; HttpOnly; SameSite` cookie is the only credential
that works across `fetch`, `EventSource` and WebSocket alike. Cookies
reintroduce CSRF, which needs scoping to the state-changing routes.

That work is not required to keep the gateway safe. It is required to ship the
SPA.

**Deliberately not done:** no "allow anonymous `/api`" escape hatch was added
for local SPA development. A switch that turns authentication off is the exact
shape of defect DGS-112 was. Local SPA work needs token plumbing in `api.ts`,
which it needs anyway.

---

## Remaining

1. **Deploy.** The running binary predates this change; the Caddy `/api/*` 403
   in `deploy/Caddyfile` is what protects the host until then.
2. **The edge block** can come off once this is deployed and the CLI is
   configured — it becomes redundant defence in depth rather than the only
   control. Removing it is a separate, deliberate step.
3. `/mesh/*` DID-signature auth, if federation grows.
