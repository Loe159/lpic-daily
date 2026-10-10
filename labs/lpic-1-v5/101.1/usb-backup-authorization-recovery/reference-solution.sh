#!/usr/bin/env bash
set -euo pipefail
usbdev="$(cat /var/lib/lpic-usb-rescue/device-path)"
test -f "$usbdev/authorized"
printf '1\n' > "$usbdev/authorized"
udevadm trigger --action=change --subsystem-match=block
udevadm settle
test -b /dev/sda
mount /srv/lpic-usb-backup
test "$(cat /srv/lpic-usb-backup/manifest.txt)" = 'usb-backup=preserve-78'
