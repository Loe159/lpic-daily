#!/usr/bin/env bash
set -euo pipefail

parted --script /dev/vdb mklabel gpt
parted --script /dev/vdb mkpart data ext4 1MiB 385MiB
parted --script /dev/vdb mkpart swap linux-swap 385MiB 100%
partprobe /dev/vdb
if command -v udevadm >/dev/null 2>&1; then
  udevadm settle --timeout=10
fi
for device in /dev/vdb1 /dev/vdb2; do
  for _ in $(seq 1 100); do
    [[ -b "$device" ]] && break
    sleep 0.1
  done
  [[ -b "$device" ]]
done

mkfs.ext4 -F /dev/vdb1
mkswap /dev/vdb2
