#!/usr/bin/env bash
set -euo pipefail
img=/workspace/disks/audit.ext4
set +e
e2fsck -fy "$img"
rc=$?
set -e
if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then exit "$rc"; fi
tune2fs -L audit-ready -m 1 "$img"
