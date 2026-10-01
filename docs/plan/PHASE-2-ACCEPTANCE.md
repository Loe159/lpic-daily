# Phase 2 acceptance criteria

Status: **Implementation present; execution acceptance pending — 2026-09-29**

The checklist below remains deliberately unchecked until the corresponding acceptance evidence has actually run. The Phase-2 branch now contains implementations/tests for the control plane, image supply, isolated networking, multi-VM scenarios, serial console, reboot-aware 102.2 lab, crash reaper and real-KVM safety checks. This file is an acceptance gate, not an implementation-progress checklist.

Current blocker: only the real-KVM/libvirt acceptance evidence remains unavailable in generic hosted CI. The supported non-KVM CI gates are green on the Phase-3 integration branch; KVM-dependent criteria remain deliberately unchecked.

Review remediation completed through 2026-09-30:
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
- isolated-network real-KVM coverage now checks blocked public ICMP, blocked public TCP, a host-uplink TCP sentinel and a host-bridge-gateway TCP sentinel instead of relying on a single ping probe;
- isolated VM NICs now reference LPIC Daily-owned libvirt nwfilters; DHCP to the bridge gateway is allowed while other IPv4 host-gateway traffic and guest IPv6 egress are blocked;
- VM non-interactive command output is sanitized before rendering on the host terminal; raw control sequences remain restricted to the explicit `:console` passthrough;
- resetting one VM in a multi-machine scenario preserves the scenario's shared isolated network/filter instead of creating a private replacement network;
- the 102.2 checker now requires the persistent source configuration in `/etc/default/grub`, the generated `grub.cfg`, and the post-reboot kernel command line;
- every generated root overlay and scratch QCOW2 file is forced to mode `0600`, independent of the caller's umask.
- multi-VM scenario teardown now persists partial cleanup progress and repeated `DestroyScenario()` calls are idempotent; a late network-filter teardown failure can be retried without re-querying an already undefined network.

## Parallel Phase-3 work

Phase-3 Exam-101 curriculum work may proceed while this real-host KVM acceptance execution is pending. This does not change any checkbox in this file and does not imply Phase 2 acceptance. A fully validated release candidate still requires the canonical real-host acceptance command below to pass.

## Control plane
- [x] application process remains non-root;
- [x] only local `qemu:///system` is accepted by the canonical backend;
- [x] failed/missing libvirt authorization fails closed;
- [x] no VM lab falls back to Podman or host execution;
- [x] concrete libvirt dependency is isolated behind project-owned interfaces;
- [ ] abandoned-resource inventory requires LPIC Daily ownership metadata, not only a name prefix.

## Image supply chain
- [x] curriculum references an opaque image ID, never a path/URL;
- [x] catalog entry includes SHA-256, provenance, format, architecture and firmware compatibility;
- [x] base-image integrity is verified before overlay creation;
- [ ] base image is opened read-only from the VM design perspective and remains byte-identical after a lab;
- [x] floating image identities are rejected;
- [ ] default VM image/state paths are provisioned under the system-libvirt image tree (or an explicitly verified equivalent) so qemu:///system DAC/SELinux access is testable.

## VM isolation
- [x] generated domain names are LPIC Daily-namespaced;
- [x] domain XML cannot contain arbitrary host filesystem mounts;
- [x] domain XML cannot contain host PCI/USB devices;
- [x] no arbitrary QEMU command-line extension is accepted from content;
- [x] memory/CPU/time and writable-disk size are bounded;
- [ ] learner commands run only inside the VM.

## Storage lifecycle
- [x] every prepared run uses a fresh overlay;
- [x] reset destroys and recreates disposable writable state;
- [x] destroy removes domain + overlay + scenario scratch disks in fake-client lifecycle coverage;
- [x] cleanup is idempotent and retryable in unit/fake-client coverage;
- [ ] abandoned resource reaping is tested.

## Networking
- [x] `network=none` domain policy creates no guest NIC;
- [x] `network=isolated` XML uses an LPIC Daily-owned network with no forwarding and an owned isolation filter;
- [ ] two guests in the same scenario can communicate when required;
- [ ] guests cannot reach the public Internet/LAN in the default isolated mode;
- [ ] guests cannot reach host bridge services in isolated mode except DHCP required for address assignment.

## Interaction/checking
- [ ] learner has a serial console path that works before normal userland login;
- [x] console cancellation closes only the dedicated console stream in unit/fake-client coverage;
- [x] structured QGA command/storage probes are covered without host command fallback;
- [x] hostile guest non-console output is sanitized; raw control sequences are restricted to the explicit serial-console surface;
- [ ] checker success depends on observable state, not exact commands.

## Curriculum proof
- [ ] one 104.1 storage lab succeeds end-to-end;
- [ ] one 102.2 bootloader lab succeeds end-to-end including reboot;
- [ ] failed reference state is rejected;
- [ ] test-only reference solution passes;
- [ ] both labs have graduated hints and original debriefs.

## Distribution pipeline
- [x] Fedora base manifest/recipe;
- [x] Debian base manifest/recipe;
- [x] openSUSE base manifest/recipe;
- [x] image-source manifests/recipes require pinned upstream integrity and generated catalogs preserve provenance/final SHA-256 contracts;
- [ ] byte-for-byte reproducibility is not claimed unless package repositories are snapshot-pinned.

## Verification
- [x] unit tests for XML/path/name policies;
- [x] fake-client lifecycle tests;
- [ ] real KVM/libvirt host-sentinel test;
- [ ] base-image immutability test;
- [ ] isolated-network no-forwarding test includes real guest public ICMP/TCP failure probes, a host-uplink TCP sentinel and a host-bridge-gateway TCP sentinel;
- [ ] both Phase-2 reference solutions are executed by the real-KVM harness, with 102.2 rebooted before grading;
- [x] `python3 scripts/validate_foundation.py`;
- [x] `go test ./...`;
- [x] `go vet ./...`;
- [x] CI green on supported non-KVM jobs.

Canonical one-command acceptance entry point on a compatible host: `LPIC_DAILY_RUN_KVM_INTEGRATION=1 scripts/run_phase2_acceptance.sh`.
