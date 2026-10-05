# Curriculum/content authoring rules

## Canonical mapping
Every LPIC-1 learning artifact must map to one or more IDs in `curriculum/lpic-1-v5/objectives.json`. Machine-readable manifests are canonical; Markdown documents explain them to humans and agents.

## Originality and licensing
Use the official LPI objective list only as the factual syllabus reference. Do not copy, translate, closely paraphrase, remix or adapt LPI Learning Materials. Build original explanations from the objective facts plus independent primary Linux documentation/man pages.

## Required layers per concept
Every schedulable concept must include all three learner-facing surfaces:
- exactly one focused `introduce` lesson with the explanation/mental model and worked examples where useful;
- at least one deterministic non-assessment quiz question;
- at least one machine-checked practical lab/rep that maps observable state back to that concept.

A concept is not considered covered when one of these three surfaces is missing. Recall-capable questions do not substitute for practical evidence.

Concept IDs must also be granular enough to represent the full official objective. If one concept groups independently examinable knowledge so broadly that a course, quiz set or lab cannot meaningfully exercise all of it, split the concept before declaring the objective complete.

An objective must not become schedulable until every mapped concept satisfies these course + quiz + lab gates.

## Modern versus legacy
Use the canonical labels:
- `lpic-required`;
- `lpic-legacy`;
- `modern-practice`.

Never silently replace examinable legacy knowledge with a modern equivalent.

## Lab authoring
Labs specify desired state, environment and checker semantics. Avoid command-string grading. Reference solutions are private authoring/test fixtures, not learner hints.

Every state check must declare the stable `concept_ids` it proves. The union of check mappings must cover the concepts claimed by the lab.

### Podman labs
- run rootless;
- `network=none` until an explicitly isolated container-network contract is implemented;
- setup/reference paths stay inside the lab directory;
- `:latest` is forbidden;
- released image identity/provenance remains pinned;
- bootstrap may explicitly pull a trusted base image only after user consent, then builds the project lab image with `--pull=never`.

### VM labs
- use the libvirt backend only;
- never fall back to Podman or host execution;
- curriculum references an opaque trusted image ID, never a host path or URL;
- required VM images may be prepared lazily by the bootstrap flow after explicit consent;
- runtime verifies the installed image catalog/digest before creating disposable overlays.

## Source policy
Prefer authoritative references: LPI objectives for syllabus scope; upstream projects, man-pages and distribution documentation for Linux behavior; established educational research for learning claims.
