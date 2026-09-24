# Proposed architecture

## Decision summary
The recommended foundation is a greenfield Go application with a terminal UI and pluggable lab runners.

```text
                         +--------------------+
                         |  TUI / CLI (Go)    |
                         | Bubble Tea v2      |
                         +----------+---------+
                                    |
         +--------------------------+--------------------------+
         |                          |                          |
+--------v---------+      +---------v---------+      +---------v---------+
| Curriculum       |      | Learning engine   |      | Progress store     |
| parser/validator |      | schedule/mastery  |      | SQLite             |
+--------+---------+      +---------+---------+      +-------------------+
         |                          |
         +--------------------------+
                                    |
                           +--------v---------+
                           | Lab orchestrator |
                           +---+----------+---+
                               |          |
                     +---------v-+      +-v----------------+
                     | Podman     |      | libvirt/KVM/QEMU |
                     | rootless   |      | QCOW2 overlays   |
                     +------------+      +------------------+
```

## Why Go
- Small, readable language surface suits human and agent maintenance.
- Fast builds/tests encourage frequent verification.
- Straightforward distribution as a native CLI/TUI binary.
- Bubble Tea v2 provides a mature Elm-style terminal UI architecture.
- SQLite has mature Go options, including pure-Go drivers if avoiding CGO becomes important.
- libvirt can be integrated with official C bindings or through a pure-Go RPC client; the runner interface keeps that choice replaceable.

Rust remains a credible alternative, but the product is primarily orchestration, content validation and terminal UX rather than memory-unsafe low-level code. Go minimizes implementation complexity without weakening the isolation boundary, which is provided by the OS/hypervisor rather than language memory safety.

## Why greenfield rather than fork
- Shell Gym: highly relevant learning-path/check model, but PolyForm Noncommercial restricts the reuse target.
- Arc Academy Terminal: useful TUI/product reference, but GPL-2.0, very young, and its simulated/playground isolation does not cover full LPIC system administration.
- SkillCoco: MIT and useful adaptive-learning ideas, but its Tauri/React generic desktop architecture, Docker/host-shell fallback and open-core product boundaries do not match our terminal-first security model.

Ideas may be researched; code should only be incorporated after explicit license/dependency review. No copied architecture by default.

## Application modules
- `curriculum`: typed schemas, versioning, validation and dependency graph.
- `learning`: mastery evidence, review scheduling, session construction.
- `progress`: SQLite repositories and migrations.
- `runner`: backend-neutral lifecycle (`prepare`, `start`, `exec/connect`, `check`, `reset`, `destroy`).
- `runner/podman`: rootless fast labs.
- `runner/libvirt`: full-system labs.
- `checker`: structured state probes and assertions.
- `tui`: dashboard, lesson reader, quiz, lab control, progress/explanations.
- `cli`: automation/admin commands such as validate, doctor, export and reset.

## Privilege model
The main process must run as the user. VM administration should use libvirt's established authorization model rather than making the entire learning application root. Any helper introduced later must expose a minimal structured API, use explicit authorization, and be threat-modeled separately.

## Content model
Curriculum content and executable lab definitions are data, not application code. All content must be versioned, schema-validated and auditable. A lab definition may request a capability profile but must not be able to silently expand its own host privileges.

## Dependency policy
Prefer standard library and a small number of established libraries. Every new dependency should have a clear purpose, active maintenance signal, compatible license, and no unnecessary network/cloud requirement.
