# Roadmap

## Phase 0 — foundations (complete)
- product interview and requirement freeze;
- agent-first repository structure;
- architecture/security ADRs;
- complete LPIC-1 v5 objective inventory;
- learning/content model;
- Apache-2.0 code + CC BY 4.0 content licensing decision;
- baseline threat model and Phase-1 acceptance criteria.

Exit criterion: no major product/security question required for Phase 1 is hidden only in chat; implementation decisions are explicit and testable.

## Phase 1 — vertical slice
Build the smallest complete adaptive daily loop for 2–3 representative objectives:
- curriculum parser/schema;
- SQLite progress/migrations;
- Bubble Tea TUI;
- inspectable scheduler/mastery model;
- XP/streak/achievement events separated from mastery;
- graduated hints;
- rootless Podman lab;
- deterministic state-based checker;
- daily desktop-notification adapter on Fedora;
- CI/security invariant tests.

Exit criteria are canonical in `docs/plan/PHASE-1-ACCEPTANCE.md`.

## Phase 2 — VM runner
Add libvirt/QEMU/KVM images and cover one boot objective + one storage objective end to end. Establish Fedora, Debian and openSUSE guest-image pipeline and isolated multi-machine lab networking.

## Phase 3 — Exam 101 curriculum
Complete topics 101–104 with automated coverage checks, progressive lessons, practical reps and cross-topic challenges.

## Phase 4 — Exam 102 curriculum
Complete topics 105–110, including networking, services and crypto. Ensure modern-versus-legacy labeling remains accurate.

## Phase 5 — adaptive practice depth
Strengthen spaced retrieval, prerequisite graph, distribution transfer, cross-topic incidents, achievement system and mastery explainability. Add adaptive short assessments.

## Phase 6 — exam simulation and packaging
Add original weighted exam simulation, including full-length mode if still desired. Harden RPM packaging first, then additional distro packaging; add update/signing, accessibility, backup/export and final security review.
