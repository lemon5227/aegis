#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$APP_DIR"

DURATION_HOURS="${LAMPORT_SOAK_HOURS:-24}"
if [[ ! "$DURATION_HOURS" =~ ^[0-9]+$ ]] || [[ "$DURATION_HOURS" -le 0 ]]; then
  echo "[LAMPORT-24H] invalid LAMPORT_SOAK_HOURS=$DURATION_HOURS" >&2
  exit 1
fi

END_TS=$(( $(date +%s) + DURATION_HOURS * 3600 ))
ROUND=0

echo "[LAMPORT-24H] start duration=${DURATION_HOURS}h"
while [[ $(date +%s) -lt $END_TS ]]; do
  ROUND=$((ROUND + 1))
  echo "[LAMPORT-24H] round=$ROUND running TestLamportThreeNodeSoakConvergence"
  go test ./... -run TestLamportThreeNodeSoakConvergence -count=1
done

echo "[LAMPORT-24H] completed rounds=$ROUND"
