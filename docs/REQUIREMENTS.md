# Product requirements

Status: **Frozen for Phase 1**  
Source: product-owner interview completed 2026-09-24.

This document is the canonical product requirements source. Architecture documents and implementation plans must remain consistent with it.

## R-001 — Primary outcome
The primary product outcome is **LPIC-1 certification readiness** for the current LPIC-1 v5.0 objectives (101-500 and 102-500).

The product should build real Linux administration skill because practical understanding improves certification readiness, but extra material must not crowd out syllabus coverage.

Acceptance rules:
- every active LPIC-1 objective is represented in the machine-readable curriculum and decomposed finely enough to cover all independently examinable concepts;
- every mapped concept receives all three surfaces: a focused lesson, at least one deterministic quiz question, and at least one machine-checked practical lab/rep;
- an objective is not schedulable or complete while any mapped concept is missing one of those surfaces;
- modern material outside the syllabus is clearly labeled and must not hide examinable legacy material.

## R-002 — Adaptive daily session
Daily study duration is not a fixed number of minutes. The engine builds an adaptive session from the work due that day and the selected activity type.

The learner must be able to choose a shorter session when needed. A future scheduler may estimate duration, but Phase 1 must not hard-code one universal daily duration.

## R-003 — Notification-first daily integration
The application must **notify once per day** that a session is available. It must not force-open a terminal/TUI at graphical login.

The notification action should open the daily session. Manual launch must always remain available.

## R-004 — Gamification
The product includes:
- streaks;
- XP;
- achievements.

Gamification is motivational metadata, not mastery evidence. Missing a day must not reduce mastery or erase previously earned learning evidence.

## R-005 — Language
The curriculum is **French-first**.

Technical terms, command names, protocol names, file names and conventional Linux vocabulary remain in their normal English form when translation would be unnatural or misleading. Example: use `filesystem`, `systemd unit`, `pipe`, `target`, `socket`, `stdout`, etc. where that is clearer than forced French terminology.

The first occurrence of an English technical term should normally be explained in French.

## R-006 — Progressive depth
Concepts are introduced incrementally. A first encounter should be short and operational; later sessions deepen the mental model, edge cases and combinations with other concepts.

The product must avoid front-loading a comprehensive lecture before the learner has practical context.

## R-007 — Prerequisite-driven curriculum order
The learning order does not have to follow LPIC objective numbering. The scheduler may reorder material according to:
- prerequisites;
- prior mastery;
- due review;
- objective weight;
- conceptual relationships;
- variety/interleaving.

Coverage reporting must still map every activity back to official LPIC objective IDs.

## R-008 — Modern practice plus exam requirements
Teach modern Linux practice in addition to the exact LPIC syllabus.

Every relevant item should be classified as one of:
- `lpic-required` — directly required by current objectives;
- `lpic-legacy` — examinable/explicitly listed but legacy in normal modern administration;
- `modern-practice` — recommended current practice beyond or alongside the exam requirement.

When an older command is examinable, teach it even if a newer tool is preferred. Make the distinction explicit.

## R-009 — Full virtualization accepted
The complete product may require KVM/QEMU/libvirt and several gigabytes of VM images. This is acceptable in exchange for authentic boot, storage, recovery, networking and system-administration labs.

A reduced installation may support container-only material, but it must report unavailable full-system objectives honestly and must never emulate completion of them.

## R-010 — Distribution strategy
The primary learner environment is Fedora. Labs and examples should nevertheless include Debian-family and openSUSE environments often enough to develop distribution-independent LPIC competence.

Rules:
- Fedora may be the default host/install target for Phase 1;
- Debian and openSUSE are first-class guest learning environments;
- package-management exercises must cover the ecosystems expected by LPIC;
- assessment should sometimes vary the distribution without changing the underlying objective.

## R-011 — Network isolation
Labs are isolated from the public Internet by default.

Networking exercises should prefer local simulated infrastructure (DNS, HTTP, SSH, routing peers, mail, etc.) in dedicated container/VM networks. Internet access must be an explicit capability granted only to a scenario that demonstrably requires it.

## R-012 — AI deferred
No AI tutor, AI-generated exercise, cloud model, local model or model API is part of the initial product scope.

Core learning, hints, grading, scheduling, labs and debriefs must be deterministic and work offline. AI may be reconsidered by a future ADR without changing the core contracts.

## R-013 — Graduated hints
Labs and challenges support progressive help, authored and validated with the content:
1. light hint;
2. directional hint;
3. precise guidance;
4. explicit solution/debrief.

Using hints may reduce the strength of mastery evidence, but must not block learning progress.

## R-014 — Strict mastery
A concept/objective must not reach full mastery after one passive lesson or one successful quiz.

High mastery requires evidence over time and at least one successful practical task without revealing the full solution. The model should prefer demonstrated transfer over repetition of identical tasks.

## R-015 — Adaptive assessment before full exam simulation
The initial assessment experience should prioritize adaptive short sessions and targeted checks. A faithful 60-question/90-minute exam simulator may be added later, but it is not required for the Phase-1 learning loop.

## R-016 — Open-source project
The project is designed from the start as a general open-source project rather than a machine-specific private tool.

Machine-specific integration must remain an adapter/configuration layer. Defaults should be usable by other Linux learners.

## R-017 — Commercial reuse permitted
Commercial reuse by third parties is acceptable.

Selected licensing:
- application code and scripts: **Apache License 2.0**;
- original educational/documentation content: **Creative Commons Attribution 4.0 International (CC BY 4.0)**, unless a file states otherwise;
- third-party assets remain under their original licenses and must be recorded.

## R-018 — Safety invariants
The product must never silently trade safety for convenience:
- no lab execution on the host as a fallback;
- no generic privileged container mode;
- no arbitrary host mounts requested by curriculum content;
- no Internet by default;
- no secrets copied into guests by default;
- destructive system labs run in disposable environments.
