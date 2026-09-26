---
name: coverage-audit
description: Audit LPIC syllabus, graph and concept coverage and detect gaps
---

# coverage-audit

1. Run `python3 scripts/validate_foundation.py`.
2. Compare objective IDs and weights with the current official LPI v5.0 objectives when syllabus freshness is part of the task.
3. Verify every active objective has one graph node and stable concepts.
4. For each objective, inspect concept coverage across explanation, retrieval, lab and review modes.
5. Flag concepts that have only passive reading or only quiz coverage.
6. Preserve the distinction between certification-required legacy knowledge and current operational practice.
7. Check any vertical slice is closed over hard prerequisites.
8. Produce gaps by objective/concept ID; do not invent a numeric mastery claim unsupported by artifacts.
