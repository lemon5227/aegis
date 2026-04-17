#!/usr/bin/env bash
set -euo pipefail

REPO="lemon5227/aegis"
BINARY_NAME="aegis-relay"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/aegis-relay"
SERVICE_NAME="aegis-relay"
DEFAULT_PORT=40100
DEFAULT_HTTP_PORT=40101

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

UNINSTALL=false
PORT=""
HTTP_PORT=""
PUBLIC_IP=""
ANNOUNCE_ADDRS=""
DB_PATH=""
TRUSTED_ADMINS=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --uninstall)    UNINSTALL=true; shift ;;
        --port)         PORT="$2"; shift 2 ;;
        --http-port)    HTTP_PORT="$2"; shift 2 ;;
        --public-ip)    PUBLIC_IP="$2"; shift 2 ;;
        --announce)     ANNOUNCE_ADDRS="$2"; shift 2 ;;
        --db-path)      DB_PATH="$2"; shift 2 ;;
        --trusted-admins) TRUSTED_ADMINS="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --port PORT          P2P listen port (default: ${DEFAULT_PORT})"
            echo "  --http-port PORT     HTTP monitoring port (default: ${DEFAULT_HTTP_PORT})"
            echo "  --public-ip IP       Public IP for announce addresses"
            echo "  --announce ADDRS     Full multiaddr for announce (overrides --public-ip)"
            echo "  --db-path PATH       Database file path"
            echo "  --trusted-admins PK  Comma-separated admin pubkeys"
            echo "  --uninstall          Remove aegis-relay from this system"
            echo "  -h, --help           Show this help"
            exit 0
            ;;
        *) error "Unknown option: $1"; exit 1 ;;
    esac
done

if [[ "$(id -u)" -ne 0 ]]; then
    error "This script must be run as root (use sudo)"
    exit 1
fi

if $UNINSTALL; then
    info "Uninstalling aegis-relay..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    rm -rf "$CONFIG_DIR"
    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    systemctl daemon-reload
    info "aegis-relay uninstalled successfully"
    exit 0
fi

ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

info "Detecting latest release..."
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$LATEST_TAG" ]]; then
    error "Could not detect latest release. Check that the repository has published releases."
    exit 1
fi
info "Latest release: ${LATEST_TAG}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${BINARY_NAME}"
TMP_FILE=$(mktemp)
info "Downloading ${BINARY_NAME}..."
if ! curl -fsSL -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    error "Download failed from ${DOWNLOAD_URL}"
    error "Make sure the release contains an '${BINARY_NAME}' asset for linux/${GOARCH}"
    rm -f "$TMP_FILE"
    exit 1
fi

info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
chmod +x "$TMP_FILE"
mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"

mkdir -p "$CONFIG_DIR"

PORT="${PORT:-$DEFAULT_PORT}"
HTTP_PORT="${HTTP_PORT:-$DEFAULT_HTTP_PORT}"

ENV_FILE="${CONFIG_DIR}/env"
cat > "$ENV_FILE" <<EOF
AEGIS_P2P_PORT=${PORT}
AEGIS_RELAY_HTTP_PORT=${HTTP_PORT}
AEGIS_RELAY_SERVICE_ENABLED=true
AEGIS_RELAY_CANDIDATE=true
AEGIS_AUTOSTART_P2P=true
AEGIS_DB_PATH=${DB_PATH:-/var/lib/aegis-relay/aegis_node.db}
EOF

if [[ -n "$ANNOUNCE_ADDRS" ]]; then
    echo "AEGIS_ANNOUNCE_ADDRS=${ANNOUNCE_ADDRS}" >> "$ENV_FILE"
elif [[ -n "$PUBLIC_IP" ]]; then
    echo "AEGIS_PUBLIC_IP=${PUBLIC_IP}" >> "$ENV_FILE"
fi

if [[ -n "$TRUSTED_ADMINS" ]]; then
    echo "AEGIS_TRUSTED_ADMINS=${TRUSTED_ADMINS}" >> "$ENV_FILE"
fi

DB_DIR=$(dirname "${DB_PATH:-/var/lib/aegis-relay/aegis_node.db}")
mkdir -p "$DB_DIR"

cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Aegis P2P Relay Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
EnvironmentFile=${ENV_FILE}
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

WorkingDirectory=${DB_DIR}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

info "Waiting for relay to start..."
sleep 2

if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "aegis-relay is running!"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    info "Relay is active on port ${PORT}"
    info "HTTP monitoring on port ${HTTP_PORT}"
    info "  Health:  http://localhost:${HTTP_PORT}/health"
    info "  Metrics: http://localhost:${HTTP_PORT}/metrics"
    info "  Peers:   http://localhost:${HTTP_PORT}/peers"
    echo ""

    PEER_ID=$(curl -fsSL "http://localhost:${HTTP_PORT}/health" 2>/dev/null | grep -oP '"peer_id":"\K[^"]+' || echo "<unknown>")
    if [[ -n "$PUBLIC_IP" ]]; then
        RELAY_ADDR="/ip4/${PUBLIC_IP}/tcp/${PORT}/p2p/${PEER_ID}"
        info "Relay multiaddr for clients:"
        echo ""
        echo "  ${RELAY_ADDR}"
        echo ""
        info "Set in client env:"
        echo "  AEGIS_RELAY_PEERS=${RELAY_ADDR}"
        echo "  AEGIS_BOOTSTRAP_PEERS=${RELAY_ADDR}"
    elif [[ -n "$ANNOUNCE_ADDRS" ]]; then
        info "Relay announce address: ${ANNOUNCE_ADDRS}"
    else
        warn "No --public-ip or --announce set. Clients behind NAT may not find this relay."
        warn "Re-run with --public-ip <YOUR_IP> to configure announce addresses."
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    error "aegis-relay failed to start. Check logs:"
    echo "  journalctl -u ${SERVICE_NAME} -n 50"
    exit 1
fi
