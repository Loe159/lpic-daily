#!/usr/bin/env bash
set -euo pipefail
uuid=$(blkid -s UUID -o value /dev/vdb)
test -n "$uuid"
sed -i -E "s|^UUID=00000000-0000-0000-0000-000000000000[[:space:]]+/srv/archive[[:space:]]+.*$|UUID=$uuid /srv/archive ext4 defaults,nodev,nosuid,nofail 0 2|" /etc/fstab
systemctl daemon-reload
mount /srv/archive
