#!/usr/bin/env python3
import argparse
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
OBJECTIVES_PATH = ROOT / "curriculum" / "lpic-1-v5" / "objectives.json"
CONCEPTS_PATH = ROOT / "curriculum" / "lpic-1-v5" / "concepts.json"

SURFACE_GLOBS = {
    "labs": "labs/lpic-1-v5/*/*/lab.json",
    "lessons": "content/lpic-1-v5/lessons/**/*.json",
    "questions": "content/lpic-1-v5/questions/**/*.json",
}


def load_json(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ValueError(f"{path.relative_to(ROOT)}: invalid JSON: {exc}") from exc


def phase3_scope():
    objectives = load_json(OBJECTIVES_PATH).get("objectives", [])
    selected_objectives = [
        item["id"] for item in objectives
        if item.get("active") is True and item.get("exam") == "101"
    ]
    if not selected_objectives:
        raise ValueError("no active Exam 101 objectives found")

    selected_set = set(selected_objectives)
    concepts = load_json(CONCEPTS_PATH).get("concepts", [])
    ordered_concepts = [
        item["id"] for item in concepts
        if item.get("active") is True and item.get("objective_id") in selected_set
    ]
    concept_to_objective = {
        item["id"]: item["objective_id"] for item in concepts
        if item.get("active") is True and item.get("objective_id") in selected_set
    }
    if len(ordered_concepts) != len(concept_to_objective):
        raise ValueError("duplicate active Exam 101 concept id")
    return selected_objectives, ordered_concepts, concept_to_objective


def artifact_records(surface, pattern):
    records = []
    for path in sorted(ROOT.glob(pattern)):
        data = load_json(path)
        artifact_id = data.get("id")
        declared_concepts = data.get("concept_ids")
        objective_ids = data.get("objective_ids")
        if not isinstance(artifact_id, str) or not artifact_id:
            raise ValueError(f"{path.relative_to(ROOT)}: missing id")
        if not isinstance(declared_concepts, list) or not all(isinstance(item, str) for item in declared_concepts):
            raise ValueError(f"{path.relative_to(ROOT)}: concept_ids must be a list of strings")
        if not isinstance(objective_ids, list) or not all(isinstance(item, str) for item in objective_ids):
            raise ValueError(f"{path.relative_to(ROOT)}: objective_ids must be a list of strings")

        evidenced_concepts = set(declared_concepts)
        if surface == "labs":
            checks = data.get("checks")
            if not isinstance(checks, list) or not checks:
                raise ValueError(f"{path.relative_to(ROOT)}: lab requires state checks")
            evidenced_concepts = set()
            for index, check in enumerate(checks, start=1):
                concept_ids = check.get("concept_ids")
                if not isinstance(concept_ids, list) or not concept_ids:
                    raise ValueError(f"{path.relative_to(ROOT)}: check {index} has no concept evidence mapping")
                unknown = set(concept_ids) - set(declared_concepts)
                if unknown:
                    raise ValueError(
                        f"{path.relative_to(ROOT)}: check {index} maps undeclared concepts "
                        + ", ".join(sorted(unknown))
                    )
                evidenced_concepts.update(concept_ids)
            missing = set(declared_concepts) - evidenced_concepts
            if missing:
                raise ValueError(
                    f"{path.relative_to(ROOT)}: declared lab concepts lack state-check evidence: "
                    + ", ".join(sorted(missing))
                )

        records.append({
            "id": artifact_id,
            "concept_ids": evidenced_concepts,
            "objective_ids": set(objective_ids),
            "path": path,
        })
    return records


def audit():
    selected_objectives, ordered_concepts, concept_to_objective = phase3_scope()
    mapped = {concept_id: {"labs": [], "lessons": [], "questions": []} for concept_id in ordered_concepts}
    seen_artifact_ids = set()

    for surface, pattern in SURFACE_GLOBS.items():
        for record in artifact_records(surface, pattern):
            artifact_id = record["id"]
            if artifact_id in seen_artifact_ids:
                raise ValueError(f"duplicate learning artifact id {artifact_id}")
            seen_artifact_ids.add(artifact_id)
            for concept_id in sorted(record["concept_ids"]):
                objective_id = concept_to_objective.get(concept_id)
                if objective_id is None:
                    continue
                if objective_id not in record["objective_ids"]:
                    raise ValueError(
                        f"{record['path'].relative_to(ROOT)}: concept {concept_id} belongs to "
                        f"Exam-101 objective {objective_id}, but that objective is not referenced"
                    )
                mapped[concept_id][surface].append(artifact_id)

    concepts = [{
        "concept_id": concept_id,
        "objective_id": concept_to_objective[concept_id],
        "surfaces": {surface: sorted(mapped[concept_id][surface]) for surface in ("labs", "lessons", "questions")},
    } for concept_id in ordered_concepts]

    objective_with_lab = {
        objective_id: any(item["objective_id"] == objective_id and item["surfaces"]["labs"] for item in concepts)
        for objective_id in selected_objectives
    }
    summary = {
        "objectives": len(selected_objectives),
        "concepts": len(concepts),
        "with_lesson": sum(bool(item["surfaces"]["lessons"]) for item in concepts),
        "with_question": sum(bool(item["surfaces"]["questions"]) for item in concepts),
        "with_lab": sum(bool(item["surfaces"]["labs"]) for item in concepts),
        "objectives_with_lab": sum(objective_with_lab.values()),
    }
    return summary, concepts


def main():
    parser = argparse.ArgumentParser()
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--require-complete", action="store_true")
    args = parser.parse_args()

    try:
        summary, concepts = audit()
    except ValueError as exc:
        print(f"Phase-3 coverage audit FAILED: {exc}")
        return 1

    print(
        "Phase-3 Exam 101 coverage: "
        f"{summary['objectives']} objectives / {summary['concepts']} concepts; "
        f"{summary['with_lesson']} with lesson; "
        f"{summary['with_question']} with question; "
        f"{summary['with_lab']} with practical lab evidence; "
        f"{summary['objectives_with_lab']} objectives with at least one lab"
    )
    if args.check:
        return 0

    missing_lesson = [item["concept_id"] for item in concepts if not item["surfaces"]["lessons"]]
    missing_question = [item["concept_id"] for item in concepts if not item["surfaces"]["questions"]]
    if missing_lesson:
        print("Phase-3 acceptance FAILED: concepts without lesson: " + ", ".join(missing_lesson))
    if missing_question:
        print("Phase-3 acceptance FAILED: concepts without deterministic retrieval question: " + ", ".join(missing_question))
    if missing_lesson or missing_question:
        return 1

    print(
        "Phase-3 theory/retrieval coverage complete. "
        "Practical-objective and cross-topic challenge gates remain documented in "
        "docs/plan/PHASE-3-ACCEPTANCE.md."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
