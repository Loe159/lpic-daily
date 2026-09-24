# Learning model

## Goal
Optimize for LPIC-1 readiness while building enough practical understanding that knowledge transfers to unfamiliar Linux systems.

## Evidence hierarchy
Mastery is inferred from multiple forms of evidence, ordered roughly from strongest to weakest:
1. successful authentic lab/challenge without solution reveal;
2. successful focused practical rep;
3. free recall / fill-in response;
4. recognition-style quiz;
5. passive lesson completion (records exposure, not mastery).

XP, streaks and achievements are never mastery evidence.

## Progressive depth
A concept is not taught once as a complete monolith. Content should move through layers such as:
1. minimal mental model and first useful operation;
2. focused retrieval/practice;
3. common options and failure modes;
4. cross-concept application;
5. diagnostic/incident use;
6. exam-focused recall where appropriate.

This supports short first encounters without sacrificing eventual depth.

## Daily session design
There is no universal fixed duration. The scheduler constructs a session from due work and the learner's requested intensity.

A typical session may mix:
- 2–5 retrieval prompts from due concepts;
- one focused explanation or worked example;
- one or more practical reps/lab;
- corrective feedback/debrief;
- scheduling of future retrieval.

A session may be review-only or incident-only when that is the best use of time.

## Scheduling
Use spaced retrieval and prerequisite readiness as design principles, but do not hard-code a claim that one named algorithm is scientifically optimal.

The first implementation should use an inspectable scheduler based on:
- concept mastery estimate;
- recency;
- repeated errors;
- LPIC objective weight;
- evidence strength;
- prerequisite readiness;
- need for distribution variation;
- desired session intensity/length.

The learner must be able to inspect why an item was scheduled.

## Curriculum ordering
LPIC objective numbers are reporting identifiers, not the mandatory teaching sequence. Concepts should be arranged as a dependency graph and may be interleaved across official topics.

Every learning item must still map back to one or more objective IDs so certification coverage remains measurable.

## Interleaving and transfer
Later practice should combine concepts across objectives. A learner who passed `grep` in isolation should later encounter it naturally while diagnosing logs, permissions, networking or services.

Distribution transfer matters too: the same underlying task may later recur in Fedora, Debian or openSUSE with appropriate tooling differences.

## Hints
Larger labs/challenges use authored graduated hints:
1. light conceptual nudge;
2. direction/category of tool to inspect;
3. precise next action or diagnostic path;
4. explicit solution and debrief.

A success after stronger hints is still useful learning evidence but carries less mastery weight than an unaided success.

## Mastery constraints
Full mastery must require evidence across time. For practical objectives, a single quiz or passive lesson can never produce full mastery.

The mastery model should distinguish at least:
- exposure;
- recall;
- guided practice;
- independent practice;
- demonstrated transfer.

Exact numeric thresholds remain implementation details for Phase 1 experiments, but tests must protect the invariant that passive completion alone is insufficient.

## Legacy versus modern practice
Learning items may carry one or more pedagogical labels:
- `lpic-required`;
- `lpic-legacy`;
- `modern-practice`.

When two approaches differ, explain which one the exam expects and which one is preferred on contemporary systems.

## Assessment and exam preparation
Initial assessment is adaptive and targeted. It should identify weak concepts and select appropriate next practice.

A later exam mode may reflect the public LPIC format and objective weights, but must use original questions and never dumps. Passing practice tests must not override missing hands-on evidence in the mastery model.
