# Application-code agent instructions

Phase 1 implementation is active. Go application code currently lives under `cmd/` and `internal/`; this legacy `src/` scope is retained only as a guardrail if code is later placed here.

Read `docs/ARCHITECTURE.md`, `docs/CONTENT_MODEL.md`, `docs/LEARNING_MODEL.md` and relevant ADRs before changing application behavior.

- Target Go 1.27 unless an ADR explicitly changes the toolchain baseline.
- Keep the main process unprivileged.
- Isolate side effects behind interfaces so runners/checkers are testable.
- Do not add a host-shell lab backend.
- Prefer typed/structured arguments over shell-string construction.
- Treat mastery evidence as append-only; projections are derived.
- Keep gamification storage/logic separate from mastery.
- Update tests and relevant architecture/plan docs with behavioral changes.
