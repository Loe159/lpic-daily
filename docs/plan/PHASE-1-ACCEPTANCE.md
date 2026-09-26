# Phase 1 — vertical-slice acceptance criteria

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
- [ ] Daily session can be built from curriculum metadata and progress state.
- [ ] Session can mix review and one new progressive concept.
- [ ] French-first lesson renders natural English technical terminology correctly.
- [ ] Learner can request graduated hints up to an explicit solution/debrief.
- [ ] XP/streak/achievement events are stored separately from mastery evidence.
- [ ] Mastery cannot become complete from passive lesson completion alone.
- [ ] Scheduler can explain why an item was selected and which prerequisites made it eligible.
- [ ] Objective coverage remains traceable to official LPIC IDs.
- [ ] Concept-level coverage remains traceable to stable concept IDs.
- [ ] Content can label `lpic-required`, `lpic-legacy`, and `modern-practice` where applicable.
- [ ] All 22 slice concepts appear in the coverage matrix.

## Technical acceptance
- [ ] Go application builds and tests on Fedora.
- [ ] TUI uses Bubble Tea v2.
- [ ] SQLite schema has versioned migrations.
- [ ] Curriculum/content schemas are versioned and fail closed on invalid input.
- [ ] Rootless Podman runner implements prepare/start/check/reset/destroy lifecycle.
- [ ] State-based checker accepts at least two different valid command paths to the same correct final state.
- [ ] PTY path supports the 103.5 jobs/process scenario.
- [ ] No host-shell execution fallback exists.
- [ ] No Internet access is required for the complete vertical slice after dependencies/images are installed.

## Security acceptance
- [ ] Lab fails closed when Podman is unavailable.
- [ ] Tests reject host-path mounts and privileged/capability escalation outside predefined profiles.
- [ ] Default lab network is disabled or explicitly isolated.
- [ ] CPU/memory/PID/time constraints exist in runner configuration.
- [ ] Host filesystem safety test runs a destructive lab and verifies a sentinel outside the sandbox is unchanged.
- [ ] Untrusted output rendered in the TUI has an explicit control-sequence handling policy and tests.
- [ ] `setup.execution_scope` cannot be changed from `sandbox`.

## Learning-model acceptance
- [ ] Evidence is append-only and mastery is a derived projection.
- [ ] A lesson completion only records exposure.
- [ ] Practical full mastery requires independent practical evidence and later confirmation/transfer.
- [ ] Solution reveal prevents that attempt from counting as independent practice.
- [ ] Initial assessment can satisfy prerequisites without fabricating lesson completion.
- [ ] Scheduler uses hard prerequisites as eligibility and recommended prerequisites as ranking only.

## Agent/repository acceptance
- [ ] `AGENTS.md` instructions are sufficient for a fresh coding agent to discover product, architecture, security, schemas and validation rules without chat history.
- [ ] Repetitive validation workflow is executable through scripts/skills.
- [ ] CI runs curriculum/graph validation, Go tests, format/lint checks and security-invariant tests.
- [ ] ADR status matches actual implementation choices.

## Explicitly out of Phase 1
- libvirt/QEMU implementation;
- complete LPIC content;
- AI tutor/generation;
- full exam simulator;
- public Internet-enabled labs;
- Debian/openSUSE host packaging.
