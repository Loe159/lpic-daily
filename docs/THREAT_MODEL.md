# Threat model

Status: **Phase-2 baseline**

## Assets to protect
- learner host OS, home directory and credentials;
- SSH/GPG/API credentials and desktop-session secrets;
- local network and other machines reachable from the host;
- libvirt/Podman control sockets;
- LPIC Daily progress database;
- trusted base images and curriculum metadata;
- terminal integrity and user trust in displayed instructions.

## Trust zones
1. **Host application** — trusted code, runs unprivileged.
2. **Curriculum/lab pack** — data treated as potentially malicious, including imported community packs.
3. **Rootless container lab** — isolated execution sharing the host kernel.
4. **VM lab** — stronger guest boundary with its own kernel/full system.
5. **Local simulated lab network** — untrusted services, no forwarding to public Internet/LAN by default.
6. **Image/update source** — external supply-chain boundary; artifacts require provenance/integrity validation.

## Primary abuse cases and required mitigations

### T-001 Malicious lab requests host command execution
Mitigations:
- no host-shell fallback;
- typed runner APIs;
- never interpolate arbitrary lesson text into host shell strings;
- validate lab schemas before execution;
- setup/check code executes inside the assigned sandbox unless a narrowly defined trusted host operation exists.

### T-002 Container escape / excessive privilege
Mitigations:
- rootless Podman;
- no generic `--privileged`;
- capability allowlists by predefined lab profile;
- no host PID/IPC namespaces;
- no device passthrough by default;
- resource limits;
- prefer VM backend for tasks genuinely requiring powerful privilege.

### T-003 Host data exfiltration through mounts/sockets
Mitigations:
- never mount `$HOME`, `/`, SSH agent, GPG agent, Docker/Podman socket or arbitrary host paths;
- explicit scratch directories owned by LPIC Daily only;
- path canonicalization and traversal checks.

### T-004 Lab attacks host/LAN/Internet
Mitigations:
- network disabled or isolated by default;
- VM networks use no forwarding by default;
- isolated VM interfaces use a host-enforced libvirt nwfilter that permits DHCP to the bridge gateway but drops other IPv4 traffic to the host bridge address and drops guest IPv6 egress;
- scenario-local DNS/HTTP/SSH peers;
- explicit reviewed capability required for external networking.

### T-005 Fork bomb/disk/memory exhaustion
Mitigations:
- PID, CPU, memory and wall-clock limits;
- bounded writable storage;
- VM disk overlays with quotas/size controls where practical;
- cleanup/reaper for abandoned labs.

### T-006 Malicious terminal escape sequences
Mitigations:
- sanitize/encode untrusted guest output before rendering in non-interactive CLI/TUI surfaces;
- bound output size;
- permit raw control sequences only in explicit terminal passthrough surfaces such as the VM serial console.

### T-007 Malicious archive/content pack path traversal
Mitigations:
- reject absolute paths and `..` escapes;
- canonicalize extraction destination;
- reject symlink/hardlink escapes;
- validate all files against schema and size/count limits before activation.

### T-008 Libvirt authorization abuse
Mitigations:
- main process remains unprivileged;
- use established libvirt authorization/polkit boundaries;
- never make the full application setuid/root;
- if a helper is ever introduced, give it a minimal structured API and separate threat model.

### T-009 Base-image/supply-chain tampering
Mitigations:
- pin image identities/checksums;
- document provenance;
- verify before first use and updates;
- immutable base + per-lab overlay;
- avoid floating `latest` image references in released curriculum.

### T-010 Progress database corruption/manipulation
Mitigations:
- SQLite migrations and integrity checks;
- transactional writes;
- export/backup path;
- mastery engine treats local data as learner-owned, not security-authoritative.

## Security acceptance invariants for Phase 1
- a missing Podman backend fails closed;
- curriculum cannot request arbitrary host mounts;
- default lab network is `none`/isolated;
- no lab requires host root;
- lab success is determined by state checks in the sandbox;
- destructive test fixture proves the host working directory remains unaffected.


## Security acceptance invariants for Phase 2
- VM labs remain non-root and use local `qemu:///system` only;
- curriculum cannot request host mounts, devices, arbitrary QEMU arguments or host execution;
- isolated VM networking has no LAN/Internet forwarding and cannot reach host bridge services except the DHCP endpoint required to acquire an address;
- VM disk overlays/scratch disks are forced to mode `0600`;
- non-interactive VM output is sanitized; only explicit console passthrough is raw;
- reset recreates disposable VM state without detaching a scenario guest from its shared isolated network;
- libvirt domains, networks and nwfilters require matching LPIC Daily ownership metadata before lifecycle/reaping operations.
