# AGENTS.md

## Mission
Build LPIC Daily as a safe, terminal-first learning system that teaches transferable Linux administration skills and covers 100% of LPIC-1 v5.0 without teaching exam dumps.

## Read before changing anything
1. Read `docs/PRODUCT.md` and `docs/REQUIREMENTS.md`.
2. Read the ADR(s) that govern the area you will change.
3. For curriculum/content work, read `curriculum/AGENTS.md`, `docs/CONTENT_AUTHORING.md`, `docs/CONTENT_MODEL.md` and `docs/CURRICULUM_GRAPH.md`.
4. For labs/isolation work, read `labs/AGENTS.md` and `docs/SECURITY.md`.
5. For application code, read `docs/ARCHITECTURE.md` plus the ADRs governing the touched module.

Do not load every document by default. Keep context task-specific.

## Source-of-truth hierarchy
- Product requirements: `docs/REQUIREMENTS.md`
- Product behavior summary: `docs/PRODUCT.md`
- Architecture decisions: `docs/adr/`
- LPIC objective coverage: `curriculum/lpic-1-v5/objectives.json`
- Learning prerequisites: `curriculum/lpic-1-v5/prerequisites.json`
- Stable concept IDs: `curriculum/lpic-1-v5/concepts.json`
- Content contracts: `schemas/`
- Security boundaries: `docs/SECURITY.md` and `docs/THREAT_MODEL.md`
- Content rules: `docs/CONTENT_AUTHORING.md`
- Current execution plan: `docs/plan/`

If sources conflict, resolve the conflict in canonical documentation before implementation.

## Non-negotiable rules
- Never execute learner lab commands directly on the host as a fallback.
- Treat lab content as untrusted input.
- Keep the normal application unprivileged; privileged operations must be narrow, explicit and auditable.
- Every LPIC objective must remain traceable to its official objective ID and version.
- Concept IDs are immutable once published.
- Do not copy or closely adapt LPI Learning Materials.
- Prefer deterministic state-based lab checks over command matching.
- Mastery is derived from append-only evidence; gamification never counts as mastery.
- Core learning, hints, grading and lab execution remain deterministic and offline-capable.
- A change is incomplete until relevant tests/validators pass and docs/coverage are updated.

## Application-code rules
- Preserve the module boundaries in `docs/ARCHITECTURE.md`.
- Isolate side effects behind interfaces.
- Do not add a host-shell lab backend or fallback.
- Prefer structured argv over shell-string construction.
- Preserve disclosure, evidence, sandbox and cleanup invariants across retries/restarts.
- Update tests and architecture/security documentation with behavioral changes.

## Change workflow
Use the matching skill under `.agents/skills/` before substantial work.

Run at minimum:

```bash
python3 scripts/validate_foundation.py
```

plus implementation-specific tests. Report what was actually executed.

## Scope
Nested `AGENTS.md` files may add stricter rules for their directory. The nearest applicable file wins for local details; this root file remains authoritative for project-wide invariants.
