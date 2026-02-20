# Aegis Beta Rollout Plan (BETA-0 to BETA-3)

Date: 2026-02-20

## Goal

Ship a controlled beta to real users with acceptable stability, clear rollback paths, and measurable network health.

## Scope

This plan focuses on:

1. reliable cold-start networking with minimal centralized dependency,
2. safe relay behavior under constrained budget,
3. observability and gate checks before user invite expansion,
4. staged beta cohort rollout.

## Phase Breakdown

## BETA-0: Preflight Hardening (Now)

### Deliverables

1. Documented deployment flow for single public relay cold-start.
2. Known-peer persistence + peer exchange enabled.
3. G6 automated gate script passing (`go`, `frontend build`, API contract sanity, migration safety scan).

### Exit Criteria

1. Two-node and three-node smoke passes complete.
2. Relay startup output includes `announce_addrs`.

## BETA-1: Auto Announce and Join Reliability

### Deliverables

1. Automatic public IP detection for announce candidates when explicit env is not set.
2. Fallback precedence:
   - explicit announce env,
   - cloud metadata public IP,
   - safe external IP probe.
3. Bootstrapping sequence:
   - known peers first,
   - relay/bootstrap fallback second.

### Exit Criteria

1. Public-capable node can join and announce without manual public IP input.
2. Private node still joins via relay path.

## BETA-2: Abuse Guard and Stability Controls

### Deliverables

1. Relay connection/rate limits enabled and documented.
2. Peer quality scoring and dialing preference hardened.
3. Incident runbook update with relay overload recovery.

### Exit Criteria

1. Synthetic burst does not collapse relay availability.
2. Join latency remains within target under moderate load.

## BETA-3: User Rollout

### Deliverables

1. Invite cohorts:
   - cohort A (10 users),
   - cohort B (30 users),
   - cohort C (100 users).
2. Weekly health report:
   - join success,
   - relay dependence ratio,
   - anti-entropy lag,
   - content/media fetch success.

### Exit Criteria

1. Cohort C stable for 7 days without critical incident.
2. No blocker-level data-loss or sync divergence bug open.

## Gate Checklist Before Sending Beta Invites

1. `./aegis-app/scripts/run_g6_gate_checks.sh` passes.
2. Relay node exposes correct `announce_addrs` and reachable TCP port.
3. At least one rollback command path is tested (restart, config revert, relay seed fallback).
4. Governance A3/A4 behavior verified in multi-node test:
   - lamport-first acceptance,
   - policy-gated media serving.

## Current Status

- BETA-0: Done
- BETA-1: Done (auto public IP announce detection added)
- BETA-2: Pending
- BETA-3: Pending

## Notes

1. Auto announce can be disabled with `AEGIS_AUTO_ANNOUNCE=0`.
2. Explicit config always wins:
   - `AEGIS_ANNOUNCE_ADDRS`
   - `AEGIS_PUBLIC_IP`
