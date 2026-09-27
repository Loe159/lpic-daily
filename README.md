# LPIC Daily

LPIC Daily is a terminal-first, local-first learning environment designed to build durable Linux administration skills while covering the complete LPIC-1 v5.0 syllabus (101-500 and 102-500).

**Phase 1 is complete.** The adaptive scheduler, append-only mastery evidence, separate XP/streak/achievement projection, Bubble Tea daily dashboard, SQLite progress store, three rootless Podman reference labs, Fedora desktop-notification adapter, Fedora 44 CI and real rootless Podman host-isolation test are implemented and validated. The current roadmap phase is **Phase 2: libvirt/QEMU/KVM VM runner**, before scaling content to the full LPIC-1 curriculum.

## Current runnable slice

The current runnable labs are:

```text
lpic1.103.1.shell-environment-repair
lpic1.103.5.stuck-worker
lpic1.104.5.shared-dropbox
```

`shell-environment-repair` validates PATH resolution, exported variables and quoting in fresh login shells. `stuck-worker` exercises process inspection/selection, foreground/background shell jobs and signals through the persistent PTY. `shared-dropbox` practices ownership, rwx/octal permissions, SGID, sticky bit and directory semantics.

Phase 2 also contains the authored VM lab `lpic1.104.1.partition-filesystems`. The CLI now dispatches `libvirt` labs only to the system libvirt backend; there is no Podman fallback. VM runs require a trusted local image catalog at `$XDG_DATA_HOME/lpic-daily/vm-images/catalog.json` (or the same path under `LPIC_DAILY_VM_IMAGE_DIR`). The reproducible VM image build/install pipeline is still in progress, so this lab is not yet part of the default runnable slice. `lpic doctor` reports `qemu-img`, trusted VM catalog and `qemu:///system` readiness separately.

Current daily workflow:

```bash
go run ./cmd/lpic tui
go run ./cmd/lpic today
go run ./cmd/lpic assess
go run ./cmd/lpic learn lpic1.103.1.lesson.shell-sequences
go run ./cmd/lpic question lpic1.103.1.q.sequence-and
go run ./cmd/lpic validate
go run ./cmd/lpic doctor
go run ./cmd/lpic lab list
go run ./cmd/lpic lab show lpic1.103.1.shell-environment-repair
go run ./cmd/lpic lab show lpic1.103.5.stuck-worker
go run ./cmd/lpic lab show lpic1.104.5.shared-dropbox
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

The optional `lpic assess` flow tests the seven 103.1 foundation concepts through recall questions. Successful answers can satisfy the normal prerequisite-readiness calculation without fabricating lesson completion.

The application never pulls lab images implicitly. Missing images or an unusable/rootful Podman service fail closed.

## Daily desktop notification

The Fedora adapter uses `notify-send` and a user-level systemd timer. The notification is sent at most once per local day when work is due; selecting **Ouvrir** launches `lpic tui` through `xdg-terminal-exec`. A custom launcher can be supplied with `LPIC_DAILY_TERMINAL_LAUNCHER`.

For a local development install:

```bash
go build -o ~/.local/bin/lpic ./cmd/lpic
mkdir -p ~/.config/systemd/user ~/.local/share/applications
cp packaging/systemd/lpic-daily-notify.* ~/.config/systemd/user/
cp packaging/desktop/lpic-daily.desktop ~/.local/share/applications/
systemctl --user daemon-reload
systemctl --user enable --now lpic-daily-notify.timer
```

Test immediately with:

```bash
lpic notify --force
```

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
- Terminal UI: Bubble Tea v2.
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
