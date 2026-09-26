# Phase 1 implementation plan

Status: **In progress**

Branch: `phase1/core-foundation`

## Goal

Deliver the vertical slice defined in `PHASE-1-VERTICAL-SLICE.md` without weakening the Phase-1 acceptance or security contracts.

## Workstream A — core curriculum + learning engine

Status: **implemented and CI green**

- Go 1.27 module;
- strict loader for objectives, graph, concepts and Phase-1 slice;
- cross-file drift and DAG validation in Go;
- append-only evidence domain model;
- mastery projection stages;
- deterministic scheduler with:
  - hard prerequisites as eligibility;
  - recommended prerequisites as ranking;
  - objective weight;
  - due reviews;
  - one new concept per session;
- CLI smoke commands: `lpic validate` and `lpic plan`.

## Workstream B — executable SQLite store

Status: **implemented, validation in progress**

- pure-Go SQLite driver via `modernc.org/sqlite`;
- XDG-aware local database path;
- migration runner with fail-closed future-version check;
- evidence append/query API;
- gamification append API kept separate from mastery;
- transactional migration application;
- database triggers preventing UPDATE/DELETE of evidence and gamification history;
- round-trip, append-only, separation, migration-idempotency and future-schema tests.

## Workstream C — content runtime

Status: **next**

- JSON Schema validation in Go;
- lesson/question/hint/lab loaders;
- coverage matrix for all 22 slice concepts;
- original French Phase-1 learning content.

## Workstream D — Podman runner

Status: **planned**

- rootless availability/doctor check;
- fail-closed lifecycle;
- resource limits;
- network-none default;
- no arbitrary host mounts;
- structured checker primitives;
- PTY support for jobs/process exercises;
- host-sentinel security tests.

## Workstream E — learner UX

Status: **planned**

- Bubble Tea v2 dependency and TUI shell;
- daily dashboard/session flow;
- lesson/question rendering;
- hints/debrief;
- progress/mastery explanation;
- Fedora desktop notification adapter.

## Workstream F — reference labs + end-to-end

Status: **planned**

- `shell-environment-repair`;
- `stuck-worker`;
- `shared-dropbox`;
- at least two valid command paths accepted where appropriate;
- restart/interruption/reset tests;
- end-to-end daily session smoke.

## Implementation invariants

- No host-shell lab backend.
- No Internet by default.
- No mastery mutation from XP/streak/achievement.
- No solution-revealed attempt may count as independent practice.
- Curriculum drift fails closed.
- Phase 1 remains limited to 103.1, 103.5 and 104.5 until the vertical slice passes acceptance.

## Merge strategy

Keep Phase 1 in reviewable commits. Do not merge a later workstream to `main` while its security-critical tests are red. The final Phase-1 completion merge requires every checkbox in `PHASE-1-ACCEPTANCE.md`.
