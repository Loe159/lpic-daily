# Lab agent instructions

Labs are adversarial inputs from the host's perspective. Read `docs/SECURITY.md` before authoring or modifying lab mechanisms.

## Incident-driven authoring (mandatory)
- Write a concise French `brief_fr` around a plausible production symptom and the intended recovery. Specify paths only when the learner needs an entry point; discovery is part of the task.
- Do **not** prescribe commands or a checklist of implementation steps. Avoid prompting for explanations, written confirmations or artificial evidence files to prove Linux skills. If a report is the operational deliverable, grade its real output, not a hard-coded assertion.
- `success_criteria_fr` describe outcomes, never the procedure; `debrief_fr` may explain the underlying commands and mechanism after success.
- Four hints progress from observation to diagnosis to mechanism to near-solution. The final hint must declare `solution-revealed` when it exposes the repair.
- Each `checks[].concept_ids` must correspond to a competency genuinely established by that check. Require a broken initial state, a valid reference repair and an independent outcome-based check; prefer invariant properties over identical prescribed text.
- **Acceptance gate:** setup -> at least one check fails; reference solution -> all checks pass; alternate plausible solution -> passes where feasible; reset -> reproduces initial failure; backend security and timeout validated. Without these, register `implemented`, not `accepted`. If KVM/Podman testing is unavailable, report that plainly.
- Podman may test files, shell and user-space tools, but cannot certify actual reboot, kernel/module behavior or systemd PID 1. Never mislabel simulated evidence as machine-level acceptance.

- Choose rootless Podman unless the learning objective genuinely needs a separate machine/kernel/boot/storage stack.
- Host execution fallback is forbidden.
- Network is denied by default.
- Never mount arbitrary host paths or secrets.
- Grade observable final state, not exact command history.
- Give every practical lab a stable `practice_context`; a transfer lab must use a materially different context ID, not merely a different lab ID.
- Every lab must define timeout/reset/cleanup behavior and a test-only reference solution.
