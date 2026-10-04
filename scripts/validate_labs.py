#!/usr/bin/env python3
from pathlib import Path
import json
import sys

ROOT = Path(__file__).resolve().parents[1]
LABS = ROOT / "labs"
CURRICULUM = ROOT / "curriculum" / "lpic-1-v5"


def load_json(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ValueError(f"{path.relative_to(ROOT)}: invalid JSON: {exc}") from exc


def safe_local_ref(base, value):
    if not isinstance(value, str) or not value:
        return False
    ref = Path(value)
    if ref.is_absolute() or ".." in ref.parts:
        return False
    try:
        target = (base / ref).resolve()
        target.relative_to(base.resolve())
    except (ValueError, OSError):
        return False
    return target.is_file()


def nonempty_text(value, minimum=1):
    return isinstance(value, str) and len(value.strip()) >= minimum


def main():
    errors = []
    objectives = load_json(CURRICULUM / "objectives.json")
    concepts = load_json(CURRICULUM / "concepts.json")

    objective_ids = {o["id"] for o in objectives["objectives"] if o.get("active")}
    concept_by_id = {c["id"]: c for c in concepts["concepts"] if c.get("active")}

    labs = sorted(LABS.glob("lpic-1-v5/*/*/lab.json"))
    if not labs:
        errors.append("no lab.json files found")

    seen_lab_ids = set()
    seen_hint_ids = set()

    for lab_path in labs:
        lab_dir = lab_path.parent
        lab = load_json(lab_path)
        lab_id = lab.get("id")

        if not isinstance(lab_id, str) or not lab_id:
            errors.append(f"{lab_path.relative_to(ROOT)}: missing id")
            continue
        if lab_id in seen_lab_ids:
            errors.append(f"duplicate lab id {lab_id}")
        seen_lab_ids.add(lab_id)

        if not nonempty_text(lab.get("title_fr"), 3):
            errors.append(f"{lab_id}: title_fr is missing/too short")
        if not nonempty_text(lab.get("brief_fr"), 20):
            errors.append(f"{lab_id}: brief_fr is missing/too short")
        if not nonempty_text(lab.get("debrief_fr"), 20):
            errors.append(f"{lab_id}: debrief_fr is missing/too short")

        practice_context = lab.get("practice_context")
        if (
            not isinstance(practice_context, str)
            or not 3 <= len(practice_context) <= 64
            or any(character not in "abcdefghijklmnopqrstuvwxyz0123456789.-" for character in practice_context)
        ):
            errors.append(f"{lab_id}: practice_context must be a 3-64 character lowercase stable context id")

        criteria = lab.get("success_criteria_fr")
        if not isinstance(criteria, list) or not criteria:
            errors.append(f"{lab_id}: success_criteria_fr must be a non-empty list")
        elif any(not nonempty_text(item, 5) for item in criteria):
            errors.append(f"{lab_id}: every success criterion must contain useful text")
        elif len(criteria) != len(set(criteria)):
            errors.append(f"{lab_id}: duplicate success criteria")

        for objective_id in lab.get("objective_ids", []):
            if objective_id not in objective_ids:
                errors.append(f"{lab_id}: unknown objective {objective_id}")

        for concept_id in lab.get("concept_ids", []):
            concept = concept_by_id.get(concept_id)
            if concept is None:
                errors.append(f"{lab_id}: unknown concept {concept_id}")
                continue
            if concept["objective_id"] not in lab.get("objective_ids", []):
                errors.append(
                    f"{lab_id}: concept {concept_id} belongs to objective "
                    f"{concept['objective_id']} not listed by the lab"
                )

        environment = lab.get("environment", {})
        if environment.get("backend") not in {"podman", "libvirt"}:
            errors.append(f"{lab_id}: unsupported backend {environment.get('backend')!r}")
        backend = environment.get("backend")
        capability_profile = environment.get("capability_profile")
        if backend == "podman" and capability_profile not in {
            "baseline",
            "identity-files",
            "process-lab",
        }:
            errors.append(
                f"{lab_id}: unsupported Podman capability_profile {capability_profile!r}"
            )
        if backend == "libvirt":
            if capability_profile != "full-machine":
                errors.append(
                    f"{lab_id}: libvirt capability_profile must be 'full-machine'"
                )
            if environment.get("writable_guest_paths", []):
                errors.append(
                    f"{lab_id}: libvirt writable_guest_paths must be empty; writable state comes from VM disks"
                )
        if environment.get("network") not in {"none", "isolated"}:
            errors.append(f"{lab_id}: unsupported network mode {environment.get('network')!r}")
        image_ref = environment.get("image_ref", "")
        if ":latest" in image_ref or image_ref.endswith("/latest"):
            errors.append(f"{lab_id}: floating latest image references are forbidden")
        if environment.get("backend") == "podman" and environment.get("network") != "none":
            errors.append(
                f"{lab_id}: Phase-1 Podman content must use network=none until isolated networking is implemented"
            )

        resources = lab.get("resources", {})
        cpu_percent = resources.get("cpu_percent")
        if not isinstance(cpu_percent, int) or isinstance(cpu_percent, bool) or not 10 <= cpu_percent <= 400:
            errors.append(f"{lab_id}: cpu_percent must be an integer from 10 to 400")
        memory_mb = resources.get("memory_mb")
        if not isinstance(memory_mb, int) or isinstance(memory_mb, bool) or not 64 <= memory_mb <= 16384:
            errors.append(f"{lab_id}: memory_mb must be an integer from 64 to 16384")
        elif backend == "libvirt" and memory_mb < 256:
            errors.append(f"{lab_id}: libvirt memory_mb must be at least 256")
        pids = resources.get("pids")
        if not isinstance(pids, int) or isinstance(pids, bool) or not 16 <= pids <= 4096:
            errors.append(f"{lab_id}: pids must be an integer from 16 to 4096")
        timeout_seconds = resources.get("timeout_seconds")
        if not isinstance(timeout_seconds, int) or isinstance(timeout_seconds, bool) or not 30 <= timeout_seconds <= 7200:
            errors.append(f"{lab_id}: timeout_seconds must be an integer from 30 to 7200")

        setup = lab.get("setup", {})
        setup_scope = setup.get("execution_scope")
        if backend == "podman":
            if setup_scope != "sandbox":
                errors.append(f"{lab_id}: Podman setup must execute in sandbox")
            if not safe_local_ref(lab_dir, setup.get("script_ref")):
                errors.append(f"{lab_id}: setup script reference is missing or unsafe")
        elif backend == "libvirt":
            if setup_scope != "none":
                errors.append(f"{lab_id}: Phase-2 libvirt setup must use execution_scope=none")
            if "script_ref" in setup:
                errors.append(f"{lab_id}: libvirt setup=none must not declare script_ref")

        solution_ref = lab.get("reference_solution_ref")
        if solution_ref is not None and not safe_local_ref(lab_dir, solution_ref):
            errors.append(f"{lab_id}: reference solution is missing or unsafe")

        if lab.get("reset_policy") != "disposable":
            errors.append(f"{lab_id}: reset policy must be disposable")

        hint_files = sorted((lab_dir / "hints").glob("*.json"))
        hints = {}
        for hint_path in hint_files:
            hint = load_json(hint_path)
            hint_id = hint.get("id")
            if not isinstance(hint_id, str) or not hint_id:
                errors.append(f"{hint_path.relative_to(ROOT)}: missing/invalid hint id")
                continue
            if hint_id in seen_hint_ids:
                errors.append(f"duplicate hint id {hint_id}")
            seen_hint_ids.add(hint_id)
            hints[hint_id] = hint

            if hint.get("lab_id") != lab_id:
                errors.append(f"{hint_id}: lab_id does not match {lab_id}")
            level = hint.get("level")
            if not isinstance(level, int) or isinstance(level, bool) or level not in (1, 2, 3, 4):
                errors.append(f"{hint_id}: invalid level {level!r}")

        levels = [hint.get("level") for hint in hints.values()]
        valid_levels = (
            len(levels) == 4
            and all(
                isinstance(level, int)
                and not isinstance(level, bool)
                and level in (1, 2, 3, 4)
                for level in levels
            )
            and sorted(levels) == [1, 2, 3, 4]
        )
        if not valid_levels:
            errors.append(
                f"{lab_id}: hint ladder must contain each level 1..4 exactly once; got {levels!r}"
            )
        for hint_id, hint in hints.items():
            level = hint.get("level")
            impact = hint.get("evidence_impact")
            if level == 1 and impact not in {"none", "minor"}:
                errors.append(f"{hint_id}: level 1 must have evidence_impact none/minor")
            elif level in (2, 3) and impact != "material":
                errors.append(f"{hint_id}: level {level} must have evidence_impact material")
            elif level == 4 and impact != "solution-revealed":
                errors.append(f"{hint_id}: level 4 must have evidence_impact solution-revealed")

        requested = lab.get("hint_ids", [])
        valid_requested = (
            isinstance(requested, list)
            and all(isinstance(item, str) and item for item in requested)
        )
        if not valid_requested:
            errors.append(f"{lab_id}: hint_ids must be a list of non-empty strings")
        else:
            requested_set = set(requested)
            if len(requested) != len(requested_set):
                errors.append(f"{lab_id}: duplicate hint IDs")
            if requested_set != set(hints):
                errors.append(
                    f"{lab_id}: hint reference drift missing={sorted(set(hints)-requested_set)} "
                    f"extra={sorted(requested_set-set(hints))}"
                )

        checks = lab.get("checks", [])
        if not checks:
            errors.append(f"{lab_id}: at least one state check is required")

        declared_concepts = set(lab.get("concept_ids", []))
        checked_concepts = set()
        for index, check in enumerate(checks, start=1):
            check_type = check.get("type")
            if check_type not in {
                "file-exists",
                "file-mode",
                "file-owner",
                "file-content-regex",
                "process-running",
                "process-absent",
                "command-exit",
                "block-device-state",
            }:
                errors.append(f"{lab_id}: unknown check type {check_type!r}")

            evidence_concepts = check.get("concept_ids")
            if (
                not isinstance(evidence_concepts, list)
                or not evidence_concepts
                or any(not isinstance(item, str) or not item for item in evidence_concepts)
            ):
                errors.append(f"{lab_id}: check {index} must map to non-empty concept_ids")
                continue
            if len(evidence_concepts) != len(set(evidence_concepts)):
                errors.append(f"{lab_id}: check {index} repeats concept IDs")

            undeclared = set(evidence_concepts) - declared_concepts
            if undeclared:
                errors.append(
                    f"{lab_id}: check {index} maps undeclared concepts {sorted(undeclared)}"
                )
            checked_concepts.update(evidence_concepts)

        missing_check_evidence = declared_concepts - checked_concepts
        if missing_check_evidence:
            errors.append(
                f"{lab_id}: concepts without state-check evidence "
                f"{sorted(missing_check_evidence)}"
            )

    if errors:
        print("Lab validation FAILED:")
        for error in errors:
            print(" -", error)
        sys.exit(1)

    print(f"Lab validation OK: {len(labs)} labs; {len(seen_hint_ids)} hints")


if __name__ == "__main__":
    main()
