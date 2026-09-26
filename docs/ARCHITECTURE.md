# Architecture

## Decision summary

The foundation is a greenfield **Go** application with a terminal UI and pluggable lab runners. Go was explicitly re-evaluated before implementation; see ADR 0006.

```text
                         +--------------------+
                         |  TUI / CLI (Go)    |
                         | Bubble Tea v2      |
                         +----------+---------+
                                    |
         +--------------------------+--------------------------+
         |                          |                          |
+--------v---------+      +---------v---------+      +---------v---------+
| Curriculum       |      | Learning engine   |      | Progress store   |
| parser/validator |      | schedule/mastery  |      | SQLite           |
+--------+---------+      +---------+---------+      +------------------+
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

The choice is based on project-specific system integration:

- Podman publishes native Go bindings for its service API.
- libvirt publishes production-ready Go bindings.
- A pure-Go libvirt RPC client exists if avoiding CGO is valuable and its Phase-2 coverage is sufficient.
- Go provides straightforward native CLI distribution, fast builds/tests and a deliberately small language surface.
- Bubble Tea v2 is mature for TUI work.
- SQLite and PTY integrations are mature.
- The isolation/security boundary is Podman/KVM/libvirt, not native code embedded in the learning process.

Rust remains the strongest fallback. The detailed comparison and reversal criteria are in ADR 0006.

## Application modules

- `curriculum`: typed built-in manifests, schema/version validation and dependency graph.
- `learning`: mastery evidence, prerequisite eligibility, review scheduling, session construction.
- `progress`: SQLite repositories and migrations.
- `runner`: backend-neutral lifecycle (`prepare`, `start`, `connect`, `check`, `reset`, `destroy`).
- `runner/podman`: rootless fast labs through a structured API.
- `runner/libvirt`: full-system labs.
- `checker`: structured state probes and assertions.
- `tui`: dashboard, lesson reader, quiz, lab control, progress/explanations.
- `cli`: validate, doctor, export, reset and diagnostics.

## Dependency direction

Domain packages must not depend on the TUI or concrete runner implementations.

```text
cmd/tui
   |
   v
application services
   |
   +--> curriculum
   +--> learning
   +--> progress interfaces
   +--> runner interfaces
                |
                +--> podman adapter
                +--> libvirt adapter
```

Content schemas remain language-neutral.

## Built-in content

The released binary embeds the canonical built-in curriculum/schemas so offline use does not depend on the current working directory. Imported/community packs are loaded separately and treated as untrusted.

The source files under `curriculum/` and `schemas/` remain the repository source of truth; embedding must never introduce a duplicated copy.

## Privilege model

The main process runs as the user. It must not require root.

- Rootless Podman uses the user's Podman service/API.
- VM administration uses libvirt's authorization model.
- Any future privileged helper requires its own ADR and narrow structured interface.
- The application must not construct arbitrary shell strings for runner control.
- Go `unsafe` is forbidden by default; introducing it requires an ADR/security review.

## Content model

Curriculum content and executable lab definitions are data, not application code. All content is versioned, schema-validated and auditable. A lab may request a predefined capability profile but cannot expand its own host privileges.

## Dependency policy

Prefer the standard library and a small number of established libraries. A dependency needs:

- a concrete product requirement;
- active maintenance or stable maturity;
- compatible license;
- bounded transitive cost;
- no unnecessary cloud/network requirement.

System adapters must sit behind project-owned interfaces so dependencies can be replaced.
