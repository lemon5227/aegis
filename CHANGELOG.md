# Changelog

All notable changes to Aegis are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - Quality + Personal Features Iteration

This iteration focused on production-readiness: fixing flaky tests, improving
test coverage from 32% to 50%+, adding ESLint to the frontend, fixing a
significant SQLite connection-pool bug, and adding new client-side personal
features (mute users, post read tracking, user preferences).

### Added

#### Personal Features (client-side, not synced via P2P)

A new file `aegis-app/db_personal.go` introduces three feature areas that stay
local to a single device and do not participate in P2P consensus:

- **Mute users** — hide content from specific pubkeys without triggering a
  network-wide shadow-ban.
  - `MuteUser(pubkey, reason)`, `UnmuteUser(pubkey)`, `IsMuted(pubkey)`
  - `GetMutedUsers()`, `GetMutedPubkeys()` (returns a set for fast filtering)
- **Post read tracking** — mark posts as read, count unread per sub.
  - `MarkPostRead(postID)`, `MarkPostsRead([]postIDs)` (batched)
  - `IsPostRead(postID)`, `GetReadPostIDs()`
  - `GetUnreadPostCount(subID)`, `ClearReadHistory()`
- **User preferences** — arbitrary key-value preferences (theme, layout, etc.)
  - `SetUserPreference(key, value)`, `GetUserPreference(key)`
  - `DeleteUserPreference(key)`, `GetAllUserPreferences()`
  - Validates: 64-char key limit, 4 KiB value limit

Three new tables added via additive schema migration:
- `muted_users` — local mute list
- `post_reads` — local read tracking
- `user_preferences` — local key-value preferences

#### Documentation

- `CONTRIBUTING.md` — full developer guide covering project structure,
  testing, code style, architecture decisions, and common tasks.
- `aegis-app/README.md` — added Testing section with explicit commands for
  unit tests, P2P integration tests, and coverage reports.
- `aegis-app/README.md` — documented new personal features and APIs.

#### Frontend Tooling

- `aegis-app/frontend/.eslintrc.json` — ESLint configuration with
  `@typescript-eslint`, `react-hooks`, and `react-refresh` plugins.
- `aegis-app/frontend/package.json` — added `lint` and `lint:fix` scripts.
- `aegis-app/frontend/src/lib/useToasts.ts` — extracted from `Toast.tsx` to
  satisfy `react-refresh/only-export-components` rule.

#### Test Suites

68 new unit tests across 5 files:

- `core_test.go` (14 tests) — identity, posts, comments, favorites,
  governance, settings, sub-stats.
- `core_extended_test.go` (29 tests) — profile, privacy, voting, feed
  stream, storage, P2P config, votes, my posts, search, etc.
- `process_message_test.go` (12 tests) — `ProcessIncomingMessage` for all
  message types: POST, COMMENT, PROFILE_UPDATE, SUB_CREATE, votes,
  deletes, governance.
- `helpers_test.go` (17 tests) — pure helper functions:
  `parsePeerAddressesCSV`, `normalizePeerAddresses`, `isPublicIPv4`,
  `normalizeAuthScope`, `computeHotScore`, `voteDelta`, recommendation
  strategies, etc.
- `db_personal_test.go` (20 tests) — full coverage of new mute, read
  tracking, and preferences features.

### Fixed

#### Critical: SQLite Connection-Pool PRAGMA Bug

`PRAGMA busy_timeout` and other PRAGMAs were only being applied to the first
connection in Go's `sql.DB` connection pool, not all connections. This caused
intermittent `SQLITE_BUSY` errors under concurrent load.

Fixed in `db_schema.go`:
- PRAGMAs now set via connection-string URI parameters
  (`?_pragma=busy_timeout(10000)&...`) so they apply to every pooled connection.
- Increased `busy_timeout` from 5 s to 10 s.
- Added `synchronous=NORMAL` for better WAL performance.
- Limited connection pool: `MaxOpenConns=8`, `MaxIdleConns=4` to reduce
  write contention.

This fix dramatically improved test stability and will improve production
reliability under concurrent user actions.

#### P2P Test Stability

Three-node and multi-node P2P tests had ~30% flake rate due to libp2p TLS
handshake races between explicit `ConnectPeer` calls and concurrent mDNS
discovery. Fixed in:

- `three_node_verification_test.go` — rewrote `connectThreeNodes` with:
  - One-directional connections (libp2p connections are bidirectional, so
    A→B implies B can reach A; doing both directions caused races).
  - 2-second mDNS settle delay before explicit connects.
  - `connectPeerWithBackoffClear` retry helper that waits longer than
    libp2p's 5-second dial backoff.
- `phase_g7_trust_reliability_test.go` — applied same retry pattern.

Result: 5/5 consecutive full-suite runs now pass (previously 5-7/10).

#### Test Cleanup

WAL/SHM files weren't being cleaned up before `t.TempDir()` removal,
causing intermittent `directory not empty` errors. Fixed by running
`PRAGMA wal_checkpoint(TRUNCATE)` before `db.Close()` in test cleanup
helpers.

#### Frontend Bug

`MyPosts.tsx` had a `return` inside a `finally` block (caught by ESLint's
`no-unsafe-finally`). Fixed to use a guarded `if` instead.

#### Minor Frontend Cleanup

- Removed unused variables (`loading`, `payload`, `governanceAdmins`,
  `showPrivateKey`, `subs`).
- Fixed `Toast.tsx` Fast Refresh warning by extracting `useToasts` hook.

### Changed

- `db_schema.go` — added `muted_users`, `post_reads`, `user_preferences`
  tables via additive migration.
- All test helpers (`newTestApp`, `newTestAppWithIdentity`, `newG7TestApp`)
  now WAL-checkpoint before close to prevent cleanup issues.
- Test files use new `retryOnBusy` helper for SQLite-sensitive operations.

### Quality Metrics

| Metric | Before | After |
|---|---|---|
| Test coverage | 32.4% | 49.7% (~50%) |
| Total tests | ~10 | 100+ |
| Flaky tests | 3-5 P2P, intermittent | 0 (5/5 consecutive runs pass) |
| Frontend ESLint errors | not configured | 0 errors, 8 warnings |
| `go vet` | passes | passes |
| `tsc --noEmit` | passes | passes |

### Migration Notes

The new tables (`muted_users`, `post_reads`, `user_preferences`) are added
via `CREATE TABLE IF NOT EXISTS` in `ensureSchema`, so existing databases
will automatically gain them on next startup with no data loss.

The connection-pool PRAGMA fix is fully backwards compatible — it just means
all connections now have the correct timeout, not just the first one.

### Files Modified

```
.gitignore                                       (coverage.out exclusion)
README.md                                        (CONTRIBUTING.md reference)
CONTRIBUTING.md                                  (new)
aegis-app/README.md                              (testing + personal features)
aegis-app/db_schema.go                           (PRAGMA fix + new tables)
aegis-app/db_personal.go                         (new: 404 lines)
aegis-app/db_personal_test.go                    (new: 414 lines)
aegis-app/core_test.go                           (new: 419 lines)
aegis-app/core_extended_test.go                  (new: 537 lines)
aegis-app/helpers_test.go                        (new: 393 lines)
aegis-app/process_message_test.go                (new: 377 lines)
aegis-app/three_node_verification_test.go        (P2P retry logic)
aegis-app/phase_g7_trust_reliability_test.go     (P2P retry + retryOnBusy)
aegis-app/frontend/.eslintrc.json                (new)
aegis-app/frontend/package.json                  (lint scripts + deps)
aegis-app/frontend/package-lock.json             (ESLint deps)
aegis-app/frontend/src/App.tsx                   (import + cleanup)
aegis-app/frontend/src/components/MyPosts.tsx    (no-unsafe-finally fix)
aegis-app/frontend/src/components/SettingsPanel.tsx  (unused vars)
aegis-app/frontend/src/components/Sidebar.tsx    (unused var)
aegis-app/frontend/src/components/Toast.tsx      (extract useToasts)
aegis-app/frontend/src/lib/useToasts.ts          (new)
```
