# ADR 0011 — Rootless Podman HTTP adapter baseline

Status: **Accepted — 2026-09-26**

## Decision

Implement the fast-lab backend against Podman's versioned HTTP API over the local rootless Unix socket, using Go's standard `net/http` stack and project-owned DTOs for the small API subset LPIC Daily needs.

Do **not** import the full Podman Go module into the shipped application.

## Why the official bindings were not retained

A compile spike with `go.podman.io/podman/v6/pkg/bindings` worked conceptually but expanded the module graph by well over one hundred indirect requirements, including container-build/image/signing/telemetry packages unrelated to the runtime calls LPIC Daily needs.

The official bindings are valid and supported; the problem is proportionality for this small client. Podman documents its HTTP API as the stable external interface, so using that interface directly preserves compatibility while sharply reducing dependency/supply-chain surface.

## Transport

- local Unix socket only;
- no TCP or SSH transport in Phase 1;
- default socket: `$XDG_RUNTIME_DIR/podman/podman.sock`;
- API base: `/v6.0.0/libpod`;
- responses are size-bounded;
- redirects are rejected.

## Fail-closed behavior

- missing/unreachable service: error;
- rootful service: error;
- cgroups v1: error for Phase 1;
- missing image: error, never implicit pull;
- unknown capability profile: error;
- `network=isolated`: not implemented yet and returns `ErrNotSupported`;
- only `network=none` is currently accepted.

## Container baseline

Every Phase-1 container request:

- sets `privileged=false`;
- drops all Linux capabilities, then adds only the selected allowlisted profile;
- sets no-new-privileges;
- uses network namespace `none` (loopback only);
- uses private PID/IPC/UTS namespaces;
- has no application-defined mounts, volumes or devices in the DTO;
- ignores image-declared anonymous volumes;
- gets explicit memory, PID and wall-clock limits;
- is labeled as LPIC Daily-managed;
- is disposable.

## API surface

The project owns minimal request/response structs rather than mirroring all Podman models. Unknown response fields are ignored, while request fields are explicit and reviewable.

The adapter remains behind `runner.Runner`; later Podman API changes do not affect the learning domain.

## Exec and probe implementation checkpoint

The first executable checkpoint deliberately separates non-interactive control from the future learner PTY.

- setup/check commands use Docker-compatible v1.40 exec endpoints exposed by the same local Podman service;
- exec is detached, non-privileged and polled through exec-inspect for the final exit code;
- stdin, attached stdout/stderr and TTY requests fail with `ErrNotSupported` until the dedicated attach/PTY implementation lands;
- filesystem probes use the native container archive endpoint and parse tar metadata, including UID/GID and names when available;
- file reads are byte-bounded;
- process probes request stable `pid,comm,args` descriptors from the native top endpoint.

Podman documents that its service exposes both the native Libpod API and a Docker v1.40 compatibility API. Using the compatibility exec endpoints avoids reimplementing interactive framing for setup operations while keeping all traffic on the verified rootless Unix socket.

## Immutable local image resolution

Phase-1 content may refer to the locally installed image by its installation tag, but the adapter never uses that mutable tag directly for the lab lifecycle. `Prepare` inspects the tag through the rootless local Podman API, requires a canonical `sha256:<64 hex>` image ID, and creates the container from that ID. The resolved definition is retained, so `Reset` recreates from the same immutable ID even if the tag is moved concurrently. Implicit pulls remain forbidden. Release signing/provenance remains a packaging responsibility above this runtime boundary.
