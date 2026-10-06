# Phase 3 acceptance criteria — Exam 101

Status: **Accepted — 2026-10-06**

Canonical scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Phase 3 covers all **23 active Exam-101 objectives / 162 stable concepts** across topics 101–104.

Phase 2 real-host KVM/libvirt acceptance passed on **2026-10-04**. This Phase-3 hardening does not modify the VM runner; it strengthens generated lab definitions, evidence semantics, scheduling and Exam-101 readiness.

## Curriculum coverage
- [x] Every selected concept has at least one original French-first lesson and exactly one focused `introduce` lesson.
- [x] Every selected concept has at least one deterministic daily non-assessment question.
- [x] Every selected concept has a machine-checked practical path; recall questions do not substitute for practice.
- [x] Stable concepts cover the complete official objective without hiding independently examinable sub-concepts inside an over-broad bucket.
- [x] `python3 scripts/audit_phase3_coverage.py --require-complete` passes.
- [x] Go and Python curriculum validation reject drift in `phase3-exam101.json`.
- [x] Artifacts map to the correct objective ID and stable concept ID.
- [x] Examinable legacy material is preserved and distinguished from recommended modern practice.
- [x] Objective weights and scope match LPIC-1 v5.0 Exam 101.

## Practical coverage
- [x] All 162 concepts expose deterministic practical coverage.
- [x] Generated Exam-101 practice validates Linux state or an independently recomputed result; the old `CONCEPT/TERMS/COMMAND/OBSERVATION/EXPLANATION` declaration does not satisfy the Exam-101 practical gate.
- [x] A successful command transcript alone is never sufficient to prove practical mastery.
- [x] Diagnostic and transfer contexts have distinct deterministic contracts; CI rejects pairs that only rename the same task.
- [x] Full-system boot/storage work uses libvirt/KVM.
- [x] Container-suitable command-line work uses rootless Podman.
- [x] No learner-content path executes directly on the host.

## Progressive learning and mastery
- [x] First exposure follows lesson → recall/question → practical work.
- [x] Review defaults to retrieval practice instead of repeatedly selecting a lab after the guided stage.
- [x] Prerequisite readiness requires successful lesson exposure, successful recall, and **independent/transfer** practical evidence for every concept in the prerequisite objective.
- [x] Guided practice alone cannot unlock a prerequisite or complete Exam 101.
- [x] Exam 102 remains locked until **every one of the 162 Exam-101 concepts** has a passed lesson, independent/transfer practical evidence, and at least **two successful recall events separated by 72 hours or more**.
- [x] Exam 102 also requires a passed cumulative Exam-101 simulation.
- [x] Study focus defaults to Exam 101. `lpic focus 101` keeps Exam 102 out of scheduling even after readiness; `lpic focus all` opts back into progression.
- [x] Hints/solution reveal continue to weaken mastery evidence.

## Exam-oriented assessment
- [x] `lpic assess --exam 101` provides a cumulative 60-question simulation.
- [x] Question counts per objective equal the official Exam-101 objective weights (total weight 60).
- [x] The simulation mixes free recall and operational application questions and interleaves objectives.
- [x] Application prompts are checked for leakage of the exact concept title and answer-bearing anchor terms.
- [x] The simulation selection is deterministic and excludes initial-assessment questions.
- [x] Internal readiness requires at least 80% overall with no objective below 50%; the CLI explicitly states that this is an LPIC Daily readiness threshold, not official LPI scoring.

## Regression and safety
- [x] `python3 scripts/validate_foundation.py`.
- [x] `go test ./...`.
- [x] `go vet ./...`.
- [x] Go formatting check.
- [x] Embedded curriculum/content loader validation.
- [x] Existing Phase-1 rootless-Podman isolation/conformance suite remains part of CI.
- [x] Existing Phase-2 non-KVM tests remain green.
- [x] Phase-2 real-KVM acceptance from 2026-10-04 remains the backend acceptance reference; rerun the canonical real-KVM acceptance command when VM/runtime code changes.

The coverage audit is deliberately narrower than runtime mastery: it proves that all required learning surfaces and deterministic practical contracts exist. Learner readiness is derived separately from recorded recall/practical evidence and the cumulative assessment gate.
