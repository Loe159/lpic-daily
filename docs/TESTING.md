# Testing strategy

## Foundation
`python3 scripts/validate_curriculum.py` verifies structural LPIC coverage before application code exists.

## Future test layers
- Unit tests: curriculum parser, scheduler, mastery math, checker primitives.
- Golden/schema tests: content parsing and stable rendering.
- TUI tests: deterministic model/update/view tests without real terminals where possible.
- Podman integration tests: run every capability profile against disposable containers.
- libvirt integration tests: provision overlay, mutate guest, reboot, verify state, destroy overlay.
- Lab conformance tests: every lab setup and reference solution must lead from fresh state to pass; deliberate wrong states must fail.
- Security negative tests: path traversal, hostile packs, mount escape attempts, command injection, excessive capabilities, network-deny assertions, resource limits.
- End-to-end smoke: install -> first daily session -> lab -> progress persistence -> reset/export.

## CI tiers
Fast PR CI should not require KVM. Hypervisor scenarios run on a dedicated KVM-capable runner or scheduled integration environment. Never weaken VM tests merely to make generic hosted CI pass.
