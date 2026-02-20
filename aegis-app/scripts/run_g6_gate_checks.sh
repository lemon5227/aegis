#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_DIR="$(cd "$ROOT_DIR/.." && pwd)"

echo "[G6] 1/4 Go compile check"
(cd "$ROOT_DIR" && go test ./...)

echo "[G6] 2/4 Frontend build check"
(cd "$ROOT_DIR/frontend" && npm run build)

echo "[G6] 3/4 API contract sanity (duplicates + required exports)"
python3 - <<'PY'
import re
from pathlib import Path

repo = Path("/Users/wenbo/aegis")
app_dts = repo / "aegis-app/frontend/wailsjs/go/main/App.d.ts"
text = app_dts.read_text(encoding="utf-8")

exports = re.findall(r"^export function\s+([A-Za-z0-9_]+)\(", text, flags=re.M)
dups = sorted({name for name in exports if exports.count(name) > 1})
if dups:
    raise SystemExit(f"Duplicate App.d.ts exports: {', '.join(dups)}")

required = [
    "SearchSubs",
    "SearchPosts",
    "GetFeedStream",
    "GetFeedStreamWithStrategy",
    "SubscribeSub",
    "PublishPostStructuredToSub",
]
missing = [name for name in required if name not in exports]
if missing:
    raise SystemExit(f"Missing required App.d.ts exports: {', '.join(missing)}")

print("App.d.ts contract sanity passed")
PY

echo "[G6] 4/4 Migration safety scan"
if rg -n "DROP TABLE|DROP COLUMN|ALTER TABLE .* RENAME" "$ROOT_DIR" --glob "*.go"; then
  echo "Potential destructive migration detected. Please review output above."
  exit 1
fi

echo "G6 gate checks passed"
