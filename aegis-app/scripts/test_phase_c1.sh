#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$APP_DIR"

echo "[C1] Running governance productization regression tests..."
go test ./... -run 'TestC1GovernancePolicyAffectsHistoricalVisibility|TestC1ModerationLogsRecorded' -count=1

echo "[C1] Running baseline guard tests (B2/B3 + LAN replication)..."
go test ./... -run 'TestB2PostUpvoteDedup|TestB2CommentUpvoteDedup|TestB3HotVsNewOrdering|TestPhase3P2PReplication' -count=1

echo "[C1] All checks passed."
