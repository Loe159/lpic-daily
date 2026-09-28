#!/usr/bin/env python3
import argparse
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
SLICE_PATH = ROOT / "curriculum" / "lpic-1-v5" / "phase1-slice.json"
MATRIX_PATH = ROOT / "curriculum" / "lpic-1-v5" / "phase1-coverage.json"

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


def artifact_records(pattern):
    records = []
    for path in sorted(ROOT.glob(pattern)):
        data = load_json(path)
        artifact_id = data.get("id")
        concepts = data.get("concept_ids")
        objectives = data.get("objective_ids")
        if not isinstance(artifact_id, str) or not artifact_id:
            raise ValueError(f"{path.relative_to(ROOT)}: missing id")
        if not isinstance(concepts, list) or not all(isinstance(item, str) for item in concepts):
            raise ValueError(f"{path.relative_to(ROOT)}: concept_ids must be a list of strings")
        if not isinstance(objectives, list) or not all(isinstance(item, str) for item in objectives):
            raise ValueError(f"{path.relative_to(ROOT)}: objective_ids must be a list of strings")
        records.append((artifact_id, set(concepts), set(objectives), path))
    return records


def generate():
    phase1 = load_json(SLICE_PATH)
    selected_objectives = phase1["selected_objectives"]
    objective_concepts = phase1["objective_concepts"]

    concept_to_objective = {}
    ordered_concepts = []
    for objective_id in selected_objectives:
        for concept_id in objective_concepts[objective_id]:
            if concept_id in concept_to_objective:
                raise ValueError(f"duplicate Phase-1 concept {concept_id}")
            concept_to_objective[concept_id] = objective_id
            ordered_concepts.append(concept_id)

    mapped = {
        concept_id: {"labs": [], "lessons": [], "questions": []}
        for concept_id in ordered_concepts
    }

    seen_artifact_ids = set()
    for surface, pattern in SURFACE_GLOBS.items():
        for artifact_id, concept_ids, objective_ids, path in artifact_records(pattern):
            if artifact_id in seen_artifact_ids:
                raise ValueError(f"duplicate learning artifact id {artifact_id}")
            seen_artifact_ids.add(artifact_id)

            for concept_id in sorted(concept_ids):
                objective_id = concept_to_objective.get(concept_id)
                if objective_id is None:
                    continue
                if objective_id not in objective_ids:
                    raise ValueError(
                        f"{path.relative_to(ROOT)}: concept {concept_id} belongs to "
                        f"Phase-1 objective {objective_id}, but that objective is not referenced"
                    )
                mapped[concept_id][surface].append(artifact_id)

    concepts = []
    for concept_id in ordered_concepts:
        objective_id = concept_to_objective[concept_id]
        surfaces = {
            name: sorted(mapped[concept_id][name])
            for name in ("labs", "lessons", "questions")
        }
        concepts.append({
            "concept_id": concept_id,
            "objective_id": objective_id,
            "surfaces": surfaces,
        })

    return {
        "schema_version": "1.0.0",
        "slice_id": phase1["id"],
        "generated_from": {
            "slice": str(SLICE_PATH.relative_to(ROOT)),
            **SURFACE_GLOBS,
        },
        "summary": {
            "concepts": len(concepts),
            "with_any_surface": sum(
                any(item["surfaces"][surface] for surface in SURFACE_GLOBS)
                for item in concepts
            ),
            "with_lab": sum(bool(item["surfaces"]["labs"]) for item in concepts),
            "with_lesson": sum(bool(item["surfaces"]["lessons"]) for item in concepts),
            "with_question": sum(bool(item["surfaces"]["questions"]) for item in concepts),
        },
        "concepts": concepts,
    }


def canonical(data):
    return json.dumps(data, ensure_ascii=False, indent=2) + "\n"


def main():
    parser = argparse.ArgumentParser()
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--write", action="store_true")
    args = parser.parse_args()

    try:
        generated = generate()
    except ValueError as exc:
        print(f"Phase-1 coverage generation FAILED: {exc}")
        return 1

    if args.write:
        MATRIX_PATH.write_text(canonical(generated), encoding="utf-8")
        print(
            f"Wrote {MATRIX_PATH.relative_to(ROOT)}: "
            f"{generated['summary']['with_any_surface']}/{generated['summary']['concepts']} "
            "concepts currently have at least one authored surface"
        )
        return 0

    if not MATRIX_PATH.exists():
        print(f"Phase-1 coverage FAILED: missing {MATRIX_PATH.relative_to(ROOT)}")
        return 1

    committed = load_json(MATRIX_PATH)
    if committed != generated:
        print("Phase-1 coverage FAILED: committed matrix is stale.")
        print("Run: python3 scripts/generate_phase1_coverage.py --write")
        return 1

    summary = generated["summary"]
    if summary["with_any_surface"] != summary["concepts"]:
        uncovered = [
            item["concept_id"]
            for item in generated["concepts"]
            if not any(item["surfaces"][surface] for surface in SURFACE_GLOBS)
        ]
        print(
            "Phase-1 coverage FAILED: "
            f"{len(uncovered)} concept(s) have no authored learning surface: "
            + ", ".join(uncovered)
        )
        return 1

    print(
        "Phase-1 coverage OK: "
        f"{summary['concepts']} concepts; "
        f"{summary['with_any_surface']} with any surface; "
        f"{summary['with_lab']} with lab; "
        f"{summary['with_lesson']} with lesson; "
        f"{summary['with_question']} with question"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
