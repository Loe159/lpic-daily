#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/audit
printf 'Inventaire RPM incomplet\n' > /workspace/audit/README.txt
: > /workspace/audit/owners.tsv
: > /workspace/audit/bash-files.txt
