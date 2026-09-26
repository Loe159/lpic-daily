# LPIC Daily

LPIC Daily is a terminal-first, local-first learning environment designed to build durable Linux administration skills while covering the complete LPIC-1 v5.0 syllabus (101-500 and 102-500).

**Phase 0 and Phase 0.5 are complete. Phase 1 implementation is in progress.** Product requirements, curriculum source of truth, prerequisite graph, content contracts, isolation model, threat model, licensing and agent workflow are frozen for the first vertical slice.

## Start here

Human contributors should read:

1. `docs/PRODUCT.md`
2. `docs/REQUIREMENTS.md`
3. `docs/ARCHITECTURE.md`
4. `docs/SECURITY.md` and `docs/THREAT_MODEL.md`
5. `docs/CURRICULUM_GRAPH.md` and `docs/CONTENT_MODEL.md`
6. `docs/plan/PHASE-1-IMPLEMENTATION.md`

Coding agents must start with `AGENTS.md` and then load only the documents relevant to the current task.

## Current Phase 1 slice

The first implementation slice covers:

- 103.1 — Travail en ligne de commande
- 103.5 — Processus et jobs
- 104.5 — Permissions et ownership

Those objectives contain 22 stable concepts. See `curriculum/lpic-1-v5/phase1-slice.json`.

## Foundation decisions

- Greenfield repository: do not fork Shell Gym, Arc Academy Terminal, or SkillCoco.
- Primary implementation: Go 1.27; terminal UI will use Bubble Tea v2.
- Local persistence: SQLite.
- Fast lab backend: rootless Podman.
- Full-system lab backend: KVM/QEMU managed by libvirt, with disposable QCOW2 overlays.
- No automatic host-shell fallback for labs.
- Curriculum is original French-first content mapped to official LPIC-1 v5.0 objectives.
- Technical English terms remain English where that is the natural Linux vocabulary.
- Daily integration is notification-first; session composition is adaptive.
- Gamification (XP/streaks/achievements) is structurally separate from mastery evidence.
- AI is outside the initial scope.
- Code is Apache-2.0; original educational/documentation content is CC BY 4.0 unless stated otherwise.

## Repository map

```text
AGENTS.md                    global agent contract
.agents/skills/              reusable agent workflows
.github/                     agent adapters + CI
cmd/lpic/                    CLI entry point
internal/curriculum/         strict curriculum loader/validation
internal/learning/           evidence, mastery projection, scheduler
internal/store/              migrations and persistence layer
curriculum/lpic-1-v5/        objective graph, concepts and slice
docs/                        product, architecture, decisions and plans
labs/                        lab definitions (added during Phase 1)
schemas/                     versioned content contracts
scripts/                     repository validation
```

## Validate

```bash
python3 scripts/validate_foundation.py
go vet ./...
go test ./...
go run ./cmd/lpic validate
go run ./cmd/lpic plan
```

The Python validators protect curriculum/graph/schema invariants. The Go loader independently fails closed on cross-file drift and the Go tests protect learning/persistence behavior.
