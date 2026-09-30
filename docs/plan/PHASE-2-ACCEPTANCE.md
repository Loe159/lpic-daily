# Phase 2 acceptance criteria

Status: **Implementation present; execution acceptance pending — 2026-09-29**

The checklist below remains deliberately unchecked until the corresponding acceptance evidence has actually run. The Phase-2 branch now contains implementations/tests for the control plane, image supply, isolated networking, multi-VM scenarios, serial console, reboot-aware 102.2 lab, crash reaper and real-KVM safety checks. This file is an acceptance gate, not an implementation-progress checklist.

Current blocker: GitHub Actions runs on the Phase-2 branch are terminating before any job step starts, so fresh foundation/Go/vet evidence is unavailable. Real-KVM checks additionally require an explicit KVM/libvirt host.

Review remediation completed on 2026-09-29:
- openSUSE source integrity is pinned directly in-repository to Build 18.68 / SHA-256 `f8a2703a4355a30d531021a88748f0c9d71124b7e33d26d4d49b85c2983e20d5`; runtime build no longer trusts a mutable remote checksum file;
- isolated-network allocation uses collision-aware RFC1918 /28 selection, excluding host routes and all existing libvirt networks, with a cross-process allocation lock around inventory + definition;
- VM console input is cancellable and Ctrl-] behavior plus CLI console/reboot dispatch have unit coverage; console streaming uses a dedicated libvirt connection so escape/cancellation closes only the stream, and the real-KVM harness verifies the guest remains QGA-responsive before successful teardown.
- libvirt ownership metadata now carries a stable scope derived from effective UID + VM state root; normal lifecycle lookups and the reaper reject same-prefix resources belonging to another scope;
- isolated-network allocation now uses one root-provisioned host-global flock instead of a per-state-root lock, closing cross-user/custom-state-root allocation races against shared `qemu:///system`;
- VM lab startup explicitly refuses effective UID 0 both at the CLI boundary and in the libvirt backend constructor, so direct package use cannot bypass the non-root invariant;
- per-run VM state directories are explicitly forced to mode `0711` after creation, independent of the caller's umask, so system-libvirt QEMU can traverse to managed disks without granting directory listing/read access.
- multi-VM scenario networks and leases are now backend-owned tracked resources: `Backend.Close()` cleans them even if callers omit `DestroyScenario()`, and a failed guest rollback keeps the scenario tracked for a later cleanup retry instead of tearing its shared network down;
- scenario rollback and backend-close recovery paths now use bounded cleanup contexts instead of unbounded orchestration contexts;
- VM image recipe IDs are executable contracts rather than provenance-only labels, recipe/distribution mismatches are rejected, and `virtual_size_mb` is enforced as the exact installed virtual-size contract;
- the generated image catalog preserves the pinned upstream source-integrity algorithm/encoding/value alongside the final artifact SHA-256;
- the Fedora image recipe exposes GRUB itself on the serial console before userland, and real-KVM coverage requires observable GRUB output;
- isolated-network real-KVM coverage now checks blocked public ICMP, blocked public TCP and a host-uplink TCP sentinel instead of relying on a single ping probe.

## Control plane
- [ ] application process remains non-root;
- [ ] only local `qemu:///system` is accepted by the canonical backend;
- [ ] failed/missing libvirt authorization fails closed;
- [ ] no VM lab falls back to Podman or host execution;
- [ ] concrete libvirt dependency is isolated behind project-owned interfaces;
- [ ] abandoned-resource inventory requires LPIC Daily ownership metadata, not only a name prefix.

## Image supply chain
- [ ] curriculum references an opaque image ID, never a path/URL;
- [ ] catalog entry includes SHA-256, provenance, format, architecture and firmware compatibility;
- [ ] base-image integrity is verified;
- [ ] base image is opened read-only from the VM design perspective and remains byte-identical after a lab;
- [ ] floating image identities are rejected;
- [ ] default VM image/state paths are provisioned under the system-libvirt image tree (or an explicitly verified equivalent) so qemu:///system DAC/SELinux access is testable.

## VM isolation
- [ ] generated domain names are LPIC Daily-namespaced;
- [ ] domain XML cannot contain arbitrary host filesystem mounts;
- [ ] domain XML cannot contain host PCI/USB devices;
- [ ] no arbitrary QEMU command-line extension is accepted from content;
- [ ] memory/CPU/time and writable-disk size are bounded;
- [ ] learner commands run only inside the VM.

## Storage lifecycle
- [ ] every run uses a fresh overlay;
- [ ] reset destroys and recreates disposable writable state;
- [ ] destroy removes domain + overlay + scenario scratch disks;
- [ ] cleanup is idempotent;
- [ ] abandoned resource reaping is tested.

## Networking
- [ ] `network=none` creates no guest NIC;
- [ ] `network=isolated` uses an LPIC Daily-owned network with no forwarding;
- [ ] two guests in the same scenario can communicate when required;
- [ ] guests cannot reach the public Internet/LAN in the default isolated mode.

## Interaction/checking
- [ ] learner has a serial console path that works before normal userland login;
- [ ] console cancellation interrupts the disposable VM stream and leaves teardown possible;
- [ ] structured probes work for booted guest state;
- [ ] hostile guest output is only passed raw in an explicit terminal surface;
- [ ] checker success depends on observable state, not exact commands.

## Curriculum proof
- [ ] one 104.1 storage lab succeeds end-to-end;
- [ ] one 102.2 bootloader lab succeeds end-to-end including reboot;
- [ ] failed reference state is rejected;
- [ ] test-only reference solution passes;
- [ ] both labs have graduated hints and original debriefs.

## Distribution pipeline
- [ ] Fedora base manifest/recipe;
- [ ] Debian base manifest/recipe;
- [ ] openSUSE base manifest/recipe;
- [ ] every released image records pinned upstream source integrity, build recipe/provenance and a final SHA-256;
- [ ] byte-for-byte reproducibility is not claimed unless package repositories are snapshot-pinned.

## Verification
- [ ] unit tests for XML/path/name policies;
- [ ] fake-client lifecycle tests;
- [ ] real KVM/libvirt host-sentinel test;
- [ ] base-image immutability test;
- [ ] isolated-network no-forwarding test includes real guest public ICMP/TCP failure probes and a host-uplink TCP sentinel;
- [ ] both Phase-2 reference solutions are executed by the real-KVM harness, with 102.2 rebooted before grading;
- [ ] `python3 scripts/validate_foundation.py`;
- [ ] `go test ./...`;
- [ ] `go vet ./...`;
- [ ] CI green on supported non-KVM jobs.

Canonical one-command acceptance entry point on a compatible host: `LPIC_DAILY_RUN_KVM_INTEGRATION=1 scripts/run_phase2_acceptance.sh`.
