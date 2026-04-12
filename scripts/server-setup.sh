#!/usr/bin/env bash
# server-setup.sh — One-time bootstrap for the Hetzner VPS.
# Run this once after provisioning the server:
#   ssh root@<VPS-IP> 'bash -s' < scripts/server-setup.sh
set -euo pipefail

DEPLOY_DIR="/opt/dojo"
GHCR_REGISTRY="ghcr.io"

echo "==> Installing Docker..."
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update -qq
apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
systemctl enable --now docker
echo "    Docker $(docker --version)"

echo "==> Installing cloudflared..."
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | gpg --dearmor -o /usr/share/keyrings/cloudflare-main.gpg
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(. /etc/os-release && echo $VERSION_CODENAME) main" \
  > /etc/apt/sources.list.d/cloudflared.list
apt-get update -qq
apt-get install -y -qq cloudflared
echo "    cloudflared $(cloudflared --version)"

echo "==> Creating deploy directory: $DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR/cloudflared"
mkdir -p "$DEPLOY_DIR/data"

echo "==> Setting up daily SQLite backup cron..."
cat > /etc/cron.daily/dojo-backup << 'EOF'
#!/bin/sh
BACKUP_DIR="/opt/dojo/backups/$(date +%Y-%m-%d)"
mkdir -p "$BACKUP_DIR"
for db in memory skills auth; do
  src="/opt/dojo/data/dojo_${db}.db"
  [ -f "$src" ] && sqlite3 "$src" ".backup $BACKUP_DIR/dojo_${db}.db"
done
# Keep 30 days
find /opt/dojo/backups -maxdepth 1 -type d -mtime +30 -exec rm -rf {} +
EOF
chmod +x /etc/cron.daily/dojo-backup

echo "==> Setting up automatic Docker image updates (nightly pull + restart)..."
cat > /etc/cron.d/dojo-pull << 'EOF'
0 3 * * * root cd /opt/dojo && docker compose pull --quiet && docker compose up -d --remove-orphans 2>&1 | logger -t dojo-deploy
EOF

echo ""
echo "==> Server ready. Next steps:"
echo ""
echo "  1. On your LOCAL machine, create the Cloudflare tunnel:"
echo "       cloudflared tunnel login"
echo "       cloudflared tunnel create dojo-bridge"
echo "       cloudflared tunnel route dns dojo-bridge bridge.trespiesdesign.com"
echo ""
echo "  2. Copy tunnel credentials to VPS:"
echo "       TUNNEL_UUID=\$(cloudflared tunnel list | grep dojo-bridge | awk '{print \$1}')"
echo "       scp ~/.cloudflared/\$TUNNEL_UUID.json root@<VPS-IP>:/opt/dojo/cloudflared/\$TUNNEL_UUID.json"
echo ""
echo "  3. Update deployments/prod/cloudflared/config.yaml — replace <TUNNEL-UUID> with \$TUNNEL_UUID"
echo ""
echo "  4. Copy deploy files to VPS:"
echo "       scp deployments/prod/docker-compose.yml root@<VPS-IP>:/opt/dojo/docker-compose.yml"
echo "       scp deployments/prod/cloudflared/config.yaml root@<VPS-IP>:/opt/dojo/cloudflared/config.yaml"
echo "       scp gateway-config.yaml root@<VPS-IP>:/opt/dojo/gateway-config.yaml"
echo "       scp otel-config.yaml root@<VPS-IP>:/opt/dojo/otel-config.yaml"
echo ""
echo "  5. Write .env on VPS (use actual values):"
echo "       ssh root@<VPS-IP> 'cat > /opt/dojo/.env << EOF"
echo "       export DOJO_CREDENTIAL_BACKEND=infisical"
echo "       export DOJO_INFISICAL_PROJECT_ID=<your-project-id>"
echo "       export DOJO_INFISICAL_CLIENT_ID=<your-client-id>"
echo "       export DOJO_INFISICAL_CLIENT_SECRET=<your-client-secret>"
echo "       EOF'"
echo ""
echo "  6. Start the stack:"
echo "       ssh root@<VPS-IP> 'cd /opt/dojo && source .env && docker compose pull && docker compose up -d'"
echo ""
echo "  7. Add GitHub secrets for the deploy workflow (see .github/workflows/deploy.yml):"
echo "       HETZNER_HOST — VPS IP address"
echo "       HETZNER_SSH_KEY — private SSH key (ssh-keygen -t ed25519)"
echo "       INFISICAL_CLIENT_ID"
echo "       INFISICAL_CLIENT_SECRET"
echo "       INFISICAL_PROJECT_ID"
