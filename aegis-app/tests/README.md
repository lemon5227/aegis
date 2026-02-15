# Test File Policy

Purpose: keep the repository clean and make test intent explicit.

## Rules
- Long-term regression tests that must run with `go test` and rely on `package main` stay in `aegis-app/` root.
- One-off acceptance/debug tests must NOT stay in root after verification.
- Temporary test assets go under `aegis-app/tests/ad-hoc/` with a short description of:
  - why it exists,
  - how to run it,
  - when it should be removed.

## Naming
- Permanent regression tests: `phase*_*.go` in root, with clear test name and purpose comment.
- Temporary test notes/scripts: place in `tests/ad-hoc/` and prefix with date or phase, e.g. `2026-02-a4-profile-sync.md`.

## Cleanup
- After temporary verification is done, either:
  1) promote into a permanent regression test with clear scope, or
  2) delete it.
