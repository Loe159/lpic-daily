# Podman client dependency spike — 2026-09-26

## Result

The initial proposal used Podman's official Go bindings. A real `go mod tidy` spike showed that adding those bindings expanded the project module graph from the small SQLite dependency set to more than one hundred indirect requirements.

Examples included Buildah, containers/image, sigstore, gpgme, OpenTelemetry, Docker/Moby API modules and multiple image/archive stacks. Many are legitimate Podman dependencies, but LPIC Daily only needs a narrow service-client surface.

## Decision

Use Podman's versioned HTTP API directly over the local Unix socket with Go standard-library HTTP/JSON.

This is not an attempt to reimplement Podman. The client only owns DTOs for:
- system info/rootless verification;
- local image existence;
- container create/start/remove/reset;
- later: exec, stat/read/process probes.

## Why this is preferable here

- dramatically smaller dependency graph;
- easier auditing and SBOM review;
- no CGO introduced by Podman transitive packages;
- HTTP requests are straightforward to fake in unit tests over a Unix socket;
- the app still uses Podman's documented stable external API;
- vendor-specific code remains confined to one adapter.

If the direct client grows large enough to approximate a general Podman SDK, revisit this decision rather than continuing to hand-maintain broad API coverage.
