#!/usr/bin/env bash
set -euo pipefail
bdf="$(cat /var/lib/lpic-onboard-sata/pci-bdf)"
test -d "/sys/bus/pci/devices/$bdf"
printf '%s\n' "$bdf" > /sys/bus/pci/drivers/ahci/bind
for attempt in $(seq 1 100); do
  test -b /dev/sda && break
  sleep 0.2
done
test -b /dev/sda
udevadm settle
mount -o ro -L LPICONBOARD /srv/lpic-onboard-archive
test "$(cat /srv/lpic-onboard-archive/manifest.txt)" = 'onboard-sata=retain-81'
