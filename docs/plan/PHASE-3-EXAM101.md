# Phase 3 — Exam 101 curriculum

Status: **In progress — started 2026-10-01**

Machine-readable scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Official scope was rechecked against LPIC-1 v5.0:
- exam code: **101-500**;
- topics: **101–104**;
- active objectives in scope: **23**;
- stable concepts in scope: **153**.

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
6. Cross-topic challenges without hidden prerequisites.
7. Full Phase-3 acceptance audit and regression suite.

## Coverage audit

`python3 scripts/audit_phase3_coverage.py --check` validates mappings and reports progress without requiring Phase 3 to be complete.

`python3 scripts/audit_phase3_coverage.py --require-complete` is the course/daily-question/practical-lab exit gate. Cross-topic challenge gates remain explicit in `PHASE-3-ACCEPTANCE.md`.

The normal foundation validation runs the non-blocking audit so incomplete curriculum work does not make every intermediate commit fail while invalid mappings still fail closed.

## Current authored tranche

Objective **103.4** currently provides:
- 7 stable concepts;
- 7 progressive micro-lessons;
- 11 deterministic daily questions.

The daily scheduler exposes an objective only when every mapped concept has exactly one focused introduction, a non-assessment daily question and a machine-checked practical lab. Recall-only coverage is insufficient. Incomplete Exam-101 objectives remain unschedulable.
