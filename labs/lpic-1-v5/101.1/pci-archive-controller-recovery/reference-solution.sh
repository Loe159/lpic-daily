#!/usr/bin/env bash
set -euo pipefail
found=0
for pci in /sys/bus/pci/devices/*; do
  test -e "$pci/vendor" && test -e "$pci/device" || continue
  test "$(cat "$pci/vendor")" = 0x1af4 || continue
  case "$(cat "$pci/device")" in
    0x1001|0x1042) ;;
    *) continue ;;
  esac
  test -L "$pci/driver" && continue
  echo "$(basename "$pci")" > /sys/bus/pci/drivers/virtio-pci/bind
  found=1
done
test "$found" -eq 1
udevadm settle
test -b /dev/vdb
mount /srv/lpic-vault
systemctl reset-failed lpic-vault-reader.service || true
systemctl start lpic-vault-reader.service
