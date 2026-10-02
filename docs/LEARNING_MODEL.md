# Learning model

## Goal
Optimize for LPIC-1 readiness while building enough practical understanding that knowledge transfers to unfamiliar Linux systems.

## Unit of mastery

The canonical mastery unit is a **concept ID** from `curriculum/lpic-1-v5/concepts.json`.

Objective/topic/certification progress is derived by aggregating concept state with LPIC objective weights. A learner can therefore be strong on some concepts within an objective without the UI falsely marking the whole objective mastered.

## Evidence hierarchy
Mastery is inferred from multiple forms of evidence, ordered roughly from strongest to weakest:
1. successful transfer/challenge without solution reveal;
2. successful independent practical lab;
3. successful guided practical rep;
4. free recall / fill-in response;
5. recognition-style quiz;
6. passive lesson completion (exposure only).

XP, streaks and achievements are never mastery evidence.

Raw evidence is append-only using `mastery-evidence.schema.json`. Mastery state is a recalculable projection, not the source record.

## Mastery stages

The UI/model distinguishes:

- **unseen** — no evidence;
- **exposed** — concept has been introduced;
- **recall** — learner can retrieve key knowledge;
- **guided** — succeeds with meaningful guidance;
- **independent** — succeeds practically without solution reveal;
- **transfer** — succeeds later in a changed/interleaved context.

Exact numeric scoring remains an implementation detail, but the following invariants are fixed:

- passive exposure cannot produce full mastery;
- a single recognition question cannot produce full mastery;
- practical concepts require practical evidence;
- revealing the solution prevents that attempt from counting as independent;
- high mastery requires evidence separated in time;
- transfer evidence is stronger than repeating an identical lab.

## Progressive depth
A concept is not taught once as a complete monolith. Content moves through:
1. minimal mental model and first useful operation;
2. focused retrieval/practice;
3. common options and failure modes;
4. cross-concept application;
5. diagnostic/incident use;
6. exam-focused recall where appropriate.

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

The scheduler uses:
- concept mastery projection;
- recency;
- repeated errors;
- LPIC objective weight;
- evidence strength;
- hard prerequisite eligibility;
- recommended prerequisite preference;
- need for distribution variation;
- desired session intensity/length.

The learner can inspect why an item was scheduled. A successful recognition-only answer does not become recall evidence. When such a practical concept is reviewed while still `exposed`, the default daily action should prefer an available lab instead of trapping the learner in repeated recognition questions. Once independent practical evidence is old enough for transfer, scheduling should prefer an unused practical context when one exists.

An initial/adaptive assessment may satisfy a prerequisite by evidence; the system must not invent a fake lesson completion.

## Curriculum ordering
LPIC objective numbers are reporting identifiers, not the teaching sequence.

`curriculum/lpic-1-v5/prerequisites.json` defines:
- hard prerequisites: eligibility;
- recommended prerequisites: ranking only.

Concept-to-concept dependencies are added only where actual authored content demonstrates a dependency.

## Interleaving and transfer
Later practice combines concepts across objectives. A learner who passed a tool in isolation should encounter it naturally in logs, permissions, networking, services or incidents.

Distribution transfer matters too: later evidence should sometimes vary Fedora, Debian or openSUSE.

## Hints
Labs/challenges use authored graduated hints:
1. light conceptual nudge;
2. direction/category of tool;
3. precise next action/diagnostic path;
4. explicit solution/debrief.

The mastery event records the highest hint level and whether the solution was revealed.

## Legacy versus modern practice
Learning items may carry:
- `lpic-required`;
- `lpic-legacy`;
- `modern-practice`.

When approaches differ, explain which one the exam expects and which one is preferred on contemporary systems.

## Assessment and exam preparation
Initial assessment is adaptive and targeted. It identifies weak concepts, can satisfy already-known prerequisites, and selects next practice.

A later exam mode may reflect the public LPIC format/objective weights, but uses original questions and never dumps. Practice-test performance cannot override missing practical evidence.
