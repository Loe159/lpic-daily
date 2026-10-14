# Lab agent instructions

Labs are adversarial inputs from the host's perspective. Read `docs/SECURITY.md` before authoring or modifying lab mechanisms.

## Incident-driven authoring (mandatory)
- Write a concise French `brief_fr` around a plausible production symptom and the intended recovery. Specify paths only when the learner needs an entry point; discovery is part of the task.
- Do **not** prescribe commands or a checklist of implementation steps. Avoid prompting for explanations, written confirmations or artificial evidence files to prove Linux skills. If a report is the operational deliverable, grade its real output, not a hard-coded assertion.
- `success_criteria_fr` describe outcomes, never the procedure; `debrief_fr` may explain the underlying commands and mechanism after success.
- Four hints progress from observation to diagnosis to mechanism to near-solution. The final hint must declare `solution-revealed` when it exposes the repair.
- Each `checks[].concept_ids` must correspond to a competency genuinely established by that check. Require a broken initial state, a valid reference repair and an independent outcome-based check; prefer invariant properties over identical prescribed text.
- **Publication versus conformance:** `accepted` publishes a scenario for progressive learner testing (owner decision 2026-10-12), not a claim of backend success. Independently validate setup red -> reference green -> reset red, an alternate repair where feasible, and backend isolation/timeouts. Record failed or unavailable tests explicitly; fix regressions before merging even when a lab is published.
- Podman may test files, shell and user-space tools, but cannot certify actual reboot, kernel/module behavior or systemd PID 1. Never mislabel simulated evidence as machine-level acceptance.

- Choose rootless Podman unless the learning objective genuinely needs a separate machine/kernel/boot/storage stack.
- Host execution fallback is forbidden.
- Network is denied by default.
- Never mount arbitrary host paths or secrets.
- Grade observable final state, not exact command history.
- Give every practical lab a stable `practice_context`; a transfer lab must use a materially different context ID, not merely a different lab ID.
- Every lab must define timeout/reset/cleanup behavior and a test-only reference solution.

## Outcome-first instructions (mandatory, 2026-10-14)

A learner-facing `brief_fr` is **not a repair procedure**. State only:
1. The operational symptom and enough context to investigate it.
2. The **observable target** (including quantities, business constraints, legitimate paths and named external interfaces).
3. What must remain intact or must not happen.

Avoid revealing the configuration knob, command, flag, exact mode, solution file, cause already diagnosed or ordered steps when finding them is the LPIC competency being tested. Examples:
- **Avoid**: `mets HISTSIZE=2000 et HISTFILESIZE=4000 dans .bashrc`.
- **Prefer**: `un nouveau shell doit conserver 2 000 commandes en session et 4 000 sur disque`.
- **Avoid**: `utilise chmod 2770 puis configure umask 0007`.
- **Prefer**: `les fichiers de l’équipe doivent conserver leur groupe, et les nouveaux documents ne doivent pas être accessibles aux tiers`.
- **Avoid**: `utilise xargs puis sort`.
- **Prefer**: `l’outil doit traiter les chemins reçus en entrée standard et restituer un rapport trié`.

Do not confuse a solution with a necessary contract: a report path, required output format, target capacity, or externally required identifier may be named; a particular Linux mechanism should be discovered through the exercise. `success_criteria_fr` describes the evidence and precise expected result **without instructing on the repair method**. Put debugging orientation in level-1/2 hints, mechanism in level-3 hints, and possible commands only in level-4 solution-revealing hints or the reference solution. Keep the mission reasonably short: prefer roughly 300–500 characters when the problem allows; put longer expectations in the criteria rather than duplicating them. 

Before accepting any lab, ask: **Could a learner copy the brief's implementation instructions and pass without diagnosing the problem?** If yes, rewrite the brief.
