# `/api/*` route disposition

**Status:** partial — the SEV-1 primitive is closed, the authentication story is not.
**Issues:** DGS-108 (RCE, closed), DGS-100 (auth, open), DGS-101 (registration, open), DGS-103 Track 0.
**Last verified against the code:** 2026-08-04.

This table exists because `/api/*` is currently blocked wholesale at the edge, and
nobody should have to re-derive the caller inventory to decide when that block can
come off. It is the input to the DGS-100 auth design, not the design itself.

---

## Why the edge block is still up

Two independent defects share the `/api/*` prefix. Only one is fixed.

| | Defect | Layer | Status |
|---|---|---|---|
| **A** | A workflow step could execute a shell command as the gateway process | capability | **closed** — `WORKFLOW_RUN_COMMAND_ENABLED`, off in production |
| **B** | `/api/*` has no authentication at all | authn/authz | **open** — needs the design below |

Fixing B would not have fixed A: `POST /auth/register` is open (DGS-101), so an
anonymous caller can mint a `role=user` token and re-enter any authenticated route.
Fixing A does not fix B: the routes still serve anyone on the internet.

**The edge block comes off when B is resolved.** Until then, `deploy/Caddyfile`
returns 403 for `/api/*` on `gateway.trespies.dev`.

---

## The two-handler-path trap

`/api/*` is served by **two independent registration paths** that share no
middleware. Any fix must cover both, and a grep of one proves nothing about the
other.

| Path | Where | Middleware it can carry |
|---|---|---|
| 1 — Gin | `server/router.go` `setupRoutes()`, on the root `s.router` | Gin middleware, but none is attached today |
| 2 — ServeMux | `workflow/api/handler.go` `RegisterRoutes`, mounted via `gin.WrapH` | **none** — a bare `http.ServeMux` |

The global `OptionalAuthMiddleware` never rejects; it assigns a guest UUID and
calls `c.Next()`. Blocking happens only where a group explicitly adds
`AuthMiddleware()`. `/v1/*` does. Nothing registered directly on `s.router` does.

---

## Caller inventory

Verified by reading every call site in this repo and in `cli/`. The decisive
column is the third one: **no browser caller sends a token today.**

| Route | Live callers | Can it carry `Authorization`? | Disposition |
|---|---|---|---|
| `POST /api/cas/gc` | none | n/a | **Done** — `AdminAuthMiddleware` (DGS-100) |
| `GET/POST /api/cas/tags`, `POST /api/cas/content`, `GET /api/cas/refs`, `/status`, `/delta`, `PUT /batch` | `dojo` CLI (`cmd_skill.go`) | Yes, but token is empty by default | **Blocked** — needs CLI token rollout first |
| `GET /api/workflows`, `GET /api/skills`, `POST /api/workflows`, `PUT /:name/canvas`, `POST /:name/validate` | workflow-builder SPA (browser `fetch`) | **No** — `api.ts` sets only `Content-Type` | **Blocked** — needs SPA auth |
| `POST /api/workflows/:name/execute` | SPA + `dojo` CLI | Mixed | **Blocked** — SPA leg |
| `GET /api/workflows/:runId/execution` | SPA via **`EventSource`** + CLI via manual SSE `GET` | **Structurally impossible** for the SPA | **Blocked** — decides the whole design |
| `GET /api/ws/workflow` | none (WebSocket; SPA support unimplemented) | No (browser WS can't set headers) | **Free win** — gate now, zero blast radius |
| `POST /api/ada/validate` | none — zero references repo-wide, no test file | n/a | **Free win** — gate now, zero blast radius |
| `POST/GET /api/telemetry/*` | separate Cloudflare Worker | — | **Out of scope** — not this router |

### The constraint that decides the design

`workflow-builder/src/lib/api.ts:92`:

```ts
const es = new EventSource(`${BASE}/api/workflows/${encodeURIComponent(runId)}/execution`);
```

A browser `EventSource` **cannot** set an `Authorization` header. Not "does not" —
cannot. So bearer-token auth cannot cover the SPA's execution stream, and any
design that assumes it will break the workflow builder the day it ships.

---

## What a resolution needs

1. **A credential the browser can send.** A `Secure; HttpOnly; SameSite` session
   cookie is the only mechanism that works for `fetch`, `EventSource` and
   WebSocket alike. Query-param tokens are the alternative and are worse: they
   land in access logs and referrers.
2. **CSRF protection**, which cookies reintroduce and bearer tokens did not have
   to think about. Scope it to the state-changing routes.
3. **A CLI token that exists by default.** Gating `/api/cas/*` breaks every `dojo`
   user whose `~/.dojo/settings.json` has no `gateway.token` — which is the
   default today. Sequence the rollout, don't just attach middleware.
4. **Coverage of both handler paths.** Path 2 is a bare `ServeMux`; it cannot take
   Gin middleware, so it needs either a wrapping handler or migration onto Gin.
5. **`/auth/register` closed** (DGS-101), or authentication on `/api/*` only
   raises the cost of entry by one HTTP request.

### Not recommended

A warn-only `STRICT_API_AUTH` ramp, as sketched in the v1.2.0 spec. A warn-only
mode on a route surface that is *already* blocked at the edge buys no migration
safety — it just extends the window in which the code is wrong. Land the cookie
design behind the edge block, verify against the real SPA over the tunnel, then
lift the block.

---

## Free wins available now

`/api/ada/validate` and `/api/ws/workflow` have **zero callers**. Neither is part
of the SPA auth problem, and gating them cannot break anything. They are held back
here only to keep the DGS-108 change reviewable as one idea; they should land as
their own small change rather than waiting on the full design.
