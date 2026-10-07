#!/usr/bin/env python3
import argparse
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
SCOPE_PATH = ROOT / "curriculum" / "lpic-1-v5" / "phase3-exam101.json"
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
    scope = load_json(SCOPE_PATH)
    selected_objectives = scope.get("selected_objectives")
    objective_concepts = scope.get("objective_concepts")
    if not isinstance(selected_objectives, list) or not selected_objectives:
        raise ValueError("phase3 scope has no selected_objectives")
    if not isinstance(objective_concepts, dict):
        raise ValueError("phase3 scope has no objective_concepts")
    if set(objective_concepts) != set(selected_objectives):
        raise ValueError("phase3 objective_concepts keys differ from selected_objectives")

    concepts = load_json(CONCEPTS_PATH).get("concepts", [])
    known = {item["id"]: item for item in concepts if item.get("active") is True}
    ordered_concepts = []
    concept_to_objective = {}
    for objective_id in selected_objectives:
        ids = objective_concepts.get(objective_id)
        if not isinstance(ids, list) or not ids:
            raise ValueError(f"phase3 objective {objective_id} has no concept list")
        for concept_id in ids:
            concept = known.get(concept_id)
            if concept is None:
                raise ValueError(f"phase3 references unknown active concept {concept_id}")
            if concept.get("objective_id") != objective_id:
                raise ValueError(
                    f"phase3 concept {concept_id} belongs to {concept.get('objective_id')}, "
                    f"not {objective_id}"
                )
            if concept_id in concept_to_objective:
                raise ValueError(f"duplicate phase3 concept {concept_id}")
            concept_to_objective[concept_id] = objective_id
            ordered_concepts.append(concept_id)

    if scope.get("concept_count") != len(ordered_concepts):
        raise ValueError(
            f"phase3 concept_count={scope.get('concept_count')}, mapped={len(ordered_concepts)}"
        )
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
        if not isinstance(declared_concepts, list) or not all(
            isinstance(item, str) for item in declared_concepts
        ):
            raise ValueError(f"{path.relative_to(ROOT)}: concept_ids must be a list of strings")
        if not isinstance(objective_ids, list) or not all(
            isinstance(item, str) for item in objective_ids
        ):
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
                    raise ValueError(
                        f"{path.relative_to(ROOT)}: check {index} has no concept evidence mapping"
                    )
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
            "counts_for_coverage": not (
                surface == "questions" and data.get("usage") == "initial-assessment"
            ),
            "focused_introduction": (
                surface == "lessons"
                and data.get("stage") == "introduce"
                and len(declared_concepts) == 1
            ),
            "recall_question": (
                surface == "questions"
                and data.get("usage") != "initial-assessment"
                and data.get("evidence_kind_on_success") == "recall"
            ),
        })
    return records


def audit():
    selected_objectives, ordered_concepts, concept_to_objective = phase3_scope()
    mapped = {
        concept_id: {
            "labs": [],
            "lessons": [],
            "questions": [],
            "introductions": [],
            "recall_questions": [],
            "verified_labs": [],
        }
        for concept_id in ordered_concepts
    }
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
                if record["counts_for_coverage"]:
                    mapped[concept_id][surface].append(artifact_id)
                    if surface == "labs" and ".standalone-" not in artifact_id:
                        mapped[concept_id]["verified_labs"].append(artifact_id)
                if record["focused_introduction"]:
                    mapped[concept_id]["introductions"].append(artifact_id)
                if record["recall_question"]:
                    mapped[concept_id]["recall_questions"].append(artifact_id)

    # Mirror the deterministic runtime synthesis so this audit describes
    # the learner-visible product rather than only hand-authored JSON files.
    for concept_id in ordered_concepts:
        objective_id = concept_to_objective[concept_id]
        if not mapped[concept_id]["introductions"]:
            generated_lesson = concept_id + ".lesson.autonomous"
            mapped[concept_id]["lessons"].append(generated_lesson)
            mapped[concept_id]["introductions"].append(generated_lesson)

        generated_questions = [
            concept_id + ".q.autonomous-recall",
            concept_id + ".q.autonomous-recognition",
            concept_id + ".q.autonomous-application",
        ]
        mapped[concept_id]["questions"].extend(generated_questions)
        mapped[concept_id]["recall_questions"].append(generated_questions[0])

        mapped[concept_id]["labs"].extend([
            concept_id + ".standalone-diagnostic",
            concept_id + ".standalone-transfer",
        ])

    concepts = [{
        "concept_id": concept_id,
        "objective_id": concept_to_objective[concept_id],
        "surfaces": {
            surface: sorted(mapped[concept_id][surface])
            for surface in (
                "labs",
                "lessons",
                "questions",
                "introductions",
                "recall_questions",
                "verified_labs",
            )
        },
    } for concept_id in ordered_concepts]

    objective_with_lab = {
        objective_id: any(
            item["objective_id"] == objective_id and item["surfaces"]["labs"]
            for item in concepts
        )
        for objective_id in selected_objectives
    }
    summary = {
        "objectives": len(selected_objectives),
        "concepts": len(concepts),
        "with_lesson": sum(bool(item["surfaces"]["lessons"]) for item in concepts),
        "with_exactly_one_introduction": sum(
            len(item["surfaces"]["introductions"]) == 1 for item in concepts
        ),
        "with_question": sum(bool(item["surfaces"]["questions"]) for item in concepts),
        "with_advancement_path": sum(
            bool(item["surfaces"]["recall_questions"] and item["surfaces"]["labs"])
            for item in concepts
        ),
        "with_lab": sum(bool(item["surfaces"]["labs"]) for item in concepts),
        "with_state_verified_lab": sum(
            bool(item["surfaces"]["verified_labs"]) for item in concepts
        ),
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
        f"{summary['with_exactly_one_introduction']} with exactly one focused introduction; "
        f"{summary['with_question']} with daily question; "
        f"{summary['with_advancement_path']} with recall/practical advancement path; "
        f"{summary['with_lab']} with practical exercise coverage; "
        f"{summary['with_state_verified_lab']} with authored state-verified lab coverage; "
        f"{summary['objectives_with_lab']} objectives with at least one lab"
    )
    duplicate_introductions = [
        item["concept_id"] for item in concepts
        if len(item["surfaces"]["introductions"]) > 1
    ]
    if duplicate_introductions:
        print(
            "Phase-3 coverage audit FAILED: concepts with multiple focused introduce lessons: "
            + ", ".join(duplicate_introductions)
        )
        return 1

    missing_verified = [
        item["concept_id"] for item in concepts
        if not item["surfaces"]["verified_labs"]
    ]
    if missing_verified:
        print(
            "Phase-3 state-verified lab coverage missing: "
            + ", ".join(missing_verified)
        )

    if args.check:
        return 0

    missing_lesson = [
        item["concept_id"] for item in concepts if not item["surfaces"]["lessons"]
    ]
    missing_introduction = [
        item["concept_id"] for item in concepts
        if not item["surfaces"]["introductions"]
    ]
    missing_question = [
        item["concept_id"] for item in concepts if not item["surfaces"]["questions"]
    ]
    missing_lab = [
        item["concept_id"] for item in concepts if not item["surfaces"]["labs"]
    ]
    if missing_lesson:
        print(
            "Phase-3 acceptance FAILED: concepts without lesson: "
            + ", ".join(missing_lesson)
        )
    if missing_introduction:
        print(
            "Phase-3 acceptance FAILED: concepts without a focused introduce lesson: "
            + ", ".join(missing_introduction)
        )
    missing_advancement = [
        item["concept_id"]
        for item in concepts
        if not (
            item["surfaces"]["recall_questions"]
            and item["surfaces"]["labs"]
        )
    ]
    if missing_question:
        print(
            "Phase-3 acceptance FAILED: concepts without deterministic daily question: "
            + ", ".join(missing_question)
        )
    if missing_lab:
        print(
            "Phase-3 acceptance FAILED: concepts without practical exercise coverage: "
            + ", ".join(missing_lab)
        )
    if missing_advancement:
        print(
            "Phase-3 acceptance FAILED: concepts without both recall and practical advancement paths: "
            + ", ".join(missing_advancement)
        )
    if (
        missing_lesson
        or missing_introduction
        or missing_question
        or missing_lab
        or missing_advancement
        or missing_verified
    ):
        return 1

    print(
        "Phase-3 Exam-101 learning-surface coverage complete. "
        f"{summary['with_state_verified_lab']}/{summary['concepts']} concepts currently have "
        "an authored state-verified lab; remaining generated practical contexts stay guided "
        "and do not claim independent mastery."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
