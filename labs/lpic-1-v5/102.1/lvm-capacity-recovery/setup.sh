#!/usr/bin/env bash
set -euo pipefail
command -v pvcreate >/dev/null && command -v vgcreate >/dev/null && command -v lvcreate >/dev/null
test -b /dev/vdb
mkdir -p /srv/app-data
test -z "$(lsblk -no FSTYPE /dev/vdb | tr -d '[:space:]')"
