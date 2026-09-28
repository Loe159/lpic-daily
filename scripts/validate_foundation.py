#!/usr/bin/env python3
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = [
    ROOT / "scripts" / "validate_curriculum.py",
    ROOT / "scripts" / "validate_learning_graph.py",
    ROOT / "scripts" / "validate_schemas.py",
    ROOT / "scripts" / "validate_labs.py",
    ROOT / "scripts" / "validate_vm_image_sources.py",
    ROOT / "scripts" / "generate_phase1_coverage.py",
]

for script in SCRIPTS:
    command = [sys.executable, str(script)]
    if script.name == "generate_phase1_coverage.py":
        command.append("--check")
    completed = subprocess.run(command, cwd=ROOT)
    if completed.returncode != 0:
        sys.exit(completed.returncode)

print("Foundation validation OK")
