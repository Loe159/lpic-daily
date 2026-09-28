# Application-code agent instructions

Application code is active. Preserve the module boundaries in `docs/ARCHITECTURE.md`, accepted decisions in `docs/adr/`, and Phase-1 learning/security invariants when extending later phases.

- Keep the main process unprivileged.
- Isolate side effects behind interfaces so runners/checkers are testable.
- Do not add a host-shell lab backend or fallback.
- Prefer typed/structured arguments over shell-string construction.
- Preserve disclosure, mastery-evidence, sandbox and cleanup invariants across retries/restarts.
- Update tests and relevant architecture docs with behavioral changes.
