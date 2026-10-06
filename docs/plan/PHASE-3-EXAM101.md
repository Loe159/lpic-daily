# Phase 3 — Exam 101 curriculum

Status: **Complete / accepted — 2026-10-06**

Machine-readable scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Official LPIC-1 v5.0 scope:
- exam code: **101-500**;
- topics: **101–104**;
- active objectives: **23**;
- stable concepts: **162**;
- total official objective weight: **60**.

Phase 2 real-host KVM/libvirt acceptance passed on **2026-10-04**. Exam-101 practice therefore uses rootless Podman for container-suitable tasks and libvirt/KVM for full-system boot/storage behavior.

## Accepted learning contract

Every Exam-101 concept has:
- exactly one focused `introduce` lesson;
- deterministic daily questions, including free recall;
- an operational application question whose prompt must not reveal the exact concept title or answer-bearing anchor terms;
- two generated concept-scoped practical contexts plus any bespoke authored labs;
- deterministic state/result verification for generated Exam-101 practice.

Generated practical exercises no longer count as mastery merely because the learner writes a proof file or because one command returned zero. Checks inspect resulting Linux state or recompute the exact expected transformation/observation. CI also requires diagnostic and transfer contexts to have different deterministic contracts.

## Progression and readiness

The normal learner path is **lesson → recall → practical work**. Once a concept is in review, pressing Enter prefers retrieval practice before another lab.

Prerequisite readiness requires all mapped concepts to have:
1. successful lesson exposure;
2. successful recall evidence;
3. independent/transfer practical evidence.

Exam 102 is stricter still. It remains locked until every Exam-101 concept has:
- a passed lesson;
- independent/transfer practical evidence;
- at least two successful recall events separated by **72 hours or more**.

A passed cumulative Exam-101 simulation is also required. Guided practice or one recognition question can no longer mark Exam 101 ready.

Study focus defaults to Exam 101. `lpic focus 101` keeps the scheduler on 101; `lpic focus all` permits Exam-102 scheduling only after the readiness gates above pass.

## Cumulative simulation

`lpic assess --exam 101` builds a deterministic **60-question** simulation. The number of questions from each objective equals that objective's official weight. Questions are interleaved across topics and mix free recall with operational application.

LPIC Daily currently treats the simulation as passed at **80% overall with no objective below 50%**. This is an internal readiness threshold, not an official LPI scoring rule.

## Coverage audit

`python3 scripts/audit_phase3_coverage.py --check` reports mappings and coverage.

`python3 scripts/audit_phase3_coverage.py --require-complete` is the structural exit gate. It requires lesson, daily-question, recall advancement and deterministic practical coverage for all 162 concepts. Generated guided/declarative fallbacks are reported separately and cannot satisfy the deterministic-practice gate.

The normal foundation validation invokes this complete gate, while Go tests independently validate generated learner surfaces, distinct transfer contracts, question leakage protection, scheduling/readiness behavior and weighted exam construction.
