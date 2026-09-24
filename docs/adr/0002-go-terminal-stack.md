# ADR 0002 — Go terminal application stack

Status: **Accepted — 2026-09-24**

## Decision
Use Go for the core application. Use Bubble Tea v2 for the TUI and SQLite for local persistence.

## Rationale
The application is mostly orchestration, validation, learning logic and terminal UX. Go offers a small readable language, rapid tests/builds and straightforward native distribution. Bubble Tea v2 is a mature terminal framework. The lab security boundary comes from Podman/KVM, not from embedding untrusted code in-process.

## Alternatives
Rust + Ratatui: strong option and attractive for low-level software, but greater implementation complexity with limited security benefit for this architecture.
Tauri/React: richer GUI but conflicts with terminal-first/offline-simple goals and adds a JS toolchain.
Python: fast authoring but weaker single-binary distribution and dependency/environment management for a system utility.

## Reversibility
Runner/content schemas should remain language-agnostic enough that this decision could be revisited before Phase 1 implementation starts.
