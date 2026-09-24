# Curriculum/content authoring rules

## Canonical mapping
Every piece of LPIC-1 content must map to one or more IDs in `curriculum/lpic-1-v5/objectives.json`. The JSON manifest is the canonical coverage inventory; Markdown explains it to humans/agents.

## Originality and licensing
The official LPI objective list is used as the factual syllabus reference. LPI Learning Materials are useful for checking understanding but are distributed under a restrictive Creative Commons BY-NC-ND license; do not copy, translate, paraphrase closely, remix or adapt their lesson text into this project. Build original explanations from the objective facts plus independent primary Linux documentation/man pages.

## Required layers per concept
A concept is not considered fully taught merely because it appears in prose. Mature content should eventually include:
- explanation/mental model;
- worked example where useful;
- retrieval question;
- practical rep or lab when the skill is observable;
- debrief/common failure modes;
- scheduled review hooks.

## Modern versus legacy
LPIC v5.0 intentionally contains tools and mechanisms that may be old in modern distributions. Mark them explicitly:
- `exam_required`: required by current syllabus;
- `legacy`: still examinable but not preferred for new systems;
- `modern_alternative`: current operational approach taught alongside it.

Never silently replace an examinable legacy term with a modern equivalent.

## Lab authoring
Labs specify desired state, environment and checker semantics. Avoid command-string grading. Reference solutions are private authoring/test fixtures, not hints exposed by default.

## Source policy
Prefer authoritative references: LPI objectives for syllabus scope; upstream projects/man-pages/distribution documentation for Linux behavior; peer-reviewed or established educational research for learning claims.
