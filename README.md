# LPIC Daily

LPIC Daily is a terminal-first, local-first learning environment designed to build durable Linux administration skills while covering the complete LPIC-1 v5.0 syllabus (101-500 and 102-500).

**Phases 1 and 2 are complete and accepted.** The adaptive scheduler, append-only mastery evidence, separate XP/streak/achievement projection, Bubble Tea daily dashboard, SQLite progress store, rootless Podman labs, desktop notifications and the libvirt/QEMU/KVM VM runner are implemented and validated. Real-host KVM acceptance passed on 2026-10-04. **Phase 3 Exam-101 curriculum is now in progress.**

## Current runnable slice

Phase 3 currently adds complete lesson/deterministic-question coverage for objective **103.4 — streams, pipes and redirections**. Once 103.1 readiness is satisfied, the daily scheduler may select 103.4; unfinished Phase-3 objectives remain excluded until every mapped concept has exactly one focused `introduce` lesson, a daily non-assessment question, and a path beyond recognition through either recall-capable retrieval or practical evidence.

The current runnable labs are:

```text
lpic1.103.1.shell-environment-repair
lpic1.103.1.transfer-shell-handoff
lpic1.103.5.stuck-worker
lpic1.103.5.transfer-operator-session
lpic1.104.5.shared-dropbox
lpic1.104.5.transfer-team-share-audit
```

`shell-environment-repair` and `transfer-shell-handoff` cover all seven 103.1 concepts in two contexts. `stuck-worker` and `transfer-operator-session` cover all seven 103.5 concepts, including PTY job control, `nohup`/SIGHUP behavior and terminal multiplexing. `shared-dropbox` and `transfer-team-share-audit` cover all eight 104.5 concepts, including behaviorally checked `umask`, SUID auditing, SGID and sticky-bit semantics. After independent evidence ages beyond the transfer gap, the scheduler prefers an unused lab context.

Phase 2 contains the authored VM labs `lpic1.104.1.partition-filesystems` and `lpic1.102.2.grub-kernel-parameter`. The CLI dispatches `libvirt` labs only to the system libvirt backend; there is no Podman fallback. VM labs refuse to run as root. Real-host Phase-2 acceptance passed on 2026-10-04.

Runtime prerequisites are now bootstrapped by LPIC Daily itself. User-scoped configuration is applied automatically; privileged or network-heavy actions are shown and require confirmation. VM storage and the trusted Fedora guest image are prepared lazily when the first VM lab needs them.

## Installation and first run

Build or install the `lpic` binary, then simply run:

```bash
lpic
```

On the first interactive launch LPIC Daily automatically configures the user-scoped pieces it can safely manage itself:

- the daily user-systemd notification timer;
- the desktop entry;
- the rootless Podman socket when Podman is already available;
- automatic terminal-launcher detection for notification actions.

Lab images are lazy. The first Podman lab prepares its image automatically. If the Fedora base image must be downloaded, LPIC Daily asks first and then performs an explicit `podman pull` followed by an offline `podman build --pull=never`.

The first VM lab similarly checks KVM/libvirt prerequisites and asks before package installation, `sudo` storage provisioning, or downloading/building the trusted Fedora VM image.

To proactively prepare everything:

```bash
lpic install
```

To prepare user integration and Podman but defer the heavier VM setup:

```bash
lpic install --no-vm
```

`--yes` accepts bootstrap confirmations for unattended development setups.

Normal use is then:

```bash
lpic
lpic today
lpic assess
lpic doctor
lpic lab list
lpic lab run lpic1.104.5.shared-dropbox
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

The application never pulls lab images implicitly. Before create, the Podman adapter resolves the configured local tag to its immutable `sha256:` image ID; resets reuse that same ID instead of resolving the tag again. Missing images, malformed image identities or an unusable/rootful Podman service fail closed.

## Daily desktop notification

The Fedora adapter uses `notify-send` and a user-level systemd timer. The timer/service files are installed automatically on first interactive launch or by `lpic install`.

Selecting **Ouvrir** starts `lpic tui` in an automatically detected terminal. LPIC Daily prefers `xdg-terminal-exec` when available and otherwise supports Kitty, foot, WezTerm, GNOME Terminal, Konsole, Alacritty and xterm. `LPIC_DAILY_TERMINAL_LAUNCHER` remains available as an explicit override.

Test the notification without consuming the day's automatic-delivery marker:

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
