# Deployment Guide

**Release line:** `v3.x` — latest tag `v3.2.2`. Production (`gateway.trespies.dev`, Hetzner) runs `v3.2.2+dynmodels`.
**Runtime:** a single Go binary (`agentic-gateway`) listening on **port 7340** by default.

See [`README.md`](./README.md) for the architecture overview and [`ARCHITECTURE.md`](./ARCHITECTURE.md) for system design.

---

## Quick Start

### Option 1: Build from source (local development)

```bash
# 1. Clone and enter the repo
git clone https://github.com/DojoGenesis/gateway.git
cd gateway

# 2. Configure — copy the example env and add provider API keys
cp .env.example .env

# 3. Build the binary (outputs to bin/agentic-gateway)
make build

# 4. Run it
./bin/agentic-gateway            # listening on :7340
```

Or run without building: `go run main.go`.

Verify it is live:

```bash
curl http://localhost:7340/health
```

### Option 2: Docker

```bash
# Build the image (multi-stage: Go 1.25 Alpine builder → distroless runtime, non-root UID 65534)
docker build -t agentic-gateway .

# Run it
docker run -p 7340:7340 --env-file .env agentic-gateway
```

The image `EXPOSE`s 7340 and ships a self-contained health probe (`agentic-gateway --health-check`) so no `curl`/`wget` is needed inside the distroless container.

### Option 3: Production VPS

For a TLS-terminated production deployment on Hetzner (Caddy + systemd), see
[**VPS Production Deployment**](#vps-production-deployment) below and the dedicated guide at
[`deploy/README.md`](./deploy/README.md).

---

## Configuration

Configuration is layered, lowest precedence first:

1. `.env` — loaded on startup if present (existing environment variables are **not** overridden)
2. `gateway-config.yaml` (or the path passed via `-config` / `MCP_CONFIG_PATH`)
3. Process environment variables — highest precedence

### Provider API keys

Set the corresponding key in `.env` to enable a provider. Providers without a key are **silently skipped at startup** — no error, they just do not register.

| Provider | Env var |
|----------|---------|
| Anthropic (Claude) | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` (+ optional `OPENAI_BASE_URL`) |
| Google (Gemini) | `GOOGLE_API_KEY` |
| Groq | `GROQ_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| DeepSeek | `DEEPSEEK_API_KEY` (+ optional `DEEPSEEK_BASE_URL`) |
| Kimi (Moonshot) | `KIMI_API_KEY` (+ optional `KIMI_BASE_URL`) |
| Ollama | `OLLAMA_HOST` (auto-detected on `localhost:11434`) |

The full set of variables is documented in [`.env.example`](./.env.example).

### Common environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `7340` | HTTP listen port |
| `ENVIRONMENT` | `development` | `production` enables structured/JSON logging |
| `ALLOWED_ORIGINS` | (none) | Comma-separated CORS origins (each must include scheme) |
| `MEMORY_DB_PATH` | `~/.dojo/memory.db` | Conversation-memory SQLite path — **use an absolute path** |
| `DOJO_CAS_PATH` | `~/.dojo/skills.db` | Content-addressable skill/workflow store |
| `AUTH_DB_DIR` | `.dojo/` (CWD-relative) | Auth DB directory — set absolute for deploys |
| `MCP_CONFIG_PATH` | `gateway-config.yaml` | MCP host configuration file |
| `MCP_APPS_ENABLED` | `false` | Enable the MCP Apps bridge |

> **Relative DB paths bite.** A relative `MEMORY_DB_PATH` or `DOJO_CAS_PATH` silently creates a new empty database whenever the working directory changes (e.g. across restarts). Always use absolute paths in any non-local deployment.

### YAML config

`gateway-config.yaml` controls runtime behaviour (feature flags, MCP servers, routing). Example feature block:

```yaml
features:
  tool_calling: true            # agentic tool-calling loop
  get_document_tool: true       # document fetch endpoint
  patch_intent: true            # extract patch intents from responses
  provider_key_management: true # accept API keys via settings endpoint
  ollama_tool_fallback: true    # text-mode fallback for Ollama
```

### Observability (optional)

The gateway supports OpenTelemetry trace export and Langfuse:

```
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_SERVICE_NAME=agentic-gateway
```

Use [`docker-compose.example.yml`](./docker-compose.example.yml) for a full local observability stack (gateway + OTEL Collector + Langfuse + PostgreSQL).

---

## Health & Metrics

### Health check

```bash
curl http://localhost:7340/health
```

Example response:

```json
{
  "status": "healthy",
  "version": "1.1.0",
  "timestamp": "2026-07-19T12:00:00Z",
  "providers": { "anthropic": "healthy" },
  "dependencies": {
    "memory_store": "healthy",
    "tool_registry": "healthy",
    "orchestration_engine": "healthy"
  },
  "uptime_seconds": 0,
  "requests_processed": 0
}
```

`status` is `degraded` if any registered provider fails its info probe. The `version` field is the `server` module version injected at build time via
`-ldflags "-X github.com/DojoGenesis/gateway/server.Version=<version>"`; an un-injected build reports the source default (`1.1.0`).

### Metrics

```bash
curl http://localhost:7340/metrics                 # Prometheus-style, unauthenticated
curl http://localhost:7340/admin/metrics/prometheus # admin surface (requires admin auth)
```

### Version scheme

Releases are cut as git tags `vMAJOR.MINOR.PATCH` (latest: `v3.2.2`). A running build may carry semver build metadata after a `+` — e.g. production runs `v3.2.2+dynmodels`.

---

## VPS Production Deployment

Production runs on Hetzner behind Caddy (automatic Let's Encrypt TLS) with the Gateway managed by systemd. The full step-by-step guide, including the GitHub OAuth app setup and secret placement, is in [`deploy/README.md`](./deploy/README.md). Summary:

```bash
# On the server (Ubuntu 24.04), from the repo's deploy/ directory:
sudo bash deploy/provision.sh --dry-run   # preview every step
sudo bash deploy/provision.sh             # idempotent — safe to re-run
```

`provision.sh` installs Caddy, creates the unprivileged `dojo` system user, downloads the release binary to `/usr/local/bin/dojo-gateway`, installs the config + systemd unit, and starts the services.

**Layout on the server:**

| Path | Purpose |
|------|---------|
| `/usr/local/bin/dojo-gateway` | The Gateway binary (release tarball `agentic-gateway_<version>_linux_amd64.tar.gz`) |
| `/etc/dojo/config.yaml` | Gateway config ([`deploy/gateway-config.yaml`](./deploy/gateway-config.yaml) template) |
| `/etc/dojo/env` | Secrets — `DOJO_JWT_SECRET`, provider keys, GitHub OAuth creds (`chmod 640`, `root:dojo`) |
| `/var/lib/dojo` | Data dir (memory + CAS SQLite) |
| `/etc/caddy/Caddyfile` | TLS reverse proxy → `localhost:7340` ([`deploy/Caddyfile`](./deploy/Caddyfile)) |
| `dojo-gateway.service` | systemd unit ([`deploy/gateway.service`](./deploy/gateway.service)) |

**Service management:**

```bash
systemctl status dojo-gateway
journalctl -u dojo-gateway -f       # Gateway logs
journalctl -u caddy -f              # TLS / proxy logs
curl https://gateway.trespies.dev/health
```

> **Version pin:** `deploy/provision.sh` sets `GATEWAY_VERSION` and only re-downloads when the installed binary's version differs. Bump that variable and re-run to upgrade.

---

## Production Considerations

### Security

- Generate a strong JWT secret: `openssl rand -hex 32` → `DOJO_JWT_SECRET` in `/etc/dojo/env`.
- Keep `registration_enabled: false` in production config; create the first admin out-of-band.
- The Docker image already runs as non-root (UID 65534) on a distroless base; the systemd unit adds `NoNewPrivileges`, `ProtectSystem=strict`, and `ProtectHome`.
- Caddy sets HSTS, `X-Content-Type-Options`, `X-Frame-Options`, and `Referrer-Policy`; keep only ports 80/443 open in the firewall.

### Secrets management

Store provider keys and the JWT secret in `/etc/dojo/env` (systemd `EnvironmentFile`), not in the tracked YAML. For orchestrated deployments use the platform's secret store (Kubernetes secrets, cloud secret managers) rather than baking keys into images.

---

## Troubleshooting

### A provider is missing from `/health`

Providers without an API key are skipped silently at startup. Confirm the key is present in the process environment (`.env` is only read if it exists in the working directory) and re-check `/health` → `providers`.

### Port already in use

The default port is `7340`. Override with `PORT=<port>` (the `--health-check` probe reads the same variable, so it follows the override).

### TLS / 502 from Caddy

Caddy reverse-proxies to `localhost:7340`. If Caddy returns 502, the Gateway process is down or bound to a different port — check `journalctl -u dojo-gateway` and confirm the `PORT`/config match the Caddyfile upstream.

### Memory or skills "reset" after a restart

Almost always a relative DB path. Set `MEMORY_DB_PATH` and `DOJO_CAS_PATH` to absolute paths (the systemd unit already points them at `/var/lib/dojo`).

---

## References

- [`README.md`](./README.md) — architecture, providers, API routes, key commands
- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — system design deep-dive
- [`deploy/README.md`](./deploy/README.md) — full VPS provisioning walkthrough
- [`.env.example`](./.env.example) — every environment variable, documented
- [`CHANGELOG.md`](./CHANGELOG.md) — release history

---

**Questions?** Open an issue on GitHub or consult the documentation above.
