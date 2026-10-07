#!/usr/bin/env bash
set -euo pipefail
parted -s /dev/vdb mklabel gpt
parted -s /dev/vdb mkpart primary ext4 1MiB 300MiB
parted -s /dev/vdb mkpart primary ext4 300MiB 650MiB
parted -s /dev/vdb mkpart primary ext4 650MiB 100%
partprobe /dev/vdb
mkfs.ext4 -F -L LPICVAR /dev/vdb1
mkfs.ext4 -F -L LPICHOME /dev/vdb2
mkfs.ext4 -F -L LPICBOOT /dev/vdb3
mkdir -p /mnt/layout/{var,home,boot,lvm-data}
mount /dev/vdb1 /mnt/layout/var
mount /dev/vdb2 /mnt/layout/home
mount /dev/vdb3 /mnt/layout/boot
mkswap -L LPICSWAP /dev/vdc
swapon /dev/vdc
parted -s /dev/vdd mklabel gpt
parted -s /dev/vdd mkpart ESP fat32 1MiB 100%
parted -s /dev/vdd set 1 esp on
partprobe /dev/vdd
mkfs.vfat -n LPICESP /dev/vdd1
pvcreate -ff -y /dev/vde
vgcreate lpicvg /dev/vde
lvcreate -L 256M -n data lpicvg
mkfs.ext4 -F -L LPICLVM /dev/lpicvg/data
mount /dev/lpicvg/data /mnt/layout/lvm-data
