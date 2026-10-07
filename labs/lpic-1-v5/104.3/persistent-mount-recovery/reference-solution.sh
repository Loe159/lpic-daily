#!/usr/bin/env bash
set -euo pipefail
mkfs.ext4 -F -L LPICDATA /dev/vdb
mkdir -p /srv/lpic-data
mount /dev/vdb /srv/lpic-data
umount /srv/lpic-data
uuid=$(blkid -s UUID -o value /dev/vdb)
printf 'UUID=%s /srv/lpic-data ext4 defaults,nodev 0 2\n' "$uuid" >> /etc/fstab
systemctl daemon-reload
systemctl start srv-lpic\x2ddata.mount
mkfs.vfat -n LPICUSB /dev/vdc
mkdir -p /media/lpic-removable
mount /dev/vdc /media/lpic-removable
printf 'removable-ok\n' > /media/lpic-removable/marker.txt
sync
umount /media/lpic-removable
