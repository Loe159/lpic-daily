# Curriculum agent instructions

Read `docs/CONTENT_AUTHORING.md`, `docs/CONTENT_STYLE.md`, `docs/CONTENT_MODEL.md`, `docs/CURRICULUM_GRAPH.md` and the appropriate `docs/lpic1/topic-*.md` before changing curriculum data.

Canonical files:
- `lpic-1-v5/objectives.json`: objective inventory and weights;
- `lpic-1-v5/prerequisites.json`: objective-level pedagogical graph;
- `lpic-1-v5/concepts.json`: immutable concept IDs;
- `schemas/`: content contracts.

Rules:
- Keep official IDs and weights exact.
- Use original explanatory language; do not adapt LPI lesson prose.
- Every active objective needs concepts, examinable terms, assessment modes and a recommended isolation backend.
- Tag legacy knowledge rather than deleting it when it remains in LPIC v5.
- Never rename an existing concept ID because its French wording changed.
- New hard prerequisites require rationale and must preserve DAG validity.
- Do not add concept-level dependencies speculatively; add them when authored content demonstrates a real dependency.

Run `python3 scripts/validate_foundation.py` after any curriculum/schema change.
