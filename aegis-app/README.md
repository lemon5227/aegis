# Aegis

Decentralized forum app built with Wails (Go + React + TypeScript).

Aegis is designed for real peer-to-peer usage, not just local demos:

- multi-node sync with anti-entropy,
- governance with Lamport-first consistency,
- shadow-ban storage policy controls,
- comment media attachments with distributed fetch paths,
- practical cold-start relay strategy for low-budget deployments.

## Why Aegis

- No central backend required for core forum flow.
- Works with mixed NAT/public nodes.
- Keeps governance behavior deterministic under message reordering.
- Separates private vs shared storage responsibility.
- Ships desktop builds (Windows, Linux, macOS ARM) via GitHub Actions.

## Current Product Status

- Governance A1/A2/A3/A4: implemented.
- Comment media blob unification: implemented.
- Beta rollout:
  - BETA-0: done
  - BETA-1: done
  - BETA-2: done
  - BETA-3: ready for cohort rollout

See rollout docs in `../docs/`:

- `BETA_ROLLOUT_PLAN_2026-02-20.md`
- `LAMPORT_AND_SHADOWBAN_STORAGE_GOVERNANCE_PLAN_2026-02-18.md`
- `COMMENT_MEDIA_BLOB_UNIFICATION_PLAN_2026-02-20.md`
- `RELAY_SELF_BOOTSTRAP_PLAN_2026-02-20.md`

## Architecture Overview

- `backend (Go)`
  - libp2p pubsub and relay support
  - anti-entropy summary sync
  - content/media fetch request-response channels
  - governance state + logs with Lamport ordering
  - SQLite persistence (`modernc.org/sqlite`)

- `frontend (React + TS)`
  - feed/discover/post detail/settings
  - profile/privacy/governance UI
  - comment attachment UX (thumbnails, zoom)
  - updates page backed by release API

## Key Features

### Networking

- P2P start/stop, peer connection, relay fallback.
- Known-peer persistence and peer exchange for faster joins.
- Auto announce support:
  - explicit: `AEGIS_ANNOUNCE_ADDRS` or `AEGIS_PUBLIC_IP`
  - automatic public IP detection when explicit config is absent.

### Governance and Consistency

- Lamport clock tracking for public content and moderation events.
- Lamport-first acceptance with timestamp fallback.
- Deterministic moderation conflict resolution.
- Shadow-ban policy applied consistently in:
  - inbound message acceptance,
  - query filtering,
  - sync summary generation,
  - content/media serving policy.

### Storage Policy

- Distinguishes shared responsibility vs private responsibility.
- Prevents disallowed content from becoming shared network burden.

### User Experience

- Post creation with local image upload and compression.
- Comment attachments via structured references (`media_cid`, `external_url`).
- Click-to-zoom images in post and comment views.

## Quick Start (Dev)

From `aegis-app`:

```bash
go test ./...
cd frontend && npm install && npm run build && cd ..
wails dev
```

## Build

```bash
wails build
```

## Relay Node (Production Bootstrap)

Build relay binary:

```bash
go build -tags relay -o aegis-relay .
```

Run relay with public announce:

```bash
AEGIS_PUBLIC_IP=<your-public-ip> ./aegis-relay
```

or:

```bash
AEGIS_ANNOUNCE_ADDRS="/ip4/<your-public-ip>/tcp/40100" ./aegis-relay
```

Verify startup logs include `announce_addrs` with reachable public address.

## Important Environment Variables

- `AEGIS_DB_PATH`: SQLite database path.
- `AEGIS_AUTOSTART_P2P`: auto-start P2P (`true` by default).
- `AEGIS_P2P_PORT`: preferred listen port (default `40100`).
- `AEGIS_BOOTSTRAP_PEERS`: initial peer seeds.
- `AEGIS_RELAY_PEERS`: static relay list.
- `AEGIS_ANNOUNCE_ADDRS`: explicit announced addresses.
- `AEGIS_PUBLIC_IP`: simple announce helper.
- `AEGIS_AUTO_ANNOUNCE`: auto public IP detection toggle (`1` default).

Abuse/stability controls:

- `AEGIS_MAX_CONNECTED_PEERS`
- `AEGIS_FETCH_REQUEST_LIMIT`
- `AEGIS_FETCH_REQUEST_WINDOW_SEC`
- `AEGIS_MSG_MAX_BYTES`
- `AEGIS_MSG_RATE_LIMIT`
- `AEGIS_MSG_RATE_WINDOW_SEC`
- `AEGIS_RELAY_SERVICE_ENABLED`

## CI/CD and Releases

Workflows:

- `.github/workflows/desktop-build-matrix.yml`
  - Windows + Linux builds
- `.github/workflows/macos-arm-build.yml`
  - macOS ARM build

Release behavior:

- Push `v*` tag -> publish official release assets.
- Manual `workflow_dispatch` can publish draft prerelease assets.

## Formal Release Command

```bash
git tag -a v0.1.1 -m "release v0.1.1"
git push origin v0.1.1
```

## WSL2 One-Click Setup

From repository root:

```bash
./scripts/setup_wsl2_env.sh
```

Auto-start after setup:

```bash
./scripts/setup_wsl2_env.sh --start
```

## Ops and Troubleshooting

- Gate checks:

```bash
./scripts/run_g6_gate_checks.sh
```

- Observability and incident guidance:
  - `../docs/R5_OBSERVABILITY_RUNBOOK.md`

## Security and Practical Notes

- Relay address is operationally public once clients connect; do not rely on IP secrecy as primary protection.
- Use rate limits, peer policies, and monitoring for real protection.
- For production relay, enforce firewall/security-group policy and keep logs/alerts active.
