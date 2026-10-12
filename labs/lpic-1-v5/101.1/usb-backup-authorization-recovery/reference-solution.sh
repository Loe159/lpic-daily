#!/usr/bin/env bash
set -euo pipefail
usbdev="$(cat /var/lib/lpic-usb-rescue/device-path)"
test -f "$usbdev/authorized"
printf '1\n' > "$usbdev/authorized"
for attempt in $(seq 1 100); do
  test -b /dev/sda && break
  sleep 0.2
done
test -b /dev/sda
udevadm settle
mount /srv/lpic-usb-backup
test "$(cat /srv/lpic-usb-backup/manifest.txt)" = 'usb-backup=preserve-78'
