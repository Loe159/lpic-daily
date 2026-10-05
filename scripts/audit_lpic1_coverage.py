#!/usr/bin/env python3
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
CURRICULUM = ROOT / "curriculum" / "lpic-1-v5"

def load(name):
    return json.loads((CURRICULUM / name).read_text(encoding="utf-8"))

def fail(errors):
    if errors:
        print("LPIC-1 standalone coverage FAILED:")
        for error in errors:
            print(" -", error)
        return 1
    return 0

def main():
    objectives_file = load("objectives.json")
    concepts_file = load("concepts.json")
    guides_file = load("objective-study-guides.json")
    phase3 = load("phase3-exam101.json")
    phase4 = load("phase4-exam102.json")

    objectives = [o for o in objectives_file["objectives"] if o.get("active")]
    concepts = [c for c in concepts_file["concepts"] if c.get("active")]
    guides = {g["objective_id"]: g for g in guides_file["guides"]}
    concept_by_id = {c["id"]: c for c in concepts}
    concepts_by_objective = {}
    for concept in concepts:
        concepts_by_objective.setdefault(concept["objective_id"], []).append(concept)

    errors = []
    if len(objectives) != 42:
        errors.append(f"expected 42 active objectives, got {len(objectives)}")
    expected_concepts = sum(len(o["concepts"]) for o in objectives)
    if len(concepts) != expected_concepts:
        errors.append(f"concept inventory mismatch: expected {expected_concepts}, got {len(concepts)}")

    scoped = []
    for scope, exam in ((phase3, "101"), (phase4, "102")):
        expected_objectives = [o["id"] for o in objectives if o["exam"] == exam]
        if scope["selected_objectives"] != expected_objectives:
            errors.append(f"Exam {exam} objective scope drift")
        for objective_id in scope["selected_objectives"]:
            ids = scope["objective_concepts"].get(objective_id, [])
            expected_ids = [
                c["id"] for c in sorted(
                    concepts_by_objective.get(objective_id, []),
                    key=lambda row: row["pedagogy_order"],
                )
            ]
            if ids != expected_ids:
                errors.append(f"{objective_id}: concept scope drift")
            scoped.extend(ids)

    if len(scoped) != len(concepts) or set(scoped) != set(concept_by_id):
        errors.append("Exam 101 + 102 scopes do not cover every active concept exactly once")

    for objective in objectives:
        guide = guides.get(objective["id"])
        if guide is None:
            errors.append(f"{objective['id']}: missing standalone study guide")
            continue
        for field in ("overview", "practice", "pitfalls"):
            if len(str(guide.get(field, "")).strip()) < 40:
                errors.append(f"{objective['id']}: study guide {field} is too shallow")
        if not objective.get("terms_files_utilities"):
            errors.append(f"{objective['id']}: no official terms/files/utilities")
        if not objective.get("assessment_evidence"):
            errors.append(f"{objective['id']}: no assessment evidence targets")

    # The runtime deterministically synthesizes exactly one focused lesson when
    # an authored one is absent, plus two daily questions and two practical
    # contexts for every active concept. Check authored duplicates here because
    # the runtime fails closed rather than hiding them.
    authored_intros = {}
    for path in ROOT.glob("content/lpic-1-v5/lessons/**/*.json"):
        data = json.loads(path.read_text(encoding="utf-8"))
        ids = data.get("concept_ids", [])
        if data.get("stage") == "introduce" and len(ids) == 1:
            authored_intros[ids[0]] = authored_intros.get(ids[0], 0) + 1
    duplicates = sorted(cid for cid, count in authored_intros.items() if count > 1)
    if duplicates:
        errors.append("multiple authored focused introductions: " + ", ".join(duplicates))

    for concept in concepts:
        objective = next(o for o in objectives if o["id"] == concept["objective_id"])
        if not concept.get("title_fr"):
            errors.append(f"{concept['id']}: empty title")
        if "lpic-required" not in concept.get("classification", []):
            errors.append(f"{concept['id']}: missing lpic-required classification")
        # Runtime coverage contract.
        if authored_intros.get(concept["id"], 0) not in (0, 1):
            errors.append(f"{concept['id']}: invalid focused introduction count")
        if not objective["terms_files_utilities"]:
            errors.append(f"{concept['id']}: objective has no technical anchors")

    if fail(errors):
        return 1

    print(
        "LPIC-1 standalone coverage OK: "
        f"{len(objectives)} objectives; {len(concepts)} concepts; "
        f"Exam 101={phase3['concept_count']}; Exam 102={phase4['concept_count']}; "
        "every concept receives one focused introduction, two daily questions "
        "(including recall), and two practical contexts at runtime."
    )
    return 0

if __name__ == "__main__":
    sys.exit(main())
