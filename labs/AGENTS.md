# Lab agent instructions

Labs are adversarial inputs from the host's perspective. Read `docs/SECURITY.md` before authoring or modifying lab mechanisms.

- Choose rootless Podman unless the learning objective genuinely needs a separate machine/kernel/boot/storage stack.
- Host execution fallback is forbidden.
- Network is denied by default.
- Never mount arbitrary host paths or secrets.
- Grade observable final state, not exact command history.
- Every lab must define timeout/reset/cleanup behavior and a test-only reference solution.
