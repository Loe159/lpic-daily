# Application-code agent instructions

Implementation has not started. When Phase 1 begins, preserve module boundaries in `docs/ARCHITECTURE.md` and decisions in `docs/adr/`.

- Keep the main process unprivileged.
- Isolate side effects behind interfaces so runners/checkers are testable.
- Do not add a host-shell lab backend.
- Prefer typed/structured arguments over shell-string construction.
- Update tests and relevant architecture docs with behavioral changes.
