# Phase 3 — Exam 101 curriculum

Status: **Core curriculum complete — final acceptance hardening 2026-10-06**

Machine-readable scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Official scope was rechecked against LPIC-1 v5.0:
- exam code: **101-500**;
- topics: **101–104**;
- active objectives in scope: **23**;
- stable concepts in scope: **162**.

Phase 2 real-host KVM/libvirt acceptance passed on **2026-10-04**. Phase 3 can therefore build on both validated execution backends: rootless Podman for container-suitable tasks and libvirt/KVM for full-system behavior.

## Goal

Complete Exam 101 with original, French-first, deterministic learning content while preserving objective/concept traceability and the existing isolation model.

For every selected concept:
- provide exactly one focused `introduce` lesson suitable for first exposure;
- provide at least one deterministic daily non-assessment question;
- provide at least one machine-checked practical lab/rep;
- keep the concept granularity fine enough that all independently examinable knowledge in the official objective is actually taught, quizzed and practised;
- retain examinable legacy knowledge and label modern practice separately;
- avoid exam dumps and copied/adapted LPI learning prose.

## Implementation order

1. **103.4 — streams, pipes and redirections**: first post-slice objective, authored and schedulable.
2. Remaining command-line objectives: 103.2, 103.3, 103.6, 103.7 and 103.8.
3. Filesystem/storage coverage for topic 104, using the validated VM backend where needed.
4. Package/library objectives 102.3–102.6 and complete 102.1/102.2 theory coverage.
5. System-architecture objectives 101.1–101.3.
6. Final learner-path hardening: concept-scoped fallback labs, strict course + quiz + practice completion, and Exam-102 gating.
7. Full Phase-3 acceptance audit and regression suite.

Cross-topic incidents and adaptive extra depth are intentionally tracked in Phase 5; they are enrichment beyond the complete Exam-101 course/quiz/practice path, not Phase-3 exit gates.

## Coverage audit

`python3 scripts/audit_phase3_coverage.py --check` validates mappings and reports progress without requiring Phase 3 to be complete.

`python3 scripts/audit_phase3_coverage.py --require-complete` is the course/daily-question/practical-lab exit gate. It requires both recall and practical advancement paths for every Exam-101 concept.

The normal foundation validation now runs `audit_phase3_coverage.py --require-complete`; any regression that removes a required Exam-101 lesson, quiz, recall path or practical path fails CI.

## Current state

Exam 101 now covers all **23 objectives / 162 concepts**. Every concept receives one focused introduction, deterministic recall/recognition coverage, an applied scenario question, and two concept-scoped fallback practice contexts in addition to any bespoke lab.

The daily planner keeps Exam 102 locked until every Exam-101 concept has successful **course + quiz + practical** evidence. Objective prerequisites use the same complete-surface rule, so partial recall coverage cannot skip the remaining concepts of a section. Generated fallback labs remain guided evidence and rotate from diagnostic to transfer context after a successful first practice.
