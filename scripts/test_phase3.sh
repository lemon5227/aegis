#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$ROOT_DIR/aegis-app"

cd "$APP_DIR"
echo "[Aegis] Running Phase 3 one-click P2P replication test..."
go test -run TestPhase3P2PReplication -v .
echo "[Aegis] Phase 3 P2P replication test passed."
