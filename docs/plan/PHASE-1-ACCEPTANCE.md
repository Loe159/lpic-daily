# Phase 1 — vertical-slice acceptance criteria

Goal: prove the product loop and security model before scaling curriculum production.

## Scope
Implement 2–3 representative objectives that exercise different learning modes but do not yet require the VM backend. Recommended candidates should include:
- one shell/text-processing objective;
- one permissions/filesystem-userland objective;
- one package/process/service-oriented objective feasible safely in rootless Podman.

The exact objectives are selected at Phase-1 kickoff from the prerequisite graph; do not choose them merely because they are easiest to demo.

## Product acceptance
- [ ] Daily session can be built from curriculum metadata and progress state.
- [ ] Session can mix review and one new progressive concept.
- [ ] French-first lesson renders natural English technical terminology correctly.
- [ ] Learner can request at least one graduated hint in a lab.
- [ ] XP/streak/achievement events are stored separately from mastery evidence.
- [ ] Mastery cannot become complete from passive lesson completion alone.
- [ ] Scheduler can explain why an item was selected.
- [ ] Objective coverage remains traceable to official LPIC IDs.
- [ ] Content can label `lpic-required`, `lpic-legacy`, and `modern-practice` where applicable.

## Technical acceptance
- [ ] Go application builds and tests on Fedora.
- [ ] TUI uses Bubble Tea v2.
- [ ] SQLite schema has versioned migrations.
- [ ] Curriculum schema is versioned and validates before use.
- [ ] Rootless Podman runner implements prepare/start/check/reset/destroy lifecycle.
- [ ] State-based checker accepts at least two different valid command paths to the same correct final state.
- [ ] No host-shell execution fallback exists.
- [ ] No Internet access is required for the complete vertical slice after dependencies/images are installed.

## Security acceptance
- [ ] Lab fails closed when Podman is unavailable.
- [ ] Tests reject host-path mounts and privileged/capability escalation outside predefined profiles.
- [ ] Default lab network is disabled or explicitly isolated.
- [ ] CPU/memory/PID/time constraints exist in runner configuration.
- [ ] Host filesystem safety test runs a destructive lab and verifies a sentinel outside the sandbox is unchanged.
- [ ] Untrusted output rendered in the TUI has an explicit control-sequence handling policy and tests.

## Agent/repository acceptance
- [ ] `AGENTS.md` instructions are sufficient for a fresh coding agent to discover product, architecture, security and validation rules without chat history.
- [ ] Repetitive validation workflow is executable through scripts/skills.
- [ ] CI runs curriculum validation, Go tests, format/lint checks and security-invariant tests.
- [ ] ADR status matches actual implementation choices.

## Explicitly out of Phase 1
- libvirt/QEMU implementation;
- complete LPIC content;
- AI tutor/generation;
- full exam simulator;
- public Internet-enabled labs;
- Debian/openSUSE host packaging.
