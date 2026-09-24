# Research sources — 2026-09-24

This file records primary/high-value sources that justify the Phase-0 recommendations. URLs are intentionally kept here so future agents can re-check freshness rather than relying on chat history.

## Agent-first repositories
- AGENTS.md specification: https://agents.md/
- OpenAI, using skills with coding agents: https://developers.openai.com/codex/skills/
- GitHub Copilot repository custom instructions: https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot
- Anthropic, effective context engineering for AI agents: https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents

Key conclusions: keep global agent instructions short/high-signal; put scoped rules near affected code/content; represent repeatable workflows as skills/scripts; make important contracts machine-checkable.

## Existing learning tools
- Shell Gym: https://github.com/iximiuz/shellgym — PolyForm Noncommercial; useful state-check/learning-path reference.
- Arc Academy Terminal: https://github.com/metarobb/arc-academy-terminal — GPL-2.0; useful TUI/daily UX reference.
- SkillCoco: https://github.com/skillcoco/skillcoco — MIT; adaptive-learning/BKT/SM-2 ideas, but different product architecture.

## Isolation/runtime
- Podman rootless docs: https://docs.podman.io/en/stable/markdown/podman.1.html
- libvirt disk backing chains: https://www.libvirt.org/kbase/backing_chains.html
- QEMU qcow2 docs: https://www.qemu.org/docs/master/system/images.html
- Firecracker rootfs/kernel setup: https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md
- libvirt Go bindings: https://github.com/libvirt/libvirt-go-module
- pure-Go libvirt RPC client: https://github.com/digitalocean/go-libvirt

## Application stack
- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Bubbles: https://github.com/charmbracelet/bubbles
- modernc SQLite: https://pkg.go.dev/modernc.org/sqlite

## LPIC-1
- Official overview: https://www.lpi.org/our-certifications/lpic-1-overview/
- Official objectives v5.0: https://www.lpi.org/en/exam-101-102-objectives/ (language paths can vary; English is canonical when translations differ)
- French objectives: https://www.lpi.org/fr/exam-101-102-objectives/
- LPI learning portal: https://learning.lpi.org/

Current syllabus baseline at research date: LPIC-1 v5.0, exams 101-500 and 102-500; both exams required; each exam is 90 minutes with 60 multiple-choice/fill-in questions according to LPI's overview.

## Learning science
Use retrieval practice and spacing as principles, not magic algorithms. Good starting literature includes modern reviews of retrieval practice/distributed practice; any specific mastery formula introduced in code must be separately documented and validated.
