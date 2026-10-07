#!/usr/bin/env bash
set -euo pipefail
cat /workspace/input/targets.txt | xargs /workspace/bin/describe-file | sort > /workspace/output/processed.log
