# Working with coding agents

## Philosophy
Agents perform better when the repository contains a small global contract and discoverable, scoped context. Do not turn `AGENTS.md` into a handbook. Put durable domain knowledge in canonical docs, reusable procedures in skills, and machine-checkable contracts in schemas/tests.

## Context loading
For each task, an agent should load only:
- root `AGENTS.md`;
- nearest nested `AGENTS.md`;
- relevant ADR(s);
- the specific design/content docs needed for the change.

## Plans
Large changes should create/update a plan in `docs/plan/` containing goal, invariants, affected modules, risks, validation, and completion state. Plans are execution artifacts; ADRs record durable decisions.

## LPIC scenario authoring and review
- Canonical migration rules: `docs/plan/SCENARIO-LABS-MIGRATION.md`; agent workflow: `.agents/skills/author-lab/SKILL.md`; scope policy: `labs/AGENTS.md`.
- Review a lab as a learner would: Is there a concrete symptom? Can the student investigate without being told which command to run? Does the final state demonstrate the target competency? Do all hints unlock progressively?
- Review wording from both sides: without reading the checker, can the learner know **exactly** which observable facts and output format are required? Without the reference solution, do they still have to choose the appropriate commands and options? Reject both vague contracts and command-level spoilers in `brief_fr` and `success_criteria_fr`.
- Review the grader adversarially: a marker file or copied statement must not bypass Linux behavior; legitimate alternate solutions must pass; and every claimed concept must have a relevant check.
- `accepted` is the publication flag used to expose authored scenarios to learners for progressive field testing. It is **not** proof of automated or manual conformance. A PR must separately state actual setup-red/reference-green/reset-red, Podman/KVM runs, CI state and known limitations; failures remain blocking for merges even after a scenario is published.
- For a large migration, use small coherent objective groups and update the coverage matrix in the same commit. Keep legacy evidence backwards-compatible.

## Agent adapters
`CLAUDE.md`, `GEMINI.md`, Copilot instructions and future vendor-specific instruction files should point to canonical project rules instead of copying them. This limits drift.

## Agent-generated content
Treat generated lessons/labs like generated code: review sources, run validators, test scenarios, and verify factual claims. AI is an authoring accelerator, never an authority.
