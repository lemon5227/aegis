# Test File Policy

Purpose: keep the repository clean and make test intent explicit.

## Rules
- Long-term regression test sources are stored under `aegis-app/tests/regression/`.
- Root-level `phase*_test.go` are symlink entrypoints so `go test ./...` still runs against `package main`.
- One-off acceptance/debug tests must NOT stay in root after verification.
- Temporary test assets go under `aegis-app/tests/ad-hoc/` with a short description of:
  - why it exists,
  - how to run it,
  - when it should be removed.

## Naming
- Permanent regression tests: source file name keeps `phase*_*.go` pattern under `tests/regression/`.
- Temporary test notes/scripts: place in `tests/ad-hoc/` and prefix with date or phase, e.g. `2026-02-a4-profile-sync.md`.

## Cleanup
- After temporary verification is done, either:
  1) promote into a permanent regression test with clear scope, or
  2) delete it.
