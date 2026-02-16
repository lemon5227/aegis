#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_DIR="$ROOT_DIR/aegis-app"
FRONTEND_DIR="$APP_DIR/frontend"

GO_VERSION="${GO_VERSION:-1.22.7}"
NVM_VERSION="${NVM_VERSION:-v0.40.1}"
NVM_NODE_ALIAS="${NVM_NODE_ALIAS:---lts}"

SEED_MULTIADDR_DEFAULT="/ip4/51.107.0.10/tcp/40100/p2p/12D3KooWLweFn4GFfEa9X1St4d78HQqYYzXaH2oy5XahKrwar6w7"
AEGIS_DB_PATH_DEFAULT="aegis_wsl2.db"

with_start=false
for arg in "$@"; do
  case "$arg" in
    --start)
      with_start=true
      ;;
    *)
      echo "Unknown argument: $arg"
      echo "Usage: $0 [--start]"
      exit 1
      ;;
  esac
done

if [[ ! -d "$APP_DIR" ]]; then
  echo "Aegis app directory not found: $APP_DIR"
  echo "Please run this script from the repository workspace."
  exit 1
fi

if [[ ! -f /proc/version ]] || ! grep -qi "microsoft" /proc/version; then
  echo "This script is intended for WSL/WSL2."
  echo "Current environment does not look like WSL."
fi

echo "[Aegis][WSL2] Installing system dependencies..."
sudo apt-get update -y
sudo apt-get install -y \
  ca-certificates \
  curl \
  wget \
  git \
  gnupg \
  lsb-release \
  build-essential \
  pkg-config \
  libgtk-3-dev \
  libayatana-appindicator3-dev \
  libsqlite3-dev

if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
  sudo apt-get install -y libwebkit2gtk-4.1-dev
else
  sudo apt-get install -y libwebkit2gtk-4.0-dev
fi

echo "[Aegis][WSL2] Installing Go ${GO_VERSION}..."
if ! command -v go >/dev/null 2>&1 || [[ "$(go version 2>/dev/null || true)" != *"go${GO_VERSION}"* ]]; then
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
fi

if ! grep -q '/usr/local/go/bin' "$HOME/.bashrc"; then
  echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$HOME/.bashrc"
fi
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"

echo "[Aegis][WSL2] Installing Node.js via nvm..."
if [[ ! -d "$HOME/.nvm" ]]; then
  curl -fsSL "https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh" | bash
fi

export NVM_DIR="$HOME/.nvm"
if [[ -s "$NVM_DIR/nvm.sh" ]]; then
  # shellcheck disable=SC1090
  . "$NVM_DIR/nvm.sh"
fi

nvm install "$NVM_NODE_ALIAS"
nvm use "$NVM_NODE_ALIAS"

echo "[Aegis][WSL2] Installing Wails CLI..."
go install github.com/wailsapp/wails/v2/cmd/wails@latest

if ! grep -q '\$HOME/go/bin' "$HOME/.bashrc"; then
  echo 'export PATH=$PATH:$HOME/go/bin' >> "$HOME/.bashrc"
fi
export PATH="$PATH:$HOME/go/bin"

echo "[Aegis][WSL2] Installing frontend dependencies..."
cd "$FRONTEND_DIR"
npm install

echo "[Aegis][WSL2] Environment ready."
echo ""
echo "Versions:"
go version || true
node -v || true
npm -v || true
wails version || true

echo ""
echo "Default run command:"
echo "AEGIS_DB_PATH=${AEGIS_DB_PATH_DEFAULT} \\\n+AEGIS_AUTOSTART_P2P=1 \\\n+AEGIS_P2P_PORT=40100 \\\n+AEGIS_BOOTSTRAP_PEERS=${SEED_MULTIADDR_DEFAULT} \\\n+AEGIS_RELAY_PEERS=${SEED_MULTIADDR_DEFAULT} \\\n+wails dev"

if [[ "$with_start" == true ]]; then
  echo ""
  echo "[Aegis][WSL2] Starting Aegis now..."
  cd "$APP_DIR"
  AEGIS_DB_PATH="${AEGIS_DB_PATH:-$AEGIS_DB_PATH_DEFAULT}" \
  AEGIS_AUTOSTART_P2P="${AEGIS_AUTOSTART_P2P:-1}" \
  AEGIS_P2P_PORT="${AEGIS_P2P_PORT:-40100}" \
  AEGIS_BOOTSTRAP_PEERS="${AEGIS_BOOTSTRAP_PEERS:-$SEED_MULTIADDR_DEFAULT}" \
  AEGIS_RELAY_PEERS="${AEGIS_RELAY_PEERS:-$SEED_MULTIADDR_DEFAULT}" \
  wails dev
fi
