# Core Capabilities Waterfall Plan

## Purpose

Track the remaining work required to turn Aegis into a production-grade distributed forum, with explicit emphasis on distributed availability, trust, and operational durability.

This document complements:
- [MULTI_NODE_AVAILABILITY_EXECUTION_PLAN_2026-03-09.md](/Users/wenbo/aegis/docs/MULTI_NODE_AVAILABILITY_EXECUTION_PLAN_2026-03-09.md)

## Core Capability Gaps

### C1. Distributed Availability

The system must absorb distributed complexity so users can continue safely under weak connectivity without being pushed into network reasoning during normal use.

Work:
- Quiet network reliability
- Partial content recovery
- Pending local actions / unsynced writes
- Offline and reconnect UX
- Multi-node recovery tooling

Status:
- In progress

### C2. Distributed Trust And Identity Integrity

Users must be able to trust authorship and state authenticity across nodes.

Work:
- Message signing for posts, comments, profiles, governance actions
- Signature verification on receive paths
- Key lifecycle, restore, rotation, device migration
- Tamper and replay resistance

Status:
- In progress

Completed on March 9, 2026:
- Signed posts, comments, profiles, vote-state broadcasts, and governance messages
- Signature verification on receive paths for signed message types
- Forged-payload regression coverage

### C3. Content Durability And Convergence

Nodes must converge predictably even with late-arriving data, partial media, and temporary partitions.

Work:
- Minimal, human-centered convergence messaging
- Conflict-safe refresh and stale-state handling only when task-relevant
- Late-arrival reconciliation for posts/comments/media
- Durable recovery and repair workflows

Status:
- Pending / partially started via repair UI

### C4. Community Operations

Communities need tools to organize and sustain activity.

Work:
- Sticky / featured / locked posts
- Configurable rules and community metadata
- Report and review queue
- Announcement and onboarding surfaces

Status:
- In progress

Completed on March 9, 2026:
- Persistent `message_outbox` for durable local writes
- Automatic outbox replay with bounded backoff after peer recovery
- Multi-node regression covering queued post replay after peer connect

### C5. Creation And Consumption Completeness

Forum usage needs complete loops for drafting, resuming, citing, sharing, and continuing reading.

Work:
- Draft inbox, history, continue reading
- Quote and reference flows
- Better deep links and context restoration
- Richer markdown and content rendering

Status:
- In progress

### C6. Release And Verification Quality

Distributed systems cannot be shipped on manual confidence alone.

Work:
- Multi-node automated tests
- Offline/reconnect regression matrix
- Late-media and partial-sync scenarios
- Release gates and rollback validation

Status:
- In progress

Completed on March 9, 2026:
- Multi-node trust/reliability regression tests
- Source-level G6 gate checks for unsigned publish payload regressions

## Execution Priority

1. C1 Distributed Availability
2. C3 Content Durability And Convergence
3. C5 Creation And Consumption Completeness
4. C4 Community Operations
5. C2 Distributed Trust And Identity Integrity
6. C6 Release And Verification Quality

## Current Active Track

Active:
- C1 Distributed Availability

Current completed slices:
- Internal health derivation and primary recovery actions
- Manual sync trigger
- Partial post/media recovery entry points

Current next slices:
- Pending local actions / unsynced writes
- Simpler offline and reconnect UX
- Multi-node recovery diagnostics
