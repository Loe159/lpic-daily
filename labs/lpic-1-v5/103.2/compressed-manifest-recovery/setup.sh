#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/input /workspace/output
seed="$(od -An -N2 -tu2 /dev/urandom | tr -d '[:space:]')"
printf 'alpha-%s\nbeta\n' "$((seed % 10000))" | gzip -c > /workspace/input/day1.txt.gz
printf 'gamma\n' | bzip2 -c > /workspace/input/day2.txt.bz2
printf 'delta\nepsilon\n' | xz -c > /workspace/input/day3.txt.xz
sha256sum /workspace/input/day1.txt.gz /workspace/input/day2.txt.bz2 > /workspace/input/manifest.sha256
printf '%064d  %s\n' 0 /workspace/input/day3.txt.xz >> /workspace/input/manifest.sha256
