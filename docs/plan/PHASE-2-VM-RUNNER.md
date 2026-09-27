# Phase 2 — VM runner

Status: **In progress — 2026-09-27**

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
    - reproducible provenance/checksum metadata;
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
