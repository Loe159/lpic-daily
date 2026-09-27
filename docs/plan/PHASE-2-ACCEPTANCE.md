# Phase 2 acceptance criteria

Status: **Draft — 2026-09-27**

## Control plane
- [ ] application process remains non-root;
- [ ] only local `qemu:///system` is accepted by the canonical backend;
- [ ] failed/missing libvirt authorization fails closed;
- [ ] no VM lab falls back to Podman or host execution;
- [ ] concrete libvirt dependency is isolated behind project-owned interfaces.

## Image supply chain
- [ ] curriculum references an opaque image ID, never a path/URL;
- [ ] catalog entry includes SHA-256, provenance, format, architecture and firmware compatibility;
- [ ] base-image integrity is verified;
- [ ] base image is opened read-only from the VM design perspective and remains byte-identical after a lab;
- [ ] floating image identities are rejected.

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
- [ ] every released image has reproducible provenance/checksum metadata.

## Verification
- [ ] unit tests for XML/path/name policies;
- [ ] fake-client lifecycle tests;
- [ ] real KVM/libvirt host-sentinel test;
- [ ] base-image immutability test;
- [ ] isolated-network no-forwarding test;
- [ ] `python3 scripts/validate_foundation.py`;
- [ ] `go test ./...`;
- [ ] `go vet ./...`;
- [ ] CI green on supported non-KVM jobs.
