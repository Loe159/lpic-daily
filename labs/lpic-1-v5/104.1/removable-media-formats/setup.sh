#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb && test -b /dev/vdc
mkdir -p /srv/media/legacy /srv/media/modern
# The disposable VM disks are supplied blank. Never modify /dev/vda.
test -z "$(blkid -s TYPE -o value /dev/vdb 2>/dev/null || true)"
test -z "$(blkid -s TYPE -o value /dev/vdc 2>/dev/null || true)"
