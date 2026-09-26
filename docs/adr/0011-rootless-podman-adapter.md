# ADR 0011 — Rootless Podman adapter baseline

Status: **Accepted — 2026-09-26**

## Decision

Implement the fast-lab backend through Podman's native Go service bindings.

Phase 1 connects only to a **rootless** Podman service and requires cgroups v2. It does not invoke the `podman` CLI.

## Fail-closed behavior

- missing/unreachable service: error;
- rootful service: error;
- cgroups v1: error for Phase 1;
- missing image: error, never implicit pull;
- unknown capability profile: error;
- `network=isolated`: not implemented yet and returns `ErrNotSupported`;
- only `network=none` is currently accepted.

The isolated-network mode will be implemented only when LPIC Daily can create a lab-internal network without accidental WAN/LAN forwarding.

## Container baseline

Every Phase-1 container:

- is explicitly non-privileged;
- drops all Linux capabilities, then adds only the chosen allowlisted profile;
- sets no-new-privileges;
- uses no network namespace except loopback;
- uses private PID/IPC/UTS namespaces;
- has no application-provided host mounts or devices;
- ignores image-declared anonymous volumes;
- gets explicit memory, PID and wall-clock limits;
- is labeled as LPIC Daily-managed;
- is disposable.

The application requires the image to already exist in the rootless local image store. Image acquisition/provenance is a separate trusted installation/update workflow.

## API boundary

The Podman package implements the project-owned `runner.Runner` interface. Phase 1 lands lifecycle first, then exec/probes/PTY as separately testable capabilities.

Interactive TTY/stdin execution is intentionally deferred until the PTY path is implemented and tested. There is no host-shell fallback.

## Dependency

Pin the Podman v6 bindings rather than tracking an unversioned branch. The initial pin is v6.1.2.
