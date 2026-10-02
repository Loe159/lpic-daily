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
- read-only rootfs for Phase-1 Podman labs, with declared writable guest paths backed by size-bounded tmpfs mounts;
- never mount `$HOME`, `/`, container-engine sockets, SSH agent sockets, credentials or arbitrary host paths;
- network `none` by default; isolated lab network only when objective requires networking;
- CPU/memory/PID/time limits;
- fresh disposable container per lab unless a scenario explicitly defines a controlled multi-step state;
- resolve a mutable local image tag to Podman's immutable `sha256:` image ID before create, and retain that ID across resets.

## Backend B: KVM/QEMU + libvirt
Use when objectives require the complete guest system: firmware/bootloader, kernel/initramfs, systemd boot behavior, block devices/partitioning/filesystem recovery, realistic network incidents or privileged administration.

Base image remains immutable. Each run gets a QCOW2 overlay/backing chain that is destroyed on reset. Prefer KVM acceleration, QEMU emulation where acceleration is unavailable only if explicitly supported.

Network defaults to an isolated libvirt network with no forwarding to the host LAN/Internet. Each VM NIC on an isolated network also references an LPIC Daily-owned libvirt nwfilter: DHCP to the libvirt gateway remains allowed, other IPv4 traffic to the host bridge address is dropped, and guest IPv6 egress is dropped. Nwfilter ownership uses namespaced metadata plus an owner-scoped deterministic UUID fallback for libvirt versions/backends that do not round-trip custom nwfilter metadata; contradictory metadata is never accepted. Scenarios needing DNS/HTTP/SSH should normally provide local simulated peer services.

Administrative commands inside full-system lab VMs use QEMU Guest Agent only as a structured transport. Trusted images provide an LPIC Daily exec trampoline. On SELinux guests it uses the distribution's `virt_qemu_ga_run_unconfined` transition so legitimate lab administration is not accidentally blocked by the guest-agent confinement domain. This does not grant host privilege: the helper exists only inside the disposable guest, receives structured argv, and the host process remains unprivileged.

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
Checks should query observable guest/container state through structured probes or narrowly scoped commands. Lab success must not depend on an exact command sequence when equivalent state can prove the skill. For inherently interactive shell-job objectives, LPIC Daily may additionally collect structured shell events (for example a successful `bg` builtin) and terminal control events such as Ctrl-Z; raw command-line text is not parsed for grading. Non-interactive guest command output is sanitized before it reaches the host terminal; raw control sequences are reserved for explicit terminal passthrough surfaces such as `:console`.

Authored JSON is validated against the embedded Draft 2020-12 schemas by the Go loaders before semantic validation and execution. Security-critical runtime limits are enforced again by the runner even if a caller bypasses authoring-time validation.

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


## Real rootless Podman host-safety integration test

Unit tests verify that generated Podman requests are non-privileged, drop capabilities before applying a predefined allowlist, use private namespaces, expose no host mounts/devices and default to no network. A separate integration test validates the boundary against a real rootless Podman service.

The test creates a sentinel file on the host, starts a disposable lab whose setup deliberately deletes and overwrites the **same absolute path** inside the guest, destroys the lab, then verifies that the host sentinel still exists with identical contents.

Run on a Linux machine with rootless Podman:

```bash
./scripts/test_podman_host_safety.sh
```

Set `LPIC_DAILY_SKIP_IMAGE_BUILD=1` only when `localhost/lpic-daily/fedora-phase1:1` is already present locally. Normal CI compiles this integration test to prevent drift; it is not claimed as executed until a compatible rootless Podman environment actually runs the script.
