#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_DIR="$ROOT_DIR/aegis-app"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass=0
fail=0

run_check() {
    local name="$1"
    shift
    echo ""
    echo "━━━ $name ━━━"
    if "$@"; then
        echo -e "${GREEN}PASS${NC}: $name"
        ((pass++))
    else
        echo -e "${RED}FAIL${NC}: $name"
        ((fail++))
    fi
}

echo "╔══════════════════════════════════════════════╗"
echo "║     Aegis R6 Release Gate Checks             ║"
echo "╚══════════════════════════════════════════════╝"

run_check "1/6 Go compile + unit tests" bash -c "cd $APP_DIR && go test ./... -count=1 -timeout 300s"

run_check "2/6 Frontend build" bash -c "cd $APP_DIR/frontend && npm run build"

run_check "3/6 Relay binary build" bash -c "cd $APP_DIR && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags relay -o /dev/null ."

run_check "4/6 Three-node integration (A2/A3/B1/C1)" bash -c "cd $APP_DIR && go test -run 'TestA2PostTitleBodyReplicatesAcrossNodes' -count=1 -timeout 60s . && go test -run 'TestA3HotNewSortingConsistentAcrossNodes' -count=1 -timeout 60s . && go test -run 'TestB1NestedCommentReplicatesAcrossNodes' -count=1 -timeout 60s . && go test -run 'TestC1GovernancePolicyAppliesAcrossNodes' -count=1 -timeout 60s ."

run_check "5/6 Migration safety (no DROP/RENAME)" bash -c "! rg -n 'DROP TABLE|DROP COLUMN|ALTER TABLE .* RENAME' $APP_DIR --glob '*.go'"

run_check "6/6 API contract sanity" bash -c "cd $APP_DIR && python3 -c \"
import re
from pathlib import Path
app_dts = Path('frontend/wailsjs/go/main/App.d.ts')
text = app_dts.read_text()
exports = re.findall(r'^export function\\\\s+([A-Za-z0-9_]+)\\\\(', text, flags=re.M)
dups = sorted({name for name in exports if exports.count(name) > 1})
if dups: raise SystemExit(f'Duplicate exports: {dups}')
required = ['SearchSubs', 'SearchPosts', 'GetFeedStream', 'SubscribeSub', 'PublishPostStructuredToSub', 'AddFavorite', 'GetFavorites', 'GetMyPosts', 'GetNotifications', 'GetP2PStatus', 'GetReleaseMetrics']
missing = [name for name in required if name not in exports]
if missing: raise SystemExit(f'Missing exports: {missing}')
print('API contract OK')
\""

echo ""
echo "╔══════════════════════════════════════════════╗"
echo -e "║  Results: ${GREEN}${pass} passed${NC}, ${RED}${fail} failed${NC}                    ║"
echo "╚══════════════════════════════════════════════╝"

if [[ $fail -gt 0 ]]; then
    echo -e "${RED}R6 gate FAILED. Do not release.${NC}"
    exit 1
fi

echo -e "${GREEN}R6 gate PASSED. Ready for release.${NC}"
