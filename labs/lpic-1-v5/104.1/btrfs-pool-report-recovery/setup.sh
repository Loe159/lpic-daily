#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb && test -b /dev/vdc
mkfs.btrfs -f -L lpic-pool /dev/vdb /dev/vdc >/dev/null
mkdir -p /srv/pool
mount /dev/vdb /srv/pool
printf 'pool-manifest=retain-736\n' > /srv/pool/manifest.txt
