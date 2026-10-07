#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-fs /mnt/ext /mnt/xfs
mkfs.ext4 -F -L LPICMAINT /dev/vdb
tune2fs -m 1 -c 20 /dev/vdb
mount /dev/vdb /mnt/ext
for n in $(seq 1 100); do printf x > "/mnt/ext/file-$n"; done
dd if=/dev/zero of=/mnt/ext/payload.bin bs=1M count=8 status=none
df -P /mnt/ext > /root/lpic-fs/df-blocks.txt
df -Pi /mnt/ext > /root/lpic-fs/df-inodes.txt
du -sk /mnt/ext > /root/lpic-fs/du.txt
set +e
dd if=/dev/zero of=/mnt/ext/fill.bin bs=1M count=400 status=none 2> /root/lpic-fs/enospc.txt
set -e
rm -f /mnt/ext/fill.bin
sync
umount /mnt/ext
tune2fs -E force_fsck /dev/vdb
e2fsck -f -y /dev/vdb
mkfs.xfs -f -L LPICXFSMAINT /dev/vdc
mount /dev/vdc /mnt/xfs
printf 'sample\n' > /mnt/xfs/sample.txt
xfs_info /mnt/xfs > /root/lpic-fs/xfs-info.txt
umount /mnt/xfs
xfs_repair -n /dev/vdc > /root/lpic-fs/xfs-repair.txt 2>&1
xfs_db -r -c 'sb 0' -c 'p magicnum' /dev/vdc > /root/lpic-fs/xfs-db.txt 2>&1
