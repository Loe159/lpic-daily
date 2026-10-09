#!/usr/bin/env bash
set -euo pipefail
mkdir -p /workspace/disks
img=/workspace/disks/audit.ext4
truncate -s 64M "$img"
mkfs.ext4 -q -F -L audit-old "$img"
printf 'ticket=inc-2817-do-not-delete\n' > /workspace/disks/evidence-source.txt
debugfs -w -R 'write /workspace/disks/evidence-source.txt /evidence.txt' "$img" >/dev/null 2>&1
tune2fs -m 5 "$img" >/dev/null 2>&1
debugfs -w -R 'set_super_value state 2' "$img" >/dev/null 2>&1
