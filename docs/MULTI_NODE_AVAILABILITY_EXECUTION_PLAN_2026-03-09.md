# Multi-Node Availability Execution Plan

## Goal

Push Aegis from a locally-usable forum client to a multi-node distributed forum that remains understandable and usable under weak connectivity, delayed replication, missing peers, partial media, and eventual consistency.

This plan explicitly prioritizes **multi-node availability** over new feature breadth.

## Product UX Principle

Distributed complexity is an implementation concern, not a default user-facing concept.

The product should behave like a calm, reliable forum client:

1. Users should normally see simple outcomes, not protocol state.
2. Recovery should happen automatically whenever possible.
3. Warnings should appear only when they materially affect the user's task.
4. Detailed network/replication diagnostics belong in settings, support, or operator tooling.
5. Primary UI should favor reassurance and continuity over raw system visibility.

## Product Definition Of Done

Aegis is considered "multi-node product ready" only when:

1. Users can publish while offline or under degraded connectivity without losing intent.
2. The app automatically retries and converges without requiring users to understand network state.
3. Missing or late-arriving distributed content degrades gracefully instead of feeling broken.
4. When user action is required, the product gives one clear next step instead of protocol details.
5. Replication conflicts and late-arriving updates resolve consistently and predictably.
6. Advanced network and recovery state remain available for support/debugging without cluttering the default UX.
7. The system is covered by multi-node automated verification, not only single-node tests.

## Workstreams

### W1. Quiet Network Reliability

Objective: keep the product feeling reliable without making users think about nodes, peers, or anti-entropy internals.

Tasks:
- Prefer silent recovery and automatic retries.
- Reduce primary-surface network messaging to minimal, task-relevant guidance.
- Keep detailed diagnostics in settings or support surfaces.
- Distinguish internal states for the system, while exposing only simplified user messaging by default.

Exit criteria:
- Users are not required to reason about distributed internals during normal use.
- Support/debugging remains possible through dedicated tooling.

### W2. Pending Replication Queue

Objective: make writes durable under connectivity issues without forcing users to manage a replication queue manually.

Tasks:
- Persist a local pending-outbox for posts, comments, votes, edits, deletes, profile updates.
- Mark mutations as `pending`, `replicated`, `failed`, `needs retry`.
- Retry automatically on reconnect with bounded backoff.
- Expose pending state only when it meaningfully affects user trust or requires action.

Exit criteria:
- Users never lose intent because a peer was unavailable at publish time.
- The system can explain delayed sync when necessary, but default UX stays lightweight.

### W3. Offline And Weak-Network UX

Objective: support intermittent connectivity as a first-class mode while keeping the product calm and simple.

Tasks:
- Allow local creation while disconnected.
- Use minimal, non-technical status messaging for offline/reconnect transitions.
- Gate actions that truly require peers, but never discard local drafts or pending writes.
- Preserve navigability when peer-driven data is incomplete.

Exit criteria:
- Users can keep reading and writing locally while disconnected.
- The app recovers without manual data reconstruction after reconnect.

### W4. Partial Content Recovery

Objective: degrade gracefully when distributed content arrives late or incompletely, without making the UI feel diagnostic-heavy.

Tasks:
- Add placeholders for missing post bodies, comments, and media.
- Add retry and repair flows for per-entity fetch.
- Use human-centered messaging for stale/partial entities instead of protocol language.
- Improve attachment/media fetch observability and recovery actions.

Exit criteria:
- Missing distributed data does not silently appear broken.
- Users can request recovery without restarting the app.

### W5. Consistency And Conflict UX

Objective: make eventual consistency feel predictable without surfacing more distributed-system detail than necessary.

Tasks:
- Surface refresh prompts only when stale content meaningfully affects the current task.
- Expose conflict-safe refresh points in detail views with simple language.
- Improve operation timelines from dev-only tooling toward operator-grade diagnostics.
- Add clear semantics for "latest known state" vs "still converging".

Exit criteria:
- Late-arriving remote changes feel predictable.
- Users do not misinterpret convergence as random UI breakage.

### W6. Multi-Node Test Matrix

Objective: stop shipping distributed behavior that is only manually tested.

Tasks:
- Add scripted multi-node scenarios: publish/replicate/edit/delete/reconnect.
- Add delayed-peer and no-peer test scenarios.
- Add media late-arrival and partial-sync scenarios.
- Add regression checks for pending queue replay after reconnect.

Exit criteria:
- Core distributed workflows are covered by repeatable automated checks.

## Execution Order

1. W1 Quiet network reliability
2. W4 Partial content recovery
3. W2 Pending replication queue
4. W3 Offline and weak-network UX
5. W5 Consistency and conflict UX
6. W6 Multi-node test matrix

## Immediate Implementation Sequence

### Phase A
- Build internal health derivation first.
- Keep primary-surface messaging minimal and action-oriented.
- Move detailed diagnostics behind settings/support surfaces.

### Phase B
- Add partial-content placeholders and retry actions.
- Expose per-post recovery entry points.

### Phase C
- Build persistent pending replication queue.
- Inline pending/failure states for local actions.

### Phase D
- Add offline banner and reconnect UX.
- Add durable retry/replay behavior.

### Phase E
- Expand automated multi-node coverage and release gates.

## Non-Goals For This Track

- Token/DAO features
- Identity-hardening details beyond what blocks availability work
- New content surfaces that do not improve distributed usability

## Current Status

Started on March 9, 2026.

In progress:
- W1 quiet network reliability

Pending:
- W3 offline and weak-network UX
- W5 consistency and conflict UX

Completed / substantially advanced:
- W2 pending replication queue
  - persistent backend `message_outbox`
  - bounded retry scheduling
  - replay after peer recovery
- W4 partial content recovery
  - partial post/media placeholders
  - per-post repair entry points
- W6 multi-node test matrix
  - automated forged-message rejection test
  - automated outbox replay after peer-connect test
