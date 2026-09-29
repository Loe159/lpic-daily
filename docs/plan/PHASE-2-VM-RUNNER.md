# Phase 2 — VM runner

Status: **Implementation ready for acceptance validation — 2026-09-29**

## Goal

Add the full-system backend required for boot, storage and later realistic networking objectives without weakening the Phase-1 host-safety invariants.

First end-to-end targets:
- **102.2 — Installation et configuration du bootloader**
- **104.1 — Partitions et création de systèmes de fichiers**

## Technical baseline

- local `qemu:///system`;
- LPIC Daily process remains non-root;
- libvirt authorization/polkit controls access;
- pure-Go libvirt RPC client behind a project-owned interface;
- immutable verified QCOW2 bases;
- disposable per-run QCOW2 overlays;
- serial console for learner interaction;
- deterministic structured probes/checkers;
- network none by default, isolated libvirt network when requested;
- no Internet/LAN forwarding in Phase 2.

## Current checkpoint

Implemented on `phase-2-vm-runner-complete`:
- system-libvirt control plane remains restricted to local `qemu:///system`, behind project-owned interfaces;
- immutable catalogued QCOW2 bases, SHA-256 runtime verification, bounded disposable overlays and scratch disks;
- explicit serial-console learner surface (`:console`) with local Ctrl-] escape, terminal restoration and a dedicated libvirt stream connection; escape/cancellation closes the console stream without powering off the guest;
- explicit VM reboot (`:reboot`) that waits for a changed guest `boot_id` through QEMU Guest Agent before returning;
- 104.1 partition/filesystem state-based VM lab;
- 102.2 GRUB 2 lab that requires a persistent kernel argument and verifies it after a real reboot;
- LPIC Daily-owned isolated libvirt networks with no forwarding and shared-network multi-VM scenarios; subnet allocation excludes host/libvirt routes and is serialized across users/processes by one root-provisioned host-global lock;
- per-run leases plus startup reaping for abandoned LPIC Daily domains, networks and disposable state without reaping a live concurrent run; ownership is verified through namespaced libvirt metadata carrying a stable effective-UID + state-root scope, and every lifecycle lookup rejects foreign scopes rather than trusting a name prefix;
- explicit Fedora, Debian and openSUSE image-supply manifests/build pipeline with source digests pinned directly in-repository (no mutable remote checksum trust at build time);
- default VM storage provisioned below `/var/lib/libvirt/images/lpic-daily/<uid>`, with per-run state directories forced to execute-traversable `0711` even under restrictive umasks and a shared root-owned network-allocation lock at the parent level; this keeps qemu:///system disks out of private home/XDG paths while coordinating its host-global network namespace;
- opt-in real-KVM integration coverage for host sentinel preservation, base-image immutability, private two-guest communication, failed public egress, crash reaping and both 104.1/102.2 reference solutions (including the 102.2 reboot);
- generic CI compiles the KVM integration test but does not pretend to execute it without a KVM/libvirt host.

VM lab startup refuses effective UID 0 both at the CLI boundary and inside the libvirt backend constructor, keeping the non-root invariant fail-closed even for direct package use. Custom image/state roots remain supported, but they intentionally share the provisioned host-global network-allocation lock because `qemu:///system` has one host-wide network namespace.

Acceptance is intentionally not declared complete yet. On 2026-09-29 the repository GitHub Actions runs for this branch are failing before any job step starts, so the branch has not obtained fresh `validate_foundation`, `go test`, `go vet` or format results from CI. The real-KVM test also still requires an explicit compatible host run with `LPIC_DAILY_RUN_KVM_INTEGRATION=1`.

Next action is validation, not additional Phase-2 feature scope: restore/diagnose CI execution, run the standard validation suite, build/install the trusted Fedora image, then execute both VM reference labs and the opt-in KVM conformance test on a real host.

## Work order

1. **Contracts and threat model**
   - ADR 0015;
   - VM-specific lab schema fields with tight bounds;
   - trusted image catalog contract;
   - resource naming/path policy.

2. **Dependency spike**
   - pin pure-Go libvirt client;
   - measure module graph;
   - connect only to local `qemu:///system`;
   - capability/version probe;
   - fake client tests.

3. **Image/overlay layer**
   - image manifest + SHA-256 verification;
   - state/data directory layout;
   - structured `qemu-img` overlay creation;
   - atomic cleanup/reset;
   - disk-space and virtual-size bounds.

4. **Domain XML/lifecycle**
   - Q35/virtio baseline;
   - memory/CPU limits;
   - serial console;
   - UEFI/BIOS profile from trusted image compatibility;
   - define/start/stop/undefine;
   - namespaced metadata.

5. **Guest interaction**
   - serial PTY/passthrough;
   - trusted guest probe transport for booted-state checks;
   - timeout/cancellation;
   - no host credential forwarding.

6. **Networking**
   - LPIC Daily-owned isolated libvirt network;
   - no `<forward>`;
   - per-lab lifecycle;
   - multi-machine-ready addressing.

7. **Storage devices**
   - bounded empty QCOW2 scratch disks;
   - attach as virtio;
   - stable per-scenario device identity.

8. **Reference lab 104.1**
   - additional disk;
   - partition + filesystem/swap;
   - checker validates partition table/filesystem state;
   - reset restores pristine state.

9. **Reference lab 102.2**
   - deliberately broken/alternate GRUB scenario;
   - learner works through serial/boot path;
   - reboot is part of evaluation;
   - checker proves requested boot state without accepting a command transcript.

10. **Guest image pipeline**
    - Fedora, Debian and openSUSE manifests/build recipes;
    - pinned source integrity + recorded build provenance/final checksum metadata;
    - guest probe + serial console baseline.

11. **Real security integration**
    - host sentinel;
    - base-image immutability;
    - isolated network verification;
    - crash/timeout reaper.

## Explicitly deferred

- public Internet access;
- USB/PCI passthrough;
- arbitrary ISO attachment;
- shared host folders;
- nested virtualization;
- Windows guests;
- VM snapshots as learner-visible persistence;
- live migration.

## Exit condition

Phase 2 is complete only when `PHASE-2-ACCEPTANCE.md` is green.
