# Product interview — completed 2026-09-24

The product-owner interview is complete. Stable answers are incorporated into `docs/REQUIREMENTS.md`; this file preserves the decision record.

| Area | Decision |
|---|---|
| Primary target | LPIC-1 certification is the primary goal. |
| Daily duration | Adaptive; depends on the format and work due. |
| Login behavior | Daily notification, not forced TUI launch. |
| Motivation | Include streaks, XP and achievements. |
| Language | French explanations; keep natural technical English vocabulary. |
| Lesson depth | Progressive: small first exposure, deepen over time. |
| Learning order | Reorganize around prerequisites/mastery rather than official numeric order. |
| Beyond syllabus | Yes: teach modern practice alongside examinable material. |
| Full-system labs | KVM/libvirt and multi-GB guest images are acceptable. |
| Distribution strategy | Fedora primary; regularly train on Debian and openSUSE too. |
| Network access | Isolated/local simulated networks by default. |
| AI tutor | Not in the initial scope. |
| Hints | Use progressive authored hints; no AI required. |
| Mastery | Intentionally strict; practical evidence over time is required. |
| Assessment | Prefer adaptive short/targeted assessment initially over full-length exam simulation. |
| Publication | Open source from the start. |
| Commercial reuse | Allowed. |

## Interpretation note
One interview response around AI-generated exercises versus hints was ambiguous in numbering. Because AI was explicitly rejected for the initial scope, the conservative product decision is: **no AI generation**, while preserving a deterministic graduated-hint system. This can be revisited only through an explicit future decision.

## Deferred questions (non-blocking for Phase 1)
These do not block the vertical slice and should not be invented by implementation agents:
- exact XP curve and achievement catalog;
- exact notification integration mechanism beyond a Linux desktop notification adapter;
- maximum disk budget for optional VM image packs;
- whether multi-day persistent lab worlds are desirable;
- exact mastery numeric thresholds after initial learning-model experiments;
- full exam-simulator milestone and UX.
