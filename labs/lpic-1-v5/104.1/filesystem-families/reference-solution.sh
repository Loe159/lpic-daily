#!/usr/bin/env bash
set -euo pipefail
mkfs.xfs -f -L LPICXFS /dev/vdb
parted -s /dev/vdc mklabel gpt
parted -s /dev/vdc mkpart primary fat32 1MiB 384MiB
parted -s /dev/vdc mkpart primary 384MiB 100%
partprobe /dev/vdc
mkfs.vfat -n LPICVFAT /dev/vdc1
mkfs.exfat -L LPICEXFAT /dev/vdc2
mkfs.btrfs -f -L LPICBTRFS /dev/vdd /dev/vde
mkdir -p /mnt/btrfs
mount -o compress=zstd /dev/vdd /mnt/btrfs
btrfs subvolume create /mnt/btrfs/data
