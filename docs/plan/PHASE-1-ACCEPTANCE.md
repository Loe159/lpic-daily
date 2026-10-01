# Phase 1 — vertical-slice acceptance criteria

Status: **Accepted — 2026-09-27**

Validation baseline at acceptance:
- foundation/schema/coverage validation in CI;
- Go tests, vet and formatting on Ubuntu;
- complete Go core/content validation on Fedora 44;
- real rootless Podman destructive host-sentinel integration test;
- six built-in Phase-1 labs; all 22 concepts have two distinct lab contexts; generated coverage validation enforces that invariant.

Goal: prove the product loop and security model before scaling curriculum production.

Canonical slice: `docs/plan/PHASE-1-VERTICAL-SLICE.md` and `curriculum/lpic-1-v5/phase1-slice.json`.

## Scope

The selected objectives are:

- **103.1 — Travail en ligne de commande** (foundation);
- **103.5 — Processus et jobs**;
- **104.5 — Permissions et ownership**.

The slice is closed over hard prerequisites: 103.5 and 104.5 require 103.1, which is included.

All **22 concepts** currently mapped to these three objectives must have traceable coverage before Phase-1 exit.

## Product acceptance
- [x] Daily session can be built from curriculum metadata and progress state.
- [x] Session can mix review and one new progressive concept.
- [x] French-first lesson renders natural English technical terminology correctly.
- [x] Learner can request graduated hints up to an explicit solution/debrief.
- [x] XP/streak/achievement events are stored separately from mastery evidence.
- [x] Mastery cannot become complete from passive lesson completion alone.
- [x] Scheduler can explain why an item was selected and which prerequisites made it eligible.
- [x] Objective coverage remains traceable to official LPIC IDs.
- [x] Concept-level coverage remains traceable to stable concept IDs.
- [x] Content can label `lpic-required`, `lpic-legacy`, and `modern-practice` where applicable.
- [x] All 22 slice concepts appear in the coverage matrix.
- [x] Every Phase-1 concept has exactly one focused `introduce` lesson and at least one deterministic daily non-assessment question; initial-assessment questions cannot satisfy daily coverage. Recognition questions remain weaker mastery evidence than free recall.
- [x] Every Phase-1 concept has at least two labs with distinct machine-checked `practice_context` values so independent evidence can later become transfer evidence.

## Technical acceptance
- [x] Go application builds and tests on Fedora.
- [x] TUI uses Bubble Tea v2.
- [x] SQLite schema has versioned migrations.
- [x] Curriculum/content schemas are versioned and fail closed on invalid input.
- [x] Rootless Podman runner implements prepare/start/check/reset/destroy lifecycle.
- [x] Mutable local image tags are resolved to an immutable image ID before create; reset keeps the same ID.
- [x] State-based checker accepts at least two different valid command paths to the same correct final state.
- [x] PTY path supports the 103.5 jobs/process scenario.
- [x] No host-shell execution fallback exists.
- [x] No Internet access is required for the complete vertical slice after dependencies/images are installed.

## Security acceptance
- [x] Lab fails closed when Podman is unavailable.
- [x] Tests reject host-path mounts and privileged/capability escalation outside predefined profiles.
- [x] Default lab network is disabled or explicitly isolated.
- [x] CPU/memory/PID/time constraints exist in runner configuration.
- [x] Host filesystem safety test runs a destructive lab and verifies a sentinel outside the sandbox is unchanged.
- [x] Untrusted output rendered in the TUI has an explicit control-sequence handling policy and tests.
- [x] `setup.execution_scope` cannot be changed from `sandbox`.

## Learning-model acceptance
- [x] Evidence is append-only and mastery is a derived projection.
- [x] A lesson completion only records exposure.
- [x] Practical full mastery requires independent practical evidence and later confirmation/transfer.
- [x] Once the transfer gap is satisfied, the study plan prefers an unused lab context when available.
- [x] Solution reveal prevents that attempt from counting as independent practice.
- [x] Initial assessment can satisfy prerequisites without fabricating lesson completion.
- [x] Scheduler uses hard prerequisites as eligibility and recommended prerequisites as ranking only.

## Agent/repository acceptance
- [x] `AGENTS.md` instructions are sufficient for a fresh coding agent to discover product, architecture, security, schemas and validation rules without chat history.
- [x] Repetitive validation workflow is executable through scripts/skills.
- [x] CI runs curriculum/graph validation, Go tests, format/lint checks and security-invariant tests.
- [x] ADR status matches actual implementation choices.

## Explicitly out of Phase 1
- libvirt/QEMU implementation;
- complete LPIC content;
- AI tutor/generation;
- full exam simulator;
- public Internet-enabled labs;
- Debian/openSUSE host packaging.
