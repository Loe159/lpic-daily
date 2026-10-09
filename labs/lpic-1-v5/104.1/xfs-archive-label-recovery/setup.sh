#!/usr/bin/env bash
set -euo pipefail
test -b /dev/vdb
mkfs.xfs -f -L wrongarchive /dev/vdb >/dev/null
mkdir -p /srv/xfsarchive
mount /dev/vdb /srv/xfsarchive
printf 'archive-id=INC-9012\n' > /srv/xfsarchive/manifest.txt
umount /srv/xfsarchive
printf '\nLABEL=archive2026 /srv/xfsarchive xfs defaults,nofail 0 2\n' >> /etc/fstab
