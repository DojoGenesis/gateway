#!/bin/bash
set -euo pipefail

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

DOMAIN="gateway.trespies.dev"
GATEWAY_VERSION="v3.0.0"
GATEWAY_PORT=7340
GATEWAY_USER="dojo"
GATEWAY_HOME="/opt/dojo"
CONFIG_DIR="/etc/dojo"
DATA_DIR="/var/lib/dojo"
BINARY_PATH="/usr/local/bin/dojo-gateway"
SERVICE_NAME="dojo-gateway"

# TODO: Update URL to match actual goreleaser output path
RELEASE_URL="https://github.com/DojoGenesis/gateway/releases/download/${GATEWAY_VERSION}/gateway_linux_amd64"

log() { echo "[$(date '+%H:%M:%S')] $*"; }

run() {
    if $DRY_RUN; then
        log "[DRY RUN] Would execute: $*"
    else
        log "Executing: $*"
        eval "$@"
    fi
}

# ---------------------------------------------------------------------------
# Preflight checks
# ---------------------------------------------------------------------------
if [[ $EUID -ne 0 ]] && ! $DRY_RUN; then
    echo "ERROR: This script must be run as root (or with sudo)." >&2
    exit 1
fi

log "Starting Gateway provisioning (DRY_RUN=${DRY_RUN})"
log "Domain:  ${DOMAIN}"
log "Version: ${GATEWAY_VERSION}"
log "Port:    ${GATEWAY_PORT}"

# ---------------------------------------------------------------------------
# Phase 1: System packages
# ---------------------------------------------------------------------------
log "=== Phase 1: System packages ==="
run "apt-get update -qq"
run "apt-get install -y caddy sqlite3 jq curl"

# ---------------------------------------------------------------------------
# Phase 2: Create dojo user (idempotent)
# ---------------------------------------------------------------------------
log "=== Phase 2: Create system user ==="
if id "${GATEWAY_USER}" &>/dev/null; then
    log "User '${GATEWAY_USER}' already exists — skipping creation"
else
    run "useradd -r -s /sbin/nologin -d '${GATEWAY_HOME}' -m '${GATEWAY_USER}'"
fi

# ---------------------------------------------------------------------------
# Phase 3: Create directories
# ---------------------------------------------------------------------------
log "=== Phase 3: Create directories ==="
for dir in "${CONFIG_DIR}" "${DATA_DIR}"; do
    if [[ -d "${dir}" ]]; then
        log "Directory ${dir} already exists — skipping"
    else
        run "mkdir -p '${dir}'"
    fi
done

# ---------------------------------------------------------------------------
# Phase 4: Download gateway binary (skip if current version already installed)
# ---------------------------------------------------------------------------
log "=== Phase 4: Download gateway binary ==="

NEEDS_DOWNLOAD=true
if [[ -x "${BINARY_PATH}" ]]; then
    INSTALLED_VERSION=$("${BINARY_PATH}" --version 2>/dev/null | awk '{print $NF}' || true)
    if [[ "${INSTALLED_VERSION}" == "${GATEWAY_VERSION}" ]]; then
        log "Binary ${BINARY_PATH} is already at ${GATEWAY_VERSION} — skipping download"
        NEEDS_DOWNLOAD=false
    else
        log "Binary exists but version mismatch (installed: '${INSTALLED_VERSION}', want: '${GATEWAY_VERSION}') — re-downloading"
    fi
fi

if $NEEDS_DOWNLOAD; then
    run "curl -fSL '${RELEASE_URL}' -o '${BINARY_PATH}'"
    run "chmod 755 '${BINARY_PATH}'"
fi

# ---------------------------------------------------------------------------
# Phase 5: Install config files
# ---------------------------------------------------------------------------
log "=== Phase 5: Install config files ==="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

run "cp '${SCRIPT_DIR}/Caddyfile' /etc/caddy/Caddyfile"
run "cp '${SCRIPT_DIR}/gateway-config.yaml' '${CONFIG_DIR}/config.yaml'"
run "cp '${SCRIPT_DIR}/gateway.service' /etc/systemd/system/${SERVICE_NAME}.service"

# ---------------------------------------------------------------------------
# Phase 6: Set permissions
# ---------------------------------------------------------------------------
log "=== Phase 6: Set permissions ==="
run "chown -R '${GATEWAY_USER}:${GATEWAY_USER}' '${DATA_DIR}'"
run "chmod 750 '${DATA_DIR}'"
run "chown root:'${GATEWAY_USER}' '${CONFIG_DIR}'"
run "chmod 750 '${CONFIG_DIR}'"
run "chown root:'${GATEWAY_USER}' '${CONFIG_DIR}/config.yaml'"
run "chmod 640 '${CONFIG_DIR}/config.yaml'"

# Ensure /etc/dojo/env exists (EnvironmentFile is non-fatal if absent, but create
# it empty so operators know where to place secrets)
if [[ ! -f "${CONFIG_DIR}/env" ]]; then
    run "touch '${CONFIG_DIR}/env'"
    run "chown root:'${GATEWAY_USER}' '${CONFIG_DIR}/env'"
    run "chmod 640 '${CONFIG_DIR}/env'"
    log "Created empty ${CONFIG_DIR}/env — populate with secrets before starting the service"
fi

# ---------------------------------------------------------------------------
# Phase 7: Enable and start services
# ---------------------------------------------------------------------------
log "=== Phase 7: Enable and start services ==="
run "systemctl daemon-reload"
run "systemctl enable --now '${SERVICE_NAME}'"
run "systemctl enable --now caddy"

# ---------------------------------------------------------------------------
# Phase 8: Health check
# ---------------------------------------------------------------------------
log "=== Phase 8: Health check ==="
if $DRY_RUN; then
    log "[DRY RUN] Would check: curl -sf https://${DOMAIN}/health"
else
    # Allow up to 15 seconds for services to come up
    sleep 3
    if curl -sf "https://${DOMAIN}/health" > /dev/null; then
        log "Gateway is live! https://${DOMAIN}/health returned 200"
    else
        log "WARNING: health check failed — check 'journalctl -u ${SERVICE_NAME}' and 'journalctl -u caddy'"
        exit 1
    fi
fi

log "Provisioning complete."
