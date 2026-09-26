# Content authoring instructions

Scope: `content/` only.

- Author original French-first educational material. Do not copy official LPI learning text or exam dumps.
- Preserve natural Linux/Unix technical terms in English when that is the normal vocabulary.
- Every artifact must reference stable objective IDs and concept IDs from `curriculum/lpic-1-v5/`.
- Lessons explain; questions test. Reading a lesson is exposure, never proof of mastery.
- Questions must be deterministically gradable offline from the declared grading strategy.
- Prefer one clear concept per recall question. Avoid trick wording and trivia not required by the mapped concept.
- Do not encode a lab reference solution in a lesson or low-level hint.
- After changing content, run `python3 scripts/generate_phase1_coverage.py --write` and the full foundation validation.
