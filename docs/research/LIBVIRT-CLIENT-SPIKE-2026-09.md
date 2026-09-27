# Libvirt client / VM runner research — 2026-09-27

## Scope

Phase 2 needs a full-system backend for LPIC objectives that cannot be represented faithfully in a container: firmware/bootloader, kernel/initramfs, partitions, filesystems, recovery and realistic isolated multi-machine networking.

Primary sources reviewed:
- libvirt Go bindings: https://libvirt.org/golang.html
- QEMU driver/security architecture: https://libvirt.org/drvqemu.html
- libvirt connection URIs: https://libvirt.org/uri.html
- disk image chains: https://libvirt.org/kbase/backing_chains.html
- domain XML: https://libvirt.org/formatdomain.html
- DigitalOcean pure-Go client: https://github.com/digitalocean/go-libvirt

## Connection mode

### qemu:///session

Advantages:
- QEMU runs with the learner user's UID/GID;
- no system libvirt authorization is required.

Important mismatch:
- libvirt documents user-session networking as substantially more limited;
- isolated libvirt networks/bridges needed by later multi-machine labs fit the system instance better;
- sVirt/SELinux separation is not equivalent to the system instance.

### qemu:///system — selected

LPIC Daily connects as the regular desktop user to the local system libvirt service. It does not become root. Authorization remains libvirt/polkit territory.

Benefits:
- standard libvirt virtual networks;
- normal Fedora QEMU/libvirt confinement path;
- suitable for future multi-machine DNS/HTTP/SSH/routing labs;
- standard storage/device lifecycle APIs.

Constraints imposed by LPIC Daily:
- local URI only;
- no remote TCP/SSH libvirt URIs;
- no arbitrary host devices;
- no arbitrary host file paths from curriculum;
- every created resource is namespaced and labeled as LPIC Daily-owned;
- failure to authorize is a hard error, never a fallback.

## Go client options

### libvirt.org/go/libvirt

This is the modern upstream Go binding listed by libvirt. It is the closest binding to the native API but brings a native libvirt/CGO build dependency.

Strengths:
- upstream;
- broad API coverage;
- native API semantics.

Cost for this project:
- Fedora/Debian build environments need libvirt development headers/libraries;
- cross-compilation/distribution becomes more coupled to the host;
- the application binary is no longer a mostly self-contained Go artifact.

### github.com/digitalocean/go-libvirt — selected for Phase 2 spike

The package speaks libvirt's RPC/XDR protocol in pure Go and had fresh published code in August 2026. License is Apache-2.0.

Caveat from its own documentation: the API is not considered stable and the project recommends vendoring.

Mitigation:
- pin an exact pseudo-version;
- expose none of its public types outside `internal/runner/libvirt`;
- wrap the small subset LPIC Daily needs behind a project-owned client interface;
- maintain contract tests against that interface;
- reconsider official CGO bindings if the pure-Go client lacks required APIs or becomes insufficiently maintained.

We do not vendor the dependency initially because that would add a large generated-code copy to this repository. Exact pinning + the adapter boundary gives us deterministic builds while keeping third-party source ownership clear.

## Base images and overlays

Curriculum never provides a host path.

A trusted image catalog maps a stable image ID to:
- absolute installed base-image path;
- distribution/version/architecture;
- disk format;
- virtual size;
- SHA-256;
- provenance metadata;
- firmware compatibility;
- guest-agent capability where applicable.

Before use, the base image checksum is verified.

Every run gets a project-owned QCOW2 overlay. The base image remains immutable. Reset destroys/recreates the overlay.

For the first implementation, overlay creation may use the trusted host utility `qemu-img` with structured argv only. This is a control-plane helper, not a learner shell and not curriculum-defined execution. All input paths come from the trusted image catalog or an LPIC Daily state directory.

## Networking

Default VM networking remains isolated.

- `network=none`: no guest NIC.
- `network=isolated`: attach only to an LPIC Daily-owned libvirt network with no forwarding element.
- Internet/LAN forwarding is outside Phase 2.
- Multi-machine scenarios later share one per-lab isolated network.

## Guest control

The VM runner needs two channels with different purposes:

1. serial console: learner interaction, including bootloader/recovery work where the normal OS may not be available;
2. structured guest probe: deterministic checks after a booted guest is available.

The first probe implementation should use a preinstalled guest-side agent/transport in trusted base images, not SSH credentials copied from the host. Boot/recovery labs must not rely exclusively on that agent for success because the learning task may deliberately break normal boot.

## First end-to-end objectives

- **102.2** — bootloader: repair/modify GRUB and prove the guest still boots to the requested state.
- **104.1** — storage: partition additional virtual disks and create the requested filesystem/swap state.

These objectives exercise the VM-specific boundary without prematurely scaling all 101–104 content.
