# LPIC Daily

LPIC Daily is a terminal-first, local-first learning environment designed to build durable Linux administration skills while covering the complete LPIC-1 v5.0 syllabus (101-500 and 102-500).

This repository has completed **Phase 0: foundations** and is ready for the Phase 1 vertical slice. There is intentionally no application implementation in the foundation snapshot. Product requirements, curriculum source of truth, isolation model, threat model, licensing and agent workflow are frozen for Phase 1.

## Start here

Human contributors should read:

1. `docs/PRODUCT.md`
2. `docs/REQUIREMENTS.md`
3. `docs/ARCHITECTURE.md`
4. `docs/SECURITY.md` and `docs/THREAT_MODEL.md`
5. `docs/lpic1/README.md`
6. `docs/plan/ROADMAP.md`

Coding agents must start with `AGENTS.md` and then load only the documents relevant to the current task.

## Foundation decisions

- Greenfield repository: do not fork Shell Gym, Arc Academy Terminal, or SkillCoco.
- Primary implementation: Go, terminal UI with Bubble Tea v2.
- Local persistence: SQLite.
- Fast lab backend: rootless Podman.
- Full-system lab backend: KVM/QEMU managed by libvirt, with disposable QCOW2 overlays.
- No automatic host-shell fallback for labs.
- Curriculum is original French-first content mapped to the official LPIC-1 v5.0 objectives; LPI learning materials may be consulted but must not be copied or adapted.
- Technical English terms remain English where that is the natural Linux vocabulary.
- Daily integration is notification-first; session composition is adaptive.
- Gamification (XP/streaks/achievements) is separate from mastery.
- AI is outside the initial scope.
- Machine-readable curriculum metadata is the canonical coverage source.
- Code is Apache-2.0; original educational/documentation content is CC BY 4.0 unless stated otherwise.

These choices are documented in `docs/REQUIREMENTS.md` and accepted ADRs.

## Repository map

```text
AGENTS.md                    global agent contract
.agents/skills/              reusable agent workflows
.github/                     adapters for GitHub/Copilot
curriculum/lpic-1-v5/        machine-readable LPIC coverage model
docs/                        product, architecture, research and decisions
docs/lpic1/                  agent-oriented LPIC knowledge map
labs/                        future lab definitions (no labs in Phase 0)
schemas/                     future content schemas
scripts/                     validation utilities
src/                         future application code
```

## Validate the foundation

```bash
python3 scripts/validate_curriculum.py
```

The validator checks both exams have total objective weight 60, all expected active objectives are present, IDs are unique, and every objective contains concepts, terms and an assessment strategy.
