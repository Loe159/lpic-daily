# LPIC-1 v5 curriculum model

Research baseline: 2026-09-24. Official current version at that date is 5.0/5.0.0 with exam codes 101-500 and 102-500.

## Canonical files

- `objectives.json` — 42 active objective IDs, exam/topic mapping, weights, broad concepts and terms.
- `prerequisites.json` — complete objective-level pedagogical DAG with hard and recommended edges.
- `concepts.json` — stable mastery IDs for the 295 current concept entries.
- `phase1-slice.json` — accepted first implementation slice.

Objective weights sum to 60 for each exam. Objective 104.4 is not an active v5 objective and is recorded only as removed metadata in `objectives.json`.

The concepts and assessment descriptions are original project decomposition, not copied lesson content. Future agents must verify the official objectives if LPI publishes a new exam version rather than silently updating these files.

Validation:

```bash
python3 scripts/validate_foundation.py
```
