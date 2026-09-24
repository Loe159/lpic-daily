---
name: coverage-audit
description: Audit LPIC syllabus coverage and detect gaps
---

# coverage-audit


1. Run `python3 scripts/validate_curriculum.py`.
2. Compare objective IDs and weights with the current official LPI v5.0 objectives.
3. For each objective, inspect concept coverage across explanation, retrieval, lab and review modes.
4. Flag concepts that have only passive reading or only quiz coverage.
5. Preserve the distinction between certification-required legacy knowledge and current operational practice.
6. Produce gaps by objective ID; do not invent a numeric mastery claim unsupported by artifacts.

