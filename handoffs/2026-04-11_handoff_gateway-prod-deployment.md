# Handoff: Gateway Production Deployment Orchestration
**From:** Sonnet (build session) | **To:** Opus | **Date:** 2026-04-11

## Objective

> Execute the five-phase production deployment of the Dojo Agentic Gateway stack to a Hetzner VPS with Named Cloudflare Tunnel, so that `bridge.trespiesdesign.com` is live, the Slack adapter is connected via Infisical, and every subsequent push to `main` auto-deploys via GitHub Actions.

---

## Required Context

### What Was Built This Session

All deployment artifacts are committed (or ready to commit) in `AgenticGatewayByDojoGenesis/`. Nothing in this repo needs to be written — your job is orchestration, not implementation.

**Deployment artifacts created:**

| File | Purpose |
|---|---|
| `Dockerfile.bridge` | Builds `dojo-bridge` binary from `./cmd/dojo/` |
| `deployments/prod/docker-compose.yml` | Full prod stack definition |
| `deployments/prod/cloudflared/config.yaml` | Named tunnel config (`<TUNNEL-UUID>` placeholder) |
| `deployments/prod/cloudflared/.gitignore` | Excludes `*.json` credential files from git |
| `scripts/server-setup.sh` | One-time VPS bootstrap (Docker + cloudflared + crons) |
| `.github/workflows/deploy.yml` | CI/CD: test → build+push bridge image → SSH deploy |
| `Procfile` | Local dev: `overmind start` / `foreman start` |
| `scripts/slack-app-manifest.yaml` | Slack app manifest with permanent prod URL |

### Architecture Overview

```
Internet
  │
  ▼
bridge.trespiesdesign.com  (Cloudflare Named Tunnel — permanent URL)
  │
  ▼  cloudflared container
bridge:8090  (dojo-bridge — receives webhooks)
  │ fetches credentials from
  ▼
Infisical Cloud  (DOJO_CREDENTIAL_BACKEND=infisical)
  │
  ▼ (credential store provides SLACK_TOKEN, SLACK_SIGNINGSECRET, etc.)

gateway:7340  (agentic-gateway — LLM routing: Claude, Kimi, Gemini)
  └── localhost only, accessed via SSH tunnel for CLI use
```

### Tech Stack
- Go workspace (go.work), pure-Go SQLite (`modernc.org/sqlite`, `CGO_ENABLED=0`)
- Docker Compose profiles — `observability` profile opt-in for OTEL/Langfuse/Postgres
- GHCR registry: `ghcr.io/trespies-source/`
- Domain: `bridge.trespiesdesign.com`
- Deploy dir on VPS: `/opt/dojo/`
- Bridge port: `8090` (public via CF tunnel)
- Gateway port: `7340` (localhost only — SSH tunnel for remote CLI)

### Infisical Credential Configuration
- Project ID: `c261d0a2-1f97-422b-bc8f-92ccd9f6e3bf`
- Client ID: `309209bf-9966-4690-86e4-531bfb4c244e`
- Client Secret: **user must provide** (was rotated this session — not stored here)
- Environment: `prod`
- Secret path: `/channel`
- Secrets stored: `SLACK_TOKEN`, `SLACK_SIGNINGSECRET`

### Key Decisions Made
- Route 2 (Hetzner VPS + Named CF Tunnel) chosen over Fly.io and Cloudflare Workers. Rationale: gateway requires SQLite WAL, NATS, and MCP subprocess spawning — not possible in serverless environments.
- Bridge image published to GHCR (`ghcr.io/trespies-source/dojo-bridge`), not same image as gateway.
- No Infisical Go SDK dependency — `InfisicalHTTPClient` implements the `InfisicalClient` interface directly via REST (no new go.mod dependency).
- `DOJO_CREDENTIAL_BACKEND=infisical` env var switches bridge from env store to Infisical at runtime.

---

## Task Definition

Execute in order. Each phase gates the next.

### Phase 1 — Provision VPS (Hetzner Cloud)

1. Log into Hetzner Cloud console and create a new server:
   - Type: **CX21** (2 vCPU, 4GB RAM, 40GB SSD) — sufficient for solo operator
   - OS: **Ubuntu 22.04**
   - Location: Nuremberg or Helsinki (lowest latency to Cloudflare EU PoPs)
   - Add SSH key: generate a fresh `ed25519` key pair for deploy use
     ```sh
     ssh-keygen -t ed25519 -C "dojo-deploy" -f ~/.ssh/dojo_deploy_ed25519
     ```
   - Note the VPS IP address — you'll need it as `HETZNER_HOST`

2. Add the deploy public key to the server's authorized_keys during creation.

3. Test SSH access:
   ```sh
   ssh -i ~/.ssh/dojo_deploy_ed25519 root@<VPS-IP> 'echo ok'
   ```

### Phase 2 — Bootstrap VPS

Run the server setup script once:
```sh
ssh root@<VPS-IP> 'bash -s' < AgenticGatewayByDojoGenesis/scripts/server-setup.sh
```

This installs Docker, cloudflared, creates `/opt/dojo/`, and sets up backup + nightly pull crons.

### Phase 3 — Create Named Cloudflare Tunnel (local machine)

Run these on your **local** machine (not the VPS):
```sh
cloudflared tunnel login
cloudflared tunnel create dojo-bridge
cloudflared tunnel route dns dojo-bridge bridge.trespiesdesign.com
TUNNEL_UUID=$(cloudflared tunnel list | grep dojo-bridge | awk '{print $1}')
echo "Tunnel UUID: $TUNNEL_UUID"
```

Copy tunnel credentials to VPS:
```sh
scp ~/.cloudflared/${TUNNEL_UUID}.json root@<VPS-IP>:/opt/dojo/cloudflared/${TUNNEL_UUID}.json
```

Update `deployments/prod/cloudflared/config.yaml` — replace both `<TUNNEL-UUID>` occurrences with `$TUNNEL_UUID`.

### Phase 4 — Deploy Files to VPS

```sh
scp AgenticGatewayByDojoGenesis/deployments/prod/docker-compose.yml root@<VPS-IP>:/opt/dojo/docker-compose.yml
scp AgenticGatewayByDojoGenesis/deployments/prod/cloudflared/config.yaml root@<VPS-IP>:/opt/dojo/cloudflared/config.yaml
scp AgenticGatewayByDojoGenesis/gateway-config.yaml root@<VPS-IP>:/opt/dojo/gateway-config.yaml
scp AgenticGatewayByDojoGenesis/otel-config.yaml root@<VPS-IP>:/opt/dojo/otel-config.yaml
```

Write `.env` on VPS (**user must supply actual `DOJO_INFISICAL_CLIENT_SECRET`**):
```sh
ssh root@<VPS-IP> 'cat > /opt/dojo/.env' << 'EOF'
export DOJO_CREDENTIAL_BACKEND=infisical
export DOJO_INFISICAL_PROJECT_ID=c261d0a2-1f97-422b-bc8f-92ccd9f6e3bf
export DOJO_INFISICAL_CLIENT_ID=309209bf-9966-4690-86e4-531bfb4c244e
export DOJO_INFISICAL_CLIENT_SECRET=<ROTATED-VALUE-FROM-USER>
EOF
```

### Phase 5 — Add GitHub Secrets

In the `DojoGenesis/gateway` repo (Settings → Secrets and variables → Actions), add:

| Secret name | Value |
|---|---|
| `GHCR_TOKEN` | PAT from **TresPies-source** GitHub account, with `write:packages` scope — create at github.com/settings/tokens |
| `HETZNER_HOST` | VPS IP address |
| `HETZNER_SSH_KEY` | Contents of `~/.ssh/dojo_deploy_ed25519` (private key) |

To get the GHCR_TOKEN: user must go to github.com → TresPies-source account → Settings → Developer settings → Personal access tokens → Tokens (classic) → New token → check `write:packages` + `read:packages`.

Note: `INFISICAL_CLIENT_ID/SECRET/PROJECT_ID` are NOT GitHub secrets — they live only in `/opt/dojo/.env` on the VPS (never in CI).

### Phase 6 — First Manual Deploy

```sh
ssh root@<VPS-IP> 'cd /opt/dojo && source .env && docker compose pull && docker compose up -d'
```

Verify the stack is healthy:
```sh
ssh root@<VPS-IP> 'cd /opt/dojo && docker compose ps'
ssh root@<VPS-IP> 'docker compose -f /opt/dojo/docker-compose.yml logs bridge --tail 30'
```

Expected: bridge logs show `credential_backend=infisical` and `registered adapter slack`.

### Phase 7 — Reconnect Slack App

The Slack app's Request URL still points to the old `trycloudflare.com` dev URL. Update it:

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → Dojo Gateway app
2. Event Subscriptions → Request URL → change to:
   ```
   https://bridge.trespiesdesign.com/webhook/slack/events
   ```
3. Slack will send a `url_verification` challenge — the bridge handles this automatically (challenge bypass is already implemented in `channel/gateway.go`)
4. Confirm "Verified" green checkmark appears

Test end-to-end: send a DM to the Dojo bot in Slack, confirm the bridge logs show the event.

### Phase 8 — Verify CI/CD Auto-Deploy

1. Make a trivial commit to `main` in `DojoGenesis/gateway`
2. Watch `.github/workflows/deploy.yml` run in GitHub Actions
3. Confirm: test passes → bridge image builds + pushed → SSH deploy runs
4. Confirm `docker compose ps` on VPS shows updated containers

---

## Definition of Done

- [ ] `ssh root@<VPS-IP> 'docker compose -f /opt/dojo/docker-compose.yml ps'` shows `bridge`, `tunnel`, `gateway` all `Up`
- [ ] `curl -sf https://bridge.trespiesdesign.com/health` returns 200
- [ ] Bridge logs show `credential_backend=infisical` and `registered adapter slack`
- [ ] Slack DM to Dojo bot produces a log entry in bridge (event received)
- [ ] GitHub Actions `deploy.yml` completes successfully on a push to `main`
- [ ] `ghcr.io/trespies-source/dojo-bridge:latest` visible in GitHub Packages under TresPies-source

---

## Constraints & Boundaries

- **DO NOT** modify any Go source files — the implementation is complete and committed
- **DO NOT** commit `/opt/dojo/.env` or the Cloudflare tunnel `.json` credential file to git
- **DO NOT** expose gateway port 7340 publicly — it must remain `127.0.0.1:7340` (SSH tunnel only)
- **MUST** use `gh auth switch --user DojoGenesis` before any `git push` or `gh` commands against the gateway repo
- **MUST** use `export KEY=value` syntax (not bare `KEY=value`) in any `.env` file — bare assignment does not export to child processes
- The `observability` Docker Compose profile (OTEL/Langfuse/Postgres) is **opt-in** — do not start it unless the user explicitly asks

---

## Key File Paths

All relative to `AgenticGatewayByDojoGenesis/` (absolute: `/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis/`):

- `Dockerfile.bridge` — bridge image build
- `deployments/prod/docker-compose.yml` — full stack
- `deployments/prod/cloudflared/config.yaml` — tunnel config (update `<TUNNEL-UUID>`)
- `scripts/server-setup.sh` — VPS bootstrap
- `.github/workflows/deploy.yml` — CI/CD pipeline
- `scripts/slack-app-manifest.yaml` — Slack app config (prod URL already set)
- `channel/infisical_http_client.go` — Infisical REST adapter (no SDK)
- `cmd/dojo/bridge.go` — bridge entrypoint, credential store wiring

---

## Next Steps After Completion

- Notify user that `bridge.trespiesdesign.com` is live and auto-deploy is active
- The gateway (`localhost:7340` on VPS) is accessible via `ssh -L 7340:localhost:7340 root@<VPS-IP>` for remote CLI use
- Optional next: enable observability profile (`docker compose --profile observability up -d`) for Langfuse + OTEL tracing
- Optional next: add Discord adapter (credentials: `DISCORD_TOKEN`, `DISCORD_APP_ID` in Infisical `/channel` path)
