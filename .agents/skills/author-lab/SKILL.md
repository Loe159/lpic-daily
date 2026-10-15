---
name: author-lab
description: Author realistic incident-driven, state-validated LPIC Linux labs
---

# author-lab

1. Read `AGENTS.md`, `labs/AGENTS.md`, `docs/SECURITY.md`, `docs/plan/SCENARIO-LABS-MIGRATION.md` and the objective's exam concepts.
2. Define a believable Linux incident: broken initial state, **observable symptom**, expected operational recovery, non-negotiable preservation constraints. The French learner brief must be short and **must not be a list of commands**. Do not mention exact repair paths if discovery is being assessed. Be precise about the **meaning** of requested information (e.g. kernel name, not an unspecified "system identity"), and any output schema (field/line order, paths provided as deliverables); do not name the discovery commands, options or expected values.
3. Group **naturally coherent** concepts rather than generating one micro-lab per concept. Tie every claimed concept to at least one check that actually measures the skill. Real Linux behavior comes first; fictitious compliance reports, `CONCEPT/TERMS/OBSERVATION` artifacts and reflection prompts are not assessment evidence.
4. Choose the least-privileged **faithful** environment: rootless Podman for shell, user space and file operations; libvirt/KVM for actual boot, partitions/mounts that demand privilege, kernel, init, systemd and multi-VM interactions. A simulated directory tree must not stand in for actual OS behavior.
5. Author a deterministic setup that begins **incorrect**, permits meaningful investigation, and leaves essential evidence accessible. Declare network disabled by default, writable paths, CPU/memory/PID limits, timeout and disposable reset.
6. Write `success_criteria_fr` in terms of independently verifiable end states. Apply the same calibration as the brief: specify observable results and format, not command recipes or checker expressions. Implement checks that accept any valid repair path and preserve important original data. Associate each check with only the relevant concept IDs. Checks must not be bypassable by creating a declaration file. If a mission asks learners to change the *current shell* (e.g. `export`/`unset`), grade that live shell state; do not silently require changes to `.bashrc` or re-run the check inside a fresh shell.
7. Supply four progressive hints: observable signal, investigation area, implicated mechanism, near-solution. Mark the final hint `solution-revealed` when appropriate. Put command-level detail in hints and post-lab debrief, not the mission brief.
8. Add the test-only `reference-solution.sh`. Verify **red** after setup, **green** after reference, reset->red again, plus timeout/interruption cleanup and (where practical) a different valid learner solution. Execute the actual backend; syntax-only review is insufficient.
9. Update `curriculum/lpic-1-v5/scenario-coverage.json` and the migration plan. The owner has authorized publishing authored scenarios as `accepted` before manual pedagogical review. Keep CI, runtime, setup-red/reference-green/reset-red, alternative solutions and KVM/Podman conformance as separately tracked verification work; never describe publication as proof that tests passed. An objective is coverage-migrated when its active concepts have authored accepted scenarios and no fallback is scheduled.
10. Run `python3 scripts/validate_foundation.py`, `python3 scripts/validate_labs.py`, `python3 scripts/audit_scenario_coverage.py --check` and the appropriate Podman/KVM acceptance tests when supported. Report **exactly** which checks ran, failed or could not run. Never infer green CI from authored code.


## Learner-brief quality gate

Before publication, read `brief_fr` and `success_criteria_fr` **without** opening the checker or solution. Reject a brief if a competent learner cannot tell precisely which outcome is requested, including distinctions such as kernel name versus version or command category versus path. Reject it also if it names the Linux command/flag that constitutes the answer. Explicit output contracts are not spoilers; prescribed procedures are. Keep the answer method in hints/debrief/reference fixtures, and keep the checker faithful to the public contract.

## Checker performance rule

Do not use interactive Bash (`-i`, `-ic`, or `--rcfile ... -ic`) in authored `command-exit` checks or reference solutions. Use `bash --noprofile --norc -c 'source /path/to/rc; ...'` only when the lab specifically tests an rcfile; otherwise use direct noninteractive tools and filesystem state checks. Avoid hidden TTY requirements. Prefer quick side-effect-free checks with explicit `timeout_seconds` when needed; all commands have a short default deadline. Built-in tests reject interactive grading checks across every active scenario.
