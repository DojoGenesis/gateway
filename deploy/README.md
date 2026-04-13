# Dojo Gateway — VPS Deployment Guide

Target: Hetzner CPX21, Ubuntu 24.04 LTS, `gateway.trespies.dev`

## 1. Prerequisites

- Hetzner CPX21 (2 vCPU, 4 GB RAM, 40 GB NVMe) with Ubuntu 24.04 LTS
- SSH root or sudo access to the server
- DNS A record: `gateway.trespies.dev` → server public IP (must propagate before running the script)

## 2. Create a GitHub OAuth App

1. Go to GitHub → Settings → Developer settings → OAuth Apps → New OAuth App
2. Set **Homepage URL**: `https://gateway.trespies.dev`
3. Set **Authorization callback URL**: `https://gateway.trespies.dev/auth/github/callback`
4. Note the **Client ID** and generate a **Client Secret**

## 3. Set environment variables in /etc/dojo/env

After the first run of `provision.sh`, create `/etc/dojo/env` (the file is created empty by the script):

```
DOJO_JWT_SECRET=<random 64-char hex string>
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
GITHUB_OAUTH_CLIENT_ID=<from step 2>
GITHUB_OAUTH_CLIENT_SECRET=<from step 2>
```

Generate a strong JWT secret:
```bash
openssl rand -hex 32
```

Protect the file (the script sets this, but verify):
```bash
chmod 640 /etc/dojo/env
chown root:dojo /etc/dojo/env
```

## 4. Dry run first

Upload the `deploy/` directory to the server (or clone the repo), then:

```bash
sudo bash deploy/provision.sh --dry-run
```

Review the output. Every step is printed as "[DRY RUN] Would execute: ..." so you can verify what will happen before committing.

## 5. Run the provisioner

```bash
sudo bash deploy/provision.sh
```

The script is idempotent — re-running it is safe. It skips:
- User creation if `dojo` already exists
- Directory creation if the path already exists
- Binary download if the installed version matches `GATEWAY_VERSION`

## 6. Verify the deployment

```bash
curl https://gateway.trespies.dev/health
```

Expected response: HTTP 200 with a JSON body indicating service status.

Check service logs if something is wrong:
```bash
journalctl -u dojo-gateway -f
journalctl -u caddy -f
```

## 7. Create the first admin user

Connect to the server and use the CLI or a direct API call:

```bash
dojo-gateway create-admin --email admin@example.com
```

Or via API (replace with actual admin-creation endpoint):

```bash
curl -X POST https://gateway.trespies.dev/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","role":"admin"}'
```

## Files in this directory

| File | Purpose |
|---|---|
| `provision.sh` | Idempotent provisioning script |
| `Caddyfile` | Caddy reverse proxy config (TLS + security headers) |
| `gateway.service` | systemd unit for the Gateway process |
| `gateway-config.yaml` | Template Gateway config (env var placeholders) |
| `README.md` | This file |

## Notes

- The Gateway listens on `localhost:7340`. Caddy terminates TLS and reverse-proxies to it.
- TLS certificates are managed automatically by Caddy via Let's Encrypt. Port 80 and 443 must be open in the Hetzner firewall.
- The `dojo` system user runs the Gateway process. It has no login shell and no home directory write access beyond `/var/lib/dojo`.
- To update the binary: change `GATEWAY_VERSION` in `provision.sh` and re-run. The version check will detect the mismatch and re-download.
- The release binary URL in `provision.sh` contains a `TODO` comment — verify the exact goreleaser artifact path before the first production run.
