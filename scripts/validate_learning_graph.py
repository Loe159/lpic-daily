#!/usr/bin/env python3
from collections import defaultdict, deque
from pathlib import Path
import json
import sys

ROOT = Path(__file__).resolve().parents[1]
CURRICULUM = ROOT / "curriculum" / "lpic-1-v5"


def load(name):
    return json.loads((CURRICULUM / name).read_text(encoding="utf-8"))


def fail(errors):
    if errors:
        print("Learning graph validation FAILED:")
        for error in errors:
            print(" -", error)
        sys.exit(1)


def check_acyclic(nodes, include_recommended, errors):
    ids = [n["objective_id"] for n in nodes]
    id_set = set(ids)
    prereqs = {}
    reverse = defaultdict(list)
    indegree = {}

    for node in nodes:
        deps = set(node["hard_prerequisites"])
        if include_recommended:
            deps |= set(node["recommended_prerequisites"])
        prereqs[node["objective_id"]] = deps
        indegree[node["objective_id"]] = len(deps)
        for dep in deps:
            if dep in id_set:
                reverse[dep].append(node["objective_id"])

    queue = deque([node for node in ids if indegree[node] == 0])
    seen = []
    while queue:
        current = queue.popleft()
        seen.append(current)
        for nxt in reverse[current]:
            indegree[nxt] -= 1
            if indegree[nxt] == 0:
                queue.append(nxt)

    if len(seen) != len(ids):
        kind = "hard+recommended" if include_recommended else "hard"
        remaining = sorted(set(ids) - set(seen))
        errors.append(f"{kind} prerequisite graph contains a cycle involving {remaining}")


def main():
    objectives = load("objectives.json")
    graph = load("prerequisites.json")
    concepts = load("concepts.json")
    phase1 = load("phase1-slice.json")
    phase3 = load("phase3-exam101.json")
    phase4 = load("phase4-exam102.json")

    errors = []
    active = [o for o in objectives["objectives"] if o.get("active")]
    active_ids = [o["id"] for o in active]
    active_set = set(active_ids)

    nodes = graph["nodes"]
    graph_ids = [n["objective_id"] for n in nodes]

    if len(graph_ids) != len(set(graph_ids)):
        errors.append("duplicate objective IDs in prerequisite graph")
    if set(graph_ids) != active_set:
        errors.append(
            "prerequisite graph objective set differs from active objectives: "
            f"missing={sorted(active_set-set(graph_ids))}, extra={sorted(set(graph_ids)-active_set)}"
        )

    node_by_id = {n["objective_id"]: n for n in nodes}
    for node in nodes:
        hard = node["hard_prerequisites"]
        recommended = node["recommended_prerequisites"]
        for dep in hard + recommended:
            if dep not in active_set:
                errors.append(f"{node['objective_id']}: unknown prerequisite {dep}")
            if dep == node["objective_id"]:
                errors.append(f"{node['objective_id']}: self prerequisite")
        overlap = set(hard) & set(recommended)
        if overlap:
            errors.append(f"{node['objective_id']}: dependency listed as hard and recommended: {sorted(overlap)}")

    check_acyclic(nodes, False, errors)
    check_acyclic(nodes, True, errors)

    order = graph["reference_topological_order"]
    if len(order) != len(active_ids) or set(order) != active_set:
        errors.append("reference_topological_order must contain every active objective exactly once")
    else:
        pos = {objective_id: index for index, objective_id in enumerate(order)}
        for node in nodes:
            for dep in node["hard_prerequisites"]:
                if pos[dep] >= pos[node["objective_id"]]:
                    errors.append(
                        f"reference order violates hard prerequisite {dep} -> {node['objective_id']}"
                    )

    concept_rows = concepts["concepts"]
    concept_ids = [c["id"] for c in concept_rows]
    if len(concept_ids) != len(set(concept_ids)):
        errors.append("duplicate concept IDs")

    concepts_by_objective = defaultdict(list)
    for concept in concept_rows:
        objective_id = concept["objective_id"]
        if objective_id not in active_set:
            errors.append(f"{concept['id']}: unknown objective {objective_id}")
            continue
        concepts_by_objective[objective_id].append(concept)
        inherited = concept.get("inherited_hard_objective_prerequisites", [])
        expected = node_by_id[objective_id]["hard_prerequisites"]
        if inherited != expected:
            errors.append(
                f"{concept['id']}: inherited hard prerequisites drift; expected {expected}, got {inherited}"
            )

    for objective in active:
        rows = concepts_by_objective[objective["id"]]
        expected_titles = objective["concepts"]
        actual_titles = [row["title_fr"] for row in sorted(rows, key=lambda x: x["pedagogy_order"])]
        if actual_titles != expected_titles:
            errors.append(f"{objective['id']}: concept inventory is not synchronized with objectives.json")

    selected = phase1["selected_objectives"]
    selected_set = set(selected)
    for objective_id in selected:
        if objective_id not in active_set:
            errors.append(f"Phase 1 slice references unknown objective {objective_id}")
            continue
        missing = set(node_by_id[objective_id]["hard_prerequisites"]) - selected_set
        if missing:
            errors.append(
                f"Phase 1 slice is not closed over hard prerequisites for {objective_id}: {sorted(missing)}"
            )

    phase1_concepts = phase1["objective_concepts"]
    concept_set = set(concept_ids)
    for objective_id in selected:
        expected = {c["id"] for c in concepts_by_objective[objective_id]}
        actual = set(phase1_concepts.get(objective_id, []))
        if actual != expected:
            errors.append(
                f"Phase 1 concept set drift for {objective_id}: "
                f"missing={sorted(expected-actual)}, extra={sorted(actual-expected)}"
            )
        unknown = actual - concept_set
        if unknown:
            errors.append(f"Phase 1 unknown concept IDs for {objective_id}: {sorted(unknown)}")


    expected_phase3_objectives = [o["id"] for o in active if o["exam"] == "101"]
    if phase3.get("selected_objectives") != expected_phase3_objectives:
        errors.append(
            "Phase 3 objective scope drift: "
            f"expected={expected_phase3_objectives}, got={phase3.get('selected_objectives')}"
        )
    if phase3.get("exam") != "101" or phase3.get("exam_code") != "101-500":
        errors.append("Phase 3 must describe Exam 101 / 101-500")
    if phase3.get("topics") != ["101", "102", "103", "104"]:
        errors.append(f"Phase 3 topics drift: {phase3.get('topics')}")
    if phase3.get("scope_source") != objectives.get("canonical_scope_source"):
        errors.append("Phase 3 scope source differs from objectives.json canonical source")

    phase3_selected = phase3.get("selected_objectives", [])
    phase3_selected_set = set(phase3_selected)
    phase3_concepts = phase3.get("objective_concepts", {})
    if set(phase3_concepts) != phase3_selected_set:
        errors.append(
            "Phase 3 objective_concepts keys differ from selected objectives: "
            f"missing={sorted(phase3_selected_set-set(phase3_concepts))}, "
            f"extra={sorted(set(phase3_concepts)-phase3_selected_set)}"
        )

    mapped_phase3_concepts = 0
    for objective_id in phase3_selected:
        if objective_id not in active_set:
            errors.append(f"Phase 3 references unknown objective {objective_id}")
            continue
        if next(o for o in active if o["id"] == objective_id)["exam"] != "101":
            errors.append(f"Phase 3 references non-Exam-101 objective {objective_id}")
        missing = set(node_by_id[objective_id]["hard_prerequisites"]) - phase3_selected_set
        if missing:
            errors.append(
                f"Phase 3 scope is not closed over hard prerequisites for {objective_id}: {sorted(missing)}"
            )
        expected = [c["id"] for c in sorted(concepts_by_objective[objective_id], key=lambda x: x["pedagogy_order"])]
        actual = phase3_concepts.get(objective_id, [])
        mapped_phase3_concepts += len(actual)
        if actual != expected:
            errors.append(
                f"Phase 3 concept set drift for {objective_id}: expected={expected}, got={actual}"
            )

    if phase3.get("concept_count") != mapped_phase3_concepts:
        errors.append(
            f"Phase 3 concept_count={phase3.get('concept_count')}, mapped={mapped_phase3_concepts}"
        )


    expected_phase4_objectives = [o["id"] for o in active if o["exam"] == "102"]
    if phase4.get("selected_objectives") != expected_phase4_objectives:
        errors.append(
            "Phase 4 objective scope drift: "
            f"expected={expected_phase4_objectives}, got={phase4.get('selected_objectives')}"
        )
    if phase4.get("exam") != "102" or phase4.get("exam_code") != "102-500":
        errors.append("Phase 4 must describe Exam 102 / 102-500")
    if phase4.get("topics") != ["105", "106", "107", "108", "109", "110"]:
        errors.append(f"Phase 4 topics drift: {phase4.get('topics')}")
    if phase4.get("scope_source") != objectives.get("canonical_scope_source"):
        errors.append("Phase 4 scope source differs from objectives.json canonical source")

    phase4_selected = phase4.get("selected_objectives", [])
    phase4_selected_set = set(phase4_selected)
    phase4_concepts = phase4.get("objective_concepts", {})
    if set(phase4_concepts) != phase4_selected_set:
        errors.append(
            "Phase 4 objective_concepts keys differ from selected objectives: "
            f"missing={sorted(phase4_selected_set-set(phase4_concepts))}, "
            f"extra={sorted(set(phase4_concepts)-phase4_selected_set)}"
        )

    mapped_phase4_concepts = 0
    for objective_id in phase4_selected:
        if objective_id not in active_set:
            errors.append(f"Phase 4 references unknown objective {objective_id}")
            continue
        if next(o for o in active if o["id"] == objective_id)["exam"] != "102":
            errors.append(f"Phase 4 references non-Exam-102 objective {objective_id}")
        expected = [c["id"] for c in sorted(concepts_by_objective[objective_id], key=lambda x: x["pedagogy_order"])]
        actual = phase4_concepts.get(objective_id, [])
        mapped_phase4_concepts += len(actual)
        if actual != expected:
            errors.append(
                f"Phase 4 concept set drift for {objective_id}: expected={expected}, got={actual}"
            )

    if phase4.get("concept_count") != mapped_phase4_concepts:
        errors.append(
            f"Phase 4 concept_count={phase4.get('concept_count')}, mapped={mapped_phase4_concepts}"
        )

    fail(errors)
    print(
        "Learning graph validation OK: "
        f"{len(active_ids)} objectives; {len(concept_rows)} concepts; "
        f"Phase 1 objectives={','.join(selected)}; "
        f"Exam-101 concepts={mapped_phase3_concepts}; Exam-102 concepts={mapped_phase4_concepts}"
    )


if __name__ == "__main__":
    main()
