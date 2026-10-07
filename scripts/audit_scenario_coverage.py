#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CURRICULUM = ROOT / "curriculum" / "lpic-1-v5"
LABS = ROOT / "labs" / "lpic-1-v5"
COVERAGE = CURRICULUM / "scenario-coverage.json"

def load_json(path: Path):
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)

concepts_doc = load_json(CURRICULUM / "concepts.json")
coverage = load_json(COVERAGE)

active = {item["id"]: item for item in concepts_doc["concepts"] if item.get("active", False)}
active_by_objective = defaultdict(set)
for concept_id, concept in active.items():
    active_by_objective[concept["objective_id"]].add(concept_id)

lab_by_id = {}
for path in LABS.glob("**/lab.json"):
    lab = load_json(path)
    lab_by_id[lab["id"]] = lab

accepted = [row for row in coverage.get("scenarios", []) if row.get("status") == "accepted"]
covered = defaultdict(set)
strengths = Counter()
errors = []

for row in accepted:
    scenario_id = row["scenario_id"]
    lab = lab_by_id.get(scenario_id)
    if lab is None:
        errors.append(f"accepted scenario missing lab: {scenario_id}")
        continue
    if "scenario-accepted" not in lab.get("labels", []):
        errors.append(f"accepted scenario missing scenario-accepted label: {scenario_id}")
    if lab.get("environment", {}).get("backend") != row.get("backend"):
        errors.append(f"backend mismatch for {scenario_id}")

    declared = set(row.get("concept_ids", []))
    lab_concepts = set(lab.get("concept_ids", []))
    if declared != lab_concepts:
        errors.append(f"coverage/lab concept mismatch for {scenario_id}")

    evidenced = set()
    for check in lab.get("checks", []):
        evidenced.update(check.get("concept_ids", []))
    missing_evidence = declared - evidenced
    if missing_evidence:
        errors.append(f"{scenario_id} concepts without check evidence: {sorted(missing_evidence)}")

    unknown = declared - active.keys()
    if unknown:
        errors.append(f"{scenario_id} references unknown/inactive concepts: {sorted(unknown)}")

    for concept_id in declared:
        covered[concept_id].add(scenario_id)
    strengths[row.get("evidence_strength", "unknown")] += 1

complete_objectives = set(coverage.get("scenario_complete_objectives", []))
for objective_id in complete_objectives:
    expected = active_by_objective.get(objective_id, set())
    got = {concept_id for concept_id in expected if covered.get(concept_id)}
    missing = expected - got
    if missing:
        errors.append(f"{objective_id} marked scenario-complete but missing: {sorted(missing)}")

fallback_only = sorted(set(active) - set(covered))
single_context = sorted(concept_id for concept_id, scenarios in covered.items() if len(scenarios) == 1)

print(f"Active concepts: {len(active)}")
print(f"Scenario-covered concepts: {len(covered)}")
print(f"Fallback-only concepts: {len(fallback_only)}")
print(f"Accepted scenarios: {len(accepted)}")
print("Evidence strengths: " + ", ".join(f"{key}={value}" for key, value in sorted(strengths.items())))
print(f"Single-scenario concepts: {len(single_context)}")
print("Scenario-complete objectives: " + (", ".join(sorted(complete_objectives)) or "none"))

if errors:
    for error in errors:
        print(f"ERROR: {error}", file=sys.stderr)
    sys.exit(1)

print("Scenario coverage audit OK")
