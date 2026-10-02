# Phase 3 acceptance criteria — Exam 101

Status: **In progress — 2026-10-01**

Canonical scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Phase 3 covers all 23 active Exam-101 objectives across topics 101–104 and all 153 active stable concepts currently mapped to them.

## Curriculum coverage
- [ ] Every selected concept has at least one original French-first lesson and a focused `introduce` lesson suitable for first exposure.
- [ ] Every selected concept has at least one deterministic daily non-assessment question; `initial-assessment` questions do not satisfy this gate. Each concept must also have a path beyond recognition through a recall-capable daily question or practical lab evidence.
- [ ] `python3 scripts/audit_phase3_coverage.py --require-complete` passes.
- [ ] The Go curriculum loader and Python graph validator both reject drift in `phase3-exam101.json`.
- [ ] Every learning artifact maps to the correct official objective ID and stable concept ID.
- [ ] Examinable legacy material remains present and explicitly distinguished from modern practice where relevant.
- [ ] Objective weights and scope still match the current LPIC-1 v5.0 Exam 101 objectives.

## Practical coverage
- [ ] Every objective with safely observable hands-on competence has at least one practical rep/lab or an explicit documented reason why practical evidence is not appropriate.
- [ ] Full-system boot/storage tasks use libvirt rather than pretending a container reproduces kernel/bootloader behavior.
- [ ] Container-suitable command-line tasks remain rootless and isolated.
- [ ] Lab success is based on observable state/evidence rather than one exact command transcript.
- [ ] No new learner-content path can execute directly on the host.

## Progressive learning
- [ ] New concepts are introduced incrementally rather than as one large lecture.
- [ ] An objective becomes selectable by the daily scheduler only when every mapped concept has a focused introduction, deterministic daily-question coverage, and a recall-or-practical mastery-advancement path, once hard prerequisites are ready.
- [ ] Incomplete Phase-3 objectives are not scheduled merely because they are listed in the complete Exam-101 scope.
- [ ] When several daily questions cover one concept, automatic recommendations rotate toward the least-attempted question instead of permanently selecting the lexicographically first ID.
- [ ] Higher-weight objectives receive proportionally deeper review/practice coverage without reducing lower-weight objectives to zero coverage.
- [ ] At least one cross-topic challenge exists for each topic 101, 102, 103 and 104.
- [ ] Cross-topic challenges do not invent undocumented hard prerequisites.
- [ ] Solution reveal/hints continue to weaken mastery evidence according to the existing mastery contract.

## Regression and safety
- [ ] `python3 scripts/validate_foundation.py`.
- [ ] `go test ./...`.
- [ ] `go vet ./...`.
- [ ] Go formatting check passes.
- [ ] Existing Phase-1 Podman isolation/security tests remain green.
- [ ] Existing non-KVM Phase-2 tests remain green.
- [ ] No Phase-2 KVM acceptance checkbox is marked complete without actual real-host execution.

## Phase-2 deferred validation rule

Phase 3 implementation may become content-complete while the Phase-2 real-KVM acceptance run is still pending. That pending KVM gate must remain visible and is required before the project is treated as a fully validated release candidate.
