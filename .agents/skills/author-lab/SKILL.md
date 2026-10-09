---
name: author-lab
description: Author realistic incident-driven, state-validated LPIC Linux labs
---

# author-lab

1. Read `AGENTS.md`, `labs/AGENTS.md`, `docs/SECURITY.md`, `docs/plan/SCENARIO-LABS-MIGRATION.md` and the objective's exam concepts.
2. Define a believable Linux incident: broken initial state, **observable symptom**, expected operational recovery, non-negotiable preservation constraints. The French learner brief must be short and **must not be a list of commands**. Do not mention exact repair paths if discovery is being assessed.
3. Group **naturally coherent** concepts rather than generating one micro-lab per concept. Tie every claimed concept to at least one check that actually measures the skill. Real Linux behavior comes first; fictitious compliance reports, `CONCEPT/TERMS/OBSERVATION` artifacts and reflection prompts are not assessment evidence.
4. Choose the least-privileged **faithful** environment: rootless Podman for shell, user space and file operations; libvirt/KVM for actual boot, partitions/mounts that demand privilege, kernel, init, systemd and multi-VM interactions. A simulated directory tree must not stand in for actual OS behavior.
5. Author a deterministic setup that begins **incorrect**, permits meaningful investigation, and leaves essential evidence accessible. Declare network disabled by default, writable paths, CPU/memory/PID limits, timeout and disposable reset.
6. Write `success_criteria_fr` in terms of independently verifiable end states. Implement checks that accept any valid repair path and preserve important original data. Associate each check with only the relevant concept IDs. Checks must not be bypassable by creating a declaration file.
7. Supply four progressive hints: observable signal, investigation area, implicated mechanism, near-solution. Mark the final hint `solution-revealed` when appropriate. Put command-level detail in hints and post-lab debrief, not the mission brief.
8. Add the test-only `reference-solution.sh`. Verify **red** after setup, **green** after reference, reset->red again, plus timeout/interruption cleanup and (where practical) a different valid learner solution. Execute the actual backend; syntax-only review is insufficient.
9. Update `curriculum/lpic-1-v5/scenario-coverage.json` and the migration plan. New scenarios remain `implemented` until runtime acceptance tests and manual pedagogical review prove all gates. Only then mark `accepted`; objectives are migrated only when every active concept is accepted and no fallback is scheduled.
10. Run `python3 scripts/validate_foundation.py`, `python3 scripts/validate_labs.py`, `python3 scripts/audit_scenario_coverage.py --check` and the appropriate Podman/KVM acceptance tests when supported. Report **exactly** which checks ran, failed or could not run. Never infer green CI from authored code.

