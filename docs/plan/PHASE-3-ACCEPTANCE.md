# Phase 3 acceptance criteria — Exam 101

Status: **In progress — updated 2026-10-04**

Canonical scope: `curriculum/lpic-1-v5/phase3-exam101.json`.

Phase 3 covers all 23 active Exam-101 objectives across topics 101–104 and all 153 stable concepts mapped to them.

Phase 2 is already accepted on a real KVM/libvirt host (2026-10-04); it is now a regression dependency, not a deferred gate.

## Curriculum coverage
- [ ] Every selected concept has at least one original French-first lesson and exactly one focused `introduce` lesson.
- [ ] Every selected concept has at least one deterministic daily non-assessment question.
- [ ] Every selected concept has at least one machine-checked practical lab/rep; recall questions do not substitute for a lab.
- [ ] Stable concepts are granular enough to cover the complete official objective without hiding independently examinable sub-concepts inside an over-broad bucket.
- [ ] `python3 scripts/audit_phase3_coverage.py --require-complete` passes.
- [x] Go and Python curriculum validation reject drift in `phase3-exam101.json`.
- [ ] Every artifact maps to the correct objective ID and stable concept ID.
- [ ] Legacy examinable material is preserved and distinguished from modern practice.
- [ ] Objective weights and scope match LPIC-1 v5.0 Exam 101.

## Practical coverage
- [ ] Every selected concept has practical evidence in at least one appropriate sandboxed lab/rep.
- [x] Full-system boot/storage tasks use libvirt/KVM.
- [x] Container-suitable tasks use rootless Podman.
- [x] Lab success is based on observable state rather than exact command transcripts.
- [x] No learner-content path executes directly on the host.

## Progressive learning
- [ ] New concepts are introduced incrementally.
- [x] An objective is schedulable only when all mapped concepts satisfy the course + quiz + practical-lab gates.
- [x] Incomplete Phase-3 objectives are not scheduled merely because they exist in the Exam-101 scope.
- [x] Multiple daily questions rotate toward the least-attempted option.
- [ ] Higher-weight objectives receive proportionally deeper practice without eliminating lower-weight coverage.
- [ ] At least one cross-topic challenge exists for topics 101, 102, 103 and 104.
- [ ] Cross-topic challenges introduce no undocumented hard prerequisite.
- [x] Hints/solution reveal continue to weaken mastery evidence.

## Regression and safety
- [ ] `python3 scripts/validate_foundation.py`.
- [ ] `go test ./...`.
- [ ] `go vet ./...`.
- [ ] Go formatting check passes.
- [ ] Existing Phase-1 real Podman isolation/conformance coverage remains green.
- [ ] Existing Phase-2 non-KVM tests remain green.
- [ ] Phase-2 real-KVM acceptance remains reproducible with the canonical acceptance command when VM/runtime code changes.
