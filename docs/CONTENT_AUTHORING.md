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
Use the canonical machine-readable labels from `common.schema.json`:
- `lpic-required`: required by the current LPIC syllabus;
- `lpic-legacy`: still examinable but generally legacy practice;
- `modern-practice`: contemporary operational practice taught alongside exam knowledge.

Never silently replace an examinable legacy term with a modern equivalent.

## Lab authoring
Labs specify desired state, environment and checker semantics. Avoid command-string grading. Reference solutions are private authoring/test fixtures, not hints exposed by default.

For Phase-1 Podman content:
- `network` must be `none`;
- setup/reference paths must stay inside the lab directory;
- images are preinstalled by a trusted workflow; the app never pulls them implicitly;
- `:latest` is forbidden;
- released image identities must be digest/provenance pinned even when a local development alias is used while authoring.

## Source policy
Prefer authoritative references: LPI objectives for syllabus scope; upstream projects/man-pages/distribution documentation for Linux behavior; peer-reviewed or established educational research for learning claims.
