# Lab agent instructions

Labs are adversarial inputs from the host's perspective. Read `docs/SECURITY.md` before authoring or modifying lab mechanisms.

## Authored labs only (mandatory)
- Never generate, synthesize, or offer automatic lab definitions from individual concepts or command anchors.
- Every practical lab must be an explicitly versioned `labs/lpic-1-v5/<objective>/<scenario>/lab.json` with a real setup, independent observable checks, progressive hints, and a reference solution.
- An unimplemented practical skill remains unverified. The learner may continue with courses and quizzes; do not create a dummy fallback or mark practical mastery from theory.
- Preserve historical evidence events from legacy generated labs in the progress store, but do not reload or schedule those lab IDs.

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

## Outcome precision without giving away the method (mandatory)

There are two independent requirements:

1. **No guessing about the contract.** Name the exact information, object, unit, scope, and externally relevant output format. If an operational output file is required, state its path, number of lines/fields, order, and intended meaning. Distinguish kernel **name** from kernel version, architecture or operating-system information; distinguish a shell command's **classification** from its resolved absolute path. The learner should not need to see a checker, a hint, or a reference solution to infer what to deliver.
2. **Discovery remains part of the assessment.** Do not specify the command, option, pipeline, configuration key or step sequence that obtains the information or fixes the failure. Do not print the expected values when determining them is the skill. Never replace the objective with an implicit answer key.

Example for a shell report:
- **Too vague:** `Identifie le système et le type de l'outil de diagnostic.`
- **Too explicit:** `Écris la sortie de uname -s sur la première ligne et de type -t report-status sur la seconde.`
- **Balanced:** `Dans /run/lpic/system-summary, écris deux lignes sans étiquette : le nom du noyau, puis la catégorie de report-status reconnue par Bash, sans chemin ni commentaire.`

Apply this balance to **both** `brief_fr` and `success_criteria_fr`. The latter may clarify observable results but must not expose commands or checker internals. Technical commands belong in progressive hints (especially levels 3/4), the post-lab debrief and private authoring fixtures. Conversely, do not withhold the required output schema or accepted objective merely to make a lab harder.

During review, ask: **Can a learner restate exactly what success looks like without knowing the repair command? Does the wording leave the Linux technique for them to discover?** If either answer is no, rewrite.

Before accepting any lab, ask: **Could a learner copy the brief's implementation instructions and pass without diagnosing the problem?** If yes, rewrite the brief.

## Lab verification messages (mandatory)

- Every `checks[]` item must have a specific, learner-readable French `description_fr`, including `command-exit`, file, process, block-device, and event checks.
- Reuse the lab's measurable success criteria but split them when several checks validate different aspects of one objective. Do not label a process-termination check as process-running, and do not repeat the same broad wording for several independent checks.
- The message must identify the **observable result** to fix, not the command to type. Avoid `Comportement attendu`, `Vérification`, opaque technical IDs, and unverifiable claims.
- When a check requires a file, name its path and explain its useful role and expected content. A raw `exit=N, expected=0` can supplement the failure, but must not be its only meaningful explanation.
- Validate the entire `:check` output before publishing: deliberately fail all checks, review each failure label, then run the reference solution and ensure the corresponding labels turn green.
