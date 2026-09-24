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

## Agent adapters
`CLAUDE.md`, `GEMINI.md`, Copilot instructions and future vendor-specific instruction files should point to canonical project rules instead of copying them. This limits drift.

## Agent-generated content
Treat generated lessons/labs like generated code: review sources, run validators, test scenarios, and verify factual claims. AI is an authoring accelerator, never an authority.
