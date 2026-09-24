# AGENTS.md

## Mission
Build LPIC Daily as a safe, terminal-first learning system that teaches transferable Linux administration skills and covers 100% of LPIC-1 v5.0 without teaching exam dumps.

## Read before changing anything
1. Read `docs/PRODUCT.md` and `docs/REQUIREMENTS.md`.
2. Read the ADR(s) that govern the area you will change.
3. For curriculum/content work, read `curriculum/AGENTS.md` and `docs/CONTENT_AUTHORING.md`.
4. For labs/isolation work, read `labs/AGENTS.md` and `docs/SECURITY.md`.
5. For application code, read `src/AGENTS.md` and `docs/ARCHITECTURE.md`.

Do not load every document by default. Keep context task-specific.

## Source-of-truth hierarchy
- Product requirements: `docs/REQUIREMENTS.md`
- Product behavior summary: `docs/PRODUCT.md`
- Architecture decisions: `docs/adr/`
- LPIC coverage: `curriculum/lpic-1-v5/objectives.json`
- Security boundaries: `docs/SECURITY.md` and `docs/THREAT_MODEL.md`
- Content rules: `docs/CONTENT_AUTHORING.md`
- Current execution plan: `docs/plan/`

If sources conflict, stop and resolve the conflict in documentation before implementation.

## Non-negotiable rules
- Never execute learner lab commands directly on the host as a fallback.
- Treat lab content as untrusted input.
- Keep the normal application unprivileged; privileged operations must be narrow, explicit and auditable.
- Every LPIC objective must remain traceable to its official objective ID and version.
- Do not copy or adapt LPI Learning Materials. Write original explanations and exercises from factual objectives and independently researched Linux documentation.
- Prefer deterministic state-based lab checks over matching command strings.
- Do not add AI tutor/generation to the initial scope. Core learning, hints, grading, lab execution and offline operation are deterministic.
- A change is incomplete until relevant tests/validators pass and docs/coverage are updated.

## Change workflow
Use the matching skill under `.agents/skills/` before substantial work. At minimum, run the repository validators/tests relevant to modified areas and report what was actually executed.

## Scope
Nested `AGENTS.md` files add or tighten instructions for their directory. The nearest applicable file wins for local details; this root file remains authoritative for project-wide invariants.
