# Security and isolation model

## Security objective
A learner must be able to perform destructive Linux administration tasks without trusting lesson content enough to protect the host manually.

## Trust boundaries
Treat as untrusted:
- commands typed by the learner;
- lab setup/reference-solution scripts;
- imported curriculum packs;
- files produced inside a lab;
- network services exposed by a lab.

The host, LPIC Daily binary, signed/bundled base-image metadata and progress database are trusted components. Trust in third-party images must be minimized and their provenance pinned.

## Backend A: rootless Podman
Use for filesystem, shell, text processing, processes, package-query simulations and other tasks that do not require a separate kernel/boot stack.

Baseline controls:
- rootless user namespace;
- no host PID/IPC namespace;
- no privileged mode;
- drop Linux capabilities by default and add only predefined profile capabilities;
- no host device passthrough by default;
- read-only rootfs where compatible, explicit writable scratch volumes;
- never mount `$HOME`, `/`, container-engine sockets, SSH agent sockets, credentials or arbitrary host paths;
- network `none` by default; isolated lab network only when objective requires networking;
- CPU/memory/PID/time limits;
- fresh disposable container per lab unless a scenario explicitly defines a controlled multi-step state.

## Backend B: KVM/QEMU + libvirt
Use when objectives require the complete guest system: firmware/bootloader, kernel/initramfs, systemd boot behavior, block devices/partitioning/filesystem recovery, realistic network incidents or privileged administration.

Base image remains immutable. Each run gets a QCOW2 overlay/backing chain that is destroyed on reset. Prefer KVM acceleration, QEMU emulation where acceleration is unavailable only if explicitly supported.

Network defaults to an isolated libvirt network with no forwarding to the host LAN/Internet. Scenarios needing DNS/HTTP/SSH should normally provide local simulated peer services.

## Why not Firecracker as primary VM backend
Firecracker's normal model supplies a kernel image and root filesystem directly to the microVM. That is excellent for fast workloads but bypasses exactly the firmware/bootloader path needed to teach BIOS/UEFI/GRUB and related LPIC objectives. It could become an optional backend later, not the canonical full-system backend.

## Why not bubblewrap/systemd-nspawn as the only boundary
Namespace sandboxes/containers share the host kernel. They can be useful, but they cannot faithfully teach every boot, kernel and block-device objective. The product needs a hypervisor boundary anyway, so security architecture should not pretend one sandbox solves all LPIC topics.

## Forbidden behavior
- No host-shell fallback when Podman/libvirt is missing or fails.
- No `--privileged` convenience mode as a generic escape hatch.
- No unvalidated interpolation of lesson strings into a host shell command.
- No automatic mounting of arbitrary host paths requested by content.
- No network access by default.
- No secrets/API keys inside lab guests unless a dedicated threat model explicitly allows it.

## Validation model
Checks should query observable guest/container state through structured probes or narrowly scoped commands. Lab success must not depend on the learner entering a specific command sequence.

## Threat-model backlog
Before Phase 1 implementation, formalize threats for:
- malicious imported course pack;
- container escape and excessive capabilities;
- libvirt authorization/polkit misuse;
- QCOW2 image provenance and tampering;
- terminal escape sequences and hostile output;
- symlink/path traversal in pack extraction;
- denial of service via fork bombs/disk exhaustion;
- local privilege escalation through helper APIs;
- isolation between simultaneous labs;
- update/signing mechanism.
