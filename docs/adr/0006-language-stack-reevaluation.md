# ADR 0006 — Language and terminal stack after explicit re-evaluation

Status: **Accepted — 2026-09-26**

## Context

The earlier Go decision was reopened before implementation because it had been influenced by a misunderstanding of conversational wording. The choice must therefore stand on technical merit alone.

LPIC Daily needs:

- Linux-first terminal UX;
- deterministic parsing/validation of a large content graph;
- local SQLite state;
- interactive PTY labs;
- first-class rootless Podman integration;
- later KVM/QEMU/libvirt orchestration;
- straightforward RPM/DEB-style binary distribution;
- a codebase that coding agents and human contributors can review reliably;
- a small attack surface around privileged/system integration.

## Decision

Use **Go** for the core application.

Phase-1 baseline:

- Go 1.27;
- standard library first;
- Bubble Tea v2 when the TUI layer is introduced;
- SQLite behind a repository interface;
- direct/rootless Podman API integration rather than shell command construction;
- libvirt behind the runner interface, preferring a typed API/RPC client over `virsh` command construction;
- PTY through a narrow library abstraction;
- forbid application use of Go `unsafe` unless a future ADR explicitly justifies it.

## Comparison

### Go

Strengths:

- Podman is itself Go-based and publishes current native Go bindings for its service API.
- libvirt publishes official Go bindings and describes them as production ready.
- A maintained pure-Go libvirt RPC client also exists, allowing a no-CGO path if its API coverage satisfies our VM runner.
- Bubble Tea v2 is mature and actively maintained.
- Mature PTY libraries exist.
- Easy native CLI distribution and fast tests/builds.
- Small language surface is favorable for agent-generated patches and review.

Trade-offs:

- Rust provides stronger compile-time ownership guarantees.
- Official libvirt Go bindings use C bindings; the pure-Go alternative must be evaluated for exact Phase-2 API coverage.
- Go's type system is less expressive than Rust's for some domain invariants, so validation must remain explicit.

### Rust

Strengths:

- Excellent memory/ownership safety.
- Ratatui is mature and highly active.
- Strong Unix primitives through `nix` and mature SQLite options.
- Bollard supports Podman as a first-class Docker-compatible runtime.

Trade-offs for this project:

- Native Podman-specific integration is less direct than using Podman's own Go bindings.
- libvirt Rust bindings are viable but have less documentation/API example coverage than the Go/Python options and independent release cadence.
- Async/runtime/ownership complexity adds implementation and review overhead without replacing our actual security boundary, which remains Podman/KVM/libvirt.

Rust remains the fallback choice if Go's system integrations prove inadequate in a real spike.

### Python

Strengths:

- Very fast content/tooling iteration.
- Textual is a capable TUI framework.
- libvirt's Python bindings are mature and broadly generated from the C API.

Trade-offs:

- Standalone distribution is heavier and less predictable across old/new glibc targets.
- Runtime dependency/packaging complexity is undesirable for a system utility that should feel native.
- Dynamic typing weakens some compile-time contracts around runner/checker APIs.

Python remains appropriate for repository tooling/validators, not the primary shipped application.

### TypeScript/Tauri/Node

Useful for rich desktop applications, but it adds a web/JS runtime/toolchain and does not improve Podman/libvirt/PTY integration for a terminal-first Linux utility. Rejected for the core.

## Why memory safety does not force Rust here

Learner commands never run in-process as native code. The critical boundary is process/namespace/hypervisor isolation and strict structured APIs. Go remains memory-safe for ordinary application code; the project forbids `unsafe` by default.

## Reversal criteria

Reopen this ADR only if a concrete implementation spike proves one of these:

- required Podman behavior cannot be expressed reliably through supported APIs;
- required libvirt behavior cannot be implemented without unacceptable CGO/distribution constraints;
- PTY integration cannot support the required jobs/process labs safely;
- measured maintainability or correctness problems materially change the trade-off.

Language preference alone is not sufficient.
