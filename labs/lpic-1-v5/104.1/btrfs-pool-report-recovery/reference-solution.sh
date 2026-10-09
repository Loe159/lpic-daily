#!/usr/bin/env bash
set -euo pipefail
mount -o remount,compress=zstd /srv/pool
btrfs subvolume create /srv/pool/reports >/dev/null
