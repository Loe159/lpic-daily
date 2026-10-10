#!/usr/bin/env bash
set -euo pipefail
for tool in lsusb lspci udevadm lsblk blkid mkfs.ext4; do command -v "$tool" >/dev/null; done
test -b /dev/sda
test "$(lsblk -dn -o TRAN /dev/sda)" = usb

usbdev="$(readlink -f /sys/class/block/sda/device)"
while [[ "$usbdev" != / ]]; do
  if [[ -f "$usbdev/authorized" && -f "$usbdev/idVendor" && -f "$usbdev/idProduct" ]]; then
    break
  fi
  usbdev="$(dirname "$usbdev")"
done
test "$usbdev" != /
[[ "$usbdev" == *"/pci"* ]]
test "$(cat "$usbdev/authorized")" = 1
install -d -m 0755 /var/lib/lpic-usb-rescue /srv/lpic-usb-backup /mnt/lpic-usb-stage
printf '%s\n' "$usbdev" > /var/lib/lpic-usb-rescue/device-path

mkfs.ext4 -F -q -L LPICUSB78 /dev/sda
mount /dev/sda /mnt/lpic-usb-stage
printf 'usb-backup=preserve-78\n' > /mnt/lpic-usb-stage/manifest.txt
umount /mnt/lpic-usb-stage
printf 'LABEL=LPICUSB78 /srv/lpic-usb-backup ext4 defaults,nofail 0 2\n' >> /etc/fstab

printf '0\n' > "$usbdev/authorized"
udevadm settle
test "$(cat "$usbdev/authorized")" = 0
! mountpoint -q /srv/lpic-usb-backup
