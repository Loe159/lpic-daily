#!/usr/bin/env bash
set -euo pipefail
for tool in lspci lsblk blkid mkfs.ext4 mount umount udevadm; do command -v "$tool" >/dev/null; done
test -b /dev/vda
test -b /dev/sda
install -d -m 0755 /var/lib/lpic-onboard-sata /srv/lpic-onboard-archive /mnt/lpic-onboard-stage
state=/var/lib/lpic-onboard-sata

# Resolve the AHCI PCI function which actually owns this guest-only SATA disk.
pci=''
location="$(readlink -f /sys/class/block/sda/device)"
while [ "$location" != / ]; do
  if [ -L "$location/driver" ] && [ "$(basename "$(readlink -f "$location/driver")")" = ahci ]; then
    pci="$location"
    break
  fi
  location="$(dirname "$location")"
done
test -n "$pci"
bdf="$(basename "$pci")"
test -d "/sys/bus/pci/devices/$bdf"
test "$(lsblk -dn -o TYPE /dev/sda)" = disk
printf '%s\n' "$bdf" > "$state/pci-bdf"
printf '%s\n' "$bdf" > "$state/pci-original"

mkfs.ext4 -F -q -L LPICONBOARD /dev/sda
mount /dev/sda /mnt/lpic-onboard-stage
printf 'onboard-sata=retain-81\n' > /mnt/lpic-onboard-stage/manifest.txt
umount /mnt/lpic-onboard-stage

# This is a real PCI-driver disconnect, not a fake /dev symlink deletion.
printf '%s\n' "$bdf" > /sys/bus/pci/drivers/ahci/unbind
udevadm settle
for attempt in $(seq 1 50); do
  test ! -b /dev/sda && break
  sleep 0.2
done
test ! -b /dev/sda
test ! -L "/sys/bus/pci/devices/$bdf/driver"
test -b /dev/vda
