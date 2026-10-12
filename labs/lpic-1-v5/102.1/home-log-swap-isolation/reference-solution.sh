#!/usr/bin/env bash
set -euo pipefail
parted --script /dev/vdb mklabel gpt
parted --script /dev/vdb mkpart users ext4 1MiB 481MiB
parted --script /dev/vdb mkpart logs ext4 481MiB 737MiB
parted --script /dev/vdb mkpart swap linux-swap 737MiB 100%
partprobe /dev/vdb
for _ in $(seq 1 100); do test -b /dev/vdb3 && break; sleep 0.1; done
test -b /dev/vdb1 && test -b /dev/vdb2 && test -b /dev/vdb3
mkfs.ext4 -q -F /dev/vdb1
mkfs.ext4 -q -F /dev/vdb2
mkswap /dev/vdb3
mkdir -p /mnt/lpic-migrate-home /mnt/lpic-migrate-log
mount /dev/vdb1 /mnt/lpic-migrate-home
mount /dev/vdb2 /mnt/lpic-migrate-log
cp -a /home/. /mnt/lpic-migrate-home/
cp -a /var/log/lpic/. /mnt/lpic-migrate-log/
umount /mnt/lpic-migrate-home /mnt/lpic-migrate-log
mount /dev/vdb1 /home
mount /dev/vdb2 /var/log/lpic
swapon /dev/vdb3
for row in '1 /home ext4 defaults 0 2' '2 /var/log/lpic ext4 defaults 0 2' '3 none swap sw 0 0'; do
  set -- $row
  uuid=$(blkid -s UUID -o value "/dev/vdb$1")
  printf 'UUID=%s %s %s %s %s %s\n' "$uuid" "$2" "$3" "$4" "$5" "$6" >> /etc/fstab
done
