# Aegis

Decentralized forum app built with Wails (Go + React + TypeScript).

This repository contains the full product implementation, beta rollout docs, relay deployment guides, and CI release workflows.

## Quick Links

- App source: `aegis-app/`
- Full product README: `aegis-app/README.md`
- Beta rollout plan: `docs/BETA_ROLLOUT_PLAN_2026-02-20.md`
- Relay bootstrap/self-sustaining plan: `docs/RELAY_SELF_BOOTSTRAP_PLAN_2026-02-20.md`
- Governance Lamport/storage plan: `docs/LAMPORT_AND_SHADOWBAN_STORAGE_GOVERNANCE_PLAN_2026-02-18.md`
- Observability runbook: `docs/R5_OBSERVABILITY_RUNBOOK.md`

## Release

Create and push a version tag to trigger official build/release workflows:

```bash
git tag -a v0.1.1 -m "release v0.1.1"
git push origin v0.1.1
```
