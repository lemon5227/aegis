# Frontend-Backend Lamport Integration Plan

Date: 2026-02-20
Status: Execution plan after backend Lamport CRUD completion

## 1. Context

Backend Lamport CRUD consistency for post/comment is now implemented (operation ordering, tombstone anti-resurrection, sync payload version fields, operation audit log, soak tests).

This document defines how frontend should connect to the completed backend capabilities without breaking current UX.

## 2. Backend Completion Checklist (Spec Alignment)

Against `docs/LAMPORT_CRUD_CONSISTENCY_SPEC_2026-02-20.md`, backend has the following done:

1. Deterministic compare key implemented globally: `lamport -> author_pubkey -> op_id`.
2. Unified CRUD semantics for post/comment with authorization checks in apply path.
3. Tombstone fields and anti-resurrection behavior in direct apply + digest apply + sync apply.
4. Sync payload metadata present: `op_id`, `op_type`, `schema_version`, `auth_scope`, `deleted_at_lamport`.
5. Idempotency by operation identity + version compare on all entity handlers.
6. Append-only entity op log implemented (`entity_ops`) and wired in apply paths.
7. Regression and multi-node soak tests implemented and passing.
8. Tombstone GC mechanism implemented with retention + stability-pass gating.

## 3. Frontend Current State (Gap Analysis)

Current UI paths (`frontend/src/App.tsx`, `PostDetail.tsx`, `CommentTree.tsx`, `SettingsPanel.tsx`) are mostly feed/comment oriented and do not expose Lamport-level diagnostics.

Main gaps:

1. No entity operation timeline view (`entity_ops`) in UI.
2. No per-post/comment consistency metadata display (`current_op_id`, `lamport`, tombstone state).
3. No admin/operator controls for tombstone GC execution/report.
4. No explicit anti-entropy/convergence inspection panel using Lamport metrics.
5. Frontend types do not include new debug/consistency fields.

## 4. Integration Contract (Backend APIs to Use)

### 4.1 New/Existing backend APIs for frontend

1. `ListEntityOps(entityType, entityID, limit)`
   - Purpose: operation timeline for debugging and moderation audit.
2. `RunTombstoneGC(retentionDays, requiredStablePasses, batchSize)`
   - Purpose: controlled tombstone cleanup in settings/admin tool.
3. `GetAntiEntropyStats()` (already available)
   - Purpose: convergence health and sync lag display.
4. Existing feed/comments endpoints remain the primary user-facing data source.
5. Voting APIs
   - `PublishPostUpvote(pubkey, postID)`
   - `PublishPostDownvote(pubkey, postID)`
   - `PublishCommentUpvote(pubkey, postID, commentID)`
   - `PublishCommentDownvote(pubkey, postID, commentID)`
   - Purpose: local-first vote apply + async network broadcast.
6. Favorites APIs (private scope)
   - `AddFavorite(postID)`, `RemoveFavorite(postID)`, `IsFavorited(postID)`, `GetFavorites(limit, cursor)`
   - Purpose: private/local favorite state; optional cross-device sync via signed favorite ops.

### 4.2 Event model

Reuse existing events:

1. `feed:updated`
2. `comments:updated`
3. `subs:updated`

Frontend diagnostics panels should poll on interval and optionally refresh on these events.

## 5. Planned Frontend Work (Implementation Phases)

### FE1 - Type and API wiring

1. Regenerate Wails bindings so `App.d.ts` includes new APIs.
2. Extend frontend types with:
   - `EntityOpRecord`
   - `TombstoneGCResult`
   - optional consistency metadata for detail panels.
3. Add API client wrappers used by UI modules.

Acceptance:

1. Typecheck passes with no `any` fallback for new APIs.
2. New methods callable from UI without runtime binding errors.

### FE2 - Consistency Diagnostics UI

1. Add a new Settings tab: `Consistency`.
2. Show anti-entropy stats:
   - requests/responses, insertions, fetch success/failure, observed lag.
3. Add operation timeline panel:
   - filters: entity type (`post`/`comment`), entity id, limit.
   - sorted by Lamport/version order (backend already sorted).
4. Add quick inspector for selected post/comment:
   - latest op id
   - lamport
   - deleted/tombstone indicators.

Acceptance:

1. Timeline reflects backend op order deterministically.
2. Deleted entities show tombstone context rather than disappearing silently in diagnostics view.

### FE3 - Tombstone GC Operator Controls

1. Add guarded action card in `Consistency` tab for `RunTombstoneGC`.
2. Inputs:
   - retention days (default 30)
   - stable passes (default 2)
   - batch size (default 200)
3. Output render:
   - scanned/deleted posts/comments counters.
4. Add two-step confirm UI to avoid accidental cleanup execution.

Acceptance:

1. GC action returns visible result and handles errors cleanly.
2. User cannot trigger dangerous operation with invalid parameters.

## 6. Validation Plan

1. Backend tests: `go test ./...` must pass.
2. Soak tests:
   - quick: `scripts/test_lamport_3node_soak.sh`
   - long run: `scripts/run_lamport_soak_24h.sh` (default 24h, configurable via `LAMPORT_SOAK_HOURS`).
3. Frontend smoke after integration:
   - create/update/delete post/comment flows still work.
   - consistency tab can query op timeline and GC endpoint.

## 7. Non-Goals for This Integration Slice

1. No redesign of main feed UX.
2. No protocol changes beyond already completed backend semantics.
3. No centralized coordinator or strong-consistency behavior.

## 8. Implementation Order

1. FE1 type/API wiring.
2. FE2 diagnostics view.
3. FE3 GC operator controls.
4. Run validation plan and close rollout.

## 9. Post Publish Latency Note (Backend)

To avoid long UI `Posting...` stalls when pubsub is slow/backpressured:

1. Post/comment create and delete write local state first (authoritative local apply).
2. Network broadcast is performed asynchronously with bounded timeout (best effort).
3. Local success is returned immediately; replication convergence is handled by pubsub + anti-entropy.

Additional local-first paths now aligned:

1. Post/comment upvote and downvote.
2. Favorite operation publish (favorite state remains local-private even if broadcast fails).

Frontend implication for Gemini:

1. Keep current loading UX, but expect faster resolve after local apply.
2. Treat eventual network replication as background behavior; no need to block UI on remote fanout completion.

## 10. Timeline Guide (How To Read)

Timeline is a developer/operator debugging view backed by `entity_ops`. It is not a user-facing feature.

### 10.1 What timeline row means

Each row is one operation on an entity (post/comment):

1. `op_type`: `CREATE` / `UPDATE` / `DELETE`
2. `entity_type`, `entity_id`: target entity
3. `lamport`: logical clock at apply time
4. `author_pubkey`: operation author
5. `op_id`: unique operation identity (idempotency + tie-break)
6. `payload_json`: extra context (sub id, source, delete marker, etc.)

### 10.2 Ordering rule (critical)

Do not interpret order by wall clock alone. Effective order is:

1. `lamport` (higher wins)
2. if equal, `author_pubkey` lexicographic compare
3. if still equal, `op_id` lexicographic compare

### 10.3 Fast troubleshooting checklist

1. Find latest row for entity.
2. If latest `op_type=DELETE`, expected visible state is tombstoned/deleted.
3. If a new op appears but state does not change, compare its version key against current winner (it may be stale).
4. If operation is missing, check whether it was rejected by auth check or never arrived via pubsub/anti-entropy.

### 10.4 Example interpretation

Rows:

1. `CREATE lamport=10`
2. `UPDATE lamport=12`
3. `DELETE lamport=15`
4. `UPDATE lamport=14` (late arrival)

Final state remains deleted because `14 < 15`.

## 11. Dev-Only Visibility Policy (Timeline Hidden In Production)

Timeline must be visible only in dev mode.

### 11.1 Backend enforcement

1. `ListEntityOps` is now gated by dev mode and returns error when disabled.
2. Dev mode switch is environment variable: `AEGIS_DEV_MODE`.
3. Enable with one of: `1`, `true`, `yes`, `on`, `dev`.

### 11.2 Frontend behavior for Gemini

1. Call `IsDevMode()` at app start.
2. Only show `Consistency` timeline UI when `IsDevMode() == true`.
3. In non-dev mode, hide timeline entry points (settings tab button, detail-page shortcut).
4. If API returns dev-only error, show quiet fallback message and no retry loop.

## 12. Voting/Favorites Frontend Wiring Notes (For Gemini)

### 12.1 Voting behavior

1. Upvote/downvote are separate operations and mutually exclusive per `(entity, voter_pubkey)`.
2. Backend score update rules:
   - fresh upvote: `+1`
   - switch downvote -> upvote: `+2`
   - fresh downvote: `-1`
   - switch upvote -> downvote: `-2`
3. Frontend should avoid hardcoded `+1` optimistic updates and instead refresh item score from backend state after action.
4. Vote broadcast is debounced and state-based in backend (not per-click immediate fanout):
   - local apply happens immediately
   - network sends compact `*_VOTE_SET` with final state (`UP`/`DOWN`/`NONE`)
   - debounce window defaults to `600ms` (`AEGIS_VOTE_BROADCAST_DEBOUNCE_MS`)

### 12.2 Favorites behavior

1. Favorites are private user data (local scope by default).
2. Favorites are persisted in backend tables (`post_favorites_state`, `post_favorite_ops`), not browser localStorage.
3. Frontend should replace localStorage-based favorites UI with backend APIs:
   - toggle by `AddFavorite` / `RemoveFavorite`
   - badge state by `IsFavorited`
   - favorites page data from `GetFavorites`
