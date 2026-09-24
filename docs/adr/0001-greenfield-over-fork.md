# ADR 0001 — Greenfield repository over a fork

Status: **Accepted — 2026-09-24**

## Decision
Start LPIC Daily as a new repository. Use Shell Gym, Arc Academy Terminal and SkillCoco as research references, not as the project base.

## Rationale
- Shell Gym has an excellent real-shell/state-check model but is PolyForm Noncommercial 1.0.0.
- Arc Academy Terminal is GPL-2.0, early-stage and does not provide the full-system isolation required for LPIC boot/storage administration.
- SkillCoco is MIT and its adaptive-learning core is interesting, but Tauri/React + generic AI-first product assumptions + Docker/host-shell fallback do not fit the target architecture.
- A fork would inherit architecture, dependency, UX and license constraints before requirements are stable.

## Consequence
We may independently implement common ideas such as mastery tracking, spaced review, state-based checks and daily challenges. Any code reuse requires explicit license and dependency review.
