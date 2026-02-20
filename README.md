# Aegis

**Aegis is a decentralized social forum built for real peer-to-peer collaboration.**

No central feed server. No fragile ordering assumptions. No generic demo UX.

It is a desktop-first product powered by **Go + libp2p + SQLite + React + Wails**, designed to run in messy real-world networks (NAT, relay fallback, mixed node quality) while keeping governance behavior deterministic and auditable.

## Why Aegis

- **Network-first architecture**: anti-entropy sync, relay-aware bootstrap, known-peer persistence, peer exchange.
- **Deterministic governance**: Lamport-first moderation ordering with clear fallback semantics.
- **Storage responsibility model**: shared vs private responsibility split to avoid abuse-driven bloat.
- **Modern user experience**: media-rich posts/comments, image attachments, zoom, profile/governance controls.
- **Shippable desktop pipeline**: automated Windows/macOS/Linux build and release workflows.

## Core Capabilities

### Distributed Forum Engine
- Sub communities, feed/search, post detail, threaded comments.
- Public and private content lanes.
- P2P propagation + anti-entropy recovery.

### Governance and Safety
- Shadow-ban policy with consistent behavior across ingest/query/sync/serving.
- Lamport clocks applied to moderation and public content acceptance.
- Moderation logs and state sync for reconciliation.

### Resilient Networking
- Bootstrap relay for cold start.
- Automatic public announce (or explicit announce override).
- Known-peer table and peer exchange to reduce long-term relay dependence.

## Current Rollout State

- BETA-0: Done
- BETA-1: Done
- BETA-2: Done
- BETA-3: In rollout preparation

Details:
- `docs/BETA_ROLLOUT_PLAN_2026-02-20.md`
- `docs/RELAY_SELF_BOOTSTRAP_PLAN_2026-02-20.md`

## Repository Map

- `aegis-app/` - Application source (backend + frontend)
- `docs/` - Architecture plans, rollout plans, runbooks
- `.github/workflows/` - CI build and release automation

## Build and Release

### Official release (tag-driven)

```bash
git tag -a v0.1.1 -m "release v0.1.1"
git push origin v0.1.1
```

This triggers platform packaging and upload to GitHub Releases.

### Manual draft beta release

Use `workflow_dispatch` with `publish_draft_release=true` for controlled beta distribution.

## More Documentation

- Full app README: `aegis-app/README.md`
- Governance plan: `docs/LAMPORT_AND_SHADOWBAN_STORAGE_GOVERNANCE_PLAN_2026-02-18.md`
- Comment media plan: `docs/COMMENT_MEDIA_BLOB_UNIFICATION_PLAN_2026-02-20.md`
- Observability runbook: `docs/R5_OBSERVABILITY_RUNBOOK.md`
