#!/usr/bin/env bash
set -euo pipefail
mkfs.xfs -f -L LPICXFS /dev/vdb
mkfs.vfat -n LPICVFAT /dev/vdc
mkfs.exfat -L LPICEXFAT /dev/vdd
mkfs.btrfs -f -L LPICBTRFS /dev/vde /dev/vdf
mkdir -p /mnt/btrfs
mount -o compress=zstd /dev/vde /mnt/btrfs
btrfs subvolume create /mnt/btrfs/data
