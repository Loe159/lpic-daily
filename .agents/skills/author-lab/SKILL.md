---
name: author-lab
description: Design a safe, state-validated Linux lab
---

# author-lab


1. Read `docs/SECURITY.md`, `labs/AGENTS.md`, and the objective metadata.
2. Select the least-privileged backend capable of teaching the skill: rootless Podman first, libvirt VM when kernel/boot/storage/system integration requires it.
3. Define initial state, learner goal, allowed capabilities, network policy, timeout, cleanup and deterministic checks.
4. Validate observable final state, not exact commands.
5. Randomize superficial values where useful while keeping grading deterministic.
6. Add a reference solution used only by tests and authoring validation.
7. Prove reset/cleanup works after success, failure, interruption and timeout.

