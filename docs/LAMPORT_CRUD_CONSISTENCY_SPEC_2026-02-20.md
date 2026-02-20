# Lamport CRUD Consistency Spec

Date: 2026-02-20
Status: Proposed implementation standard (next refactor baseline)

## 1. Goal

Define one deterministic consistency model for distributed CRUD in Aegis so that:

1. create/update/delete follow the same ordering logic,
2. offline node replay cannot revive stale state,
3. UI always converges to the latest state under eventual delivery.

This spec applies to posts/comments first, and can be reused for other replicated entities.

## 2. Core Principle

All replicated writes are **operations** with Lamport ordering, not direct row overwrites.

Canonical version key for each operation:

1. `lamport` (primary)
2. `author_pubkey` (tie-break 1)
3. `op_id` (tie-break 2, unique)

Comparison function is total-order deterministic:

- incoming is newer if:
  1) `incoming.lamport > current.lamport`, or
  2) lamport equal and `incoming.author_pubkey > current.author_pubkey`, or
  3) above equal and `incoming.op_id > current.op_id`.

## 3. Data Model

For each replicated entity (`post`, `comment`) maintain state fields:

1. `entity_id`
2. `current_lamport`
3. `current_op_id`
4. `current_author_pubkey`
5. `deleted_at_lamport` (0 if active)
6. `deleted_at_ts`
7. `deleted_by`

Optional but recommended:

- append-only `entity_ops` log for audit/debug/rebuild.

## 4. Operation Types

Unified operation enum:

1. `CREATE`
2. `UPDATE`
3. `DELETE`

Each operation payload must contain:

1. `entity_type`
2. `entity_id`
3. `op_type`
4. `op_id`
5. `author_pubkey`
6. `lamport`
7. `timestamp`
8. `body` / attachment refs / metadata (for create/update)

## 5. CRUD Semantics under Lamport

### 5.1 Create

1. If entity does not exist -> create state.
2. If entity exists -> apply only when incoming op is newer by canonical compare.
3. If existing state is tombstoned and incoming create is older than tombstone -> reject.

### 5.2 Read

1. Query current state table only.
2. Hide records where tombstone active (`deleted_at_lamport > 0`).
3. Governance filters (shadow-ban etc.) apply after state resolution.

### 5.3 Update

1. Never bypass compare function.
2. Apply only when incoming op is newer than current version.
3. If tombstone exists with newer/equal version -> reject update.

### 5.4 Delete

1. Delete is a tombstone op, not physical delete.
2. Apply only if delete op is newer than current version.
3. Once applied, any older create/update must not resurrect content.

## 6. Anti-Resurrection Rule (offline replay protection)

When receiving historical data from offline nodes:

1. compare incoming op/version against current state version,
2. if incoming older/equal -> ignore,
3. if current has newer tombstone -> ignore all non-newer non-delete ops.

This guarantees stale node replay cannot bring deleted/older content back.

## 7. Sync Protocol Requirements

Summary digests must include version metadata, not only payload pointers.

Minimum digest fields:

1. `entity_id`
2. `op_type` (or `deleted` flag)
3. `lamport`
4. `op_id`
5. `author_pubkey`

Receiver decides pull/apply by compare function before mutating local state.

## 8. UI Convergence Rules

UI update events should represent operation completion, not optimistic DB assumptions.

1. On accepted op apply, emit entity-scoped event (`feed:updated`, `comments:updated`, `subs:updated`).
2. On rejected stale op, no state regression event should be emitted.
3. Clients should periodically resync critical lists as fallback for missed events.

## 9. Conflict Scenarios

### 9.1 Concurrent update vs delete

- Larger canonical version wins.
- If delete wins, content remains hidden until a strictly newer operation appears.

### 9.2 Equal lamport from different nodes

- Resolve with `author_pubkey`, then `op_id`.
- Deterministic across all nodes.

### 9.3 Offline node reconnect with stale state

- Stale ops are ignored by compare function.
- Node converges to network-latest without reviving old data.

## 10. Implementation Rules (must-have)

1. One shared function for version compare across all CRUD paths.
2. No direct `INSERT OR REPLACE` that bypasses version checks.
3. Tombstone-aware upsert logic in:
   - incoming message processing,
   - anti-entropy apply,
   - digest upsert shortcuts.
4. Tests must include:
   - delete then stale update replay,
   - concurrent equal-lamport conflict,
   - offline reconnect anti-resurrection.

## 11. Recommended Rollout Plan

1. Phase C1: introduce shared compare utility and version fields normalization.
2. Phase C2: refactor post/comment create/update/delete apply paths to use compare utility only.
3. Phase C3: update digest/sync payloads and apply logic with tombstone metadata.
4. Phase C4: add regression suite and long-run multi-node replay test.

## 12. Non-Goals

1. This spec does not define global strong consistency.
2. This spec does not require central coordinator.
3. This spec does not define cryptographic moderation policy; it defines ordering and convergence behavior.
