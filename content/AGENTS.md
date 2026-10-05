# Content authoring instructions

Scope: `content/` only.

- Author original French-first educational material. Do not copy official LPI learning text or exam dumps.
- Preserve natural Linux/Unix technical terms in English when that is the normal vocabulary.
- Every artifact must reference stable objective IDs and concept IDs from `curriculum/lpic-1-v5/`.
- Lessons explain; questions test. Reading a lesson is exposure, never proof of mastery.
- Questions must be deterministically gradable offline from the declared grading strategy.
- Prefer one clear concept per recall question. Avoid trick wording and trivia not required by the mapped concept.
- Every schedulable concept needs course + daily quiz + machine-checked lab coverage. Do not use a recall question as a substitute for practical evidence.
- Split concepts that are too broad to exercise all independently examinable knowledge through those three surfaces.
- Do not encode a lab reference solution in a lesson or low-level hint.
- After changing content, refresh Phase-1 coverage when Phase-1 concepts are affected with `python3 scripts/generate_phase1_coverage.py --write`, run `python3 scripts/audit_phase3_coverage.py --check`, then run the full foundation validation.
