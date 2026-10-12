#!/usr/bin/env bash
set -euo pipefail
# Report the failing setup line from disposable guests without exposing secrets.
trap 'rc=$?; printf "Lab setup failed at line %s (exit=%s)\n" "$LINENO" "$rc" >&2' ERR
for tool in lsusb lspci udevadm lsblk blkid mkfs.ext4; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    printf "Missing guest command: %s (rebuild the Fedora LPIC VM image)\n" "$tool" >&2
    exit 1
  fi
done
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
for attempt in $(seq 1 50); do
  test ! -b /dev/sda && break
  sleep 0.2
done
test ! -b /dev/sda
! mountpoint -q /srv/lpic-usb-backup
