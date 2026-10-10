#!/usr/bin/env bash
set -euo pipefail
test "$(blkid -s LABEL -o value /dev/sda)" = LPICARCHIVE
umount /srv/lpic-archive
sed -i 's|^LABEL=LPICQUEUE[[:space:]]\+/srv/lpic-archive[[:space:]]|LABEL=LPICARCHIVE /srv/lpic-archive |' /etc/fstab
mount /srv/lpic-archive
test "$(cat /srv/lpic-archive/index.txt)" = 'archive-index=retain-51'
