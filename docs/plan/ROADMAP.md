# Roadmap

## Phase 0 — foundations (complete)
- product interview and requirement freeze;
- agent-first repository structure;
- architecture/security ADRs;
- complete LPIC-1 v5 objective inventory;
- learning/content model;
- Apache-2.0 code + CC BY 4.0 content licensing decision;
- baseline threat model and Phase-1 acceptance criteria.

## Phase 0.5 — pedagogical contracts (complete)
- full 42-objective prerequisite DAG;
- 309 stable concept IDs;
- versioned JSON Schemas for concepts, lessons, questions, labs, hints, evidence and achievements;
- append-only mastery evidence model;
- formal Phase-1 selection: 103.1 + 103.5 + 104.5;
- dependency/graph/schema validators and CI.

Exit criterion: a fresh implementation agent can determine what to build and validate without inventing ordering, IDs or content contracts.

## Phase 1 — vertical slice (complete)
Build the complete adaptive daily loop for **103.1, 103.5 and 104.5**:
- curriculum/schema loader;
- SQLite progress/migrations;
- Bubble Tea TUI;
- inspectable scheduler/mastery projection;
- XP/streak/achievement events separated from mastery;
- graduated hints;
- rootless Podman runner + PTY;
- deterministic state-based checker;
- three primary scenarios plus three transfer contexts, giving every Phase-1 concept two practical contexts;
- daily desktop-notification adapter on Fedora;
- CI/security invariant tests.

Exit criteria are canonical in `docs/plan/PHASE-1-ACCEPTANCE.md` and were accepted on 2026-09-27.

## Phase 2 — VM runner (complete)
Add libvirt/QEMU/KVM images and cover one boot objective + one storage objective end to end. Establish Fedora, Debian and openSUSE guest-image pipeline and isolated multi-machine lab networking. Real-host KVM/libvirt acceptance passed on 2026-10-04.

## Phase 3 — Exam 101 curriculum (core complete; acceptance hardening)
Topics 101–104 are covered across all 23 objectives / 162 concepts with automated scope checks, focused lessons, quizzes and concept-scoped practical reps. Final hardening enforces course + quiz + lab completion before prerequisite or Exam-102 progression; see `docs/plan/PHASE-3-EXAM101.md`.

## Phase 4 — Exam 102 curriculum
Complete topics 105–110, including networking, services and crypto. Ensure modern-versus-legacy labeling remains accurate.

## Cross-cutting initiative — scenario-based practical curriculum (in progress)
Replace the 618 concept-scoped fallback labs progressively with a smaller portfolio of realistic, state-based scenarios modeled after shell-environment-repair, stuck-worker and shared-dropbox. Keep full concept coverage throughout the migration, prefer scenario evidence in the scheduler, then retire fallback generation only after both exams are fully covered.

Pilot in progress: objective 103.1 is the first migration target, with accepted state-based scenarios replacing generated fallbacks in normal scheduling once coverage is complete.

Canonical plan: docs/plan/SCENARIO-LABS-MIGRATION.md.

## Phase 5 — adaptive practice depth
Strengthen spaced retrieval, concept prerequisites where demonstrated, distribution transfer, cross-topic incidents, achievement system and mastery explainability. Add adaptive short assessments.

## Phase 6 — exam simulation and packaging
Add original weighted exam simulation, including full-length mode if still desired. Harden RPM packaging first, then additional distro packaging; add update/signing, accessibility, backup/export and final security review.
