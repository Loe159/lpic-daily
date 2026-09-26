#!/usr/bin/env python3
from pathlib import Path
import json
import sys

ROOT = Path(__file__).resolve().parents[1]
SCHEMAS = ROOT / "schemas"


def walk_refs(value):
    if isinstance(value, dict):
        for key, child in value.items():
            if key == "$ref" and isinstance(child, str):
                yield child
            yield from walk_refs(child)
    elif isinstance(value, list):
        for child in value:
            yield from walk_refs(child)


def main():
    errors = []
    ids = {}
    files = sorted(SCHEMAS.glob("*.schema.json"))

    if not files:
        errors.append("no JSON Schema files found")

    for path in files:
        try:
            schema = json.loads(path.read_text(encoding="utf-8"))
        except Exception as exc:
            errors.append(f"{path.name}: invalid JSON: {exc}")
            continue

        if schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
            errors.append(f"{path.name}: must use JSON Schema Draft 2020-12")

        schema_id = schema.get("$id")
        if not schema_id:
            errors.append(f"{path.name}: missing $id")
        elif schema_id in ids:
            errors.append(f"{path.name}: duplicate $id also used by {ids[schema_id]}")
        else:
            ids[schema_id] = path.name

        # common.schema.json is a definitions-only schema; instance schemas describe objects.
        if path.name != "common.schema.json" and schema.get("type") != "object":
            errors.append(f"{path.name}: root schema must describe an object")

        for ref in walk_refs(schema):
            if ref.startswith("#") or ref.startswith("http://") or ref.startswith("https://"):
                continue
            target = ref.split("#", 1)[0]
            if target and not (SCHEMAS / target).exists():
                errors.append(f"{path.name}: missing local $ref target {target}")

    if errors:
        print("Schema validation FAILED:")
        for error in errors:
            print(" -", error)
        sys.exit(1)

    print(f"Schema validation OK: {len(files)} schema files; local refs resolved")


if __name__ == "__main__":
    main()
