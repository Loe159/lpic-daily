#!/usr/bin/env python3
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = [
    ROOT / "scripts" / "validate_curriculum.py",
    ROOT / "scripts" / "validate_learning_graph.py",
    ROOT / "scripts" / "validate_schemas.py",
]

for script in SCRIPTS:
    completed = subprocess.run([sys.executable, str(script)], cwd=ROOT)
    if completed.returncode != 0:
        sys.exit(completed.returncode)

print("Foundation validation OK")
