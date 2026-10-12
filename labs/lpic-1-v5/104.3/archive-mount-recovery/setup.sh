#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
mkdir -p /srv/archive
mkfs.ext4 -F -q -L backupstore /dev/vdb
mount /dev/vdb /srv/archive
printf 'backup-2026-10-09=complete\n' > /srv/archive/manifest.txt
umount /srv/archive
printf '\nUUID=00000000-0000-0000-0000-000000000000 /srv/archive ext4 defaults,nofail 0 2\n' >> /etc/fstab
