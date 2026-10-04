# Phase 2 acceptance criteria

Status: **Real-host acceptance passed — 2026-10-04**

The checklist distinguishes evidence that can be established in generic CI from criteria that require a real KVM/libvirt host. Non-KVM items are checked only when their corresponding unit, fake-control-plane, schema or static validation has run successfully; real-machine behavior remains unchecked until the dedicated acceptance harness executes it. This file is an acceptance gate, not an implementation-progress checklist.

Real-host KVM/libvirt acceptance passed in full on 2026-10-04 using the canonical acceptance command. Phase 2 is accepted. Repository-wide CI remains a separate regression signal; acceptance evidence must not be downgraded merely because a hosted CI run is unavailable.

Review remediation completed through 2026-10-02:
- real-host validation now waits for DHCP/default-route readiness instead of assuming QEMU Guest Agent readiness implies network readiness;
- nwfilters carry an owner-scoped deterministic UUID in addition to namespaced metadata, allowing safe cleanup when libvirt omits custom nwfilter metadata on round-trip while still rejecting contradictory ownership metadata;
- trusted v2 guest images route QEMU Guest Agent execution through a project-owned SELinux-aware trampoline using the distribution-provided `virt_qemu_ga_run_unconfined` transition, so legitimate administrative labs do not depend on permissive SELinux;
- the 104.1 reference solution waits for udev/block-device convergence after repartitioning, avoiding host-speed-dependent `/dev/vdb1`/`vdb2` races;
- real-KVM reference-solution failures now preserve stdout/stderr, host iptables/nftables incompatibilities are surfaced explicitly, and the doctor libvirt cold-start probe uses a less brittle timeout;
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
- the Fedora image recipe configures GRUB and kernel output for the serial console; real-KVM coverage verifies observable boot output before the login prompt, avoiding a race where libvirt console attachment can occur just after a fast GRUB handoff;
- isolated-network real-KVM coverage now checks blocked public ICMP, blocked public TCP, a host-uplink TCP sentinel and a host-bridge-gateway TCP sentinel instead of relying on a single ping probe;
- isolated VM NICs now reference LPIC Daily-owned libvirt nwfilters; DHCP to the bridge gateway is allowed while other IPv4 host-gateway traffic and guest IPv6 egress are blocked;
- VM non-interactive command output is sanitized before rendering on the host terminal; raw control sequences remain restricted to the explicit `:console` passthrough;
- resetting one VM in a multi-machine scenario preserves the scenario's shared isolated network/filter instead of creating a private replacement network;
- the 102.2 checker now requires the persistent source configuration in `/etc/default/grub`, the generated `grub.cfg`, and the post-reboot kernel command line;
- every generated root overlay and scratch QCOW2 file is forced to mode `0600`, independent of the caller's umask.
- multi-VM scenario teardown now persists partial cleanup progress and repeated `DestroyScenario()` calls are idempotent; a late network-filter teardown failure can be retried without re-querying an already undefined network.

## Control plane
- [x] application process remains non-root;
- [x] only local `qemu:///system` is accepted by the canonical backend;
- [x] failed/missing libvirt authorization fails closed;
- [x] no VM lab falls back to Podman or host execution;
- [x] concrete libvirt dependency is isolated behind project-owned interfaces;
- [x] abandoned-resource inventory requires LPIC Daily owner identity, not only a name prefix; nwfilters may use the deterministic owner-scoped UUID fallback only when custom metadata is absent, and tests reject contradictory/foreign ownership.

## Image supply chain
- [x] curriculum references an opaque image ID, never a path/URL;
- [x] catalog entry includes SHA-256, provenance, format, architecture and firmware compatibility;
- [x] base-image integrity is verified before overlay creation;
- [x] base image is opened read-only and remained byte-identical after the 2026-10-04 real-host acceptance run;
- [x] floating image identities are rejected;
- [x] default VM image/state paths were provisioned under the system-libvirt image tree and exercised successfully on the 2026-10-04 real-host acceptance run.

## VM isolation
- [x] generated domain names are LPIC Daily-namespaced;
- [x] domain XML cannot contain arbitrary host filesystem mounts;
- [x] domain XML cannot contain host PCI/USB devices;
- [x] no arbitrary QEMU command-line extension is accepted from content;
- [x] memory/CPU/time and writable-disk size are bounded;
- [x] learner commands run only inside the VM.

## Storage lifecycle
- [x] every prepared run uses a fresh overlay;
- [x] reset destroys and recreates disposable writable state;
- [x] destroy removes domain + overlay + scenario scratch disks in fake-client lifecycle coverage;
- [x] cleanup is idempotent and retryable in unit/fake-client coverage;
- [x] abandoned resource reaping is covered by fake-control-plane lifecycle tests, including live-lease preservation and prefixed foreign-resource rejection.

## Networking
- [x] `network=none` domain policy creates no guest NIC;
- [x] `network=isolated` XML uses an LPIC Daily-owned network with no forwarding and an owned isolation filter;
- [x] two guests in the same scenario can communicate when required (2026-10-04 real-host KVM run);
- [x] guests cannot reach the public Internet/LAN in the default isolated mode (2026-10-04 real-host KVM run);
- [x] guests cannot reach host bridge services in isolated mode except DHCP required for address assignment (2026-10-04 real-host KVM run).

## Interaction/checking
- [x] learner has a serial console path that works before normal userland login;
- [x] console cancellation closes only the dedicated console stream in unit/fake-client coverage;
- [x] structured QGA command/storage probes are covered without host command fallback;
- [x] hostile guest non-console output is sanitized; raw control sequences are restricted to the explicit serial-console surface;
- [x] checker success depends on observable state, not exact commands; checker tests accept different command histories that produce the same final state.

## Curriculum proof
- [x] one 104.1 storage lab succeeds end-to-end on the 2026-10-04 real-host run;
- [x] one 102.2 bootloader lab succeeds end-to-end including reboot on the 2026-10-04 real-host run;
- [x] failed reference state is rejected;
- [x] test-only reference solutions for both Phase-2 labs pass on the 2026-10-04 real-host run;
- [x] both VM labs have four-level graduated hint ladders and non-empty authored debriefs enforced by schema/loader validation.

## Distribution pipeline
- [x] Fedora base manifest/recipe;
- [x] Debian base manifest/recipe;
- [x] openSUSE base manifest/recipe;
- [x] image-source manifests/recipes require pinned upstream integrity and generated catalogs preserve provenance/final SHA-256 contracts;
- [x] byte-for-byte reproducibility is explicitly not claimed while package repositories are not snapshot-pinned.

## Verification
- [x] unit tests for XML/path/name policies;
- [x] fake-client lifecycle tests;
- [x] real KVM/libvirt host-sentinel test (2026-10-04 focused run);
- [x] base-image immutability test (2026-10-04 focused run);
- [x] isolated-network no-forwarding test includes real guest public ICMP/TCP failure probes, a host-uplink TCP sentinel and a host-bridge-gateway TCP sentinel (2026-10-04 focused run);
- [x] both Phase-2 reference solutions were executed successfully by the real-KVM harness on 2026-10-04, with 102.2 rebooted before grading;
- [x] `python3 scripts/validate_foundation.py`;
- [x] `go test ./...`;
- [x] `go vet ./...`;
- [ ] CI green on supported non-KVM jobs (currently blocked before step execution on GitHub Actions).

Canonical one-command acceptance entry point on a compatible host: `LPIC_DAILY_RUN_KVM_INTEGRATION=1 scripts/run_phase2_acceptance.sh`.


## Host firewall compatibility note

LPIC Daily does not mutate the host firewall or third-party VPN configuration. If libvirt reports an `iptables`/nftables parsing failure while starting an isolated VM, acceptance stops fail-closed and surfaces the host compatibility error. Operators should verify `sudo iptables -w -L` and inspect third-party nftables rules (for example Tailscale) rather than weakening the VM isolation policy.
