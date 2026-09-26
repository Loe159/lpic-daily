# ADR 0006 — Pure-Go SQLite driver

Status: **Accepted — 2026-09-26**

## Decision

Use `modernc.org/sqlite` through Go's standard `database/sql` package for the Phase-1 local progress store.

Pinned baseline: `modernc.org/sqlite v1.59.0`.

## Rationale

LPIC Daily is intended to ship as a straightforward Linux CLI/TUI binary. The modernc driver provides SQLite without requiring CGO, which keeps builds and distribution simpler than a CGO-backed driver.

The application still depends on SQLite's transactional semantics; using `database/sql` keeps the storage API isolated from the driver choice.

## Consequences

- `go.mod` and `go.sum` are committed and CI verifies `go mod tidy` is a no-op.
- Migrations are embedded and versioned.
- Opening a database newer than the binary fails closed.
- Mastery evidence and gamification events remain separate append-only logs.
- A future driver change must preserve migration compatibility and be recorded by ADR.
