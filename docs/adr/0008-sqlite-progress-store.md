# ADR 0008 — SQLite progress persistence

Status: **Accepted — 2026-09-26**

## Decision

Use SQLite for local progress state through Go's `database/sql` API.

Use the pure-Go `modernc.org/sqlite` driver for Phase 1 so the core binary does not acquire a CGO build dependency merely for its progress database.

Persistence is hidden behind project-owned interfaces.

## Data model

The first migration separates:

- `mastery_evidence`: append-only learning evidence;
- `gamification_events`: XP/streak/achievement-related events.

The separation enforces the product invariant that gamification does not become mastery evidence by schema accident.

`PRAGMA user_version` tracks ordered migrations. Migration files are embedded into the binary.

## Rationale

- local/offline and transactional;
- easy backup/export;
- no daemon/account required;
- pure-Go driver simplifies builds;
- append-only evidence keeps mastery projections recalculable;
- project-owned interfaces leave room to replace the driver without changing learning logic.

## Constraints

- no update/delete API for mastery evidence in normal application flow;
- migrations are sequential and transactional;
- timestamps are stored as UTC RFC3339Nano strings in Phase 1;
- structured lists are JSON-encoded only at the storage boundary;
- DB errors must never cause fallback to ad-hoc files or host shell behavior.

## Future

Add explicit backup/export and corruption-recovery behavior before a stable release. Gamification repositories arrive when that feature is implemented.
