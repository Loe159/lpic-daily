# Curriculum agent instructions

Read `docs/CONTENT_AUTHORING.md` and `docs/CONTENT_STYLE.md` and the appropriate `docs/lpic1/topic-*.md` before changing curriculum data.

- `lpic-1-v5/objectives.json` is the canonical objective inventory.
- Keep official IDs and weights exact.
- Use original explanatory language; do not adapt LPI lesson prose.
- Every active objective needs concepts, examinable terms, assessment modes and a recommended isolation backend.
- Tag legacy knowledge rather than deleting it when it remains in LPIC v5.
- Run `python3 scripts/validate_curriculum.py` after any change.
