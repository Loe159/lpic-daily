# Phase 3 — Exam 101 curriculum

Status: **In progress — started 2026-10-01**

Machine-readable scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Official scope was rechecked against the current LPI LPIC-1 v5.0 objectives on 2026-10-01:
- exam code: **101-500**;
- topics: **101–104**;
- active objectives in this repository scope: **23**;
- active stable concepts: **153**.

Source: https://www.lpi.org/fr/exam-101-102-objectives/

## Relationship with Phase 2

Phase 2 implementation is present but its real-host KVM/libvirt acceptance execution is still pending.

Phase 3 may proceed in parallel because most Exam-101 curriculum work is independent from the hypervisor acceptance run. This does **not** mark Phase 2 accepted and must not weaken or bypass its KVM gate. Before a release candidate is considered fully validated, the Phase-2 real-host acceptance command still has to pass.

## Goal

Complete Exam 101 with original, French-first, deterministic learning content while preserving objective/concept traceability and the existing isolation model.

For every selected concept:
- provide at least one progressive lesson surface;
- provide at least one deterministic retrieval question;
- add practical evidence when the skill is observable safely in Podman or libvirt;
- retain examinable legacy knowledge and label modern practice separately;
- avoid exam dumps and copied/adapted LPI learning prose.

## Implementation order

1. **103.4 — streams, pipes and redirections**: first post-slice objective, Podman-safe and independent of KVM.
2. Finish the remaining command-line objectives: 103.2, 103.3, 103.6, 103.7 and 103.8.
3. Expand filesystem/storage content for topic 104, reusing Phase-2 VM capabilities where full-system behavior is required.
4. Add package/library objectives 102.3–102.6 and complete 102.1/102.2 theory/retrieval coverage.
5. Add system-architecture objectives 101.1–101.3, using libvirt only where boot/target behavior requires it.
6. Add cross-topic challenges that combine concepts without creating new hidden prerequisites.
7. Run the full Phase-3 acceptance audit and regression suite.

## Coverage audit

`python3 scripts/audit_phase3_coverage.py --check` is a non-blocking progress audit used during implementation. It validates mappings and reports lesson/question/lab coverage across all 153 Exam-101 concepts.

`python3 scripts/audit_phase3_coverage.py --require-complete` is an exit gate for theory/retrieval coverage. Practical-objective and cross-topic gates remain explicit in `PHASE-3-ACCEPTANCE.md`.

The normal foundation validation runs the non-blocking `--check` mode so incomplete Phase-3 content does not make every intermediate commit red while invalid mappings still fail closed. The Go curriculum loader also loads and validates this manifest, so it cannot silently drift from `objectives.json`, `concepts.json` or the prerequisite graph.

## First tranche

The first authored tranche is objective **103.4**:
- 7 stable concepts;
- 7 progressive micro-lessons;
- 8 deterministic retrieval questions, including explicit `stdin` redirection with `<`.

Hands-on 103.4 reps/challenges follow after this initial content tranche; no artificial practical evidence is claimed before a checker can observe a meaningful result.

The daily scheduler derives its currently runnable objective scope from this Phase-3 manifest plus authored content. An objective becomes schedulable only when every mapped concept has a focused `introduce` lesson and one non-assessment daily question. This makes 103.4 reachable after 103.1 readiness without exposing unfinished Exam-101 objectives. Concepts with several daily questions recommend the least-attempted question so authored retrieval variants remain reachable over time.
