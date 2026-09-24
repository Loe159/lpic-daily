# ADR 0003 — Dual lab isolation backend

Status: **Accepted — 2026-09-24**

## Decision
Implement a backend interface with:
1. rootless Podman for lightweight labs;
2. KVM/QEMU managed by libvirt with QCOW2 overlays for full-system labs.

Never fall back to the host shell.

## Rationale
Containers are fast and cheap but share the host kernel. LPIC includes firmware, bootloader, initramfs, system boot, partitions and filesystem repair, which require a more complete guest machine. QCOW2 backing chains make disposable VM state practical.

## Rejected as primary backend
- Firecracker: fast microVM isolation, but its direct kernel/rootfs boot model skips important bootloader/firmware learning paths.
- bubblewrap/systemd-nspawn alone: useful namespace isolation but not a separate kernel/full virtual machine.
- unrestricted local shell: unacceptable host risk.
