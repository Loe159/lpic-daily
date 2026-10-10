#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/releases /run/lpic
printf 'release=v2\n' > /workspace/releases/app-v2.conf
ln -s releases/app-v1.conf /workspace/current.conf
rm -f /workspace/releases/archive-v2.conf /workspace/releases/snapshot-v2.conf /workspace/limitations.txt /run/lpic/cross-fs-link
