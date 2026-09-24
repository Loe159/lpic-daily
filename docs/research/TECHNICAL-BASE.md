# Technical foundation research synthesis

## Recommendation
Start greenfield with Go + Bubble Tea v2 + SQLite. Model lab execution behind an interface with rootless Podman and libvirt/KVM/QEMU backends. Base VMs use immutable images plus disposable QCOW2 overlays. The application itself remains unprivileged and never uses a host-shell fallback.

## Comparison

| Candidate | Strength | Blocking/important mismatch |
|---|---|---|
| Shell Gym | Excellent real-shell reps, state checks, custom paths | PolyForm Noncommercial; root/live-host operating model is not our desired foundation |
| Arc Academy Terminal | Strong TUI/daily/streak UX; Rust terminal architecture | GPL-2.0, early project, simulated/playground isolation insufficient for full LPIC |
| SkillCoco | MIT, adaptive BKT/SM-2 ideas, local-first | Tauri/React generic desktop, Docker + host-shell fallback, open-core product assumptions |
| New Go repo | Exact security/product fit, minimal baggage | More initial implementation work |

## Isolation
Rootless Podman is cheap and appropriate for command-line/file/process exercises. It cannot emulate the complete boot/kernel/storage machine boundary. KVM/QEMU/libvirt covers those objectives and QCOW2 backing chains make resets efficient. Firecracker is not the primary backend because its common direct kernel/rootfs boot path skips firmware/bootloader material we explicitly need to teach.

## Language/UI
Go optimizes iteration and maintainability. Bubble Tea v2 gives a production-used Elm-like TUI model. SQLite keeps progress local and inspectable. Runner interfaces prevent the language choice from dictating the virtualization implementation forever.

## What not to freeze yet
- Exact mastery algorithm.
- Exact lesson/lab JSON/YAML schema.
- AI provider integration.
- VM image build pipeline.
- Update/signing mechanism.
These should be driven by the interview, threat model and vertical slice.
