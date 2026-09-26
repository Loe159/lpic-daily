# LPIC Daily

LPIC Daily is a terminal-first, local-first learning environment designed to build durable Linux administration skills while covering the complete LPIC-1 v5.0 syllabus (101-500 and 102-500).

The project is currently in **Phase 1: vertical slice**. The curriculum graph, mastery/evidence model, SQLite progress store, explainable scheduler, rootless Podman boundary and first authored lab are implemented and validated in CI. The polished TUI, notifications and full curriculum content still come later.

## Current runnable slice

The current runnable labs are:

```text
lpic1.103.5.stuck-worker
lpic1.104.5.shared-dropbox
```

`stuck-worker` exercises process inspection/selection, foreground/background shell jobs and signals through the persistent PTY. `shared-dropbox` practices ownership, rwx/octal permissions, SGID, sticky bit and directory semantics.

Current command-mode workflow:

```bash
go run ./cmd/lpic validate
go run ./cmd/lpic doctor
go run ./cmd/lpic lab list
go run ./cmd/lpic lab show lpic1.104.5.shared-dropbox
go run ./cmd/lpic lab show lpic1.103.5.stuck-worker
```

For local development, prepare the Phase-1 Fedora image:

```bash
podman build -t localhost/lpic-daily/fedora-phase1:1 labs/images/fedora-phase1
systemctl --user enable --now podman.socket
```

Then run:

```bash
go run ./cmd/lpic lab run lpic1.104.5.shared-dropbox
```

Inside command mode:

```text
:shell   enter a persistent PTY-backed shell in the same sandbox
:check   evaluate the observable final state
:hint    reveal the next graduated hint
:quit    destroy the disposable lab and leave
```

Each normal input line is executed by `bash -lc` **inside the sandbox**, not on the host. Use `:shell` when the exercise needs persistent shell state, job control, interactive programs or foreground/background process handling. The PTY is created by Podman inside the same disposable lab; terminal resizes are forwarded to the guest.

The application never pulls lab images implicitly. Missing images or an unusable/rootful Podman service fail closed.

## Start here

Human contributors should read:

1. `docs/PRODUCT.md`
2. `docs/REQUIREMENTS.md`
3. `docs/ARCHITECTURE.md`
4. `docs/SECURITY.md` and `docs/THREAT_MODEL.md`
5. `docs/lpic1/README.md`
6. `docs/plan/ROADMAP.md`

Coding agents must start with `AGENTS.md` and then load only the documents relevant to the current task.

## Core decisions

- Greenfield repository: do not fork Shell Gym, Arc Academy Terminal, or SkillCoco.
- Primary implementation: Go.
- Planned terminal UI: Bubble Tea v2, once the domain/runtime slice is stable.
- Local persistence: SQLite.
- Fast lab backend: rootless Podman over its local Unix-socket HTTP API.
- Full-system lab backend: KVM/QEMU managed by libvirt, with disposable QCOW2 overlays.
- No automatic host-shell fallback for labs.
- Network denied by default.
- Curriculum is original French-first content mapped to the official LPIC-1 v5.0 objectives.
- Technical English terms remain English where that is the natural Linux vocabulary.
- Daily integration is notification-first; session composition is adaptive.
- Gamification (XP/streaks/achievements) is separate from mastery.
- AI is outside the initial scope.
- Machine-readable curriculum metadata is the canonical coverage source.
- Code is Apache-2.0; original educational/documentation content is CC BY 4.0 unless stated otherwise.

## Repository map

```text
AGENTS.md                    global agent contract
.agents/skills/              reusable agent workflows
.github/                     CI and GitHub adapters
cmd/lpic/                    current CLI entry point
internal/learning/           evidence, mastery and scheduler domain
internal/progress/           persistence interfaces + SQLite
internal/runner/             sandbox contract and Podman adapter
internal/checker/            state-based lab grading
internal/lab/                authored lab loading/orchestration
curriculum/lpic-1-v5/        machine-readable LPIC coverage model
docs/                        product, architecture, research and ADRs
docs/lpic1/                  agent-oriented LPIC knowledge map
labs/                        authored labs and lab image recipes
schemas/                     content contracts
scripts/                     validation utilities
```

## Validate

```bash
python3 scripts/validate_foundation.py
go test ./...
go vet ./...
go run ./cmd/lpic validate
```

The foundation validator checks the complete LPIC objective model, learning graph, schemas and authored lab references. Go tests cover the domain, SQLite, runner security contract, Podman HTTP protocol handling, state checkers and lab orchestration.
