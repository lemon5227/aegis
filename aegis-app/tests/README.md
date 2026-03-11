# Test File Policy

Purpose: keep the repository clean and make test intent explicit.

## Rules
- Permanent regression `_test.go` files are allowed in the repository root when they protect shipping gates.
- Keep package/layout compatibility with `package main`.
- One-off acceptance/debug tests must NOT stay in root after verification.
- Temporary test assets go under `aegis-app/tests/ad-hoc/` with a short description of:
  - why it exists,
  - how to run it,
  - when it should be removed.

## Naming
- Permanent regression tests: use `phase*_*.go` naming and keep scope explicit in the test name.
- Temporary test notes/scripts: place in `tests/ad-hoc/` and prefix with date or phase, e.g. `2026-02-a4-profile-sync.md`.

## Cleanup
- After temporary verification is done, either:
  1) promote into a permanent regression test with clear scope, or
  2) delete it.
