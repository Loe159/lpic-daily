#!/usr/bin/env bash
set -euo pipefail
for cmd in mkfs.ext4 debugfs blkid mount findmnt udevadm; do command -v "$cmd" >/dev/null; done
test -b /dev/sda && test -b /dev/vdb
[[ "$(readlink -f /sys/class/block/sda/device)" == *"/ata"* ]]
[[ "$(readlink -f /sys/class/block/vdb/device)" == *"/virtio"* ]]
mkfs.ext4 -F -q -L LPICARCHIVE /dev/sda
mkfs.ext4 -F -q -L LPICQUEUE /dev/vdb
install -d -m 0755 /srv/lpic-archive /mnt/lpic-archive-stage /mnt/lpic-queue-stage
mount /dev/sda /mnt/lpic-archive-stage
printf 'archive-index=retain-51\n' > /mnt/lpic-archive-stage/index.txt
umount /mnt/lpic-archive-stage
mount /dev/vdb /mnt/lpic-queue-stage
printf 'queue-volume=preserve-51\n' > /mnt/lpic-queue-stage/queue.txt
umount /mnt/lpic-queue-stage
printf 'LABEL=LPICQUEUE /srv/lpic-archive ext4 defaults,nofail 0 2\n' >> /etc/fstab
mount /srv/lpic-archive
mountpoint -q /srv/lpic-archive
test ! -e /srv/lpic-archive/index.txt
