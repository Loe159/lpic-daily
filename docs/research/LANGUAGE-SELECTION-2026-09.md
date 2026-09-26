# Language selection research — 2026-09-26

This document records the evidence used by ADR 0006.

## Requirements tested

1. TUI quality and terminal correctness.
2. PTY support for real interactive shell/jobs exercises.
3. Rootless Podman integration without shell-string orchestration.
4. libvirt/KVM integration for Phase 2.
5. SQLite and deterministic local state.
6. Linux binary packaging.
7. security/auditability.
8. agent/human maintainability.
9. ecosystem freshness.

## Go evidence

- Current stable Go line at research time: Go 1.27; Go 1.27.0 released 2026-08-19.
- Bubble Tea v2 is actively maintained; v2.0.10 released 2026-09-24.
- Podman v6 publishes native Go bindings for images, containers, pods, networks, manifests and other APIs.
- libvirt publishes an official Go binding and calls its API production ready/stable.
- DigitalOcean `go-libvirt` is a pure-Go RPC implementation and was active in 2026.
- `modernc.org/sqlite` offers a current CGO-free SQLite implementation.
- `creack/pty` is a mature PTY package.

Primary references:
- https://go.dev/doc/devel/release
- https://github.com/charmbracelet/bubbletea/releases
- https://pkg.go.dev/go.podman.io/podman/v6/pkg/bindings
- https://pkg.go.dev/libvirt.org/go/libvirt
- https://github.com/digitalocean/go-libvirt
- https://pkg.go.dev/modernc.org/sqlite
- https://github.com/creack/pty

## Rust evidence

- Ratatui 0.30.x is mature and actively maintained.
- `nix` exposes direct Unix PTY APIs.
- `rusqlite` / SQLx provide mature SQLite support.
- Bollard explicitly supports Docker and Podman, including rootless Podman socket discovery.
- libvirt Rust bindings exist and are usable, but current docs report materially lower documentation/example coverage than ideal for our main VM control plane.
- `portable-pty` is useful, but a 2026 Linux exec-error issue reinforces the value of keeping PTY behind our own abstraction rather than coupling the architecture to one crate.

Primary references:
- https://docs.rs/ratatui/
- https://docs.rs/nix/latest/nix/pty/
- https://docs.rs/rusqlite/
- https://docs.rs/bollard/
- https://docs.rs/crate/virt/latest
- https://github.com/wezterm/wezterm/issues/7742

## Python evidence

- Textual is a capable MIT terminal UI framework.
- libvirt's Python bindings are described as essentially complete mappings of the API.
- PyInstaller can create standalone Linux bundles but bundles the interpreter/dependencies and has glibc compatibility constraints across build targets.

Primary references:
- https://textual.textualize.io/
- https://libvirt.org/python
- https://pyinstaller.org/en/stable/

## Conclusion

Go is retained after explicit re-evaluation. The strongest project-specific reason is not generic developer productivity: it is the combination of **native Podman bindings + mature libvirt choices + simple native CLI distribution**, while still providing a strong TUI/PTY/SQLite ecosystem.

Rust is the strongest alternative and should be reconsidered if the Go integration spike exposes a concrete limitation.
