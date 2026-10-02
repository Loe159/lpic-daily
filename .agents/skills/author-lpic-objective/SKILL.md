---
name: author-lpic-objective
description: Create or revise original LPIC curriculum content
---

# author-lpic-objective


1. Locate the objective in `curriculum/lpic-1-v5/objectives.json`.
2. Confirm objective/version/weight against the official LPI objectives page.
3. Read `docs/CONTENT_AUTHORING.md`, `docs/CONTENT_STYLE.md`, and the corresponding topic file in `docs/lpic1/`.
4. Teach concepts and mental models before command memorization.
5. Mark legacy exam knowledge separately from recommended modern practice.
6. Add retrieval questions and practical evidence that map back to concepts.
7. Write all explanatory prose originally; do not adapt LPI Learning Materials.
8. Ensure every newly schedulable concept has exactly one focused `introduce` lesson and at least one non-`initial-assessment` deterministic daily question.
9. Run `python3 scripts/validate_curriculum.py`, `python3 scripts/audit_phase3_coverage.py --check`, and the relevant Go/coverage tests after edits.

