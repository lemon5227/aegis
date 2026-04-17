#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$ROOT_DIR/aegis-app"

cd "$APP_DIR"
echo "[Aegis] Running C1 governance productization tests..."
go test -run 'TestC1|TestGovernance' -v -count=1 .
echo "[Aegis] C1 governance tests passed."
