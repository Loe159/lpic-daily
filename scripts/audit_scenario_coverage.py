#!/usr/bin/env python3
import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
CONCEPTS = ROOT / "curriculum" / "lpic-1-v5" / "concepts.json"
MATRIX = ROOT / "curriculum" / "lpic-1-v5" / "scenario-coverage.json"
LAB_GLOB = "labs/lpic-1-v5/*/*/lab.json"
VALID_STATUS = {"planned", "implemented", "accepted"}
VALID_STRENGTH = {"behavior", "state", "artifact", "decision"}
VALID_BACKEND = {"podman", "libvirt"}

def load(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ValueError(f"{path.relative_to(ROOT)}: invalid JSON: {exc}") from exc

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    if not args.check:
        parser.error("--check is required")

    try:
        concepts_doc = load(CONCEPTS)
        matrix = load(MATRIX)
    except ValueError as exc:
        print(f"Scenario coverage FAILED: {exc}")
        return 1

    concepts = concepts_doc.get("concepts", concepts_doc)
    active = {c["id"]: c for c in concepts if c.get("active", True)}
    active_by_objective = defaultdict(set)
    for concept_id, concept in active.items():
        active_by_objective[concept["objective_id"]].add(concept_id)

    labs = {}
    for path in sorted(ROOT.glob(LAB_GLOB)):
        lab = load(path)
        labs[lab["id"]] = (lab, path)

    errors = []
    accepted_coverage = defaultdict(set)
    implemented_coverage = defaultdict(set)
    scenario_contexts = defaultdict(set)
    strengths = Counter()
    accepted_ids = set()
    scenarios = matrix.get("scenarios", [])
    seen = set()

    for entry in scenarios:
        scenario_id = entry.get("scenario_id")
        if not scenario_id or scenario_id in seen:
            errors.append(f"duplicate or missing scenario_id: {scenario_id!r}")
            continue
        seen.add(scenario_id)
        if entry.get("status") not in VALID_STATUS:
            errors.append(f"{scenario_id}: invalid status {entry.get('status')!r}")
        if entry.get("evidence_strength") not in VALID_STRENGTH:
            errors.append(f"{scenario_id}: invalid evidence_strength {entry.get('evidence_strength')!r}")
        if entry.get("backend") not in VALID_BACKEND:
            errors.append(f"{scenario_id}: invalid backend {entry.get('backend')!r}")
        status = entry.get("status")
        if status == "planned":
            continue

        if scenario_id not in labs:
            errors.append(f"{scenario_id}: {status} scenario has no authored lab")
            continue

        lab, path = labs[scenario_id]
        if status == "accepted":
            accepted_ids.add(scenario_id)
            strengths[entry["evidence_strength"]] += 1
            if "scenario-accepted" not in lab.get("labels", []):
                errors.append(f"{scenario_id}: accepted matrix entry lacks scenario-accepted lab label")
        elif "scenario-accepted" in lab.get("labels", []):
            errors.append(f"{scenario_id}: implemented scenario must not carry scenario-accepted label")

        for field in ("objective_ids", "concept_ids"):
            if sorted(entry.get(field, [])) != sorted(lab.get(field, [])):
                errors.append(f"{scenario_id}: matrix {field} differs from {path.relative_to(ROOT)}")
        if entry.get("backend") != lab.get("environment", {}).get("backend"):
            errors.append(f"{scenario_id}: matrix backend differs from authored lab")
        if not lab.get("reference_solution_ref"):
            errors.append(f"{scenario_id}: {status} scenario lacks reference_solution_ref")
        else:
            ref = path.parent / lab["reference_solution_ref"]
            if not ref.is_file():
                errors.append(f"{scenario_id}: missing reference solution {ref.relative_to(ROOT)}")

        checked = set()
        for check in lab.get("checks", []):
            checked.update(check.get("concept_ids", []))
        for concept_id in entry.get("concept_ids", []):
            if concept_id not in active:
                errors.append(f"{scenario_id}: unknown/inactive concept {concept_id}")
                continue
            if concept_id not in checked:
                errors.append(f"{scenario_id}: concept {concept_id} has no mapped check")
            if status == "accepted":
                accepted_coverage[concept_id].add(scenario_id)
                scenario_contexts[concept_id].add(lab.get("practice_context", ""))
            elif status == "implemented":
                implemented_coverage[concept_id].add(scenario_id)

    labelled = {
        lab_id for lab_id, (lab, _) in labs.items()
        if "scenario-accepted" in lab.get("labels", [])
    }
    for lab_id in sorted(labelled - accepted_ids):
        errors.append(f"{lab_id}: scenario-accepted lab missing accepted matrix entry")

    migrated = matrix.get("migrated_objective_ids", [])
    for objective_id in migrated:
        expected = active_by_objective.get(objective_id, set())
        if not expected:
            errors.append(f"migrated objective {objective_id} has no active concepts")
            continue
        missing = expected - set(accepted_coverage)
        if missing:
            errors.append(
                f"{objective_id}: migrated objective missing accepted scenario coverage for "
                + ", ".join(sorted(missing))
            )

    active_ids = set(active)
    accepted_ids_by_concept = set(accepted_coverage)
    implemented_only_ids = set(implemented_coverage) - accepted_ids_by_concept
    no_scenario_ids = active_ids - accepted_ids_by_concept - set(implemented_coverage)
    exam101_ids = {
        cid for cid, concept in active.items()
        if concept["objective_id"].split(".", 1)[0] in {"101", "102", "103", "104"}
    }
    fallback_generation_enabled = (ROOT / "internal" / "lab" / "generated.go").is_file()
    fallback_only = active_ids - accepted_ids_by_concept if fallback_generation_enabled else set()
    without_practice = (
        set() if fallback_generation_enabled else active_ids - accepted_ids_by_concept
    )
    strength_summary = ", ".join(f"{k}={strengths[k]}" for k in sorted(VALID_STRENGTH))
    implemented_count = sum(entry.get("status") == "implemented" for entry in scenarios)
    scenario_sizes = [len(entry.get("concept_ids", [])) for entry in scenarios if entry.get("status") == "accepted"]
    average = (sum(scenario_sizes) / len(scenario_sizes)) if scenario_sizes else 0.0
    maximum = max(scenario_sizes, default=0)
    one_context = sum(len(contexts) == 1 for contexts in scenario_contexts.values())
    fully_migrated = [
        objective_id for objective_id, ids in active_by_objective.items()
        if ids and ids.issubset(accepted_ids_by_concept)
    ]

    print(
        "Scenario coverage: "
        f"active={len(active_ids)}; accepted-covered={len(accepted_ids_by_concept)}; "
        f"fallback-only={len(fallback_only)}; without-practice={len(without_practice)}; "
        f"accepted-scenarios={len(scenario_sizes)}; implemented-scenarios={implemented_count}; "
        f"avg-concepts={average:.2f}; "
        f"max-concepts={maximum}; accepted-single-context={one_context}; "
        f"fully-migrated-objectives={len(fully_migrated)}"
    )
    print(f"Evidence strength: {strength_summary}")
    print(
        "Exam 101 scenario migration: "
        f"accepted-covered={len(exam101_ids & accepted_ids_by_concept)}; "
        f"implemented-only={len(exam101_ids & implemented_only_ids)}; "
        f"not-implemented={len(exam101_ids & no_scenario_ids)}; "
        f"active={len(exam101_ids)}"
    )
    if fully_migrated:
        print("Migrated objectives: " + ", ".join(sorted(fully_migrated)))

    if errors:
        for error in errors:
            print("Scenario coverage FAILED: " + error)
        return 1
    return 0

if __name__ == "__main__":
    sys.exit(main())
