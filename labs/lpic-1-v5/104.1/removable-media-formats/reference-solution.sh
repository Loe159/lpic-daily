#!/usr/bin/env bash
set -euo pipefail
mkfs.vfat -F 32 -n LEGACY /dev/vdb >/dev/null
mkfs.exfat -n MODERN /dev/vdc >/dev/null
mount /dev/vdb /srv/media/legacy
mount /dev/vdc /srv/media/modern
