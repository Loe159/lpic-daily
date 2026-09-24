# Product contract

## Problem
Traditional LPIC preparation tends to separate theory, command memorization and real system administration. LPIC Daily turns the complete LPIC-1 syllabus into a sustainable daily practice loop on the learner's Linux workstation while keeping dangerous exercises disposable and isolated.

## Primary outcome
The primary outcome is **LPIC-1 certification readiness**. Real Linux administration skill is the preferred means of reaching that outcome, not a substitute for exhaustive exam-objective coverage.

The canonical detailed requirements live in `docs/REQUIREMENTS.md`.

## Core experience
LPIC Daily sends one daily notification when practice is due. Opening it starts an adaptive session that may combine:
- retrieval of previously learned material;
- one short new concept or concept expansion;
- hands-on manipulation in a disposable environment;
- immediate deterministic feedback based on system state;
- graduated hints when requested;
- a concise debrief;
- scheduling of future review from demonstrated evidence;
- XP/streak/achievement updates that remain separate from mastery.

Session length is adaptive rather than fixed. The learner can request a shorter session and can always launch additional lessons, labs, challenges or assessments manually.

## Curriculum behavior
- French-first explanations; technical English remains English when that is the natural Linux terminology.
- Introduce topics progressively rather than as large up-front lectures.
- Reorder objectives according to prerequisites and mastery rather than blindly following LPIC numbering.
- Preserve traceability to official LPIC objective IDs.
- Teach both examinable legacy tools and current best practice, with explicit labels distinguishing them.
- Fedora is the primary user/host context; Debian and openSUSE recur as first-class learning environments.

## Product principles
1. **LPIC coverage is non-negotiable:** practical depth must support, not displace, complete certification scope.
2. **Practice over completion:** watching/reading does not equal mastery.
3. **State over command matching:** different valid solutions should pass.
4. **Progressive depth:** first exposure is small; later practice deepens and interleaves concepts.
5. **Safe realism:** real Linux where pedagogically valuable, disposable infrastructure where risky.
6. **Distribution breadth:** Debian-family and RPM-family administration both matter; include openSUSE/Zypper where LPIC requires it.
7. **Modern practice plus exam legacy:** teach why old commands exist and when newer equivalents are preferred.
8. **Offline-first:** core lessons, hints, progress, scheduling and grading work without cloud services or AI.
9. **Gamification without distortion:** streaks, XP and achievements motivate consistency but never determine mastery.
10. **Explainability:** the learner can inspect why a topic was selected and how mastery changed.
11. **Local ownership:** progress stays local by default and is exportable.
12. **Accessibility:** keyboard-first, readable without color, usable with common terminal accessibility tooling.

## Non-goals for initial releases
- LPIC-2/3 coverage.
- Hosted multi-user LMS.
- Mandatory account/cloud sync.
- AI tutor or AI-generated authoritative curriculum.
- Exam dumps or copied proprietary questions.
- Executing unrestricted exercises on the learner's host.
- Forcing the TUI to open at login.

## Product decisions
The product-owner interview is complete. `docs/INTERVIEW.md` records the answers and interpretation. New behavior that contradicts `docs/REQUIREMENTS.md` requires an explicit requirement/ADR change.
